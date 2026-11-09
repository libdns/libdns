package binarylane

import (
	"net/netip"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// Binary Lane splits RDATA for MX, SRV and CAA across dedicated fields. These
// tests pin the composition on read and the split on write, which is where a
// mismatch with the API would silently corrupt records.

func TestToLibdns(t *testing.T) {
	tests := []struct {
		name     string
		api      apiRecord
		wantType string
		wantData string
	}{
		{
			name:     "A passes data through",
			api:      apiRecord{ID: 1, Type: "A", Name: "www", Data: "203.0.113.10"},
			wantType: "A",
			wantData: "203.0.113.10",
		},
		{
			name:     "TXT passes data through unquoted",
			api:      apiRecord{ID: 2, Type: "TXT", Name: "_acme-challenge", Data: "token-value"},
			wantType: "TXT",
			wantData: "token-value",
		},
		{
			name:     "MX folds priority into data",
			api:      apiRecord{ID: 3, Type: "MX", Name: "@", Data: "mail.example.com.", Priority: 10},
			wantType: "MX",
			wantData: "10 mail.example.com.",
		},
		{
			name:     "SRV folds priority, weight and port into data",
			api:      apiRecord{ID: 4, Type: "SRV", Name: "_sip._tcp", Data: "sip.example.com.", Priority: 10, Weight: 20, Port: 5060},
			wantType: "SRV",
			wantData: "10 20 5060 sip.example.com.",
		},
		{
			name:     "CAA folds flags and tag into data",
			api:      apiRecord{ID: 5, Type: "CAA", Name: "@", Data: "letsencrypt.org", Flags: 0, Tag: "issue"},
			wantType: "CAA",
			wantData: `0 issue "letsencrypt.org"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toLibdns(tt.api)
			rr := got.RR()

			if rr.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", rr.Type, tt.wantType)
			}
			if rr.Data != tt.wantData {
				t.Errorf("Data = %q, want %q", rr.Data, tt.wantData)
			}
			if want := time.Duration(tt.api.TTL) * time.Second; rr.TTL != want {
				t.Errorf("TTL = %v, want %v", rr.TTL, want)
			}

			id, ok := recordID(got)
			if !ok {
				t.Fatalf("recordID() not recovered for %s", tt.wantType)
			}
			if id != tt.api.ID {
				t.Errorf("recordID() = %d, want %d", id, tt.api.ID)
			}
		})
	}
}

func TestToLibdnsUnmodelledTypeSurvives(t *testing.T) {
	// Binary Lane serves SOA, which libdns does not model. It must come back as
	// an opaque RR rather than being dropped or erroring.
	got := toLibdns(apiRecord{
		ID: 9, Type: "SOA", Name: "@", Data: "ns1.binarylane.com.au. 1 3600 600 604800 1800", TTL: 1800,
	})

	if got.RR().Type != "SOA" {
		t.Errorf("Type = %q, want SOA", got.RR().Type)
	}
	if _, ok := got.(libdns.RR); !ok {
		t.Errorf("got %T, want the opaque libdns.RR for an unmodelled type", got)
	}
}

func TestToAPI(t *testing.T) {
	tests := []struct {
		name string
		rec  libdns.Record
		want apiRecord
	}{
		{
			name: "A normalises the address",
			rec:  libdns.Address{Name: "www", TTL: time.Hour, IP: netip.MustParseAddr("203.0.113.10")},
			want: apiRecord{Type: "A", Name: "www", Data: "203.0.113.10"},
		},
		{
			name: "TXT sends the text verbatim",
			rec:  libdns.TXT{Name: "_acme-challenge", TTL: 2 * time.Minute, Text: "token-value"},
			want: apiRecord{Type: "TXT", Name: "_acme-challenge", Data: "token-value"},
		},
		{
			name: "MX splits preference out of data",
			rec:  libdns.MX{Name: "@", TTL: time.Hour, Preference: 10, Target: "mail.example.com."},
			want: apiRecord{Type: "MX", Name: "@", Data: "mail.example.com.", Priority: 10},
		},
		{
			name: "SRV reassembles the owner name and splits the numbers out",
			rec: libdns.SRV{
				Service: "sip", Transport: "tcp", Name: "example", TTL: time.Hour,
				Priority: 10, Weight: 20, Port: 5060, Target: "sip.example.com.",
			},
			want: apiRecord{
				Type: "SRV", Name: "_sip._tcp.example", Data: "sip.example.com.",
				Priority: 10, Weight: 20, Port: 5060,
			},
		},
		{
			name: "CAA splits flags and tag out of data",
			rec:  libdns.CAA{Name: "@", TTL: time.Hour, Flags: 128, Tag: "issue", Value: "letsencrypt.org"},
			want: apiRecord{Type: "CAA", Name: "@", Data: "letsencrypt.org", Flags: 128, Tag: "issue"},
		},
		{
			name: "an empty name becomes the apex marker",
			rec:  libdns.TXT{Name: "", TTL: time.Minute, Text: "v=spf1 -all"},
			want: apiRecord{Type: "TXT", Name: "@", Data: "v=spf1 -all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toAPI(tt.rec)
			if err != nil {
				t.Fatalf("toAPI() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("toAPI() =\n  %+v\nwant\n  %+v", got, tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// A record read from the API and written straight back must be unchanged,
	// or SetRecords would rewrite records it merely passed through.
	for _, api := range []apiRecord{
		{ID: 1, Type: "A", Name: "www", Data: "203.0.113.10"},
		{ID: 2, Type: "TXT", Name: "_acme-challenge", Data: "token"},
		{ID: 3, Type: "MX", Name: "@", Data: "mail.example.com.", Priority: 10},
		{ID: 4, Type: "SRV", Name: "_sip._tcp.example", Data: "sip.example.com.", Priority: 10, Weight: 20, Port: 5060},
		{ID: 5, Type: "CAA", Name: "@", Data: "letsencrypt.org", Flags: 0, Tag: "issue"},
		{ID: 6, Type: "CNAME", Name: "alias", Data: "target.example.com.", TTL: 300},
		{ID: 7, Type: "NS", Name: "sub", Data: "ns1.example.com."},
	} {
		t.Run(api.Type, func(t *testing.T) {
			back, err := toAPI(toLibdns(api))
			if err != nil {
				t.Fatalf("toAPI(toLibdns(%s)) error = %v", api.Type, err)
			}

			want := apiRecord{
				Type: api.Type, Name: api.Name, Data: api.Data,
				Priority: api.Priority, Port: api.Port, Weight: api.Weight,
				Flags: api.Flags, Tag: api.Tag,
			}
			if back != want {
				t.Errorf("round trip changed the record:\n got %+v\nwant %+v", back, want)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	// Binary Lane enforces a fixed zone TTL, so a record read back from it
	// always carries 3600 regardless of what was requested.
	existing := apiRecord{ID: 1, Type: "TXT", Name: "_acme-challenge", Data: "token", TTL: 3600}

	tests := []struct {
		name string
		want libdns.RR
		ok   bool
	}{
		{"name only is a wildcard over type, ttl and data", libdns.RR{Name: "_acme-challenge"}, true},
		{"exact match", libdns.RR{Name: "_acme-challenge", Type: "TXT", Data: "token", TTL: 3600 * time.Second}, true},
		{"a TTL the zone cannot hold never matches", libdns.RR{Name: "_acme-challenge", TTL: 120 * time.Second}, false},
		{"wrong name never matches", libdns.RR{Name: "other"}, false},
		{"wrong type", libdns.RR{Name: "_acme-challenge", Type: "A"}, false},
		{"wrong data", libdns.RR{Name: "_acme-challenge", Data: "different"}, false},
		{"wrong ttl", libdns.RR{Name: "_acme-challenge", TTL: 2 * time.Hour}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matches(existing, tt.want); got != tt.ok {
				t.Errorf("matches() = %v, want %v", got, tt.ok)
			}
		})
	}
}

func TestRecordIDAbsentForCallerBuiltRecords(t *testing.T) {
	// A record the caller constructed has no ProviderData, so DeleteRecords
	// must fall back to matching by value rather than deleting ID 0.
	if _, ok := recordID(libdns.TXT{Name: "x", Text: "y"}); ok {
		t.Error("recordID() reported an ID for a caller-built record")
	}
}

func TestUnFQDN(t *testing.T) {
	for in, want := range map[string]string{
		"example.com.": "example.com",
		"example.com":  "example.com",
		"":             "",
	} {
		if got := unFQDN(in); got != want {
			t.Errorf("unFQDN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestToAPIOmitsTTL(t *testing.T) {
	// Binary Lane rejects any TTL but its own zone default:
	//
	//	HTTP 400: The default TTL of 3600 may not be changed.
	//
	// So TTL must never reach a write request. If it did, every ACME DNS-01
	// challenge would fail, because clients set a short TTL on challenge
	// records.
	for _, rec := range []libdns.Record{
		libdns.TXT{Name: "_acme-challenge", TTL: 2 * time.Minute, Text: "token"},
		libdns.TXT{Name: "_acme-challenge", TTL: time.Hour, Text: "token"},
		libdns.Address{Name: "www", TTL: 30 * time.Second, IP: netip.MustParseAddr("203.0.113.10")},
		libdns.MX{Name: "@", TTL: time.Second, Preference: 10, Target: "mail.example.com."},
	} {
		got, err := toAPI(rec)
		if err != nil {
			t.Fatalf("toAPI() error = %v", err)
		}
		if got.TTL != 0 {
			t.Errorf("toAPI(%T with TTL %v) set TTL=%d; it must be omitted so the API applies its own",
				rec, rec.RR().TTL, got.TTL)
		}
	}
}
