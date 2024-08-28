package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/pennsieve/datasets-service/api/service"
	"github.com/pennsieve/datasets-service/service/handler"
	"github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
	"github.com/sirupsen/logrus"
	"log"
)

func init() {
	db, err := pgdb.ConnectRDS()
	if err != nil {
		panic(fmt.Sprintf("unable to connect to RDS database: %s", err))
	}
	logrus.Info("connected to RDS database")
	handler.PennsieveDB = db

	// Get SSM variables
	handler.HandlerVars, err = service.GetAppClientVars(context.Background())
	if err != nil {
		log.Fatalf("Unable to get SSM vars: %v\n", err)
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("LoadDefaultConfig: %v\n", err)
	}

	handler.S3Client = s3.NewFromConfig(cfg)
	handler.SNSClient = sns.NewFromConfig(cfg)

}

func main() {
	lambda.Start(handler.DatasetsServiceHandler)
}
