package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/pennsieve/datasets-service/api/logging"
	"github.com/pennsieve/datasets-service/api/service"
	"github.com/pennsieve/datasets-service/service/handler"
	"github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
)

func init() {
	// Single, explicit point of logging configuration. This used to be spread
	// across three competing init() blocks (here via logrus, api/store, and
	// api/logging), whose ordering decided the effective level.
	logging.SetDefaultFromEnv()

	// Every failure below means this Lambda cannot serve any request. They
	// previously used three different exit mechanisms in one short block
	// (panic, stdlib log.Fatalf, and logrus); converge on a structured slog
	// error followed by os.Exit(1).
	db, err := pgdb.ConnectRDS()
	if err != nil {
		slog.Error("unable to connect to RDS database", slog.Any(logging.ErrorKey, err))
		os.Exit(1)
	}
	slog.Info("connected to RDS database")
	handler.PennsieveDB = db

	// Get SSM variables
	handler.HandlerVars, err = service.GetAppClientVars(context.Background())
	if err != nil {
		slog.Error("unable to get SSM vars", slog.Any(logging.ErrorKey, err))
		os.Exit(1)
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		slog.Error("unable to load default AWS config", slog.Any(logging.ErrorKey, err))
		os.Exit(1)
	}

	handler.S3Client = s3.NewFromConfig(cfg)
	handler.SNSClient = sns.NewFromConfig(cfg)

}

func main() {
	lambda.Start(handler.DatasetsServiceHandler)
}
