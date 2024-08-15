package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	transport "github.com/aws/smithy-go/endpoints"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/store"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageState"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageType"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestGetTrashcanPageDeleting(t *testing.T) {
	limit := 100
	offset := 0
	rootNodeIdToExpectedPage := map[string]*models.TrashcanPage{
		"": {
			Limit: limit, Offset: offset, TotalCount: 2, Packages: []models.TrashcanItem{
				{
					ID:     4,
					Name:   "root-dir-1",
					NodeId: "N:collection:82c127ca-b72b-4d8b-a0c3-a9e4c7b14654",
					Type:   packageType.Collection.String(),
					State:  packageState.Ready.String(),
				},
				{
					ID:     5,
					Name:   "root-dir-2",
					NodeId: "N:collection:d6542ca3-31a4-473f-a7ab-490ca4fddc63",
					Type:   packageType.Collection.String(),
					State:  packageState.Ready.String(),
				},
			},
			Messages: []string{},
		},
		"N:collection:36cb9fb0-f72a-42fd-bcac-959ecb866279": {
			Limit: limit, Offset: offset, TotalCount: 0, Packages: []models.TrashcanItem{}, Messages: []string{},
		}, // an empty directory
		"N:collection:82c127ca-b72b-4d8b-a0c3-a9e4c7b14654": {
			Limit: limit, Offset: offset, TotalCount: 2, Packages: []models.TrashcanItem{
				{
					ID:     9,
					Name:   "one-dir-deleting-1",
					NodeId: "N:collection:e9bfe050-b375-43a1-91ec-b519439ad011",
					Type:   packageType.Collection.String(),
					State:  packageState.Deleting.String(),
				},
				{
					ID:     13,
					Name:   "one-dir-empty-deleting-1",
					NodeId: "N:collection:113d3c44-af35-408f-9fcc-0e4aa0b20a5d",
					Type:   packageType.Collection.String(),
					State:  packageState.Deleting.String(),
				},
			},
			Messages: []string{},
		},
		"N:collection:d6542ca3-31a4-473f-a7ab-490ca4fddc63": {
			Limit: limit, Offset: offset, TotalCount: 1, Packages: []models.TrashcanItem{
				{
					ID:     15,
					Name:   "one-dir-1",
					NodeId: "N:collection:f4136743-e930-401e-88bb-e7ef34789a88",
					Type:   packageType.Collection.String(),
					State:  packageState.Ready.String(),
				},
			},
			Messages: []string{},
		},
		"N:collection:e9bfe050-b375-43a1-91ec-b519439ad011": { // only set to DELETING. It's contents still have non-DELET* states and so expect an empty page.
			Limit: limit, Offset: offset, TotalCount: 0, Packages: []models.TrashcanItem{}, Messages: []string{},
		},
		"N:collection:113d3c44-af35-408f-9fcc-0e4aa0b20a5d": { // only set to DELETING. And it is empty, so nothing to show
			Limit: limit, Offset: offset, TotalCount: 0, Packages: []models.TrashcanItem{}, Messages: []string{},
		},
		"N:collection:f4136743-e930-401e-88bb-e7ef34789a88": {
			Limit: limit, Offset: offset, TotalCount: 1, Packages: []models.TrashcanItem{
				{
					ID:     25,
					Name:   "two-file-deleting-1.csv",
					NodeId: "N:package:d9ee5d8f-0f27-4179-ae9e-8b914a719543",
					Type:   packageType.CSV.String(),
					State:  packageState.Deleting.String(),
				},
			},
			Messages: []string{},
		},
	}
	orgId := 2
	datasetNodeId := "N:dataset:149b65da-6803-4a67-bf20-83076774a5c7"

	db := store.OpenDB(t)
	defer db.Close()

	db.ExecSQLFile("show-deleting-test.sql")
	defer db.Truncate(orgId, "packages")

	mfBucket := getEnv("MANIFEST_FILES_BUCKET", "manifest-files-bucket")

	service := NewDatasetsService(db.DB, getS3Client(), models.HandlerSSMVars{S3Bucket: mfBucket}, orgId)
	for rootId, expectedPage := range rootNodeIdToExpectedPage {
		t.Run(fmt.Sprintf("GetTrashcanPage starting at folder %s", rootId), func(t *testing.T) {
			actual, err := service.GetTrashcanPage(context.Background(), datasetNodeId, rootId, limit, offset)
			if assert.NoError(t, err) {
				assert.Equal(t, expectedPage, actual)
			}
		})
	}
}

