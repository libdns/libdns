package joohoi_acme_dns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/libdns/libdns"
)

type Provider struct {
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	Subdomain string `json:"subdomain,omitempty"`
	ClientURL string `json:"client_url,omitempty"`
}

// Implements libdns.RecordAppender.
//
// The only operation Joohoi's ACME-DNS API supports is a rolling update
// of two TXT records. The zone and record name are determined by the
// ACME-DNS account, it is not set via the API. Therefore, AppendRecords()
// ignores zone and record name information. It just adds new TXT records
// for the account defined by Provider struct fields
// (Username, Password, Subdomain, ClientURL).
func (p *Provider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	for _, record := range recs {
		if record.Type != "TXT" {
			return nil, fmt.Errorf("joohoi_acme_dns provider only supports adding TXT records")
		}
		body, err := json.Marshal(
			map[string]string{
				"subdomain": p.Subdomain,
				"txt":       record.Value,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("Error while marshalling JSON: %w", err)
		}
		req, err := http.NewRequest("POST", p.ClientURL+"/update", bytes.NewBuffer(body))
		req.Header.Set("X-Api-User", p.Username)
		req.Header.Set("X-Api-Key", p.Password)
		client := &http.Client{Timeout: time.Second * 30}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Error while reading response: %w", err)
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("Updating ACME-DNS record resulted in response code %d", resp.StatusCode)
		}
	}
	return recs, nil
}

// Implements libdns.RecordDeleter.
//
// Always returns and error since ACME-DNS API does not support record deletion.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	return nil, fmt.Errorf("joohoi_acme_dns provider does not support record deletion")
}

// Interface guards.
var (
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
