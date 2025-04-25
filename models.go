package simplydotcom

import (
	"fmt"
	"time"

	"github.com/libdns/libdns"
)

type simplyResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

// dnsRecord represents the common structure of a DNS record
type dnsRecord struct {
	Name     string  `json:"name"`
	Ttl      int     `json:"ttl"`
	Data     string  `json:"data"`
	Type     string  `json:"type"`
	Priority *uint16 `json:"priority,omitempty"`
	Comment  string  `json:"comment,omitempty"`
}

// dnsRecordResponse is used for API responses that include DNS records
type dnsRecordResponse struct {
	Id int `json:"record_id"`
	dnsRecord
}

// createRecordResponse represents the API response when creating a new record
type createRecordResponse struct {
	Record struct {
		Id int `json:"id"`
	} `json:"record,omitempty"`
	simplyResponse
}

type getRecordsResponse struct {
	Records []dnsRecordResponse `json:"records"`
	simplyResponse
}

func toSimply(rec libdns.Record) dnsRecord {
	switch rec := rec.(type) {
	case libdns.MX:
		return dnsRecord{
			Name:     rec.Name,
			Ttl:      int(rec.TTL.Seconds()),
			Data:     rec.Target,
			Type:     rec.RR().Type,
			Priority: &rec.Preference,
		}

	case libdns.SRV:
		// Simply expects priority extracted from the Data field, so we extract it here.
		return dnsRecord{
			Name:     rec.RR().Name,
			Ttl:      int(rec.TTL.Seconds()),
			Data:     fmt.Sprintf("%d %d %s", rec.Weight, rec.Port, rec.Target),
			Priority: &rec.Priority,
			Type:     "SRV",
		}

	default:
		var rr = rec.RR()
		return dnsRecord{
			Name: rr.Name,
			Ttl:  int(rr.TTL.Seconds()),
			Data: rr.Data,
			Type: rr.Type,
		}
	}
}

func (rec dnsRecordResponse) toLibdns(zone string) (libdns.Record, error) {

	switch rec.Type {
	case "MX":
		// Simply MX records have extracted Priority from the Data field, so we construct the libdns.MX record manually.
		return libdns.MX{
			Name:       rec.Name,
			TTL:        time.Duration(rec.Ttl) * time.Second,
			Target:     rec.Data,
			Preference: *rec.Priority,
		}, nil

	case "SRV":
		// Simply SRV records have extracted Priority from the Data field, so we append it here before parsing to libdns.SRV.
		return libdns.RR{
			Name: libdns.AbsoluteName(rec.Name, zone),
			Type: rec.Type,
			Data: fmt.Sprintf("%d %s", *rec.Priority, rec.Data),
			TTL:  time.Duration(rec.Ttl) * time.Second,
		}.Parse()

	default:
		return libdns.RR{
			Name: libdns.RelativeName(rec.Name, zone),
			Type: rec.Type,
			Data: rec.Data,
			TTL:  time.Duration(rec.Ttl) * time.Second,
		}.Parse()
	}
}
