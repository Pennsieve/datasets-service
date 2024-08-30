package models

type ManifestWorkerInput struct {
	OrgIntId      int    `json:"org_int_id"`
	DatasetNodeId string `json:"dataset_node_id"`
}
