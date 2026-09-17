package store

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/pennsieve/datasets-service/api/logging"
	"github.com/pennsieve/datasets-service/api/models"
	"log/slog"
)

type SnsStoreFactory interface {
	NewSimpleStore(topic string) SnsStore
}

// NewSnsStoreFactory takes the request-scoped logger so that SNS failures carry
// the same trace/request context as the rest of the invocation. Pass
// slog.Default() outside of a request.
func NewSnsStoreFactory(snsClient models.SnsAPI, logger *slog.Logger) SnsStoreFactory {
	if logger == nil {
		logger = slog.Default()
	}
	return &snsStoreFactory{SnsClient: snsClient, Logger: logger}
}

// NewSimpleStore returns a DatasetsStore instance that
// will run statements directly on database
func (d *snsStoreFactory) NewSimpleStore(topic string) SnsStore {
	return &snsStore{SnsClient: d.SnsClient, SnsTopic: topic, Logger: d.Logger}
}

type snsStoreFactory struct {
	SnsClient models.SnsAPI
	Logger    *slog.Logger
}

type SnsStore interface {
	TriggerWorkerLambda(ctx context.Context, input models.ManifestWorkerInput) error
}

type snsStore struct {
	SnsClient models.SnsAPI
	SnsTopic  string
	Logger    *slog.Logger
}

func (s *snsStore) TriggerWorkerLambda(ctx context.Context, input models.ManifestWorkerInput) error {

	jsonInput, err := json.Marshal(input)
	if err != nil {
		// Previously this error was discarded: err was overwritten by the
		// Publish call below, so a marshal failure silently published an
		// empty/garbage message body to the topic.
		s.Logger.Error("error marshalling SNS message",
			slog.Any(logging.ErrorKey, err),
			slog.String(logging.SnsTopicKey, s.SnsTopic),
			slog.Int(logging.OrgIdKey, input.OrgIntId),
			slog.String(logging.DatasetNodeIdKey, input.DatasetNodeId))
		return err
	}

	params := sns.PublishInput{
		Message:  aws.String(string(jsonInput)),
		TopicArn: aws.String(s.SnsTopic),
	}

	_, err = s.SnsClient.Publish(context.Background(), &params)
	if err != nil {
		s.Logger.Error("error publishing to SNS",
			slog.Any(logging.ErrorKey, err),
			slog.String(logging.SnsTopicKey, s.SnsTopic),
			slog.Int(logging.OrgIdKey, input.OrgIntId),
			slog.String(logging.DatasetNodeIdKey, input.DatasetNodeId))
		return err
	}

	return nil
}
