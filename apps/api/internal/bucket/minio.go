package bucket

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioBucketManager struct {
	client *minio.Client
}

func NewMinioBucketManager(endpoint, accessKey, secretKey string, useSSL bool) (*MinioBucketManager, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &MinioBucketManager{client: client}, nil
}

func (m *MinioBucketManager) Ping(ctx context.Context) error {
	_, err := m.client.ListBuckets(ctx)
	return err
}

func (m *MinioBucketManager) Client() *minio.Client {
	return m.client
}

func (m *MinioBucketManager) GenerateUploadURL(
	ctx context.Context,
	bucketName string,
	objectPath string,
	expiry time.Duration,
) (*url.URL, error) {
	return m.client.PresignedPutObject(ctx, bucketName, objectPath, expiry)
}

func (m *MinioBucketManager) GenerateDownloadURL(
	ctx context.Context,
	bucketName string,
	objectPath string,
	expiry time.Duration,
) (*url.URL, error) {
	return m.client.PresignedGetObject(ctx, bucketName, objectPath, expiry, nil)
}

func (m *MinioBucketManager) GetObject(
	ctx context.Context,
	bucketName string,
	objectPath string,
	objOptions minio.GetObjectOptions,
) (*minio.Object, error) {
	return m.client.GetObject(ctx, bucketName, objectPath, objOptions)
}

func (m *MinioBucketManager) PutObject(
	ctx context.Context,
	bucketName string,
	objectPath string,
	reader io.Reader,
	objectSize int64,
	opts minio.PutObjectOptions,
) (minio.UploadInfo, error) {
	return m.client.PutObject(ctx, bucketName, objectPath, reader, objectSize, opts)
}

func (m *MinioBucketManager) RemoveObject(
	ctx context.Context,
	bucketName string,
	objectPath string,
) error {
	return m.client.RemoveObject(ctx, bucketName, objectPath, minio.RemoveObjectOptions{})
}

func (m *MinioBucketManager) StatObject(
	ctx context.Context,
	bucketName string,
	objectPath string,
	opts minio.StatObjectOptions,
) (minio.ObjectInfo, error) {
	return m.client.StatObject(ctx, bucketName, objectPath, opts)
}
