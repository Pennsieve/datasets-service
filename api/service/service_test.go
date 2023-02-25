package service

import (
	"context"
	"errors"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/store"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/dbTable"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetTrashcanPageErrors(t *testing.T) {
	for tName, expected := range map[string]struct {
		rootNodeId string
		error      error
	}{
		"package not found error":    {"non-empty-root-node-id", models.PackageNotFoundError{OrgId: 1, NodeId: "N:package:bad-1234"}},
		"unexpected error":           {"non-empty-root-node-id", errors.New("unexpected error")},
		"root page unexpected error": {"", errors.New("unexpected error")},
	} {
		service := NewDatasetsService(&MockDatasetsStore{
			GetDatasetByNodeIdReturn:       MockReturn[*dbTable.Dataset]{ReturnValue: &dbTable.Dataset{Id: 13}},
			GetTrashcanPaginatedReturn:     MockReturn[*store.PackagePage]{Error: expected.error},
			GetTrashcanRootPaginatedReturn: MockReturn[*store.PackagePage]{Error: expected.error},
		})
		t.Run(tName, func(t *testing.T) {
			_, err := service.GetTrashcanPage(context.Background(), "N:dataset:7890", expected.rootNodeId, 10, 0)
			if assert.Error(t, err) {
				assert.Equal(t, expected.error, err)
			}
		})
	}
}

type MockReturn[T any] struct {
	ReturnValue T
	Error       error
}

type MockDatasetsStore struct {
	GetDatasetByNodeIdReturn       MockReturn[*dbTable.Dataset]
	GetTrashcanRootPaginatedReturn MockReturn[*store.PackagePage]
	GetTrashcanPaginatedReturn     MockReturn[*store.PackagePage]
}

func (m *MockDatasetsStore) GetTrashcanRootPaginated(_ context.Context, _ int64, _ int, _ int) (*store.PackagePage, error) {
	if err := m.GetTrashcanRootPaginatedReturn.Error; err != nil {
		return nil, err
	}
	return m.GetTrashcanRootPaginatedReturn.ReturnValue, nil
}

func (m *MockDatasetsStore) GetTrashcanPaginated(_ context.Context, _ int64, _ string, _ int, _ int) (*store.PackagePage, error) {
	if err := m.GetTrashcanPaginatedReturn.Error; err != nil {
		return nil, err
	}
	return m.GetTrashcanPaginatedReturn.ReturnValue, nil
}

func (m *MockDatasetsStore) GetDatasetByNodeId(_ context.Context, _ string) (*dbTable.Dataset, error) {
	if err := m.GetDatasetByNodeIdReturn.Error; err != nil {
		return nil, err
	}
	return m.GetDatasetByNodeIdReturn.ReturnValue, nil
}
