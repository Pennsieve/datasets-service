package handler

import (
	"context"
	"database/sql"
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

func LambdaHandler(ctx context.Context, params models.ManifestWorkerInput) (int, error) {

	srv := service.NewDatasetsService(PennsieveDB, S3Client, SnsClient, *HandlerVars, int(params.OrgIntId))
	err := srv.GetManifest(ctx, params)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return 0, nil

}
