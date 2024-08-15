package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"time"
)

type HandlerSSMVars struct {
	S3Bucket string
}

type WriteManifestOutput struct {
	S3Bucket string
	S3Key    string
}

// WorkspaceManifest how the file on S3 will be structured.
type WorkspaceManifest struct {
	Date          JSONDate          `json:"manifestCreatedOn"`
	DatasetId     int64             `json:"datasetId"`
	DatasetNodeId string            `json:"datasetNodeId"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	License       string            `json:"license"`
	Contributors  pgdb.Contributors `json:"contributors"`
	Tags          pgdb.Tags         `json:"tags"`
	Files         []ManifestDTO     `json:"files"`
}

type ManifestDTO struct {
	PackageNodeId string     `json:"sourcePackageId"`
	PackageName   string     `json:"sourcePackageName"`
	FileName      NullString `json:"name,omitempty"`
	Path          string     `json:"path"`
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

type JSONDate time.Time

func (t JSONDate) MarshalJSON() ([]byte, error) {
	stamp := fmt.Sprintf("\"%s\"", time.Time(t).Format(time.ANSIC))
	return []byte(stamp), nil
}

func (t *JSONDate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	tt, _ := time.Parse(time.ANSIC, s)

	*t = JSONDate(tt)
	return nil
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