func TestGetTrashcanPageEmpty(t *testing.T) {
	orgId := 7
	mockFactory := MockFactory{mockStore: &MockDatasetsStore{
		GetDatasetByNodeIdReturn:           MockReturn[*pgdb.Dataset]{Value: &pgdb.Dataset{Id: 17}},
		CountDatasetPackagesByStatesReturn: MockReturn[int]{Value: 0},
	}}
	mockS3Factory := MockS3Factory{}

	mfBucket := getEnv("MANIFEST_FILES_BUCKET", "manifest-files-bucket")
	service := NewDatasetsServiceWithFactory(&mockFactory, &mockS3Factory, models.HandlerSSMVars{S3Bucket: mfBucket}, orgId)
	page, err := service.GetTrashcanPage(context.Background(), "N:dataset:dddd", "", 100, 0)
	if assert.NoError(t, err) {
		assert.NotNil(t, page.Packages)
		assert.Empty(t, page.Packages)
	}
}

func TestGetTrashcanPageErrors(t *testing.T) {
	orgId := 7
	for tName, expected := range map[string]struct {
		rootNodeId string
		mockStore  MockDatasetsStore
	}{
		"dataset not found error": {"N:collection:8700", MockDatasetsStore{
			GetDatasetByNodeIdReturn: MockReturn[*pgdb.Dataset]{Error: models.DatasetNotFoundError{OrgId: orgId, Id: models.DatasetNodeId("N:dataset:9492034")}}}},
		"unexpected get dataset error": {"N:collection:8700", MockDatasetsStore{
			GetDatasetByNodeIdReturn: MockReturn[*pgdb.Dataset]{Error: errors.New("unexpected get dataset error")}}},
		"unexpected count deleted error": {"N:collection:8700", MockDatasetsStore{
			GetDatasetByNodeIdReturn:           MockReturn[*pgdb.Dataset]{Value: &pgdb.Dataset{Id: 13}},
			CountDatasetPackagesByStatesReturn: MockReturn[int]{Error: errors.New("unexpected count dataset error")},
		}},
		"package not found error": {"N:collection:5790", MockDatasetsStore{
			GetDatasetByNodeIdReturn:           MockReturn[*pgdb.Dataset]{Value: &pgdb.Dataset{Id: 13}},
			CountDatasetPackagesByStatesReturn: MockReturn[int]{Value: 6},
			GetDatasetPackageByNodeIdReturn:    MockReturn[*pgdb.Package]{Error: models.PackageNotFoundError{OrgId: orgId, Id: models.PackageNodeId("N:package:bad-999"), DatasetId: models.DatasetIntId(13)}},
		}},
		"unexpected trashcan error": {"N:collection:5790", MockDatasetsStore{
			GetDatasetByNodeIdReturn:           MockReturn[*pgdb.Dataset]{Value: &pgdb.Dataset{Id: 13}},
			CountDatasetPackagesByStatesReturn: MockReturn[int]{Value: 6},
			GetDatasetPackageByNodeIdReturn:    MockReturn[*pgdb.Package]{Value: &pgdb.Package{Id: 57, PackageType: packageType.Collection}},
			GetTrashcanPaginatedReturn:         MockReturn[*store.PackagePage]{Error: errors.New("unexpected error")},
		}},
		"unexpected root trashcan error": {"", MockDatasetsStore{
			GetDatasetByNodeIdReturn:           MockReturn[*pgdb.Dataset]{Value: &pgdb.Dataset{Id: 13}},
			CountDatasetPackagesByStatesReturn: MockReturn[int]{Value: 6},
			GetTrashcanRootPaginatedReturn:     MockReturn[*store.PackagePage]{Error: errors.New("unexpected root error")},
		}},
	} {
		mockFactory := MockFactory{&expected.mockStore, -1}
		mockS3Factory := MockS3Factory{}
		mfBucket := getEnv("MANIFEST_FILES_BUCKET", "manifest-files-bucket")

		service := NewDatasetsServiceWithFactory(&mockFactory, &mockS3Factory, models.HandlerSSMVars{S3Bucket: mfBucket}, orgId)
		t.Run(tName, func(t *testing.T) {
			_, err := service.GetTrashcanPage(context.Background(), "N:dataset:7890", expected.rootNodeId, 10, 0)
			if assert.Error(t, err) {
				assert.Equal(t, expected.mockStore.getExpectedErrors(), []error{err})
				assert.Equal(t, orgId, mockFactory.orgId)
			}
		})
	}
}

