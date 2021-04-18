// Package namedotcom implements a DNS record management client compatible
// with the libdns interfaces for namedotcom.

package namedotcom

import (
	"context"
	"github.com/libdns/libdns"
	"time"
)


// Provider implements the libdns interface for namedotcom
type Provider struct {
	Client
	APIToken string `json:"api_token,omitempty"`
	User string `json:"user,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	records, err := p.listAllRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var appendedRecords []libdns.Record

	for _, record := range records {
		newRecord, err := p.addRecord(ctx, zone, record)
		if err != nil {
			return nil, err
		}
		newRecord.TTL = newRecord.TTL * time.Second
		appendedRecords = append(appendedRecords, newRecord)
	}

	return appendedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var setRecords []libdns.Record

	for _, record := range records {
		setRecord, err := p.updateRecord(ctx, zone, record)
		if err != nil {
			return setRecords, err
		}
		setRecord.TTL = setRecord.TTL * time.Second
		setRecords = append(setRecords, setRecord)
	}

	return setRecords, nil
}


// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var deletedRecords []libdns.Record

	for _, record := range records {
		deletedRecord, err := p.deleteRecord(ctx, zone, record)
		if err != nil {
			return nil, err
		}
		deletedRecord.TTL = deletedRecord.TTL * time.Second
		deletedRecords = append(deletedRecords, deletedRecord)
	}

	return deletedRecords, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
