package infomaniak

import (
	"context"
	"encoding/json"
)

// IkRecord infomaniak API record return type
type IkRecord struct {
	// ID of this record on infomaniak's side
	ID string `json:"id,omitempty"`

	// Type of this record
	Type string `json:"type"`

	// Absolute Source / Name
	Source string `json:"source,omitempty"`

	// Source / Name relative to domain name
	SourceIdn string `json:"source_idn,omitempty"`

	// Value of this record
	Target string `json:"target"`

	// TTL in seconds
	TtlInSec uint `json:"ttl"`

	// Priority of this record - default value on infomaniak's side
	// for records that do not have a priority is 10
	Priority uint `json:"priority,omitempty"`
}

// IkResponse infomaniak API response
type IkResponse struct {
	// Result of the API call: either "success" or "error"
	Result string `json:"result"`

	// Data is set if API call was successful and contains the actual response
	Data json.RawMessage `json:"data,omitempty"`

	// Error is set if the API call failed and contains all errors that occurred
	Error json.RawMessage `json:"error,omitempty"`
}

// IkDomain infomaniak API domain return type
type IkDomain struct {
	// Domain's ID on infomaniak's side
	ID int `json:"id"`

	// Domain name
	Name string `json:"customer_name"`
}

// IkClient interface to abstract infomaniak client
type IkClient interface {
	// DeleteRecord deletes record with given ID
	DeleteRecord(ctx context.Context, zone string, id string) error

	// CreateOrUpdateRecord creates record if it has no ID property set, otherwise it updates the record with the given ID
	CreateOrUpdateRecord(ctx context.Context, zone string, record IkRecord) (*IkRecord, error)

	// GetDnsRecordsForZone returns all records of the given zone
	GetDnsRecordsForZone(ctx context.Context, zone string) ([]IkRecord, error)
}
