package allinkl

import (
	"time"

	"github.com/libdns/libdns"
)

type allinklRecord struct {
	ID     string `xml:"record_id,omitempty"`
	ZoneID string `xml:"record_zone"`
	Type   string `xml:"record_type"`
	Name   string `xml:"record_name"`
	Value  string `xml:"record_data"`
	TTL    int    `xml:"record_ttl,omitempty"`
}

func (record allinklRecord) toLibdnsRecord(zone string) (libdns.Record, error) {
	// Convert TTL from int to time.Duration
	ttl := time.Duration(record.TTL) * time.Second

	// Create the RR record
	rr := libdns.RR{
		Type: record.Type,
		Name: libdns.RelativeName(record.Name, zone),
		Data: record.Value,
		TTL:  ttl,
	}

	// Parse the RR into a concrete record type
	parsed, err := rr.Parse()
	if err != nil {
		// If parsing fails, return the RR as is
		return rr, nil
	}

	// Return the parsed record (which could be Address, TXT, etc.)
	return parsed, nil
}
