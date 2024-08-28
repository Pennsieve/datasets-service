package service

import (
    "context"
    "database/sql"
    "fmt"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/pennsieve/datasets-service/api/models"
    "github.com/pennsieve/datasets-service/api/store"
    "github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageState"
    "github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageType"
    "github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
    "strings"
    "time"
)

type DatasetsService interface {
    GetDataset(ctx context.Context, datasetNodeId string) (*pgdb.Dataset, error)
    GetTrashcanPage(ctx context.Context, datasetNodeId string, rootNodeId string, limit int, offset int) (*models.TrashcanPage, error)
    GetManifest(ctx context.Context, input models.ManifestWorkerInput) error
    TriggerAsyncGetManifest(ctx context.Context, datasetNodeId string) (*models.ManifestResult, error)
}

type datasetsService struct {
    StoreFactory     store.DatasetsStoreFactory
    S3StoreFactory   store.S3StoreFactory
    SnsStoreFactory  store.SnsStoreFactory
    OrgId            int
    S3ManifestBucket string
    SnsTopic         string
}

func NewDatasetsServiceWithFactory(factory store.DatasetsStoreFactory, s3factory store.S3StoreFactory, snsFactory store.SnsStoreFactory, options *models.HandlerVars, orgId int) DatasetsService {
    return &datasetsService{
        StoreFactory:     factory,
        S3StoreFactory:   s3factory,
        SnsStoreFactory:  snsFactory,
        S3ManifestBucket: options.S3Bucket,
        OrgId:            orgId,
        SnsTopic:         options.SnsTopic}
}

func NewDatasetsService(db *sql.DB, s3Client *s3.Client, snsClient models.SnsAPI, options *models.HandlerVars, orgId int) DatasetsService {
    pgFactory := store.NewPostgresStoreFactory(db)
    s3Factory := store.NewS3StoreFactory(s3Client)
    snsFactory := store.NewSnsStoreFactory(snsClient)

    datasetsSvc := NewDatasetsServiceWithFactory(pgFactory, s3Factory, snsFactory, options, orgId)
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

// TriggerAsyncGetManifest is used to signal the worker Lambda to generate the manifest.
func (s *datasetsService) TriggerAsyncGetManifest(ctx context.Context, datasetNodeId string) (*models.ManifestResult, error) {
    q := s.StoreFactory.NewSimpleStore(s.OrgId)
    ds, _ := q.GetDatasetByNodeId(ctx, datasetNodeId)
    s3 := s.S3StoreFactory.NewSimpleStore(s.S3ManifestBucket)

    // Define S3Key for Manifest (format: "datasetID/datasetID_lastUpdated.json"
    manifestFileName := fmt.Sprintf("%s/%s_%s.json",
        strings.Replace(datasetNodeId, "N:dataset:", "", -1),
        strings.Replace(datasetNodeId, "N:dataset:", "", -1),
        strings.Replace(ds.UpdatedAt.Format(time.RFC3339), ":", "_", -1))

    sns := s.SnsStoreFactory.NewSimpleStore(s.SnsTopic)

    sns.TriggerWorkerLambda(ctx, models.ManifestWorkerInput{
        OrgIntId:      s.OrgId,
        DatasetNodeId: datasetNodeId,
        ManifestS3Key: manifestFileName,
    })

    // Create Presigned URL for file on S3
    presignedUrl, err := s3.GetPresignedUrl(ctx, s.S3ManifestBucket, manifestFileName)
    if err != nil {
        return nil, err
    }

    result := models.ManifestResult{
        Url:      presignedUrl.String(),
        S3Bucket: s.S3ManifestBucket,
        S3Key:    manifestFileName,
        Status:   models.CREATING,
    }
    return &result, nil

}

// GetManifest is used by the manifest Worker to generate the manifest and store on S3.
func (s *datasetsService) GetManifest(ctx context.Context, input models.ManifestWorkerInput) error {
    q := s.StoreFactory.NewSimpleStore(s.OrgId)
    s3 := s.S3StoreFactory.NewSimpleStore(s.S3ManifestBucket)

    ds, _ := q.GetDatasetByNodeId(ctx, input.DatasetNodeId)

    manifest, err := q.GetDatasetManifest(ctx, ds.Id)

    // Map each entry to packageID for lookup to create path
    manifestMap := make(map[int]models.DatasetManifest)
    for i := 0; i < len(manifest); i++ {
        manifestMap[manifest[i].PackageId] = manifest[i]
    }

    // Generate ManifestDTO which includes full path for files
    var results []models.ManifestDTO
    for i, _ := range manifest {

        if !strings.HasPrefix(manifest[i].PackageNodeId, "N:collection") {
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
                PackageName:   manifest[i].PackageName,
                FileName:      manifest[i].FileName,
                Path:          sb.String(),
                PackageNodeId: manifest[i].PackageNodeId,
                Size:          manifest[i].Size,
                CheckSum:      manifest[i].CheckSum,
            })
        }

    }

    license := "N/A"
    if ds.License.Valid {
        license = ds.License.String
    }

    description := "N/A"
    if ds.Description.Valid {
        description = ds.Description.String
    }

    workspaceManifest := models.WorkspaceManifest{
        Date:          models.JSONDate(time.Now()),
        DatasetId:     ds.Id,
        DatasetNodeId: input.DatasetNodeId,
        Name:          ds.Name,
        Description:   description,
        License:       license,
        Contributors:  ds.Contributors,
        Tags:          ds.Tags,
        Files:         results,
    }

    // Write JSON file to S3
    _, err = s3.WriteManifestToS3(ctx, input.DatasetNodeId, input.ManifestS3Key, workspaceManifest)
    if err != nil {
        return err
    }
    return nil

}
