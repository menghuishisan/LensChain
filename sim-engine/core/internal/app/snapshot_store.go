package app

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// SnapshotPayload 是快照持久化时保存的完整载荷。
type SnapshotPayload struct {
	SessionID string          `json:"session_id"`
	Tick      int64           `json:"tick"`
	Scenes    []SnapshotScene `json:"scenes"`
}

// SnapshotScene 是单个场景的快照内容。
// SceneStateJSON 为场景算法内部状态；RenderEnvelopeJSON 为最近一帧 RenderEnvelope；
// SharedStateJSON 为联动共享状态快照（联动组场景才会填充）。
type SnapshotScene struct {
	SceneCode          string `json:"scene_code"`
	SceneStateJSON     []byte `json:"scene_state_json"`
	RenderEnvelopeJSON []byte `json:"render_envelope_json"`
	SharedStateJSON    []byte `json:"shared_state_json,omitempty"`
}

// SnapshotStore 定义快照持久化存储接口。
type SnapshotStore interface {
	Save(snapshotID string, payload SnapshotPayload) (string, error)
	Load(snapshotID string) (SnapshotPayload, error)
}

// ObjectStorageConfig 是 SimEngine 快照对象存储配置。
type ObjectStorageConfig struct {
	Endpoint        string
	AccessKey       string
	SecretKey       string
	UseSSL          bool
	Bucket          string
	Region          string
	ObjectPrefix    string
	EncryptionKey   string
	PresignDuration time.Duration
}

// MinIOSnapshotStore 是符合文档要求的 MinIO/S3 快照存储实现。
type MinIOSnapshotStore struct {
	client          *minio.Client
	bucket          string
	objectPrefix    string
	encryptionKey   []byte
	presignDuration time.Duration
}

// NewMinIOSnapshotStore 创建 MinIO/S3 快照存储。
func NewMinIOSnapshotStore(ctx context.Context, cfg ObjectStorageConfig) (*MinIOSnapshotStore, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, errors.New("snapshot storage endpoint is required")
	}
	if strings.TrimSpace(cfg.AccessKey) == "" {
		return nil, errors.New("snapshot storage access key is required")
	}
	if strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, errors.New("snapshot storage secret key is required")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, errors.New("snapshot storage bucket is required")
	}
	keyBytes, err := decodeSnapshotEncryptionKey(cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}
	if cfg.PresignDuration <= 0 {
		cfg.PresignDuration = time.Hour
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, err
		}
	}

	return &MinIOSnapshotStore{
		client:          client,
		bucket:          cfg.Bucket,
		objectPrefix:    strings.Trim(strings.TrimSpace(cfg.ObjectPrefix), "/"),
		encryptionKey:   keyBytes,
		presignDuration: cfg.PresignDuration,
	}, nil
}

// decodeSnapshotEncryptionKey 解析配置中的快照加密密钥，支持以下三种输入格式：
//   - 32 字节 ASCII 原文（len == 32）
//   - base64 标准/URL-safe 编码（解码后必须是 32 字节）
//   - hex 编码（解码后必须是 32 字节，即 64 hex 字符）
//
// 推荐使用 base64 或 hex 编码的随机字节（如
// `[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }) -as [byte[]])`），
// 安全性高于 ASCII 字符串。
func decodeSnapshotEncryptionKey(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("snapshot encryption key is required")
	}
	if len(trimmed) == 32 {
		return []byte(trimmed), nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.URLEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := hex.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	return nil, errors.New("snapshot encryption key 必须是 32 字节（ASCII/base64/hex 任一格式）")
}

// snapshotIOTimeout 是单次对象存储 I/O 的最大允许时间。
//
// 必须有限：CreateSnapshot 在前端 pause/auto-snapshot 等关键路径上同步调用 Save，
// MinIO 一次网络抖动若没有超时就能让上层调用永久阻塞——从而把 sim-engine 的全局
// engine.mu / runtime.opMu 一起拖死，前端 play/pause/step 完全失灵。
//
// 5s 足以容忍正常网络抖动；超时后调用方（CreateSnapshot 等）会得到明确错误，可走
// 重试或告警分支，不再有"沉默死锁"。
const snapshotIOTimeout = 5 * time.Second

