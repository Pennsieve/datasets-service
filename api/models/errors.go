package models

import "fmt"

type DatasetNotFoundError struct {
	OrgId  int
	NodeId string
}

func (e DatasetNotFoundError) Error() string {
	return fmt.Sprintf("no dataset with id %s found in workspace %d", e.NodeId, e.OrgId)
}
