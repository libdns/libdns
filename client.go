package namedotcom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/libdns/libdns"
	"github.com/pkg/errors"
	"net/url"
	"strings"
	"sync"
)

type Client struct {
	client *nameDotCom
	mutex sync.Mutex
}

// getClient assigns the namedotcom client to the provider
func (p *Provider) getClient() error {
	if p.client == nil {
		p.client = NewNameDotComClient(p.APIToken, p.User, p.Endpoint)
	}
	return nil
}

// listAllRecords returns all records for a zone
func (p *Provider) listAllRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.getClient()

	opts := listRecordsRequest{
		DomainName: zone,
		Page:       1,
	}

	var records []libdns.Record
	for opts.Page > 0 {
		response, err := p.listRecordsPerPage(ctx, opts)
		if err != nil {
			return nil, err
		}

		for _, record := range response.Records {
			records = append(records, record.toLibDNSRecord())
		}

		opts.Page = response.NextPage
	}

	return records, nil

}

// listRecordsPerPage returns all the records for a given page
func (p *Provider) listRecordsPerPage(ctx context.Context, opts listRecordsRequest) (*listRecordsResponse, error) {
	endpoint := fmt.Sprintf("/v4/domains/%s/records", opts.DomainName)

	values := url.Values{}
	if opts.Page != 0 {
		values.Set("page", fmt.Sprintf("%d", opts.Page))
	}

	body, err := p.client.doRequest(ctx, "GET", endpoint + "?" + values.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp := &listRecordsResponse{}

	err = json.NewDecoder(body).Decode(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// deleteRecord deletes a record from the zone
func (p *Provider) deleteRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.getClient()

	if record.ID == "" {
		record.ID ,_ = p.getRecordId(ctx, zone, record)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	var deletedRecord nameDotComRecord
	endpoint := fmt.Sprintf("/v4/domains/%s/records/%s", zone, record.ID)

	deletedRecord.fromLibDNSRecord(record)
	post := &bytes.Buffer{}
	err := json.NewEncoder(post).Encode(deletedRecord)
	if err != nil {
		return libdns.Record{}, err
	}

	body, err := p.client.doRequest(ctx, "DELETE", endpoint, post)
	if err != nil {
		return libdns.Record{}, err
	}

	err = json.NewDecoder(body).Decode(&deletedRecord)
	if err != nil {
		return libdns.Record{}, err
	}

	return deletedRecord.toLibDNSRecord(), nil
}

// updateRecord replaces a record with the target record that is passed
func (p *Provider) updateRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.getClient()

	if record.ID == "" {
		record.ID, _ = p.getRecordId(ctx, zone, record)
	}

	var updateTarget nameDotComRecord
	p.mutex.Lock()
	defer p.mutex.Unlock()

	endpoint := fmt.Sprintf("/v4/domains/%s/records/%s", zone, record.ID)

	post := &bytes.Buffer{}
	updateTarget.fromLibDNSRecord(record)

	err := json.NewEncoder(post).Encode(updateTarget)
	if err != nil {
		return libdns.Record{}, err
	}

	body, err := p.client.doRequest(ctx, "PUT", endpoint, post)
	if err != nil {
		return libdns.Record{}, err
	}

	err = json.NewDecoder(body).Decode(&updateTarget)
	if err != nil {
		return libdns.Record{}, err
	}

	return updateTarget.toLibDNSRecord(), nil
}

// addRecord creates a new record in the zone.
func (p *Provider) addRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.getClient()
	var newRecord nameDotComRecord

	endpoint := fmt.Sprintf("/v4/domains/%s/records", zone)

	newRecord.fromLibDNSRecord(record)
	post := &bytes.Buffer{}
	err := json.NewEncoder(post).Encode(newRecord)
	if err != nil {
		return libdns.Record{}, err
	}

	body, err := p.client.doRequest(ctx, "POST", endpoint, post)
	if err != nil {
		return libdns.Record{}, err
	}

	err = json.NewDecoder(body).Decode(&newRecord)
	if err != nil {
		return libdns.Record{}, err
	}

	return newRecord.toLibDNSRecord(), nil
}

// getRecordId returns the id for the target record if found
func (p *Provider) getRecordId(ctx context.Context, zone string, record libdns.Record) (string, error) {
	p.getClient()
	records, _ := p.listAllRecords(ctx, zone)
	for _, testRecord := range records {
		if strings.ToLower(record.Name) == strings.ToLower(testRecord.Name) {
			return testRecord.ID, nil
		}
	}
	return "", errors.New("id not found")
}