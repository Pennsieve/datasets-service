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
	"github.com/pennsieve/datasets-service/api/models"
	log "github.com/sirupsen/logrus"
	"net/url"
	"strings"
	"time"
)

type S3StoreFactory interface {
	NewSimpleStore(bucket string) S3Store
}

func NewS3StoreFactory(s3Client *s3.Client) S3StoreFactory {
	return &s3StoreFactory{S3Client: s3Client}
}

// NewSimpleStore returns a DatasetsStore instance that
// will run statements directly on database
func (d *s3StoreFactory) NewSimpleStore(bucket string) S3Store {
	return &s3Store{d.S3Client, bucket}
}

type s3StoreFactory struct {
	S3Client *s3.Client
	S3Bucket string
}

type S3Store interface {
	WriteManifestToS3(ctx context.Context, datasetNodeId string, manifest []models.ManifestDTO) (*models.WriteManifestOutput, error)
	GetPresignedUrl(ctx context.Context, bucket, key string) (*url.URL, error)
}

type s3Store struct {
	S3Client *s3.Client
	S3Bucket string
}

func (d *s3Store) WriteManifestToS3(ctx context.Context, datasetNodeId string, manifest []models.ManifestDTO) (*models.WriteManifestOutput, error) {

	// Method to create relatively short UUID JSON file name
	randName, _ := randomFilename16Char()
	manifestFileName := fmt.Sprintf("%s/%s.json",
		strings.Replace(datasetNodeId, "N:dataset:", "", -1), randName)

	uploader := manager.NewUploader(d.S3Client, func(u *manager.Uploader) {
		// Define a strategy that will buffer 25 MiB in memory
		u.BufferProvider = manager.NewBufferedReadSeekerWriteToPool(25 * 1024 * 1024)
	})

	serializedManifests, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(d.S3Bucket),
		Key:    aws.String(manifestFileName),
		Body:   bytes.NewReader(serializedManifests),
	})
	if err != nil {
		return nil, err
	}

	return &models.WriteManifestOutput{
		S3Bucket: d.S3Bucket,
		S3Key:    manifestFileName,
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

	p := Presigner{s3.NewPresignClient(d.S3Client)}
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
		log.Printf("Couldn't get a presigned request to get %v:%v. Here's why: %v\n",
			bucketName, objectKey, err)
	}
	return request, err
}
