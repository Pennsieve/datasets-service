package models

import (
	"fmt"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/packageInfo/packageType"
)

type DatasetNotFoundError struct {
	OrgId  int
	NodeId string
	Id     int
}

func (e DatasetNotFoundError) Error() string {
	if len(e.NodeId) > 0 && e.Id > 0 {
		return fmt.Sprintf("dataset node id (%s), id (%d) not found in workspace %d", e.NodeId, e.Id, e.OrgId)
	}
	if len(e.NodeId) > 0 {
		return fmt.Sprintf("dataset node id (%s) not found in workspace %d", e.NodeId, e.OrgId)
	}
	return fmt.Sprintf("no dataset with id (%d) found in workspace %d", e.Id, e.OrgId)
}

type PackageNotFoundError struct {
	OrgId     int
	NodeId    string
	DatasetId int64
}

func (e PackageNotFoundError) Error() string {
	if e.DatasetId == 0 {
		return fmt.Sprintf("no package with node id %q found in workspace %d", e.NodeId, e.OrgId)
	}

	return fmt.Sprintf("no package with node id %q found in dataset %d, workspace %d", e.NodeId, e.DatasetId, e.OrgId)

}

type FolderNotFoundError struct {
	OrgId      int
	NodeId     string
	ActualType packageType.Type
}

func (e FolderNotFoundError) Error() string {
	if e.ActualType < 0 {
		return fmt.Sprintf("no folder with node id %q found in workspace %d", e.NodeId, e.OrgId)
	}
	return fmt.Sprintf("no folder with node id %q found in workspace %d (actual type %s)", e.NodeId, e.OrgId, e.ActualType)
}
