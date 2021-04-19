package namedotcom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/libdns/libdns"
	"io"
	"net/url"
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
	p.mutex.Lock()
	defer p.mutex.Unlock()

	_ = p.getClient()

	var (
		err error
		records []libdns.Record
		method = "GET"
		body io.Reader
		resp = &listRecordsResponse{}
		values = url.Values{}
	)

	opts := listRecordsRequest{
		DomainName: zone,
		Page:       1,
	}

	for opts.Page > 0 {
		endpoint := fmt.Sprintf("/v4/domains/%s/records", opts.DomainName)

		if opts.Page != 0 {
			values.Set("page", fmt.Sprintf("%d", opts.Page))
			if body, err = p.client.doRequest(ctx, method, endpoint + "?" + values.Encode(), nil); err != nil{
				return nil, err
			}

			if err = json.NewDecoder(body).Decode(resp); err != nil{
				return nil, err
			}

			for _, record := range resp.Records {
				records = append(records, record.toLibDNSRecord())
			}

			opts.Page = resp.NextPage
		}
	}

	return records, nil
}



// deleteRecord deletes a record from the zone
func (p *Provider) deleteRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	_ =  p.getClient()

	var (
		err error
		deletedRecord nameDotComRecord
		body io.Reader
		method = "DELETE"
		post = &bytes.Buffer{}
	)

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
	p.mutex.Lock()
	defer p.mutex.Unlock()

	_ = p.getClient()

	var (
		updatedRecord nameDotComRecord
		method = "PUT"
		body io.Reader
		post = &bytes.Buffer{}
		err error
	)

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
	p.mutex.Lock()
	defer p.mutex.Unlock()

	_ = p.getClient()

	var (
		err error
		method = "POST"
		body io.Reader
		newRecord nameDotComRecord
		endpoint = fmt.Sprintf("/v4/domains/%s/records", zone)
		post = &bytes.Buffer{}
	)


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