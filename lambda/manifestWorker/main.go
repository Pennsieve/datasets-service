package manifestWorker

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pennsieve/datasets-service/api/service"
	"github.com/pennsieve/datasets-service/manifest-worker/handler"
	"github.com/pennsieve/pennsieve-go-core/pkg/queries/pgdb"
)

func init() {
	db, err := pgdb.ConnectRDS()
	if err != nil {
		panic(fmt.Sprintf("unable to connect to RDS database: %s", err))
	}
	handler.Logger.Info("connected to RDS database")
	handler.PennsieveDB = db

	// Get SSM variables
	handler.HandlerVars, err = service.GetAppClientVars(context.Background())
	if err != nil {
		handler.Logger.Error("Unable to get SSM vars: %v\n", err)
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		handler.Logger.Error("LoadDefaultConfig: %v\n", err)
	}

	handler.S3Client = s3.NewFromConfig(cfg)
}

func main() {
	lambda.Start(handler.LambdaHandler)
}