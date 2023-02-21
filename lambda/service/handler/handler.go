package handler

import (
	"database/sql"
	"github.com/aws/aws-lambda-go/events"
	log "github.com/sirupsen/logrus"
	"os"
)

var PennsieveDB *sql.DB

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
	handler, err := NewHandler(&request)
	if err != nil {
		return nil, err
	}
	return handler.handle()
}
