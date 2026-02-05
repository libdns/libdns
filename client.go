package arvancloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

const (
	apiBaseURL = "https://napi.arvancloud.ir/cdn/4.0"
)

// client manages communication with the ArvanCloud API.
type client struct {
	AuthAPIKey string
	BaseURL    string
	httpClient *http.Client
}

// newClient creates a new ArvanCloud API client.
func newClient(authKey string) *client {
	return &client{
		AuthAPIKey: authKey,
		BaseURL:    apiBaseURL,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

func (c *client) getDomains(ctx context.Context) ([]libdns.Zone, error) {

	u := "/domains"
	req, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	var arDomains []arDomain
	_, err = c.do(req, &arDomains)
	if err != nil {
		return nil, err
	}

	zones := make([]libdns.Zone, len(arDomains))
	for i, arDomain := range arDomains {
		zones[i] = libdns.Zone{
			// Add trailing dot to make it a FQDN
			Name: arDomain.Name + ".",
		}
	}

	return zones, nil
}

// getRecords fetches DNS records for a zone.
func (c *client) getRecords(ctx context.Context, zone string) ([]arDNSRecord, error) {
	var records []arDNSRecord
	page := 1

	for {
		u := fmt.Sprintf("/domains/%s/dns-records?page=%d&per_page=100", zone, page)
		req, err := c.newRequest(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}

		var pageRecords []arDNSRecord
		resp, err := c.do(req, &pageRecords)
		if err != nil {
			return nil, err
		}

		records = append(records, pageRecords...)

		if resp.Links.Next == nil || resp.Meta.CurrentPage >= resp.Meta.LastPage || len(pageRecords) == 0 {
			break
		}
		page++
	}
	return records, nil
}

func (p *Provider) findExistingRecords(records []arDNSRecord, rec libdns.Record, zone string) []arDNSRecord {

	arRec, err := arvancloudRecord(rec)
	if err != nil {
		return nil
	}

	var results []arDNSRecord

	for i, r := range records {
		if r.Name == "@" {
			r.Name = zone
		}
		if strings.EqualFold(r.Name, arRec.Name) && strings.EqualFold(r.Type, arRec.Type) {
			results = append(results, records[i])
		}
	}
	return results
}

// createRecord creates a new DNS record.
func (c *client) createRecord(ctx context.Context, zone string, record libdns.Record) (arDNSRecord, error) {

	arRec, err := arvancloudRecord(record)
	if err != nil {
		return arDNSRecord{}, err
	}

	// fmt.Println(string(jsonBytes))
	u := fmt.Sprintf("/domains/%s/dns-records", zone)
	req, err := c.newRequest(ctx, http.MethodPost, u, arRec)
	if err != nil {
		return arDNSRecord{}, err
	}

	var resp arDNSRecord
	if _, err := c.do(req, &resp); err != nil {
		return arDNSRecord{}, err
	}
	return resp, nil
}

// deleteRecord deletes a DNS record.
func (c *client) deleteRecord(ctx context.Context, zone string, recordID string) (arDNSRecord, error) {
	u := fmt.Sprintf("/domains/%s/dns-records/%s", zone, recordID)
	req, err := c.newRequest(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return arDNSRecord{}, err
	}

	var resp arDNSRecord
	if _, err := c.do(req, &resp); err != nil {
		return arDNSRecord{}, err
	}

	return resp, nil
}

// updateRecord updates a DNS record.
func (c *client) updateRecord(ctx context.Context, zone string, recordID string, record arDNSRecord) (arDNSRecord, error) {
	u := fmt.Sprintf("/domains/%s/dns-records/%s", zone, recordID)
	req, err := c.newRequest(ctx, http.MethodPut, u, record)
	if err != nil {
		return arDNSRecord{}, err
	}

	var resp arDNSRecord
	if _, err := c.do(req, &resp); err != nil {
		return arDNSRecord{}, err
	}

	return resp, nil
}

func (c *client) do(req *http.Request, result any) (arResponse, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return arResponse{}, err
	}
	defer resp.Body.Close()

	var respData arResponse

	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return arResponse{}, err
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return arResponse{}, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if len(respData.Errors) > 0 {
		return arResponse{}, fmt.Errorf("got errors: HTTP %d: %+v", resp.StatusCode, respData.Errors)
	}

	if len(respData.Data) > 0 && result != nil {
		err = json.Unmarshal(respData.Data, result)
		if err != nil {
			return arResponse{}, err
		}
		respData.Data = nil
	}

	return respData, err
}

func (c *client) newRequest(ctx context.Context, method, url string, payload any) (*http.Request, error) {
	var body []byte
	var req *http.Request
	var err error

	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequestWithContext(ctx, method, c.BaseURL+url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
	} else {
		req, err = http.NewRequestWithContext(ctx, method, c.BaseURL+url, nil)
		if err != nil {
			return nil, err
		}

	}
	req.Header.Set("Authorization", "Apikey "+c.AuthAPIKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func unwrapContent(content string) string {
	if strings.HasPrefix(content, `"`) && strings.HasSuffix(content, `"`) {
		content = strings.TrimPrefix(strings.TrimSuffix(content, `"`), `"`)
	}
	return content
}

