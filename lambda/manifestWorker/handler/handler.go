package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/pennsieve/datasets-service/api/logging"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/service"
	"net/http"
)

var PennsieveDB *sql.DB
var S3Client *s3.Client
var SnsClient *sns.Client
var HandlerVars *models.HandlerVars
var Logger = logging.Default

func LambdaHandler(ctx context.Context, snsEvent events.SNSEvent) (int, error) {

	message := snsEvent.Records[0].SNS.Message
	var params models.ManifestWorkerInput
	err := json.Unmarshal([]byte(message), &params)
	if err != nil {
		Logger.Error("Unable to unmarshal the manifest worker input")
		return 0, err
	}

	Logger.Info(fmt.Sprintf("params: %d, %s, %s", params.OrgIntId, params.DatasetNodeId, params.ManifestS3Key))

	srv := service.NewDatasetsService(PennsieveDB, S3Client, SnsClient, HandlerVars, int(params.OrgIntId))
	err = srv.GetManifest(ctx, params)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return 0, nil

}
