package bucket

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioBucketDriver struct {
	client *minio.Client
}

func New(endpoint, accessKey, secretKey string, useSSL bool) (*MinioBucketDriver, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &MinioBucketDriver{client: client}, nil
}

func (m *MinioBucketDriver) Ping(ctx context.Context) error {
	_, err := m.client.ListBuckets(ctx)
	return err
}

func (m *MinioBucketDriver) Client() *minio.Client {
	return m.client
}

func (m *MinioBucketDriver) GenerateUploadURL(
	ctx context.Context,
	bucketName string,
	objectPath string,
	expiry time.Duration,
) (*url.URL, error) {
	return m.client.PresignedPutObject(ctx, bucketName, objectPath, expiry)
}

func (m *MinioBucketDriver) GenerateDownloadURL(
	ctx context.Context,
	bucketName string,
	objectPath string,
	expiry time.Duration,
) (*url.URL, error) {
	return m.client.PresignedGetObject(ctx, bucketName, objectPath, expiry, nil)
}

func (m *MinioBucketDriver) GetObject(
	ctx context.Context,
	bucketName string,
	objectPath string,
	objOptions minio.GetObjectOptions,
) (*minio.Object, error) {
	return m.client.GetObject(ctx, bucketName, objectPath, objOptions)
}

func (m *MinioBucketDriver) PutObject(
	ctx context.Context,
	bucketName string,
	objectPath string,
	reader io.Reader,
	objectSize int64,
	opts minio.PutObjectOptions,
) (minio.UploadInfo, error) {
	return m.client.PutObject(ctx, bucketName, objectPath, reader, objectSize, opts)
}

func (m *MinioBucketDriver) RemoveObject(
	ctx context.Context,
	bucketName string,
	objectPath string,
) error {
	return m.client.RemoveObject(ctx, bucketName, objectPath, minio.RemoveObjectOptions{})
}

func (m *MinioBucketDriver) StatObject(
	ctx context.Context,
	bucketName string,
	objectPath string,
	opts minio.StatObjectOptions,
) (minio.ObjectInfo, error) {
	return m.client.StatObject(ctx, bucketName, objectPath, opts)
}