func TestGetManifest(t *testing.T) {
	datasetNodeId := "N:dataset:149b65da-6803-4a67-bf20-83076774a5c7"

	orgId := 2
	db := store.OpenDB(t)
	defer db.Close()

	db.ExecSQLFile("manifest-test.sql")
	defer func() {
		db.Truncate(orgId, "packages")
		db.Truncate(orgId, "files")
	}()

	mfBucket := getEnv("MANIFEST_FILES_BUCKET", "test-manifest-bucket")
	handleVars := models.HandlerSSMVars{S3Bucket: mfBucket}

	s3Client := getS3Client()
	service := NewDatasetsService(db.DB, s3Client, handleVars, orgId)

	// Generate manifest for dataset and store to S3 (and generate presigned url)
	actual, err := service.GetManifest(context.Background(), datasetNodeId)
	assert.NoError(t, err)
	assert.Equal(t, mfBucket, actual.S3Bucket)

	// Reading file from S3 which should contain array of manifestFile objects
	testResult, err := readS3Object(s3Client, actual.S3Bucket, actual.S3Key)
	assert.NoError(t, err)
	assert.Len(t, testResult.Files, 10, "Unexpected length of manifest file array from S3.")

	// Check name for single package
	check := false
	for _, w := range testResult.Files {
		if w.PackageNodeId == "N:package:1" {
			check = true
			exp := sql.NullString{String: "root-file.txt-1", Valid: true}
			assert.Equal(t, w.FileName, models.NullString{NullString: exp})
		}
	}
	assert.True(t, check, "Should have encountered an entry with packageId == 1")

}

