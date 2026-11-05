package domainchief

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is the base URL of the Domain Chief API.
const DefaultBaseURL = "https://domain.chief.app/api/v1"

const (
	maxAttempts  = 3
	retryBackoff = time.Second
	perPage      = 100
	modulePath   = "github.com/libdns/domainchief"
)

// APIError is returned when the Domain Chief API responds with an unexpected status code.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int

	// Code is the machine readable error code, if the API returned one (for example `dns.not_hosted`).
	Code string

	// Message is the human readable error message, including validation errors.
	Message string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("domain chief API error (%d, %s): %s", e.StatusCode, e.Code, e.Message)
	}

	return fmt.Sprintf("domain chief API error (%d): %s", e.StatusCode, e.Message)
}

// retryable reports whether the request may succeed when sent again.
//
// A conflict is only retried when another write for the same domain is still running,
// not when it is permanent (e.g. the domain does not use hosted DNS). Server errors are
// only retried for idempotent requests, so a create is never applied twice.
func (e *APIError) retryable(idempotent bool) bool {
	switch {
	case e.StatusCode == http.StatusConflict:
		return e.Code == "dns.zone_busy" || e.Code == "dns.zone_changed"
	case e.StatusCode == http.StatusTooManyRequests:
		return true
	case e.StatusCode >= http.StatusInternalServerError:
		return idempotent
	default:
		return false
	}
}

func isNotFound(err error) bool {
	var apiErr *APIError

	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// apiRecord is a DNS record as returned by the Domain Chief API.
type apiRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Value   string `json:"value"`
	TTL     int    `json:"ttl"`
	Prio    int    `json:"prio"`
}

// recordInput is the request body to create or update a DNS record.
type recordInput struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Prio    int    `json:"prio"`
}

// apiChallenge is an ACME challenge as returned by the Domain Chief API.
type apiChallenge struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

type apiDomain struct {
	Domain           string `json:"domain"`
	IsUsingHostedDNS bool   `json:"is_using_hosted_dns"`
}

type pagination struct {
	Meta struct {
		CurrentPage int `json:"current_page"`
		LastPage    int `json:"last_page"`
	} `json:"meta"`
}

func (p *Provider) baseURL() string {
	if p.BaseURL == "" {
		return DefaultBaseURL
	}

	return strings.TrimRight(p.BaseURL, "/")
}

func (p *Provider) httpClient() *http.Client {
	if p.HTTPClient != nil {
		return p.HTTPClient
	}

	return defaultHTTPClient
}

var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

func domainPath(zone string) string {
	return "/domains/" + url.PathEscape(zoneDomain(zone))
}

func (p *Provider) listRecords(ctx context.Context, zone string) ([]apiRecord, error) {
	var records []apiRecord

	for page := 1; ; page++ {
		var response struct {
			pagination
			Data []apiRecord `json:"data"`
		}

		query := url.Values{
			"page":     {strconv.Itoa(page)},
			"per_page": {strconv.Itoa(perPage)},
		}

		if err := p.do(ctx, http.MethodGet, domainPath(zone)+"/dns/records?"+query.Encode(), nil, true, &response); err != nil {
			return nil, err
		}

		records = append(records, response.Data...)

		if response.Meta.LastPage <= page || len(response.Data) == 0 {
			return records, nil
		}
	}
}

func (p *Provider) createRecord(ctx context.Context, zone string, input recordInput) (apiRecord, error) {
	var response struct {
		Data apiRecord `json:"data"`
	}

	err := p.do(ctx, http.MethodPost, domainPath(zone)+"/dns/records", input, false, &response)

	return response.Data, err
}

func (p *Provider) updateRecord(ctx context.Context, zone, id string, input recordInput) (apiRecord, error) {
	var response struct {
		Data apiRecord `json:"data"`
	}

	err := p.do(ctx, http.MethodPut, domainPath(zone)+"/dns/records/"+url.PathEscape(id), input, true, &response)

	return response.Data, err
}

func (p *Provider) deleteRecord(ctx context.Context, zone, id string) error {
	err := p.do(ctx, http.MethodDelete, domainPath(zone)+"/dns/records/"+url.PathEscape(id), nil, true, nil)

	if isNotFound(err) {
		return nil
	}

	return err
}

func (p *Provider) presentChallenge(ctx context.Context, fqdn, value string) (apiChallenge, error) {
	var response struct {
		Data apiChallenge `json:"data"`
	}

	err := p.do(ctx, http.MethodPost, "/acme/challenges", map[string]string{
		"name":  fqdn,
		"value": value,
	}, true, &response)

	return response.Data, err
}

func (p *Provider) cleanUpChallenge(ctx context.Context, fqdn, value string) error {
	query := url.Values{
		"name":  {fqdn},
		"value": {value},
	}

	return p.do(ctx, http.MethodDelete, "/acme/challenges?"+query.Encode(), nil, true, nil)
}

func (p *Provider) listDomains(ctx context.Context) ([]apiDomain, error) {
	var domains []apiDomain

	for page := 1; ; page++ {
		var response struct {
			pagination
			Data []apiDomain `json:"data"`
		}

		query := url.Values{
			"page":     {strconv.Itoa(page)},
			"per_page": {strconv.Itoa(perPage)},
		}

		if err := p.do(ctx, http.MethodGet, "/domains?"+query.Encode(), nil, true, &response); err != nil {
			return nil, err
		}

		domains = append(domains, response.Data...)

		if response.Meta.LastPage <= page || len(response.Data) == 0 {
			return domains, nil
		}
	}
}

// do sends the request, retrying when another DNS write for the same domain is still running.
func (p *Provider) do(ctx context.Context, method, path string, body any, idempotent bool, out any) error {
	var payload []byte

	if body != nil {
		var err error

		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}

	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = p.send(ctx, method, path, payload, out)

		var apiErr *APIError
		if !errors.As(err, &apiErr) || !apiErr.retryable(idempotent) || attempt == maxAttempts {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt) * retryBackoff):
		}
	}

	return err
}

func (p *Provider) send(ctx context.Context, method, path string, payload []byte, out any) error {
	if p.APIToken == "" {
		return errors.New("domain chief: missing API token")
	}

	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL()+path, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIToken)
	req.Header.Set("User-Agent", userAgent)

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := p.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("domain chief API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseError(resp)
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, resp.Body)

		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("domain chief API returned an invalid response: %w", err)
	}

	return nil
}

func parseError(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	apiErr := &APIError{StatusCode: resp.StatusCode}

	var decoded struct {
		Message string              `json:"message"`
		Code    string              `json:"code"`
		Errors  map[string][]string `json:"errors"`
	}

	if json.Unmarshal(raw, &decoded) == nil && decoded.Message != "" {
		apiErr.Code = decoded.Code
		apiErr.Message = decoded.Message

		for field, messages := range decoded.Errors {
			apiErr.Message += fmt.Sprintf(" [%s: %s]", field, strings.Join(messages, " "))
		}
	} else {
		apiErr.Message = strings.TrimSpace(string(raw))
	}

	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}

	return apiErr
}

var userAgent = buildUserAgent()

func buildUserAgent() string {
	version := "devel"

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == modulePath && dep.Version != "" {
				version = dep.Version
			}
		}
	}

	return "libdns-domainchief/" + version + " (+https://" + modulePath + ")"
}
