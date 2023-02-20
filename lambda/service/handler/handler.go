package handler

import (
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/datasets-service/api/service"
	"github.com/pennsieve/pennsieve-go-api/pkg/authorizer"
	log "github.com/sirupsen/logrus"
	"net/http"
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
	claims := authorizer.ParseClaims(request.RequestContext.Authorizer.Lambda)
	return handleRequest(claims, request)
}

func handleRequest(claims *authorizer.Claims, request events.APIGatewayV2HTTPRequest) (*events.APIGatewayV2HTTPResponse, error) {
	var err error
	datasetId, _ := request.QueryStringParameters["dataset_id"]
	method := request.RequestContext.HTTP.Method
	path := request.RequestContext.HTTP.Path
	reqBody := request.Body
	reqID := request.RequestContext.RequestID
	requestLogger := log.WithFields(log.Fields{
		"requestID": reqID,
		"method":    method,
		"path":      path,
	})
	requestLogger.WithFields(log.Fields{
		"datasetId":    datasetId,
		"claims":       claims,
		"request body": reqBody}).Info("DatasetServiceHandler received request")
	apiResponse := events.APIGatewayV2HTTPResponse{}
	switch path {
	case "/datasets/trashcan":
		switch method {
		case "GET":
			if err := DatasetsService.GetTrashcan(datasetId); err != nil {
				return nil, err
			}
		default:
			requestLogger.Info("method not allowed")
			return handleError(&apiResponse, "method not allowed: "+method, http.StatusMethodNotAllowed), nil
		}
	default:
		requestLogger.Info("unknown route")
		return handleError(&apiResponse, "resource not found: "+path, http.StatusNotFound), nil
	}
	return &apiResponse, err

}

func handleError(res *events.APIGatewayV2HTTPResponse, message string, status int) *events.APIGatewayV2HTTPResponse {
	res.Body = fmt.Sprintf("{'message': '%s'}", message)
	res.StatusCode = status
	// Return for convenience
	return res
}
