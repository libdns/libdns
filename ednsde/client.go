package ednsde

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultEndpoint is the production eDNS challenge API.
const DefaultEndpoint = "https://dns-challenge.edns.de"

// The only two actions the API understands.
const (
	actionAdd    = "addChallengeRecord"
	actionRemove = "removeChallengeRecord"
)

// Result codes reported in data.result_code on an HTTP 200 response.
const (
	resultAdded          = 1 // a new challenge record was created
	resultAlreadyExists  = 2 // an identical call was made before; nothing changed
	resultRemoved        = 3 // the challenge record was deleted
	resultAlreadyRemoved = 4 // the record was already gone from DNS; only the marker was cleared
	resultNotFound       = 5 // no API-created record with this value exists
)

// ednsTTL is the TTL eDNS applies to challenge records. It is fixed and cannot
// be influenced through the API, so the TTL on an input record is ignored and
// this value is what we report back to the caller.
const ednsTTL = 300 * time.Second

// Challenge values must satisfy these bounds, enforced by the API with a 400.
// We check them up front so the caller gets a useful message instead.
const (
	minValueLen = 10
	maxValueLen = 64
)

// Retry budget for transient failures. libdns expects providers to recover from
// connection errors, 429s and 5xx without adding substantial latency, so the
// whole budget stays well under a second.
const (
	maxAttempts  = 3
	retryBackoff = 200 * time.Millisecond
)

// maxResponseBytes caps how much of a response we are willing to read; the real
// API answers with a few hundred bytes.
const maxResponseBytes = 1 << 20

// apiRequest is the JSON body of every call. Subdomain is omitted for records
// on the zone apex, because sending it as an empty string is answered with 400.
type apiRequest struct {
	Action         string `json:"action"`
	Domain         string `json:"domain"`
	Subdomain      string `json:"subdomain,omitempty"`
	ChallengeToken string `json:"challenge_token"`
}

// apiResponse is the response envelope. On success "data" is an object, on
// every error it is an empty array -- which is why it stays raw until we know
// which of the two we are looking at.
type apiResponse struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// apiData is the "data" object of a successful response.
type apiData struct {
	Action         string `json:"action"`
	Domain         string `json:"domain"`
	Subdomain      string `json:"subdomain"`
	ChallengeToken string `json:"challenge_token"`
	Result         string `json:"result"`
	ResultCode     int    `json:"result_code"`
}

// APIError is returned for any non-200 answer from the eDNS API. Callers can
// use [errors.As] to reach it, for example to distinguish an authorization
// problem (401) from a malformed request (400).
type APIError struct {
	// StatusCode is the HTTP status of the response. The eDNS API also repeats
	// it in the body, where the two always agree.
	StatusCode int

	// Message is the API's own description of the failure, passed through
	// unchanged.
	Message string

	// Hint carries advice the API itself does not give, most importantly for
	// 401, where an invalid token and a token that is merely not assigned to
	// the zone are indistinguishable.
	Hint string
}

// Error implements the error interface, combining the API's own message with
// our hint where there is one.
func (e *APIError) Error() string {
	msg := fmt.Sprintf("edns api: HTTP %d: %s", e.StatusCode, e.Message)
	if e.Hint != "" {
		msg += " -- " + e.Hint
	}
	return msg
}

func hintFor(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "the access token must be valid and assigned to this zone on the zone's DNS-01-Challenge tab"
	case http.StatusBadRequest:
		return "this is a malformed request and will not succeed on retry"
	default:
		return ""
	}
}

// retryableStatus reports whether another attempt could plausibly succeed.
func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// do performs one API call, retrying transient failures.
func (p *Provider) do(ctx context.Context, req apiRequest) (*apiData, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt-1) * retryBackoff):
			}
		}

		data, retryable, err := p.attempt(ctx, body)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if !retryable || ctx.Err() != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

// attempt makes a single HTTP request and reports whether a retry is worthwhile.
func (p *Provider) attempt(ctx context.Context, body []byte) (*apiData, bool, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("building request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-TOKEN", p.APIToken)

	resp, err := p.httpClient().Do(httpReq)
	if err != nil {
		return nil, true, fmt.Errorf("calling the eDNS challenge API: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, true, fmt.Errorf("reading the response: %w", err)
	}

	var env apiResponse
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, retryableStatus(resp.StatusCode),
			fmt.Errorf("decoding the response (HTTP %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, retryableStatus(resp.StatusCode), &APIError{
			StatusCode: resp.StatusCode,
			Message:    env.Message,
			Hint:       hintFor(resp.StatusCode),
		}
	}

	var data apiData
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, false, fmt.Errorf("decoding the response data: %w", err)
	}
	return &data, false, nil
}
