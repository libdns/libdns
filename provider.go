package dynv6

import (
	"context"
	"fmt"

	"github.com/libdns/libdns"
)

// Provider for dynv6 HTTP REST API
type Provider struct {
	// Token is required for authorization.
	// You can generate one at: https://dynv6.com/keys
	Token string
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	zoneDetails, err := p.getZoneByName(ctx, zone)
	if err != nil {
		return nil, err
	}
	dynv6Records, err := p.getRecords(ctx, zoneDetails.ID)
	if err != nil {
		return nil, err
	}
	var recs []libdns.Record
	for _, r := range dynv6Records {
		recs = append(recs, r.toLibdnsRecord())
	}
	return recs, nil
}

// AppendRecords adds records to the zone and returns the records that were created.
func (p *Provider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	zoneDetails, err := p.getZoneByName(ctx, zone)
	if err != nil {
		return nil, err
	}
	results := []libdns.Record{}
	for _, r := range recs {
		dynv6Rec, err := fromLibdnsRecord(zone, &r)
		if err != nil {
			return results, err
		}
		result, err := p.addRecord(ctx, zoneDetails.ID, dynv6Rec)
		if err != nil {
			return results, err
		}
		results = append(results, result.toLibdnsRecord())
	}
	return results, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones, and returns the records that were updated.
func (p *Provider) SetRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	zoneDetails, err := p.getZoneByName(ctx, zone)
	if err != nil {
		return nil, err
	}
	existingRecords, err := p.getRecords(ctx, zoneDetails.ID)
	if err != nil {
		return nil, err
	}
	results := []libdns.Record{}
	for _, r := range recs {
		existingRecord := findRecord(existingRecords, &r)
		var result *record
		if existingRecord != nil {
			//record found, update it
			updateRecord := *existingRecord
			updateRecord.Data = r.Value
			result, err = p.updateRecord(ctx, zoneDetails.ID, &updateRecord)
			if err != nil {
				return results, err
			}
		} else {
			//no record found, add a new one
			newRecord, err := fromLibdnsRecord(zone, &r)
			if err != nil {
				return results, err
			}
			result, err = p.addRecord(ctx, zoneDetails.ID, newRecord)
			if err != nil {
				return results, err
			}
		}
		results = append(results, result.toLibdnsRecord())
	}
	return results, nil
}

// DeleteRecords deletes records from the zone and returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	zoneDetails, err := p.getZoneByName(ctx, zone)
	if err != nil {
		return nil, err
	}
	existingRecords, err := p.getRecords(ctx, zoneDetails.ID)
	if err != nil {
		return nil, err
	}
	results := []libdns.Record{}
	for _, r := range recs {
		existingRecord := findRecordWithValue(existingRecords, &r)
		if existingRecord == nil {
			return results, fmt.Errorf("Record not found: %+v", r)
		}
		err = p.deleteRecord(ctx, zoneDetails.ID, existingRecord.ID)
		if err != nil {
			return results, err
		}
		results = append(results, r)
	}
	return results, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
