// instance_service_storage.go
// 模块04 — 实验环境：快照与终端审计对象存储辅助逻辑
// 统一复用 internal/pkg/storage，避免在业务层重复实现对象存储读写

package experiment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/model/entity"
	cryptopkg "github.com/lenschain/backend/internal/pkg/crypto"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/storage"
	"go.uber.org/zap"
)

const maxCommandOutputBytes = 4 * 1024

// snapshotArchivePayload 表示写入对象存储的完整快照归档内容。
type snapshotArchivePayload struct {
	SnapshotID      int64           `json:"snapshot_id,string"`
	InstanceID      int64           `json:"instance_id,string"`
	SnapshotType    int             `json:"snapshot_type"`
	ContainerStates json.RawMessage `json:"container_states,omitempty"`
	SimEngineState  json.RawMessage `json:"sim_engine_state,omitempty"`
	Description     *string         `json:"description,omitempty"`
	CreatedAt       string          `json:"created_at"`
}

// uploadSnapshotArchive 上传完整快照归档，并允许写入带归档内容的容器运行态。
// 快照归档在写入对象存储前会执行 AES 加密，满足模块04对快照存储加密的要求。
func (s *instanceService) uploadSnapshotArchive(ctx context.Context, snapshot *entity.InstanceSnapshot, containerStates json.RawMessage, simEngineState json.RawMessage) (*string, *int64, error) {
	if snapshot == nil {
		return nil, nil, fmt.Errorf("快照不能为空")
	}
	if storage.GetClient() == nil {
		return nil, nil, fmt.Errorf("对象存储未初始化")
	}

	payload := snapshotArchivePayload{
		SnapshotID:      snapshot.ID,
		InstanceID:      snapshot.InstanceID,
		SnapshotType:    int(snapshot.SnapshotType),
		ContainerStates: containerStates,
		SimEngineState:  simEngineState,
		Description:     snapshot.Description,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("序列化快照归档失败: %w", err)
	}
	encryptedBody, err := cryptopkg.AESEncrypt(string(body))
	if err != nil {
		return nil, nil, fmt.Errorf("加密快照归档失败: %w", err)
	}

	objectKey := fmt.Sprintf("experiment/snapshots/%d/%d.enc", snapshot.InstanceID, snapshot.ID)
	encryptedBytes := []byte(encryptedBody)
	if _, err := storage.UploadFile(ctx, objectKey, bytes.NewReader(encryptedBytes), int64(len(encryptedBytes)), "application/octet-stream"); err != nil {
		return nil, nil, fmt.Errorf("上传快照归档失败: %w", err)
	}

	size := int64(len(encryptedBytes))
	return &objectKey, &size, nil
}

// loadSnapshotArchive 下载并解密实例快照归档。
func (s *instanceService) loadSnapshotArchive(ctx context.Context, snapshot *entity.InstanceSnapshot) (*snapshotArchivePayload, error) {
	if snapshot == nil || snapshot.SnapshotDataURL == "" {
		return nil, fmt.Errorf("快照归档地址为空")
	}
	if storage.GetClient() == nil {
		return nil, fmt.Errorf("对象存储未初始化")
	}
	reader, err := storage.DownloadFile(ctx, snapshot.SnapshotDataURL)
	if err != nil {
		return nil, fmt.Errorf("下载快照归档失败: %w", err)
	}
	defer reader.Close()

	encryptedBody, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取快照归档失败: %w", err)
	}
	decryptedBody, err := cryptopkg.AESDecrypt(string(encryptedBody))
	if err != nil {
		return nil, fmt.Errorf("解密快照归档失败: %w", err)
	}

	var payload snapshotArchivePayload
	if err := json.Unmarshal([]byte(decryptedBody), &payload); err != nil {
		return nil, fmt.Errorf("解析快照归档失败: %w", err)
	}
	return &payload, nil
}

// deleteSnapshotArchive 删除快照归档对象。
func (s *instanceService) deleteSnapshotArchive(ctx context.Context, snapshot *entity.InstanceSnapshot) {
	if snapshot == nil || snapshot.SnapshotDataURL == "" {
		return
	}
	if storage.GetClient() == nil {
		return
	}
	if err := storage.DeleteFile(ctx, snapshot.SnapshotDataURL); err != nil {
		logger.L.Warn("删除快照归档失败",
			zap.Int64("instance_id", snapshot.InstanceID),
			zap.Int64("snapshot_id", snapshot.ID),
			zap.String("object_key", snapshot.SnapshotDataURL),
			zap.Error(err),
		)
	}
}

// buildTerminalCommandAudit 构建终端命令审计的截断输出和附加明细。
func (s *instanceService) buildTerminalCommandAudit(ctx context.Context, instanceID int64, result *ExecResult) (*string, map[string]interface{}) {
	if result == nil {
		return nil, map[string]interface{}{}
	}

	fullOutput := strings.TrimSpace(strings.Join([]string{result.Stdout, result.Stderr}, "\n"))
	if fullOutput == "" {
		return nil, map[string]interface{}{}
	}

	commandOutput := truncateUTF8(fullOutput, maxCommandOutputBytes)
	detail := map[string]interface{}{}
	if len(commandOutput) > 0 {
		detail["truncated"] = len(commandOutput) < len(fullOutput)
	}

	if storage.GetClient() != nil {
		objectKey := fmt.Sprintf("experiment/terminal-logs/%d/%d.log", instanceID, time.Now().UTC().UnixNano())
		if _, err := storage.UploadFile(ctx, objectKey, strings.NewReader(fullOutput), int64(len([]byte(fullOutput))), "text/plain; charset=utf-8"); err == nil {
			detail["output_object_key"] = objectKey
		} else {
			logger.L.Warn("上传终端完整输出失败",
				zap.Int64("instance_id", instanceID),
				zap.String("object_key", objectKey),
				zap.Error(err),
			)
		}
	}

	return &commandOutput, detail
}

// truncateUTF8 按字节上限截断字符串，并保证结果仍是合法 UTF-8。
func truncateUTF8(input string, maxBytes int) string {
	if maxBytes <= 0 || len(input) <= maxBytes {
		return input
	}

	runes := []rune(input)
	for len(runes) > 0 {
		candidate := string(runes)
		if len([]byte(candidate)) <= maxBytes {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}
	return ""
}
