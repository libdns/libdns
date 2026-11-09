package binarylane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultAPIBase is the root of Binary Lane's API, including the version
// segment every endpoint sits under.
const DefaultAPIBase = "https://api.binarylane.com.au/v2/"

// requestTimeout bounds a single API call. Callers control overall deadlines
// through the context; this only stops one request hanging indefinitely.
const requestTimeout = 30 * time.Second

// perPage is the page size used when listing. The API's default is small
// enough that a zone of any size would need several round trips.
const perPage = 200

// apiRecord is a DNS record as the API represents it. Binary Lane keeps parts
// of the RDATA of some types in their own fields rather than in Data; see
// model.go. Priority, Port, Weight and Flags deliberately omit "omitempty",
// because zero is a meaningful value for each of them — an MX preference of 0
// or CAA flags of 0 must be sent, not dropped.
type apiRecord struct {
	ID       int64  `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	Data     string `json:"data,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
	Priority int    `json:"priority"`
	Port     int    `json:"port"`
	Weight   int    `json:"weight"`
	Flags    int    `json:"flags"`
	Tag      string `json:"tag,omitempty"`
}

type recordResponse struct {
	DomainRecord apiRecord `json:"domain_record"`
}

type recordsResponse struct {
	DomainRecords []apiRecord `json:"domain_records"`
	Meta          struct {
		Total int `json:"total"`
	} `json:"meta"`
}

type domainsResponse struct {
	Domains []struct {
		Name string `json:"name"`
	} `json:"domains"`
	Meta struct {
		Total int `json:"total"`
	} `json:"meta"`
}

// apiError is the problem-details body the API returns for a failed request.
type apiError struct {
	Status int                 `json:"status"`
	Title  string              `json:"title"`
	Detail string              `json:"detail"`
	Errors map[string][]string `json:"errors"`
}

func (e *apiError) Error() string {
	msg := new(strings.Builder)

	fmt.Fprintf(msg, "binarylane: HTTP %d", e.Status)

	if e.Title != "" {
		fmt.Fprintf(msg, ": %s", e.Title)
	}
	if e.Detail != "" {
		fmt.Fprintf(msg, ": %s", e.Detail)
	}

	for field, problems := range e.Errors {
		fmt.Fprintf(msg, "; %s: %s", field, strings.Join(problems, ", "))
	}

	return msg.String()
}

// baseURL resolves the configured endpoint, defaulting and normalising it.
// Callers must hold p.mu.
func (p *Provider) baseURL() (*url.URL, error) {
	raw := p.APIBase
	if raw == "" {
		raw = DefaultAPIBase
	}
	if !strings.HasSuffix(raw, "/") {
		// Request paths are resolved relative to this URL, so without a
		// trailing slash the final segment would be discarded.
		raw += "/"
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("binarylane: invalid APIBase %q: %w", p.APIBase, err)
	}

	return u, nil
}

// do performs one API call, decoding a JSON response into out when out is
// non-nil. Callers must hold p.mu.
func (p *Provider) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	if strings.TrimSpace(p.APIToken) == "" {
		return fmt.Errorf("binarylane: APIToken is required")
	}

	base, err := p.baseURL()
	if err != nil {
		return err
	}

	endpoint, err := base.Parse(path)
	if err != nil {
		return fmt.Errorf("binarylane: building request path %q: %w", path, err)
	}
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}

	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("binarylane: encoding request: %w", err)
		}

		payload = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), payload)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.APIToken))
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if p.HTTPClient == nil {
		p.HTTPClient = &http.Client{Timeout: requestTimeout}
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// The body should be problem details, but an error is an error even if
		// a proxy returned something else entirely.
		failure := &apiError{Status: resp.StatusCode}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		if err := json.Unmarshal(raw, failure); err != nil || (failure.Title == "" && failure.Detail == "") {
			failure.Detail = strings.TrimSpace(string(raw))
		}
		failure.Status = resp.StatusCode

		return failure
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("binarylane: decoding response: %w", err)
	}

	return nil
}

// listRecords returns every record in the zone, following pagination.
// Callers must hold p.mu.
func (p *Provider) listRecords(ctx context.Context, zone string) ([]apiRecord, error) {
	var all []apiRecord

	for page := 1; ; page++ {
		var out recordsResponse

		query := url.Values{
			"page":     {strconv.Itoa(page)},
			"per_page": {strconv.Itoa(perPage)},
		}

		if err := p.do(ctx, http.MethodGet, "domains/"+url.PathEscape(zone)+"/records", query, nil, &out); err != nil {
			return nil, fmt.Errorf("listing records for %q: %w", zone, err)
		}

		all = append(all, out.DomainRecords...)

		// Stop on a short or empty page rather than trusting the total alone,
		// so a miscounted total cannot spin this loop forever.
		if len(out.DomainRecords) < perPage || len(all) >= out.Meta.Total {
			return all, nil
		}
	}
}

// createRecord adds one record to the zone. Callers must hold p.mu.
func (p *Provider) createRecord(ctx context.Context, zone string, rec apiRecord) (apiRecord, error) {
	var out recordResponse

	rec.ID = 0
	if err := p.do(ctx, http.MethodPost, "domains/"+url.PathEscape(zone)+"/records", nil, rec, &out); err != nil {
		return apiRecord{}, fmt.Errorf("creating %s record %q: %w", rec.Type, rec.Name, err)
	}

	return out.DomainRecord, nil
}

// updateRecord replaces one record in the zone. Callers must hold p.mu.
func (p *Provider) updateRecord(ctx context.Context, zone string, id int64, rec apiRecord) (apiRecord, error) {
	var out recordResponse

	rec.ID = 0
	path := "domains/" + url.PathEscape(zone) + "/records/" + strconv.FormatInt(id, 10)
	if err := p.do(ctx, http.MethodPut, path, nil, rec, &out); err != nil {
		return apiRecord{}, fmt.Errorf("updating %s record %q: %w", rec.Type, rec.Name, err)
	}

	return out.DomainRecord, nil
}

// deleteRecord removes one record from the zone. Callers must hold p.mu.
func (p *Provider) deleteRecord(ctx context.Context, zone string, id int64) error {
	path := "domains/" + url.PathEscape(zone) + "/records/" + strconv.FormatInt(id, 10)
	if err := p.do(ctx, http.MethodDelete, path, nil, nil, nil); err != nil {
		return fmt.Errorf("deleting record %d: %w", id, err)
	}

	return nil
}

// listDomains returns the zone names the token can see. Callers must hold p.mu.
func (p *Provider) listDomains(ctx context.Context) ([]string, error) {
	var names []string

	for page := 1; ; page++ {
		var out domainsResponse

		query := url.Values{
			"page":     {strconv.Itoa(page)},
			"per_page": {strconv.Itoa(perPage)},
		}

		if err := p.do(ctx, http.MethodGet, "domains", query, nil, &out); err != nil {
			return nil, fmt.Errorf("listing domains: %w", err)
		}

		for _, d := range out.Domains {
			names = append(names, d.Name)
		}

		if len(out.Domains) < perPage || len(names) >= out.Meta.Total {
			return names, nil
		}
	}
}