// Save 将加密后的快照写入对象存储并返回预签名对象地址。
func (s *MinIOSnapshotStore) Save(snapshotID string, payload SnapshotPayload) (string, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encryptedJSON, err := encryptSnapshotPayload(s.encryptionKey, payloadJSON)
	if err != nil {
		return "", err
	}

	objectName := s.objectName(snapshotID)
	putCtx, putCancel := context.WithTimeout(context.Background(), snapshotIOTimeout)
	_, err = s.client.PutObject(
		putCtx,
		s.bucket,
		objectName,
		bytes.NewReader(encryptedJSON),
		int64(len(encryptedJSON)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	)
	putCancel()
	if err != nil {
		return "", err
	}

	presignCtx, presignCancel := context.WithTimeout(context.Background(), snapshotIOTimeout)
	signedURL, err := s.client.PresignedGetObject(
		presignCtx,
		s.bucket,
		objectName,
		s.presignDuration,
		url.Values{},
	)
	presignCancel()
	if err != nil {
		return "", err
	}
	return signedURL.String(), nil
}

// Load 从对象存储读取并解密快照。
func (s *MinIOSnapshotStore) Load(snapshotID string) (SnapshotPayload, error) {
	getCtx, getCancel := context.WithTimeout(context.Background(), snapshotIOTimeout)
	defer getCancel()
	object, err := s.client.GetObject(getCtx, s.bucket, s.objectName(snapshotID), minio.GetObjectOptions{})
	if err != nil {
		return SnapshotPayload{}, err
	}
	defer object.Close()

	encryptedJSON, err := io.ReadAll(object)
	if err != nil {
		return SnapshotPayload{}, err
	}
	payloadJSON, err := decryptSnapshotPayload(s.encryptionKey, encryptedJSON)
	if err != nil {
		return SnapshotPayload{}, err
	}

	var payload SnapshotPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return SnapshotPayload{}, err
	}
	return payload, nil
}

// objectName 返回快照对象在桶中的统一路径。
func (s *MinIOSnapshotStore) objectName(snapshotID string) string {
	if s.objectPrefix == "" {
		return path.Join("sim-engine", "snapshots", snapshotID+".bin")
	}
	return path.Join(s.objectPrefix, snapshotID+".bin")
}

// encryptSnapshotPayload 将快照 JSON 加密为 nonce+ciphertext。
func encryptSnapshotPayload(key []byte, payloadJSON []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, payloadJSON, nil)
	return append(nonce, ciphertext...), nil
}

// decryptSnapshotPayload 解密 nonce+ciphertext 格式的快照内容。
func decryptSnapshotPayload(key []byte, encryptedJSON []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(encryptedJSON) < gcm.NonceSize() {
		return nil, errors.New("encrypted snapshot payload is invalid")
	}
	nonce := encryptedJSON[:gcm.NonceSize()]
	ciphertext := encryptedJSON[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// ValidateObjectStorageConfig 校验对象存储配置完整性。
// 配置加载统一在 internal/config 中完成，本函数仅做必填项校验。
func ValidateObjectStorageConfig(cfg ObjectStorageConfig) error {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return fmt.Errorf("object_storage.endpoint 不能为空")
	}
	if strings.TrimSpace(cfg.AccessKey) == "" {
		return fmt.Errorf("object_storage.access_key 不能为空")
	}
	if strings.TrimSpace(cfg.SecretKey) == "" {
		return fmt.Errorf("object_storage.secret_key 不能为空")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return fmt.Errorf("object_storage.bucket 不能为空")
	}
	if strings.TrimSpace(cfg.EncryptionKey) == "" {
		return fmt.Errorf("object_storage.encryption_key 不能为空")
	}
	return nil
}
