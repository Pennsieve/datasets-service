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
		mockStore  MockDatasetsStore
	}{
		"dataset not found error": {"N:collection:8700", MockDatasetsStore{
			GetDatasetByNodeIdReturn: MockReturn[*dbTable.Dataset]{Error: models.DatasetNotFoundError{OrgId: 7, NodeId: "N:dataset:9492034"}}}},
		"unexpected get dataset error": {"N:collection:8700", MockDatasetsStore{
			GetDatasetByNodeIdReturn: MockReturn[*dbTable.Dataset]{Error: errors.New("unexpected get dataset error")}}},
		"package not found error": {"N:collection:5790", MockDatasetsStore{
			GetDatasetByNodeIdReturn:   MockReturn[*dbTable.Dataset]{ReturnValue: &dbTable.Dataset{Id: 13}},
			GetTrashcanPaginatedReturn: MockReturn[*store.PackagePage]{Error: models.PackageNotFoundError{OrgId: 5, NodeId: "N:package:bad-999"}},
		}},
		"unexpected trashcan error": {"N:collection:5790", MockDatasetsStore{
			GetDatasetByNodeIdReturn:   MockReturn[*dbTable.Dataset]{ReturnValue: &dbTable.Dataset{Id: 13}},
			GetTrashcanPaginatedReturn: MockReturn[*store.PackagePage]{Error: errors.New("unexpected error")},
		}},
		"unexpected root trashcan error": {"", MockDatasetsStore{
			GetDatasetByNodeIdReturn:       MockReturn[*dbTable.Dataset]{ReturnValue: &dbTable.Dataset{Id: 13}},
			GetTrashcanRootPaginatedReturn: MockReturn[*store.PackagePage]{Error: errors.New("unexpected root error")},
		}},
	} {
		service := NewDatasetsService(&expected.mockStore)
		t.Run(tName, func(t *testing.T) {
			_, err := service.GetTrashcanPage(context.Background(), "N:dataset:7890", expected.rootNodeId, 10, 0)
			if assert.Error(t, err) {
				assert.Equal(t, expected.mockStore.getExpectedErrors(), []error{err})
			}
		})
	}
}

type MockReturn[T any] struct {
	ReturnValue T
	Error       error
}

func (mr MockReturn[T]) r() (T, error) {
	if err := mr.Error; err != nil {
		var r T
		return r, err
	}
	return mr.ReturnValue, nil
}

type MockDatasetsStore struct {
	GetDatasetByNodeIdReturn       MockReturn[*dbTable.Dataset]
	GetTrashcanRootPaginatedReturn MockReturn[*store.PackagePage]
	GetTrashcanPaginatedReturn     MockReturn[*store.PackagePage]
}

func (m *MockDatasetsStore) getExpectedErrors() []error {
	expected := make([]error, 3)
	var i int
	if err := m.GetDatasetByNodeIdReturn.Error; err != nil {
		expected[i] = err
		i++
	}
	if err := m.GetTrashcanRootPaginatedReturn.Error; err != nil {
		expected[i] = err
		i++
	}
	if err := m.GetTrashcanPaginatedReturn.Error; err != nil {
		expected[i] = err
		i++
	}
	return expected[:i]
}

func (m *MockDatasetsStore) GetTrashcanRootPaginated(_ context.Context, _ int64, _ int, _ int) (*store.PackagePage, error) {
	return m.GetTrashcanRootPaginatedReturn.r()
}

func (m *MockDatasetsStore) GetTrashcanPaginated(_ context.Context, _ int64, _ string, _ int, _ int) (*store.PackagePage, error) {
	return m.GetTrashcanPaginatedReturn.r()
}

func (m *MockDatasetsStore) GetDatasetByNodeId(_ context.Context, _ string) (*dbTable.Dataset, error) {
	return m.GetDatasetByNodeIdReturn.r()
}
