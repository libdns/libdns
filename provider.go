// Package leaseweb implements a DNS record management client compatible
// with the libdns interfaces for Leaseweb.
// Upstream documentation found at:
// https://developer.leaseweb.com/api-docs/domains_v2.html
package leaseweb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/libdns/libdns"
)

const (
	LeasewebApiKeyHeader = "X-LSW-Auth"
)

// Provider facilitates DNS record manipulation with Leaseweb.
type Provider struct {
	// Leasewebs API key. Generate one in the Leaseweb customer portal -> Administration -> API Key
	APIKey string `json:"api_token,omitempty"`
	mutex  sync.Mutex
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.leaseweb.com/hosting/v2/domains/%s/resourceRecordSets", zone), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add(LeasewebApiKeyHeader, p.APIKey)

	res, err := client.Do(req)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("Received StatusCode %d from Leaseweb API", res.StatusCode)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var recordSets LeasewebRecordSets
	json.Unmarshal([]byte(data), &recordSets)

	var records []libdns.Record

	for _, resourceRecordSet := range recordSets.ResourceRecordSets {
		for _, content := range resourceRecordSet.Content {
			record := libdns.Record{
				Name:  resourceRecordSet.Name,
				Value: content,
				Type:  resourceRecordSet.Type,
				TTL:   time.Duration(resourceRecordSet.TTL) * time.Second,
			}
			records = append(records, record)
		}
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	client := &http.Client{}

	var addedRecords []libdns.Record

	for _, record := range records {
		body := &LeasewebRecordSet{
			Name:    record.Name,
			Type:    record.Type,
			Content: []string{record.Value},
			TTL:     int(record.TTL.Seconds()),
		}

		bodyBuffer := new(bytes.Buffer)
		json.NewEncoder(bodyBuffer).Encode(body)

		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.leaseweb.com/hosting/v2/domains/%s/resourceRecordSets", zone), bodyBuffer)
		if err != nil {
			return nil, err
		}

		req.Header.Add(LeasewebApiKeyHeader, p.APIKey)

		res, err := client.Do(req)
		defer res.Body.Close()
		if err != nil {
			return nil, err
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			return nil, fmt.Errorf("Received StatusCode %d from Leaseweb API", res.StatusCode)
		}

		addedRecords = append(addedRecords, record)
	}

	return addedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	client := &http.Client{}

	var updatedRecords []libdns.Record

	var resourceRecordSets []LeasewebRecordSet

	for _, record := range records {

		recordSet := LeasewebRecordSet{
			Name:    record.Name,
			Type:    record.Type,
			Content: []string{record.Value},
			TTL:     int(record.TTL.Seconds()),
		}

		resourceRecordSets = append(resourceRecordSets, recordSet)

		updatedRecords = append(updatedRecords, record)
	}

	body := &LeasewebRecordSets{
		ResourceRecordSets: resourceRecordSets,
	}

	bodyBuffer := new(bytes.Buffer)
	json.NewEncoder(bodyBuffer).Encode(body)

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("https://api.leaseweb.com/hosting/v2/domains/%s/resourceRecordSets", zone), bodyBuffer)
	if err != nil {
		return nil, err
	}

	req.Header.Add(LeasewebApiKeyHeader, p.APIKey)

	res, err := client.Do(req)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, fmt.Errorf("Received StatusCode %d from Leaseweb API", res.StatusCode)
	}

	return updatedRecords, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	client := &http.Client{}

	var deletedRecords []libdns.Record

	for _, record := range records {
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("https://api.leaseweb.com/hosting/v2/domains/%s/resourceRecordSets/%s/%s", zone, record.Name, record.Type), nil)
		if err != nil {
			return nil, err
		}

		req.Header.Add(LeasewebApiKeyHeader, p.APIKey)

		res, err := client.Do(req)
		defer res.Body.Close()
		if err != nil {
			return nil, err
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			return nil, fmt.Errorf("Received StatusCode %d from Leaseweb API", res.StatusCode)
		}

		deletedRecords = append(deletedRecords, record)
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
