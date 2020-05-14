package route53

import (
	"context"
	"time"

	r53 "github.com/aws/aws-sdk-go/service/route53"
	"github.com/libdns/libdns"
)

// Provider implements the libdns interfaces for Route53
type Provider struct {
	MaxRetries int `json:"max_retries,omitempty"`
	client     *r53.Route53
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	zoneID, err := p.getZoneID(ctx, zone)
	if err != nil {
		return nil, err
	}

	records, err := p.getRecords(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	zoneID, err := p.getZoneID(ctx, zone)
	if err != nil {
		return nil, err
	}

	var createdRecords []libdns.Record

	for _, record := range records {
		newRecord, err := p.createRecord(ctx, zoneID, record)
		if err != nil {
			return nil, err
		}
		newRecord.TTL = time.Duration(newRecord.TTL) * time.Second
		createdRecords = append(createdRecords, newRecord)
	}

	return createdRecords, nil
}

// DeleteRecords deletes the records from the zone. If a record does not have an ID,
// it will be looked up. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	zoneID, err := p.getZoneID(ctx, zone)
	if err != nil {
		return nil, err
	}

	var deletedRecords []libdns.Record

	for _, record := range records {
		deletedRecord, err := p.deleteRecord(ctx, zoneID, record)
		if err != nil {
			return nil, err
		}
		deletedRecord.TTL = time.Duration(deletedRecord.TTL) * time.Second
		deletedRecords = append(deletedRecords, deletedRecord)
	}

	return deletedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records
// or creating new ones. It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	zoneID, err := p.getZoneID(ctx, zone)
	if err != nil {
		return nil, err
	}

	var updatedRecords []libdns.Record

	for _, record := range records {
		updatedRecord, err := p.updateRecord(ctx, zoneID, record)
		if err != nil {
			return nil, err
		}
		updatedRecord.TTL = time.Duration(updatedRecord.TTL) * time.Second
		updatedRecords = append(updatedRecords, updatedRecord)
	}

	return updatedRecords, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
