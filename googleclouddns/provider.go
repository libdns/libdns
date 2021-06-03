// Package googleclouddns implements a DNS record management client compatible
// with the libdns interfaces for Google Cloud DNS.
package googleclouddns

import (
	"context"
	"sync"
	"time"

	"github.com/libdns/libdns"
	"google.golang.org/api/dns/v1"
)

// Provider facilitates DNS record manipulation with Google Cloud DNS.
type Provider struct {
	Project            string `json:"gcp_project,omitempty"`
	ServiceAccountJSON string `json:"gcp_application_default,omitempty"`
	service            *dns.Service
	zoneMap            map[string]string
	zoneMapLastUpdated time.Time
	mutex              sync.Mutex
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	return p.getCloudDNSRecords(ctx, zone)
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return p.applyRecords(ctx, zone, records, false)
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return p.applyRecords(ctx, zone, records, true)
}

func (p *Provider) applyRecords(ctx context.Context, zone string, records []libdns.Record, update bool) ([]libdns.Record, error) {
	googleDNSRecords := groupRecords(records)
	setRecords := make([]libdns.Record, 0)
	for _, recordData := range googleDNSRecords {
		sr, err := p.setCloudDNSRecord(ctx, zone, recordData.values, update)
		if err != nil {
			return setRecords, err
		}
		setRecords = append(setRecords, sr...)
	}
	return setRecords, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	googleDNSRecords := groupRecords(records)
	recordsToReturn := make([]libdns.Record, 0)
	for recordName, recordData := range googleDNSRecords {
		err := p.deleteCloudDNSRecord(ctx, zone, recordName, recordData.recordType)
		if err != nil {
			return recordsToReturn, err
		}
		recordsToReturn = append(recordsToReturn, recordData.values...)
	}
	return recordsToReturn, nil
}

type googleDNSRecord struct {
	recordType string
	values     []libdns.Record
}

// groupRecords combines libdns.Record entries into a single googleDNSRecord to ensure
// the values are sent at the same time to Google Cloud.
func groupRecords(records []libdns.Record) map[string]googleDNSRecord {
	gdrs := make(map[string]googleDNSRecord)
	for _, record := range records {
		if gdr, ok := gdrs[record.Name]; !ok {
			gdrs[record.Name] = googleDNSRecord{
				recordType: record.Type,
				values:     []libdns.Record{record},
			}
		} else {
			gdr.values = append(gdr.values, record)
			gdrs[record.Name] = gdr
		}
	}
	return gdrs
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
