package handler

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/pennsieve/datasets-service/api/models"
	"os"
)

// SSMGetParameterAPI defines the interface for the GetParameter function.
// We use this interface to test the function using a mocked service.
type SSMGetParameterAPI interface {
	GetParameter(ctx context.Context,
		params *ssm.GetParameterInput,
		optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

// FindParameter retrieves an AWS Systems Manager string parameter
// Inputs:
//
//	c is the context of the method call, which includes the AWS Region
//	api is the interface that defines the method call
//	input defines the input arguments to the service call.
//
// Output:
//
//	If success, a GetParameterOutput object containing the result of the service call and nil
//	Otherwise, nil and an error from the call to GetParameter
func FindParameter(c context.Context, api SSMGetParameterAPI, input *ssm.GetParameterInput) (*ssm.GetParameterOutput, error) {
	return api.GetParameter(c, input)
}

// GetAppClientVars returns a SSMVars struct with values from AWS SSM or ENV
func GetAppClientVars(ctx context.Context) (*models.HandlerSSMVars, error) {

	s3BucketId := os.Getenv("MANIFEST_FILES_BUCKET")

	if s3BucketId == "" {
		// Get variables from SSM
		env := os.Getenv("ENV")

		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return nil, err
		}

		client := ssm.NewFromConfig(cfg)

		input := &ssm.GetParameterInput{
			WithDecryption: aws.Bool(false),
			Name:           aws.String(fmt.Sprintf("/%s/datasets-service/manifest-bucket", env)),
		}

		results, err := FindParameter(ctx, client, input)
		if err != nil {
			return nil, err
		}

		s3BucketId = aws.ToString(results.Parameter.Value)
	}

	return &models.HandlerSSMVars{
		S3Bucket: s3BucketId,
	}, nil
}
