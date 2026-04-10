// minio.go
// MinIO 对象存储客户端封装
// 用于文件上传（头像、课件、附件）、下载、成绩单PDF存储、备份文件存储
// 遵循 S3 兼容协议

package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/logger"
)

// 全局 MinIO 客户端
var client *minio.Client
var bucketName string

// Init 初始化 MinIO 客户端
func Init(cfg *config.MinIOConfig) error {
	var err error
	client, err = minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return fmt.Errorf("初始化MinIO客户端失败: %w", err)
	}

	bucketName = cfg.Bucket

	// 确保 Bucket 存在
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("检查Bucket失败: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
			Region: cfg.Region,
		}); err != nil {
			return fmt.Errorf("创建Bucket失败: %w", err)
		}
		logger.L.Info("MinIO Bucket已创建", zap.String("bucket", bucketName))
	}

	logger.L.Info("MinIO连接成功",
		zap.String("endpoint", cfg.Endpoint),
		zap.String("bucket", bucketName),
	)

	return nil
}

// GetClient 获取 MinIO 客户端
func GetClient() *minio.Client {
	return client
}

// UploadFile 上传文件
// objectName 为对象存储路径（如 "avatars/user_123.jpg"）
// reader 为文件内容读取器
// contentType 为文件 MIME 类型
func UploadFile(ctx context.Context, objectName string, reader io.Reader, fileSize int64, contentType string) (string, error) {
	_, err := client.PutObject(ctx, bucketName, objectName, reader, fileSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("上传文件失败: %w", err)
	}

	return objectName, nil
}

// GetFileURL 获取文件的预签名下载URL
// expires 为URL有效期
func GetFileURL(ctx context.Context, objectName string, expires time.Duration) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := client.PresignedGetObject(ctx, bucketName, objectName, expires, reqParams)
	if err != nil {
		return "", fmt.Errorf("生成预签名URL失败: %w", err)
	}
	return presignedURL.String(), nil
}

// DownloadFile 下载文件
func DownloadFile(ctx context.Context, objectName string) (io.ReadCloser, error) {
	obj, err := client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %w", err)
	}
	return obj, nil
}

// DeleteFile 删除文件
func DeleteFile(ctx context.Context, objectName string) error {
	err := client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

// FileExists 检查文件是否存在
func FileExists(ctx context.Context, objectName string) (bool, error) {
	_, err := client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetUploadPresignedURL 获取预签名上传URL（前端直传）
func GetUploadPresignedURL(ctx context.Context, objectName string, expires time.Duration) (string, error) {
	presignedURL, err := client.PresignedPutObject(ctx, bucketName, objectName, expires)
	if err != nil {
		return "", fmt.Errorf("生成预签名上传URL失败: %w", err)
	}
	return presignedURL.String(), nil
}
