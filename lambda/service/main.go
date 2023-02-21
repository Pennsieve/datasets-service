package main

import (
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/pennsieve/datasets-service/api/store"
	"github.com/pennsieve/datasets-service/service/handler"
	"github.com/sirupsen/logrus"
)

func init() {
	config, err := store.PostgresConfigForRDS()
	if err != nil {
		panic("unable to get postgres config for RDS: " + err.Error())
	}
	db, err := config.OpenAtSchema("pennsieve")
	if err != nil {
		panic(fmt.Sprintf("unable to open database with config %s: %s", config.LogString(), err))
	}
	if err = db.Ping(); err != nil {
		panic(fmt.Sprintf("unable to connect to database with config %s: %s", config.LogString(), err))
	}
	logrus.Info("connected to database: ", config.LogString())
	handler.PennsieveDB = db
}

func main() {
	lambda.Start(handler.DatasetsServiceHandler)
}
