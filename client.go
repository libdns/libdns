package namedotcom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/libdns/libdns"
	"github.com/pkg/errors"
	"io"
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
		p.client = NewNameDotComClient(p.APIToken, p.User, p.APIUrl)
	}
	return nil
}

// listAllRecords returns all records for a zone
func (p *Provider) listAllRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	var (
		err error
		records []libdns.Record
		response *listRecordsResponse

	)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	_ = p.getClient()

	opts := listRecordsRequest{
		DomainName: zone,
		Page:       1,
	}

	for opts.Page > 0 {
		if response, err = p.listRecordsPerPage(ctx, opts); err != nil{
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
	var (
		err error
		method = "GET"
		body io.Reader
		resp = &listRecordsResponse{}
		values = url.Values{}
	)

	endpoint := fmt.Sprintf("/v4/domains/%s/records", opts.DomainName)

	if opts.Page != 0 {
		values.Set("page", fmt.Sprintf("%d", opts.Page))
	}

	if body, err = p.client.doRequest(ctx, method, endpoint + "?" + values.Encode(), nil); err != nil{
		return nil, err
	}


	if err = json.NewDecoder(body).Decode(resp); err != nil{
		return nil, err
	}

	return resp, nil
}

// deleteRecord deletes a record from the zone
func (p *Provider) deleteRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	_ =  p.getClient()

	var (
		err error
		deletedRecord nameDotComRecord
		body io.Reader
		method = "DELETE"
		post = &bytes.Buffer{}
	)

	if record.ID == "" {
		record.ID ,_ = p.getRecordId(ctx, zone, record)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	endpoint := fmt.Sprintf("/v4/domains/%s/records/%s", zone, record.ID)

	deletedRecord.fromLibDNSRecord(record)
	if err = json.NewEncoder(post).Encode(deletedRecord); err != nil {
		return record, err
	}

	if body, err = p.client.doRequest(ctx, method, endpoint, post); err != nil {
		return record, err
	}

	if err = json.NewDecoder(body).Decode(&deletedRecord); err != nil {
		return record, err
	}

	return deletedRecord.toLibDNSRecord(), nil
}

// updateRecord replaces a record with the target record that is passed
func (p *Provider) updateRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	var (
		updatedRecord nameDotComRecord
		method = "PUT"
		body io.Reader
		post = &bytes.Buffer{}
		err error
	)

	_ = p.getClient()

	if record.ID == "" {
		record.ID, err = p.getRecordId(ctx, zone, record)
		if err != nil {
			method = "POST"
		}
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	endpoint := fmt.Sprintf("/v4/domains/%s/records/%s", zone, record.ID)

	updatedRecord.fromLibDNSRecord(record)

	if err = json.NewEncoder(post).Encode(updatedRecord);err != nil {
		return record, err
	}

	if body, err = p.client.doRequest(ctx, method, endpoint, post); err != nil{
		return record, err
	}

	if err = json.NewDecoder(body).Decode(&updatedRecord);err != nil {
		return record, err
	}

	return updatedRecord.toLibDNSRecord(), nil
}

// addRecord creates a new record in the zone.
func (p *Provider) addRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	var (
		err error
		method = "POST"
		body io.Reader
		newRecord nameDotComRecord
		endpoint = fmt.Sprintf("/v4/domains/%s/records", zone)
		post = &bytes.Buffer{}
	)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	_ = p.getClient()


	newRecord.fromLibDNSRecord(record)

	if err := json.NewEncoder(post).Encode(newRecord); err != nil {
		return record, err
	}

	if body, err = p.client.doRequest(ctx, method, endpoint, post); err != nil{
		return record, err
	}

	if err = json.NewDecoder(body).Decode(&newRecord); err != nil{
		return record, err
	}

	return newRecord.toLibDNSRecord(), nil
}

// getRecordId returns the id for the target record if found
func (p *Provider) getRecordId(ctx context.Context, zone string, record libdns.Record) (string, error) {
	_ = p.getClient()

	records, err := p.listAllRecords(ctx, zone)
	if err != nil {
		return "", err
	}

	for _, testRecord := range records {
		if strings.ToLower(record.Name) == strings.ToLower(testRecord.Name) {
			return testRecord.ID, nil
		}
	}
	return "", errors.New("id not found")
}