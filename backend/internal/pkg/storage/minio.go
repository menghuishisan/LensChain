// minio.go
// 该文件封装平台对象存储访问能力，负责初始化 MinIO/S3 兼容客户端，并提供上传、下载、
// 删除、存在性检查和预签名 URL 等通用方法。课程附件、实验快照、成绩单 PDF、备份文件
// 等需要落对象存储的场景都应复用这里。

package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sort"
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

var errMinIONotInitialized = errors.New("MinIO客户端未初始化")

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
	if client == nil || bucketName == "" {
		return "", errMinIONotInitialized
	}
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
	if client == nil || bucketName == "" {
		return "", errMinIONotInitialized
	}
	reqParams := make(url.Values)
	presignedURL, err := client.PresignedGetObject(ctx, bucketName, objectName, expires, reqParams)
	if err != nil {
		return "", fmt.Errorf("生成预签名URL失败: %w", err)
	}
	return presignedURL.String(), nil
}

// DownloadFile 下载文件
func DownloadFile(ctx context.Context, objectName string) (io.ReadCloser, error) {
	if client == nil || bucketName == "" {
		return nil, errMinIONotInitialized
	}
	obj, err := client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %w", err)
	}
	return obj, nil
}

// DeleteFile 删除文件
func DeleteFile(ctx context.Context, objectName string) error {
	if client == nil || bucketName == "" {
		return errMinIONotInitialized
	}
	err := client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

// FileExists 检查文件是否存在
func FileExists(ctx context.Context, objectName string) (bool, error) {
	if client == nil || bucketName == "" {
		return false, errMinIONotInitialized
	}
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
	if client == nil || bucketName == "" {
		return "", errMinIONotInitialized
	}
	presignedURL, err := client.PresignedPutObject(ctx, bucketName, objectName, expires)
	if err != nil {
		return "", fmt.Errorf("生成预签名上传URL失败: %w", err)
	}
	return presignedURL.String(), nil
}

// ObjectInfo 描述对象存储中的文件元信息。
// 该结构用于备份清理、运维审计和下载前展示等需要查看对象摘要信息的场景。
type ObjectInfo struct {
	ObjectName string
	Size       int64
	LastModified time.Time
	ETag       string
}

// ListObjects 列出指定前缀下的对象。
// 备份清理等需要按目录批量遍历对象时，应通过这里统一读取，而不是在业务层直接操作客户端。
func ListObjects(ctx context.Context, prefix string, recursive bool) ([]ObjectInfo, error) {
	if client == nil || bucketName == "" {
		return nil, errMinIONotInitialized
	}

	var objects []ObjectInfo
	for object := range client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	}) {
		if object.Err != nil {
			return nil, fmt.Errorf("列出对象失败: %w", object.Err)
		}
		objects = append(objects, ObjectInfo{
			ObjectName:   object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
			ETag:         object.ETag,
		})
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].LastModified.After(objects[j].LastModified)
	})
	return objects, nil
}
