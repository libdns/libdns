package infomaniak

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/libdns/libdns"
)

// Base URL to infomaniak API
const apiBaseUrl = "https://api.infomaniak.com"

// URL of DNS record endpoint
const apiDnsRecord = apiBaseUrl + "/1/domain/%d/dns/record"

// Client that abstracts and calls infomaniak API
type Client struct {
	// infomaniak API token
	Token string

	// http client used for requests
	HttpClient *http.Client

	// cache of domains registered for the
	// current infomaniak account to prevent
	// that we have to load them for each request
	domains *[]IkDomain

	// mutex to prevent race conditions
	mu sync.Mutex
}

// GetDnsRecordsForZone loads all dns records for a given zone
func (c *Client) GetDnsRecordsForZone(ctx context.Context, zone string) ([]IkRecord, error) {
	domain, err := c.getDomainForZone(ctx, zone)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(apiDnsRecord, domain.ID), nil)
	if err != nil {
		return nil, err
	}

	var dnsRecords []IkRecord
	_, err = c.doRequest(req, &dnsRecords)
	if err != nil {
		return nil, err
	}

	zoneRecords := make([]IkRecord, 0)
	for _, rec := range dnsRecords {
		if bytes.Contains([]byte(rec.SourceIdn), []byte(zone)) {
			zoneRecords = append(zoneRecords, rec)
		}
	}
	return zoneRecords, nil
}

// CreateOrUpdateRecord creates a record if its Id property is not set, otherwise it updates the record
func (c *Client) CreateOrUpdateRecord(ctx context.Context, zone string, record IkRecord) (*IkRecord, error) {
	domain, err := c.getDomainForZone(ctx, zone)
	if err != nil {
		return nil, err
	}
	record.Source = libdns.RelativeName(record.SourceIdn, domain.Name)

	rawJson, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}

	var method = http.MethodPost
	var endpoint = fmt.Sprintf(apiDnsRecord, domain.ID)
	if record.ID != "" {
		endpoint += "/" + record.ID
		method = http.MethodPut
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewBuffer(rawJson))
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(req, nil)
	if err != nil {
		return nil, err
	}

	if record.ID == "" {
		var idString string
		err = json.Unmarshal(resp.Data, &idString)
		if err != nil {
			return nil, err
		}
		record.ID = idString
	}
	return &record, nil
}

// DeleteRecord deletes an existing dns record for a given zone
func (c *Client) DeleteRecord(ctx context.Context, zone string, id string) error {
	domain, err := c.getDomainForZone(ctx, zone)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf(apiDnsRecord, domain.ID)+"/"+id, nil)
	if err != nil {
		return err
	}
	_, err = c.doRequest(req, nil)
	return err
}

// getDomainForZone looks for the domain that this zone is under
func (c *Client) getDomainForZone(ctx context.Context, zone string) (IkDomain, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.domains == nil {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBaseUrl+"/1/product?service_name=domain", nil)
		if err != nil {
			return IkDomain{}, err
		}
		var domains []IkDomain
		_, err = c.doRequest(req, &domains)
		if err != nil {
			return IkDomain{}, err
		}
		c.domains = &domains
	}
	for _, domain := range *c.domains {
		if bytes.Contains([]byte(zone), []byte(domain.Name)) {
			return domain, nil
		}
	}
	return IkDomain{}, fmt.Errorf("Could not find a domain name for zone %s in listed services", zone)
}

// doRequest performs the API call for the given request req and parses the response's data to the given data struct - if the parameter is not nil
func (c *Client) doRequest(req *http.Request, data interface{}) (*IkResponse, error) {
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	rawResp, err := c.HttpClient.Do(req)

	if err != nil {
		return nil, err
	}
	defer rawResp.Body.Close()

	var resp IkResponse
	err = json.NewDecoder(rawResp.Body).Decode(&resp)
	if err != nil {
		return nil, err
	}

	if rawResp.StatusCode >= 400 || resp.Result != "success" {
		return nil, fmt.Errorf("got errors: HTTP %d: %+v", rawResp.StatusCode, string(resp.Error))
	}

	if data != nil {
		err = json.Unmarshal(resp.Data, data)
		if err != nil {
			return nil, err
		}
	}

	return &resp, nil
}
