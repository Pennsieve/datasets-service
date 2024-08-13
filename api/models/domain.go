package models

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3UploadAPI interface {
	PutObject(ctx context.Context, input *s3.PutObjectInput, f ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	UploadPart(ctx context.Context, input *s3.UploadPartInput, f ...func(*s3.Options)) (*s3.UploadPartOutput, error)
	CreateMultipartUpload(ctx context.Context, input *s3.CreateMultipartUploadInput, f ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error)
	CompleteMultipartUpload(ctx context.Context, input *s3.CompleteMultipartUploadInput, f ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error)
	AbortMultipartUpload(ctx context.Context, input *s3.AbortMultipartUploadInput, f ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error)
}
