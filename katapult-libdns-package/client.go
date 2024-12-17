package katapult

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var baseURL = "https://api.katapult.io/core/v1"
var errUnexpectedStatusCode = errors.New("unexpected status code")

// NewHTTPClient returns a configured HTTP client.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// DoRequest performs an HTTP request and decodes the response.
func (p *Provider) DoRequest(ctx context.Context, method, url string, body interface{}, response interface{}) error {
	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	apiURL := baseURL + url

	req, err := http.NewRequestWithContext(ctx, method, apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.APIToken)
	req.Header.Set("Content-Type", "application/json")

	client := NewHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}
		return fmt.Errorf("%w: %d, response body: %s", errUnexpectedStatusCode, resp.StatusCode, string(respBody))
	}

	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
