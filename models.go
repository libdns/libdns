package netlify

import (
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

type netlifyDNSDeleteError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
