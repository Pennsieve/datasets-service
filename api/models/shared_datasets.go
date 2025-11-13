package models

import "time"

type SharedDatasetsPage struct {
	Limit      int                  `json:"limit"`
	Offset     int                  `json:"offset"`
	TotalCount int                  `json:"totalCount"`
	Datasets   []SharedDatasetItem  `json:"datasets"`
}

type SharedDatasetItem struct {
	Id        int64     `json:"id"`
	NodeId    string    `json:"node_id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	UpdatedAt time.Time `json:"updated_at"`
}
