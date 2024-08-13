package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type HandlerSSMVars struct {
	S3Bucket string
}

type WriteManifestOutput struct {
	S3Bucket string
	S3Key    string
}

// S3ManifestFile how the file on S3 will be structured.
type S3ManifestFile struct {
	DatasetNodeId string
	Date          time.Time
	Manifest      []ManifestDTO
}

type ManifestDTO struct {
	PackageId     int        `json:"package_id"`
	PackageName   string     `json:"package_name"`
	FileName      NullString `json:"file_name,omitempty"`
	Path          string     `json:"path"`
	PackageNodeId string     `json:"package_node_id"`
	Size          NullInt    `json:"size,omitempty"`
	CheckSum      NullString `json:"checksum,omitempty"`
}

type DatasetManifest struct {
	PackageId     int             `json:"package_id"`
	PackageName   string          `json:"package_name"`
	FileName      NullString      `json:"file_name,omitempty"`
	Path          []sql.NullInt64 `json:"path"`
	PackageNodeId string          `json:"package_node_id"`
	Size          NullInt         `json:"size,omitempty"`
	CheckSum      NullString      `json:"checksum,omitempty"`
}

// NullString is a wrapper around sql.NullString
type NullString struct{ sql.NullString }

// MarshalJSON method is called by json.Marshal,
// whenever it is of type NullString
func (x *NullString) MarshalJSON() ([]byte, error) {
	if !x.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(x.String)
}

func (x *NullString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*x = NullString{NullString: sql.NullString{
		String: s, Valid: true,
	}}
	return nil
}

// NullInt is a wrapper around sql.NullInt64
type NullInt struct{ sql.NullInt64 }

// MarshalJSON method is called by json.Marshal,
// whenever it is of type NullString
func (x *NullInt) MarshalJSON() ([]byte, error) {
	if !x.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(x.Int64)
}

func (x *NullInt) UnmarshalJSON(data []byte) error {
	var s int64
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*x = NullInt{NullInt64: sql.NullInt64{
		Int64: s, Valid: true,
	}}
	return nil
}

type ManifestResult struct {
	Url      string `json:"url"`
	S3Bucket string `json:"s3_bucket"`
	S3Key    string `json:"s3_key"`
}
