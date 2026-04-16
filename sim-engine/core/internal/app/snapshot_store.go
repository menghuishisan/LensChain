package app

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
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
type SnapshotScene struct {
	SceneCode       string `json:"scene_code"`
	StateJSON       []byte `json:"state_json"`
	RenderStateJSON []byte `json:"render_state_json"`
	SharedStateJSON []byte `json:"shared_state_json"`
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
	if len(cfg.EncryptionKey) != 32 {
		return nil, errors.New("snapshot encryption key must be 32 bytes")
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
		encryptionKey:   []byte(cfg.EncryptionKey),
		presignDuration: cfg.PresignDuration,
	}, nil
}

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
	_, err = s.client.PutObject(
		context.Background(),
		s.bucket,
		objectName,
		bytes.NewReader(encryptedJSON),
		int64(len(encryptedJSON)),
		minio.PutObjectOptions{ContentType: "application/octet-stream"},
	)
	if err != nil {
		return "", err
	}

	signedURL, err := s.client.PresignedGetObject(
		context.Background(),
		s.bucket,
		objectName,
		s.presignDuration,
		url.Values{},
	)
	if err != nil {
		return "", err
	}
	return signedURL.String(), nil
}

// Load 从对象存储读取并解密快照。
func (s *MinIOSnapshotStore) Load(snapshotID string) (SnapshotPayload, error) {
	object, err := s.client.GetObject(context.Background(), s.bucket, s.objectName(snapshotID), minio.GetObjectOptions{})
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

// ParseObjectStorageConfigFromEnv 从环境变量构造对象存储配置。
func ParseObjectStorageConfigFromEnv(getenv func(string) string) (ObjectStorageConfig, error) {
	cfg := ObjectStorageConfig{
		Endpoint:        getenv("SIM_ENGINE_OBJECT_STORAGE_ENDPOINT"),
		AccessKey:       getenv("SIM_ENGINE_OBJECT_STORAGE_ACCESS_KEY"),
		SecretKey:       getenv("SIM_ENGINE_OBJECT_STORAGE_SECRET_KEY"),
		Bucket:          getenv("SIM_ENGINE_OBJECT_STORAGE_BUCKET"),
		Region:          getenv("SIM_ENGINE_OBJECT_STORAGE_REGION"),
		ObjectPrefix:    getenv("SIM_ENGINE_OBJECT_STORAGE_PREFIX"),
		EncryptionKey:   getenv("SIM_ENGINE_SNAPSHOT_ENCRYPTION_KEY"),
		PresignDuration: time.Hour,
	}
	useSSL := strings.TrimSpace(strings.ToLower(getenv("SIM_ENGINE_OBJECT_STORAGE_USE_SSL")))
	cfg.UseSSL = useSSL == "1" || useSSL == "true" || useSSL == "yes"

	if cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" || cfg.EncryptionKey == "" {
		return ObjectStorageConfig{}, fmt.Errorf("object storage env is incomplete")
	}
	return cfg, nil
}
