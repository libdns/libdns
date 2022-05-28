package netlify

import (
	"encoding/json"
	"time"

	"github.com/libdns/libdns"
	"github.com/netlify/open-api/v2/go/models"
)

type netlifyZone struct {
	*models.DNSZone
}

type netlifyDNSRecord struct {
	*models.DNSRecord
}

func (r netlifyDNSRecord) libdnsRecord(zone string) libdns.Record {
	return libdns.Record{
		Type:  r.Type,
		Name:  libdns.RelativeName(r.Hostname, zone),
		Value: r.Value,
		TTL:   time.Duration(r.TTL) * time.Second,
		ID:    r.ID,
	}
}

func netlifyRecord(r libdns.Record) netlifyDNSRecord {
	return netlifyDNSRecord{
		&models.DNSRecord{
			ID:       r.ID,
			Type:     r.Type,
			Hostname: r.Name,
			Value:    r.Value,
			TTL:      int64(r.TTL.Seconds()),
			Priority: int64(r.Priority),
		},
	}
}

// All API responses have this structure.
type netlifyResponse struct {
	Result  json.RawMessage `json:"result,omitempty"`
	Success bool            `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors,omitempty"`
	Messages   []interface{}      `json:"messages,omitempty"`
	ResultInfo *netlifyResultInfo `json:"result_info,omitempty"`
}

type netlifyResultInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Count      int `json:"count"`
	TotalCount int `json:"total_count"`
}

type netlifyResultDNSZones struct {
	Id                   string              `json:"id,omitempty"`
	Name                 string              `json:"name,omitempty"`
	Errors               []string            `json:"errors,omitempty"`
	SupportedRecordTypes []string            `json:"supported_record_types,omitempty"`
	UserID               string              `json:"user_id,omitempty"`
	CreatedAt            string              `json:"created_at,omitempty"`
	UpdatedAt            string              `json:"updated_at,omitempty"`
	Records              []*models.DNSRecord `json:"records,omitempty"`
	DnsServers           []string            `json:"dns_servers,omitempty"`
	AccountId            string              `json:"account_id,omitempty"`
	SiteId               string              `json:"site_id,omitempty"`
	AccountSlug          string              `json:"account_slug,omitempty"`
	AccountName          string              `json:"account_name,omitempty"`
	Domain               string              `json:"domain,omitempty"`
	Ipv6Enabled          bool                `json:"ipv6_enabled,omitempty"`
	Dedicated            bool                `json:"dedicated,omitempty"`
}

type netlifyDNSDeleteError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
