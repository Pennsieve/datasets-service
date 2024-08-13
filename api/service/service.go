package service

import (
	"context"
	"database/sql"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pennsieve/datasets-service/api/models"
	"github.com/pennsieve/datasets-service/api/store"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageState"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageType"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"strings"
)

type DatasetsService interface {
	GetDataset(ctx context.Context, datasetNodeId string) (*pgdb.Dataset, error)
	GetTrashcanPage(ctx context.Context, datasetNodeId string, rootNodeId string, limit int, offset int) (*models.TrashcanPage, error)
	GetManifest(ctx context.Context, datasetNodeId string) (*models.ManifestResult, error)
}

type datasetsService struct {
	StoreFactory     store.DatasetsStoreFactory
	S3StoreFactory   store.S3StoreFactory
	OrgId            int
	S3ManifestBucket string
}

func NewDatasetsServiceWithFactory(factory store.DatasetsStoreFactory, s3factory store.S3StoreFactory, ssmVars models.HandlerSSMVars, orgId int) DatasetsService {
	return &datasetsService{StoreFactory: factory, S3StoreFactory: s3factory, S3ManifestBucket: ssmVars.S3Bucket, OrgId: orgId}
}

func NewDatasetsService(db *sql.DB, s3Client *s3.Client, ssmVars models.HandlerSSMVars, orgId int) DatasetsService {
	str := store.NewDatasetsStoreFactory(db, s3Client)

	s3factory := store.NewS3StoreFactory(s3Client)

	datasetsSvc := NewDatasetsServiceWithFactory(str, s3factory, ssmVars, orgId)
	return datasetsSvc
}

func (s *datasetsService) GetTrashcanPage(ctx context.Context, datasetId string, rootNodeId string, limit int, offset int) (*models.TrashcanPage, error) {
	trashcan := models.TrashcanPage{Limit: limit, Offset: offset, Packages: []models.TrashcanItem{}, Messages: []string{}}
	err := s.StoreFactory.ExecStoreTx(ctx, s.OrgId, func(q store.DatasetsStore) error {
		dataset, err := q.GetDatasetByNodeId(ctx, datasetId)
		if err != nil {
			return err
		}
		deletedCount, err := q.CountDatasetPackagesByStates(ctx, dataset.Id, []packageState.State{packageState.Deleted, packageState.Deleting})
		if err != nil || deletedCount == 0 {
			return err
		}
		var page *store.PackagePage
		if len(rootNodeId) == 0 {
			page, err = q.GetTrashcanRootPaginated(ctx, dataset.Id, limit, offset)
		} else {
			rootPckg, pckgErr := q.GetDatasetPackageByNodeId(ctx, dataset.Id, rootNodeId)
			if pckgErr != nil {
				return pckgErr
			}
			if rootPckg.PackageType != packageType.Collection {
				return models.FolderNotFoundError{OrgId: s.OrgId, NodeId: rootNodeId, DatasetId: models.DatasetNodeId(datasetId), ActualType: rootPckg.PackageType}
			}
			page, err = q.GetTrashcanPaginated(ctx, dataset.Id, rootPckg.Id, limit, offset)
		}
		if err != nil {
			return err
		}
		packages := make([]models.TrashcanItem, len(page.Packages))
		for i, p := range page.Packages {
			packages[i] = models.TrashcanItem{
				ID:     p.Id,
				Name:   p.Name,
				NodeId: p.NodeId,
				Type:   p.PackageType.String(),
				State:  p.PackageState.String(),
			}
		}
		trashcan.TotalCount = page.TotalCount
		trashcan.Packages = packages
		return nil
	})
	return &trashcan, err
}

func (s *datasetsService) GetDataset(ctx context.Context, datasetId string) (*pgdb.Dataset, error) {
	q := s.StoreFactory.NewSimpleStore(s.OrgId)
	return q.GetDatasetByNodeId(ctx, datasetId)
}

func (s *datasetsService) GetManifest(ctx context.Context, datasetNodeId string) (*models.ManifestResult, error) {
	q := s.StoreFactory.NewSimpleStore(s.OrgId)
	s3 := s.S3StoreFactory.NewSimpleStore(s.S3ManifestBucket)

	ds, _ := q.GetDatasetByNodeId(ctx, datasetNodeId)

	manifest, err := q.GetDatasetManifest(ctx, ds.Id)

	// Map each entry to packageID for lookup to create path
	manifestMap := make(map[int]models.DatasetManifest)
	for i := 0; i < len(manifest); i++ {
		manifestMap[manifest[i].PackageId] = manifest[i]
	}

	// Generate ManifestDTO which includes full path for files
	var results []models.ManifestDTO
	for i, _ := range manifest {
		var sb strings.Builder
		for pathIndex, _ := range manifest[i].Path {
			if manifest[i].Path[pathIndex].Valid {
				if pathIndex > 1 {
					sb.WriteString("/")
				}
				sb.WriteString(manifestMap[int(manifest[i].Path[pathIndex].Int64)].PackageName)
			}
		}

		results = append(results, models.ManifestDTO{
			PackageId:     manifest[i].PackageId,
			PackageName:   manifest[i].PackageName,
			FileName:      manifest[i].FileName,
			Path:          sb.String(),
			PackageNodeId: manifest[i].PackageNodeId,
			Size:          manifest[i].Size,
			CheckSum:      manifest[i].CheckSum,
		})

	}

	// Write JSON file to S3
	writeOutput, err := s3.WriteManifestToS3(ctx, datasetNodeId, results)
	if err != nil {
		return nil, err
	}

	// Create Presigned URL for file on S3
	presignedUrl, err := s3.GetPresignedUrl(ctx, writeOutput.S3Bucket, writeOutput.S3Key)
	if err != nil {
		return nil, err
	}

	result := models.ManifestResult{
		Url:      presignedUrl.String(),
		S3Bucket: writeOutput.S3Bucket,
		S3Key:    writeOutput.S3Key,
	}
	return &result, nil

}
