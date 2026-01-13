package models

import "time"

// SharedDatasetsPage represents a paginated response of shared datasets
type SharedDatasetsPage struct {
	Limit      int                   `json:"limit"`
	Offset     int                   `json:"offset"`
	TotalCount int                   `json:"totalCount"`
	Datasets   []SharedDatasetItem   `json:"datasets"`
}

// SharedDatasetItem represents a single shared dataset in the response
type SharedDatasetItem struct {
	Content SharedDatasetContent `json:"content"`
}

// SharedDatasetContent contains the actual dataset information
type SharedDatasetContent struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description,omitempty"`
	State              string    `json:"state"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	Status             string    `json:"status"`
	Tags               []string  `json:"tags,omitempty"`
	DataUseAgreementID *int      `json:"dataUseAgreementId,omitempty"`
	IntId              int       `json:"intId"`
	WorkspaceNodeID    string    `json:"workspaceNodeId"`
	WorkspaceName      string    `json:"workspaceName"`
}