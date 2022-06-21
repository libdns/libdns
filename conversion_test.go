package rfc2136

import (
	"github.com/libdns/libdns"
	"github.com/miekg/dns"
	"net"
	"testing"
	"time"
)

const zone = "example.com."

var testCases = map[dns.RR]libdns.Record{
	&dns.TXT{
		Hdr: dns.RR_Header{
			Name:   "txt.example.com.",
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    220,
		},
		Txt: []string{"hello world"},
	}: {
		Type:  "TXT",
		Name:  "txt",
		Value: "\"hello world\"",
		TTL:   220 * time.Second,
	},
	&dns.TXT{
		Hdr: dns.RR_Header{
			Name:   "txt.example.com.",
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    220,
		},
		Txt: []string{"hello", "world"},
	}: {
		Type:  "TXT",
		Name:  "txt",
		Value: "\"hello\" \"world\"",
		TTL:   220 * time.Second,
	},

	&dns.A{
		Hdr: dns.RR_Header{
			Name:   "a.example.com.",
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    300,
		},
		A: net.ParseIP("1.2.3.4"),
	}: {
		Type:  "A",
		Name:  "a",
		Value: "1.2.3.4",
		TTL:   300 * time.Second,
	},

	&dns.AAAA{
		Hdr: dns.RR_Header{
			Name:   "aaaa.example.com.",
			Rrtype: dns.TypeAAAA,
			Class:  dns.ClassINET,
			Ttl:    150,
		},
		AAAA: net.ParseIP("1:2:3:4::"),
	}: {
		Type:  "AAAA",
		Name:  "aaaa",
		Value: "1:2:3:4::",
		TTL:   150 * time.Second,
	},
}

func TestRecordFromRR(t *testing.T) {
	for rr, expected := range testCases {
		converted := recordFromRR(rr, zone)
		if expected != converted {
			t.Errorf("converted record does not match expected\nRR: %#v\nExpected: %#v\nGot: %#v",
				rr, expected, converted)
		}
	}
}

func rrEqual(rr1, rr2 dns.RR) bool {
	return dns.IsDuplicate(rr1, rr2) && rr1.Header().Ttl == rr2.Header().Ttl
}

func TestRecordToRR(t *testing.T) {
	for expected, record := range testCases {
		converted, err := recordToRR(record, zone)
		if err != nil {
			t.Error(err)
		}
		if !rrEqual(expected, converted) {
			t.Errorf("converted rr does not match expected\nRecord: %#v\nExpected: %#v\nGot: %#v",
				record, expected, converted)
		}
	}
}
