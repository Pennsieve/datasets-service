package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pennsieve/datasets-service/api/logging"
	"github.com/pennsieve/datasets-service/api/models"
	"log/slog"
	"net/url"
	"time"
)

type S3StoreFactory interface {
	NewSimpleStore(bucket string) S3Store
}

// NewS3StoreFactory takes the request-scoped logger so that S3 failures carry
// the same trace/request context as the rest of the invocation. Pass
// slog.Default() outside of a request.
func NewS3StoreFactory(s3Client *s3.Client, logger *slog.Logger) S3StoreFactory {
	if logger == nil {
		logger = slog.Default()
	}
	return &s3StoreFactory{S3Client: s3Client, Logger: logger}
}

// NewSimpleStore returns a DatasetsStore instance that
// will run statements directly on database
func (d *s3StoreFactory) NewSimpleStore(bucket string) S3Store {
	return &s3Store{S3Client: d.S3Client, S3Bucket: bucket, Logger: d.Logger}
}

type s3StoreFactory struct {
	S3Client *s3.Client
	Logger   *slog.Logger
}

type S3Store interface {
	WriteManifestToS3(ctx context.Context, datasetNodeId string, s3Key string, manifest models.WorkspaceManifest) (*models.WriteManifestOutput, error)
	GetPresignedUrl(ctx context.Context, bucket, key string) (*url.URL, error)
}

type s3Store struct {
	S3Client *s3.Client
	S3Bucket string
	Logger   *slog.Logger
}

func (d *s3Store) WriteManifestToS3(ctx context.Context, datasetNodeId string, s3Key string, manifest models.WorkspaceManifest) (*models.WriteManifestOutput, error) {

	//// Method to create relatively short UUID JSON file name
	//randName, _ := randomFilename16Char()
	//manifestFileName := fmt.Sprintf("%s/%s.json",
	//	strings.Replace(datasetNodeId, "N:dataset:", "", -1), randName)

	uploader := manager.NewUploader(d.S3Client, func(u *manager.Uploader) {
		// Define a strategy that will buffer 25 MiB in memory
		u.BufferProvider = manager.NewBufferedReadSeekerWriteToPool(25 * 1024 * 1024)
	})

	serializedManifests, err := json.MarshalIndent(manifest, "", "    ")
	if err != nil {
		return nil, err
	}

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(d.S3Bucket),
		Key:    aws.String(s3Key),
		Body:   bytes.NewReader(serializedManifests),
	})
	if err != nil {
		return nil, err
	}

	return &models.WriteManifestOutput{
		S3Bucket: d.S3Bucket,
		S3Key:    s3Key,
	}, nil
}

func randomFilename16Char() (s string, err error) {
	b := make([]byte, 8)
	_, err = rand.Read(b)
	if err != nil {
		return
	}
	s = fmt.Sprintf("%x", b)
	return
}

func (d *s3Store) GetPresignedUrl(ctx context.Context, bucket, key string) (*url.URL, error) {

	p := Presigner{PresignClient: s3.NewPresignClient(d.S3Client), Logger: d.Logger}
	res, err := p.GetObject(bucket, key, 3600)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(res.URL)
	if err != nil {
		return nil, err
	}

	return u, nil
}

type Presigner struct {
	PresignClient *s3.PresignClient
	Logger        *slog.Logger
}

// logger is nil-safe: Presigner is exported and can be constructed directly
// without a logger.
func (presigner Presigner) logger() *slog.Logger {
	if presigner.Logger == nil {
		return slog.Default()
	}
	return presigner.Logger
}

func (presigner Presigner) GetObject(
	bucketName string, objectKey string, lifetimeSecs int64) (*v4.PresignedHTTPRequest, error) {
	request, err := presigner.PresignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(lifetimeSecs * int64(time.Second))
	})
	if err != nil {
		// Was logged via Printf, which logrus emits at Info level: a real
		// presign failure looked like an informational message.
		presigner.logger().Error("could not get a presigned request for S3 object",
			slog.Any(logging.ErrorKey, err),
			slog.String(logging.S3BucketKey, bucketName),
			slog.String(logging.S3KeyKey, objectKey))
	}
	return request, err
}
