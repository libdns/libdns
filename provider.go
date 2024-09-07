// Package nanelo implements a DNS record management client compatible
// with the libdns interfaces for Nanelo
package nanelo

import (
	"context"
	"fmt"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with Nanelo
type Provider struct {
	APIToken string `json:"api_token,omitempty"`
}

type APIResponse struct {
	OK  bool `json:"ok"`
	Error   *string `json:"error"`
	Result  *map[string]interface{} `json:"result"`
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	baseURL, _ := url.Parse("https://api.nanelo.com/v1")
	baseURL = baseURL.JoinPath(p.APIToken)
	baseURL = baseURL.JoinPath("dns")
	baseURL = baseURL.JoinPath("addrecord")

	for _, rec := range records {
		endpoint := baseURL
		query := endpoint.Query()
		query.Set("domain", zone)
		query.Set("name", rec.Name)
		query.Set("type", rec.Type)
		query.Set("value", rec.Value)
		query.Set("ttl", fmt.Sprintf("%f", rec.TTL.Seconds()))

		if rec.Priority != nil {
			query.Set("priority", fmt.Sprintf("%d", rec.Priority))
		}

		endpoint.RawQuery = query.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		var apiResponse APIResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResponse)
		if err != nil {
			return nil, err
		}

		if apiResponse.Error != nil {
			return nil, fmt.Errorf(*apiResponse.Error)
		}
		if apiResponse.OK == false {
			return nil, fmt.Errorf("Unknown Error when trying to create the DNS Record")
		}
	}
	return records, nil
}
// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	baseURL, _ := url.Parse("https://api.nanelo.com/v1")
	baseURL = baseURL.JoinPath(p.APIToken)
	baseURL = baseURL.JoinPath("dns")
	baseURL = baseURL.JoinPath("deleterecord")
	
	for _, rec := range records {
		endpoint := baseURL
		query := endpoint.Query()
		query.Set("domain", zone)
		query.Set("name", rec.Name)
		query.Set("type", rec.Type)
		query.Set("value", rec.Value)
		query.Set("ttl", fmt.Sprintf("%f", rec.TTL.Seconds()))

		if rec.Priority != nil {
			query.Set("priority", fmt.Sprintf("%d", rec.Priority))
		}

		endpoint.RawQuery = query.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
		if err != nil {
			return nil, err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		var apiResponse APIResponse
		err = json.NewDecoder(resp.Body).Decode(&apiResponse)
		if err != nil {
			return nil, err
		}

		if apiResponse.Error != nil {
			return nil, fmt.Errorf(*apiResponse.Error)
		}
		if apiResponse.OK == false {
			return nil, fmt.Errorf("Unknown Error when trying to create the DNS Record")
		}
	}
	return records, nil
}

// Interface guards
var (
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
