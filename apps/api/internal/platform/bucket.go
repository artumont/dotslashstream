package platform

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

type BucketClient interface {
	// Ping checks connectivity to the object storage service.
	Ping(ctx context.Context) error

	// Client returns the underlying MinIO client instance.
	Client() *minio.Client

	// GenerateUploadURL generates a presigned URL for uploading an object.
	GenerateUploadURL(
		ctx context.Context,
		bucketName string,
		objectPath string,
		expiry time.Duration,
	) (*url.URL, error)

	// GenerateDownloadURL generates a presigned URL for downloading an object.
	GenerateDownloadURL(
		ctx context.Context,
		bucketName string,
		objectPath string,
		expiry time.Duration,
	) (*url.URL, error)

	// GetObject retrieves an object from the bucket.
	GetObject(
		ctx context.Context,
		bucketName string,
		objectPath string,
		objOptions minio.GetObjectOptions,
	) (*minio.Object, error)

	// PutObject uploads an object directly to the bucket.
	PutObject(
		ctx context.Context,
		bucketName string,
		objectPath string,
		reader io.Reader,
		objectSize int64,
		opts minio.PutObjectOptions,
	) (minio.UploadInfo, error)

	// RemoveObject deletes an object from the bucket.
	RemoveObject(
		ctx context.Context,
		bucketName string,
		objectPath string,
	) error

	// StatObject retrieves metadata about an object without downloading it.
	StatObject(
		ctx context.Context,
		bucketName string,
		objectPath string,
		opts minio.StatObjectOptions,
	) (minio.ObjectInfo, error)
}
