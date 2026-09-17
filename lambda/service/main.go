package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/pennsieve/datasets-service/api/service"
	"github.com/pennsieve/datasets-service/service/handler"
	"github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
	"github.com/sirupsen/logrus"
)

func init() {
	// Cold-start failures below are all "this Lambda cannot serve any request"
	// conditions. They previously used three different exit mechanisms
	// (panic, stdlib log.Fatalf, logrus); converge on a single structured
	// error log followed by os.Exit(1).
	db, err := pgdb.ConnectRDS()
	if err != nil {
		logrus.WithError(err).Fatal("unable to connect to RDS database")
	}
	logrus.Info("connected to RDS database")
	handler.PennsieveDB = db

	// Get SSM variables
	handler.HandlerVars, err = service.GetAppClientVars(context.Background())
	if err != nil {
		logrus.WithError(err).Fatal("unable to get SSM vars")
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logrus.WithError(err).Fatal("unable to load default AWS config")
	}

	handler.S3Client = s3.NewFromConfig(cfg)
	handler.SNSClient = sns.NewFromConfig(cfg)

}

func main() {
	lambda.Start(handler.DatasetsServiceHandler)
}
