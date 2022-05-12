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