// readS3Object is used to read object from MINIO test s3 store for testing manifest endpoint.
func readS3Object(client *s3.Client, bucket string, key string) (*models.WorkspaceManifest, error) {
	requestInput := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := client.GetObject(context.Background(), requestInput)
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	body1, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	bodyString1 := fmt.Sprintf("%s", body1)

	var results *models.WorkspaceManifest
	decoder := json.NewDecoder(strings.NewReader(bodyString1))
	err = decoder.Decode(&results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

type MockReturn[T any] struct {
	Value T
	Error error
}

func (mr MockReturn[T]) ret() (T, error) {
	if err := mr.Error; err != nil {
		var r T
		return r, err
	}
	return mr.Value, nil
}

type MockDatasetsStore struct {
	GetDatasetByNodeIdReturn           MockReturn[*pgdb.Dataset]
	GetTrashcanRootPaginatedReturn     MockReturn[*store.PackagePage]
	GetTrashcanPaginatedReturn         MockReturn[*store.PackagePage]
	CountDatasetPackagesByStatesReturn MockReturn[int]
	GetDatasetPackageByNodeIdReturn    MockReturn[*pgdb.Package]
	GetManifestReturn                  MockReturn[[]models.DatasetManifest]
}

func (m *MockDatasetsStore) GetDatasetManifest(_ context.Context, _ int64) ([]models.DatasetManifest, error) {
	return m.GetManifestReturn.ret()
}

func (m *MockDatasetsStore) getExpectedErrors() []error {
	expected := make([]error, 5)
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
	if err := m.CountDatasetPackagesByStatesReturn.Error; err != nil {
		expected[i] = err
		i++
	}
	if err := m.GetDatasetPackageByNodeIdReturn.Error; err != nil {
		expected[i] = err
		i++
	}
	return expected[:i]
}

func (m *MockDatasetsStore) GetTrashcanRootPaginated(_ context.Context, _ int64, _ int, _ int) (*store.PackagePage, error) {
	return m.GetTrashcanRootPaginatedReturn.ret()
}

func (m *MockDatasetsStore) GetTrashcanPaginated(_ context.Context, _ int64, _ int64, _ int, _ int) (*store.PackagePage, error) {
	return m.GetTrashcanPaginatedReturn.ret()
}

func (m *MockDatasetsStore) GetDatasetByNodeId(_ context.Context, _ string) (*pgdb.Dataset, error) {
	return m.GetDatasetByNodeIdReturn.ret()
}

func (m *MockDatasetsStore) CountDatasetPackagesByStates(_ context.Context, _ int64, _ []packageState.State) (int, error) {
	return m.CountDatasetPackagesByStatesReturn.ret()
}

func (m *MockDatasetsStore) GetDatasetPackageByNodeId(_ context.Context, _ int64, _ string) (*pgdb.Package, error) {
	return m.GetDatasetPackageByNodeIdReturn.ret()
}

type MockS3Store struct {
}

func (m *MockS3Store) WriteManifestToS3(ctx context.Context, datasetNodeId string, manifest models.WorkspaceManifest) (*models.WriteManifestOutput, error) {
	return nil, nil
}
func (m *MockS3Store) GetPresignedUrl(ctx context.Context, bucket string, key string) (*url.URL, error) {
	return nil, nil
}

type MockS3Factory struct {
	mockStore *MockS3Store
}

func (m *MockS3Factory) NewSimpleStore(bucket string) store.S3Store {
	return m.mockStore
}

type MockFactory struct {
	mockStore *MockDatasetsStore
	orgId     int
}

func (m *MockFactory) NewSimpleStore(orgId int) store.DatasetsStore {
	m.orgId = orgId
	return m.mockStore
}

func (m *MockFactory) ExecStoreTx(_ context.Context, orgId int, fn func(store store.DatasetsStore) error) error {
	m.orgId = orgId
	return fn(m.mockStore)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

type Resolver struct {
	URL *url.URL
}

func (r *Resolver) ResolveEndpoint(_ context.Context, params s3.EndpointParameters) (transport.Endpoint, error) {
	u := *r.URL
	u.Path += "/" + *params.Bucket
	return transport.Endpoint{URI: u}, nil
}

func getS3Client() *s3.Client {

	testDBUri, _ := url.Parse(getEnv("MINIO_URL", "http://localhost:9008"))

	s3Client := s3.New(s3.Options{
		EndpointResolverV2: &Resolver{URL: testDBUri},
		Credentials: aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
			}, nil
		}),
	})

	mfBucket := getEnv("MANIFEST_FILES_BUCKET", "manifest-files-bucket")
	_, err := s3Client.CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket:                     aws.String(mfBucket),
		ACL:                        "",
		CreateBucketConfiguration:  nil,
		GrantFullControl:           nil,
		GrantRead:                  nil,
		GrantReadACP:               nil,
		GrantWrite:                 nil,
		GrantWriteACP:              nil,
		ObjectLockEnabledForBucket: aws.Bool(false),
		ObjectOwnership:            "",
	})

	if err != nil {
		log.Info(err)
	}

	return s3Client

}
