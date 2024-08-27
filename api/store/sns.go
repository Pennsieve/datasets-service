package store

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/pennsieve/datasets-service/api/models"
	log "github.com/sirupsen/logrus"
)

type SnsStoreFactory interface {
	NewSimpleStore(topic string) SnsStore
}

func NewSnsStoreFactory(snsClient models.SnsAPI) SnsStoreFactory {
	return &snsStoreFactory{SnsClient: snsClient}
}

// NewSimpleStore returns a DatasetsStore instance that
// will run statements directly on database
func (d *snsStoreFactory) NewSimpleStore(topic string) SnsStore {
	return &snsStore{d.SnsClient, topic}
}

type snsStoreFactory struct {
	SnsClient models.SnsAPI
}

type SnsStore interface {
	TriggerWorkerLambda(ctx context.Context, input models.ManifestWorkerInput) error
}

type snsStore struct {
	SnsClient models.SnsAPI
	SnsTopic  string
}

func (s *snsStore) TriggerWorkerLambda(ctx context.Context, input models.ManifestWorkerInput) error {

	jsonInput, err := json.Marshal(input)

	params := sns.PublishInput{
		Message:  aws.String(string(jsonInput)),
		TopicArn: aws.String(s.SnsTopic),
	}

	_, err = s.SnsClient.Publish(context.Background(), &params)
	if err != nil {
		log.Error("Error publishing to SNS: ", err)
		return err
	}

	return nil
}
