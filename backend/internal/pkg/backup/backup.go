// backup.go
// 该文件提供系统管理模块复用的数据备份基础能力，负责执行 PostgreSQL 全量导出、可选压缩、
// 加密后上传到对象存储，并提供按保留份数清理旧备份文件的公共方法。它解决的是“备份文件
// 怎么生成、怎么加密、怎么落 MinIO、怎么做对象保留策略”的通用问题，不承担备份记录入库。

package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/config"
	cryptopkg "github.com/lenschain/backend/internal/pkg/crypto"
	"github.com/lenschain/backend/internal/pkg/storage"
)

// Options 定义一次备份执行所需的参数。
// 系统模块在触发手动或自动备份时，应把数据库连接信息和对象路径规划整理到这里。
type Options struct {
	Database       config.DatabaseConfig
	ObjectPrefix   string
	FileName       string
	TempDir        string
	Encrypt        bool
	Compress       bool
	CommandTimeout time.Duration
}

// Result 描述一次备份执行后的产物信息。
// 上层可以把这里的信息持久化到 backup_records，或用于接口响应展示。
type Result struct {
	ObjectName    string
	FileSize      int64
	ContentType   string
	Duration      time.Duration
	StartedAt     time.Time
	FinishedAt    time.Time
	Compressed    bool
	Encrypted     bool
	LocalTempPath string
}

// RunPostgresBackup 执行 PostgreSQL 全量备份并上传到对象存储。
// 它使用 pg_dump 导出数据库，按配置决定是否压缩和加密，最后统一上传到 MinIO。
func RunPostgresBackup(ctx context.Context, opts Options) (*Result, error) {
	startedAt := time.Now().UTC()
	timeout := opts.CommandTimeout
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	commandCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	fileName := strings.TrimSpace(opts.FileName)
	if fileName == "" {
		fileName = buildDefaultFileName(startedAt)
	}
	objectName := filepath.ToSlash(strings.Trim(filepath.ToSlash(opts.ObjectPrefix), "/"))
	if objectName != "" {
		objectName += "/"
	}
	objectName += fileName

	dumpBytes, err := dumpDatabase(commandCtx, opts.Database)
	if err != nil {
		return nil, err
	}

	contentType := "application/sql"
	payload := dumpBytes
	if opts.Compress {
		payload, err = gzipBytes(dumpBytes)
		if err != nil {
			return nil, err
		}
		contentType = "application/gzip"
		if !strings.HasSuffix(objectName, ".gz") {
			objectName += ".gz"
		}
	}

	if opts.Encrypt {
		payload, err = cryptopkg.AESEncryptBytes(payload)
		if err != nil {
			return nil, fmt.Errorf("加密备份文件失败: %w", err)
		}
		contentType = "application/octet-stream"
		if !strings.HasSuffix(objectName, ".enc") {
			objectName += ".enc"
		}
	}

	if _, err := storage.UploadFile(ctx, objectName, bytes.NewReader(payload), int64(len(payload)), contentType); err != nil {
		// 上传失败时把当前产物落到本地临时文件，便于管理员重试或人工取回，和文档里的失败保底策略保持一致。
		localPath, saveErr := persistFailedPayload(opts.TempDir, objectName, payload)
		if saveErr != nil {
			return nil, fmt.Errorf("上传备份文件失败: %w；保存本地临时文件失败: %v", err, saveErr)
		}
		return &Result{
			ObjectName:    objectName,
			FileSize:      int64(len(payload)),
			ContentType:   contentType,
			Duration:      time.Since(startedAt),
			StartedAt:     startedAt,
			FinishedAt:    time.Now().UTC(),
			Compressed:    opts.Compress,
			Encrypted:     opts.Encrypt,
			LocalTempPath: localPath,
		}, fmt.Errorf("上传备份文件失败: %w", err)
	}

	finishedAt := time.Now().UTC()
	return &Result{
		ObjectName:  objectName,
		FileSize:    int64(len(payload)),
		ContentType: contentType,
		Duration:    finishedAt.Sub(startedAt),
		StartedAt:   startedAt,
		FinishedAt:  finishedAt,
		Compressed:  opts.Compress,
		Encrypted:   opts.Encrypt,
	}, nil
}

// CleanupRetention 按保留份数清理旧备份文件。
// 当对象数量超过 keep 时，会删除时间最早的多余文件，保留最新的 keep 份。
func CleanupRetention(ctx context.Context, prefix string, keep int) ([]string, error) {
	if keep <= 0 {
		return nil, nil
	}

	objects, err := storage.ListObjects(ctx, prefix, true)
	if err != nil {
		return nil, err
	}
	if len(objects) <= keep {
		return nil, nil
	}

	var deleted []string
	for _, object := range objects[keep:] {
		if err := storage.DeleteFile(ctx, object.ObjectName); err != nil {
			return deleted, fmt.Errorf("删除过期备份失败: %w", err)
		}
		deleted = append(deleted, object.ObjectName)
	}
	return deleted, nil
}

// dumpDatabase 调用 pg_dump 导出数据库完整 SQL 内容。
func dumpDatabase(ctx context.Context, db config.DatabaseConfig) ([]byte, error) {
	args := []string{
		"--format=plain",
		"--no-owner",
		"--no-privileges",
		"--host", db.Host,
		"--port", fmt.Sprintf("%d", db.Port),
		"--username", db.User,
		db.DBName,
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append([]string{}, "PGPASSWORD="+db.Password)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("执行 pg_dump 失败: %s", message)
	}
	return stdout.Bytes(), nil
}

// gzipBytes 对备份内容做 gzip 压缩，降低对象存储占用和下载体积。
func gzipBytes(payload []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(payload); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("压缩备份文件失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭压缩流失败: %w", err)
	}
	return io.ReadAll(&buf)
}

// buildDefaultFileName 生成默认备份文件名，方便手动和自动任务保持一致命名风格。
func buildDefaultFileName(now time.Time) string {
	return fmt.Sprintf("lenschain_%s.sql", now.Format("20060102_150405"))
}

// persistFailedPayload 在对象存储上传失败时把备份内容保存在本地临时目录。
// 这样上层即使收到失败状态，也仍然能获取可重试的文件路径，避免导出结果彻底丢失。
func persistFailedPayload(tempDir string, objectName string, payload []byte) (string, error) {
	pattern := strings.ReplaceAll(filepath.Base(objectName), ".", "_") + "_*.bak"
	file, err := os.CreateTemp(tempDir, pattern)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := file.Write(payload); err != nil {
		return "", err
	}
	return file.Name(), nil
}
