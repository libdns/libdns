package namedotcom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/libdns/libdns"
	"io"
	"sync"
)

type Client struct {
	client *nameDotCom
	mutex  sync.Mutex
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
		records []libdns.Record
		err     error

		method  = "GET"
		body    io.Reader
		resp    = &listRecordsResponse{}
		reqPage = 1
	)

	for reqPage > 0 {
		endpoint := fmt.Sprintf("/v4/domains/%s/records", zone)

		if reqPage != 0 {
			if body, err = p.client.doRequest(ctx, method, endpoint+"?page="+fmt.Sprint(reqPage), nil); err != nil {
				return nil, err
			}

			if err = json.NewDecoder(body).Decode(resp); err != nil {
				return nil, err
			}

			for _, record := range resp.Records {
				records = append(records, record.toLibDNSRecord())
			}

			reqPage = int(resp.NextPage)
		}
	}

	return records, nil
}

// deleteRecord deletes a record from the zone
func (p *Provider) deleteRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	_ = p.getClient()

	var (
		deletedRecord nameDotComRecord
		err           error

		method = "DELETE"
		body   io.Reader
		post   = &bytes.Buffer{}
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

// upsertRecord replaces a record with the target record or creates a new record if update target not found
func (p *Provider) upsertRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	_ = p.getClient()

	var (
		upsertedRecord nameDotComRecord
		err            error

		method = "PUT"
		body   io.Reader
		post   = &bytes.Buffer{}
	)

	if record.ID == "" {
		method = "POST"
	}

	endpoint := fmt.Sprintf("/v4/domains/%s/records/%s", zone, record.ID)

	upsertedRecord.fromLibDNSRecord(record)

	if err = json.NewEncoder(post).Encode(upsertedRecord); err != nil {
		return record, err
	}

	if body, err = p.client.doRequest(ctx, method, endpoint, post); err != nil {
		return record, err
	}

	if err = json.NewDecoder(body).Decode(&upsertedRecord); err != nil {
		return record, err
	}

	return upsertedRecord.toLibDNSRecord(), nil
}
