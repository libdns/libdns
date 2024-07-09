// Direct implementation of all required do.de DNS API endpoints

package dode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const baseURl = "https://www.do.de/api"

type ApiResponse struct {
	Domain  *string `json:"domain"`
	Success *bool   `json:"success"`
	Error   *string `json:"error"`
}

func (p *Provider) createACMERecord(ctx context.Context, domain string, value string) error {
	baseURL, _ := url.Parse(baseURl)
	endpoint := baseURL.JoinPath("letsencrypt")

	query := endpoint.Query()
	query.Set("token", p.APIToken)
	query.Set("domain", domain)
	query.Set("value", value)

	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	var apiResponse ApiResponse
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&apiResponse)
	if err != nil {
		return err
	}

	if apiResponse.Error != nil {
		return fmt.Errorf(*apiResponse.Error)
	}
	if apiResponse.Success == nil || !*apiResponse.Success {
		return fmt.Errorf("creating the ACME record was not successfull")
	}

	return nil
}

func (p *Provider) deleteACMERecord(ctx context.Context, domain string) error {
	baseURL, _ := url.Parse(baseURl)
	endpoint := baseURL.JoinPath("letsencrypt")

	query := endpoint.Query()
	query.Set("token", p.APIToken)
	query.Set("domain", domain)
	query.Set("action", "delete")

	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	var apiResponse ApiResponse
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&apiResponse)
	if err != nil {
		return err
	}

	if apiResponse.Error != nil {
		return fmt.Errorf(*apiResponse.Error)
	}
	if apiResponse.Success == nil || !*apiResponse.Success {
		return fmt.Errorf("creating the ACME record was not successfull")
	}

	return nil
}
