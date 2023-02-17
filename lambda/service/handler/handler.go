package handler

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/datasets-service/api/service"
	"github.com/pennsieve/pennsieve-go-api/pkg/authorizer"
	log "github.com/sirupsen/logrus"
	"os"
)

var DatasetsService service.DatasetsService

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	ll, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(ll)
	}
}

func DatasetsServiceHandler(request events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error) {
	var err error
	var apiResponse *events.APIGatewayV2HTTPResponse
	datasetId, _ := request.QueryStringParameters["dataset_id"]
	claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)
	method := request.RequestContext.HTTP.Method
	routeKey := request.RouteKey
	path := request.RequestContext.HTTP.Path
	log.WithFields(log.Fields{
		"datasetId": datasetId,
		"claims":    claims,
		"method":    method,
		"routeKey":  routeKey,
		"path":      path}).Info("DatasetServiceHandler received request")
	apiResponse, err = handleRequest()
	err = DatasetsService.GetTrashcan(datasetId)
	return apiResponse, err
}

func handleRequest() (*events.APIGatewayV2HTTPResponse, error) {
	log.Println("handleRequest() ")
	apiResponse := events.APIGatewayV2HTTPResponse{Body: "{'response':'hello'}", StatusCode: 200}

	return &apiResponse, nil
}
