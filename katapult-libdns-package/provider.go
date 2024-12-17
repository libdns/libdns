package katapult

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with Katapult.
type Provider struct {
	APIToken string `json:"api_token,omitempty"`
}

var errFailedToDeleteRecord = errors.New("failed to delete record")

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	url := "/dns_zones/_/records?dns_zone[name]=" + RemoveTrailingDot(zone)

	var apiResponse DNSRecordsAPIResponse
	if err := p.DoRequest(ctx, http.MethodGet, url, nil, &apiResponse); err != nil {
		return nil, err
	}

	var records []libdns.Record
	for _, record := range apiResponse.DNSRecords {
		records = append(records, p.ToLibDNSRecord(record))
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var addedRecords []libdns.Record
	url := "/dns_zones/_/records"

	for _, record := range records {
		properties, err := p.FromLibDNSRecord(record)
		if err != nil {
			return addedRecords, err
		}

		requestBody := map[string]interface{}{
			"dns_zone":   map[string]string{"name": RemoveTrailingDot(zone)},
			"properties": properties,
		}

		var apiResponse DNSRecordAPIResponse
		if err := p.DoRequest(ctx, http.MethodPost, url, requestBody, &apiResponse); err != nil {
			return addedRecords, err
		}

		addedRecords = append(addedRecords, p.ToLibDNSRecord(apiResponse.DNSRecord))
	}

	return addedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var updatedRecords []libdns.Record

	for _, record := range records {
		var url string
		var method string

		if record.ID == "" {
			url = "/dns_zones/_/records"
			method = http.MethodPost
		} else {
			url = "/dns_records/" + record.ID
			method = http.MethodPatch
		}

		properties, err := p.FromLibDNSRecord(record)
		if err != nil {
			return updatedRecords, err
		}

		requestBody := map[string]interface{}{
			"dns_zone":   map[string]string{"name": RemoveTrailingDot(zone)},
			"properties": properties,
		}

		var apiResponse DNSRecordAPIResponse
		if err := p.DoRequest(ctx, method, url, requestBody, &apiResponse); err != nil {
			return updatedRecords, err
		}

		updatedRecords = append(updatedRecords, p.ToLibDNSRecord(apiResponse.DNSRecord))
	}

	return updatedRecords, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var deletedRecords []libdns.Record

	for _, record := range records {
		url := "/dns_records/" + record.ID

		var apiResponse DeletionAPIResponse
		if err := p.DoRequest(ctx, http.MethodDelete, url, nil, &apiResponse); err != nil {
			return deletedRecords, err
		}

		if apiResponse.Deleted {
			deletedRecords = append(deletedRecords, record)
		} else {
			return deletedRecords, fmt.Errorf("%w: %s", errFailedToDeleteRecord, record.ID)
		}
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
