// Package libdnstemplate implements a DNS record management client compatible
// with the libdns interfaces for mijn.host.
package mijnhost

import (
	"context"
	"fmt"
	"net/http"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with mijn.host.
type Provider struct {
	ApiKey string `json:"api_token,omitempty"`
	ApiURL string `json:"api_url,omitempty"`
}

func (p *Provider) setDefaults() {
	if p.ApiURL == "" {
		p.ApiURL = "https://mijn.host/api/v2"
	}
}

func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.setDefaults()

	zone = normalizeZone(zone)
	reqURL := fmt.Sprintf("%s/domains/%s/dns", p.ApiURL, zone)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	var result RecordsResponse
	err = p.doAPIRequest(req, &result)

	recs := make([]libdns.Record, 0, len(result.Data.Records))
	for _, r := range result.Data.Records {
		recs = append(recs, r.libDNSRecord(zone))
	}

	return recs, err
}

func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.setDefaults()

	zone = normalizeZone(zone)

	var results []libdns.Record
	for _, record := range records {
		_, err := p.updateRecord(ctx, zone, record)
		if err != nil {
			return nil, err
		}

		results = append(results, record)
	}

	return results, nil
}

func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.setDefaults()

	zone = normalizeZone(zone)

	// The api does not support deleting records, so we retrieve all of them, and update the whole set
	// without the removed ones

	allRecords, err := p.GetRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	var filteredRecords []libdns.Record

	for _, record := range allRecords {
		shouldRemove := false
		for _, r := range records {
			if record.Type == r.Type && record.Name == r.Name && record.Value == r.Value {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			filteredRecords = append(filteredRecords, record)
		}
	}

	// Now call the update endpoint to set all records.
	err = p.replaceRecords(ctx, zone, filteredRecords)

	if err != nil {
		return nil, err
	}

	return records, err
}

func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.setDefaults()

	zone = normalizeZone(zone)

	var results []libdns.Record
	var resultErr error
	for _, libRecord := range records {

		_, err := p.updateRecord(ctx, zone, libRecord)
		if err != nil {
			resultErr = err
		}
		results = append(results, libRecord)

	}

	return results, resultErr
}

var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
