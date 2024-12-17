package katapult

import (
	"errors"
	"fmt"
	"time"

	"github.com/libdns/libdns"
)

var errUnsupportedRecordType = errors.New("unsupported record type")

// DNSRecord represents a DNS record in the API response.
type DNSRecord struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	TTL      int    `json:"ttl"`
	Priority uint   `json:"priority"`
	Content  string `json:"content"`
}

// DNSRecordAPIResponse represents an API response for a DNS record.
type DNSRecordAPIResponse struct {
	DNSRecord DNSRecord `json:"dns_record"`
}

// DNSRecordsAPIResponse represents an API response of DNS records.
type DNSRecordsAPIResponse struct {
	DNSRecords []DNSRecord `json:"dns_records"`
}

// DeletionAPIResponse represents an API response for a deletion.
type DeletionAPIResponse struct {
	Deleted bool `json:"deleted"`
}

// ToLibDNSRecord converts a DNSRecord to a libdns.Record.
func (p *Provider) ToLibDNSRecord(r DNSRecord) libdns.Record {
	return libdns.Record{
		ID:       r.ID,
		Type:     r.Type,
		Name:     r.Name,
		Value:    r.Content,
		TTL:      time.Duration(r.TTL) * time.Second,
		Priority: r.Priority,
	}
}

// FromLibDNSRecord converts a libdns.Record to an API request body.
func (p *Provider) FromLibDNSRecord(record libdns.Record) (interface{}, error) {
	content := map[string]interface{}{}

	switch record.Type {
	case "A", "AAAA", "IPS":
		content["ip_address"] = record.Value
	case "ALIAS", "CNAME", "MX", "NS", "PTR":
		content["hostname"] = record.Value
	case "TXT":
		content["content"] = record.Value
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedRecordType, record.Type)
	}

	return map[string]interface{}{
		"type": record.Type,
		"name": record.Name,
		"ttl":  ensureValidTTL(record.TTL),
		"content": map[string]interface{}{
			record.Type: content,
		},
	}, nil
}

func ensureValidTTL(rawTTL time.Duration) *int {
	ttl := int(rawTTL.Seconds())
	if ttl == 0 {
		return nil
	} else if ttl < 60 {
		minTTL := 60
		return &minTTL
	}
	return &ttl
}
