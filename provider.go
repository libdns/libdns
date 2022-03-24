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

var acmePrefix = "_acme-challenge."

type DomainConfig struct {
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
	FullDomain string `json:"fulldomain,omitempty"`
	ServerURL  string `json:"server_url,omitempty"`
}

type Domain = string

// Provider.Configs defines a Domain -> DomainConfig mapping.
// Configs map uses the same structure as ACME-DNS client
// JSON storage file (https://github.com/acme-dns/acme-dns-client).
type Provider struct {
	Configs map[Domain]DomainConfig `json:"config,omitempty"`
}

// Implements libdns.RecordAppender.
//
// The only operation Joohoi's ACME-DNS API supports is a rolling update
// of two TXT records. Zone and record names are used to select
// respective credentials from Provider.Configs. If relevant config exists,
// AppendRecords appends records using selected ACME-DNS account.
// Only TXT records are supported. ID, TTL and Priority fields
// of libdns.Record are ignored.
func (p *Provider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	appendedRecords := []libdns.Record{}
	for _, record := range recs {
		if record.Type != "TXT" {
			return appendedRecords, fmt.Errorf("joohoi_acme_dns provider only supports adding TXT records")
		}
		domain := record.Name + "." + zone
		if domain[len(domain)-1:] == "." {
			domain = domain[:len(domain)-1]
		}
		if domain[:len(acmePrefix)] != acmePrefix {
			return appendedRecords, fmt.Errorf("joohoi_acme_dns provider only supports adding TXT records with %s prefix", acmePrefix)
		}
		domain = domain[len(acmePrefix):]
		config, found := p.Configs[domain]
		if !found {
			return appendedRecords, fmt.Errorf("Config for domain %s not found", domain)
		}
		body, err := json.Marshal(
			map[string]string{
				"subdomain": config.Subdomain,
				"txt":       record.Value,
			},
		)
		if err != nil {
			return appendedRecords, fmt.Errorf("Error while marshalling JSON: %w", err)
		}
		req, err := http.NewRequest("POST", config.ServerURL+"/update", bytes.NewBuffer(body))
		req.Header.Set("X-Api-User", config.Username)
		req.Header.Set("X-Api-Key", config.Password)
		client := &http.Client{Timeout: time.Second * 30}
		resp, err := client.Do(req)
		if err != nil {
			return appendedRecords, fmt.Errorf("Error while reading response: %w", err)
		}
		if resp.StatusCode != 200 {
			return appendedRecords, fmt.Errorf("Updating ACME-DNS record resulted in response code %d", resp.StatusCode)
		}
		appendedRecords = append(appendedRecords, libdns.Record{Type: "TXT", Name: record.Name, Value: record.Value})
	}
	return appendedRecords, nil
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
