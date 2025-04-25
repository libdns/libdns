package simplydotcom

import (
	"net/netip"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

func TestToLibdns(t *testing.T) {
	zone := "example.com."

	// Helper function to create a uint16 pointer
	uint16Ptr := func(v uint16) *uint16 {
		return &v
	}

	tests := []struct {
		name          string
		input         dnsRecordResponse
		expected      string // Name
		expectedType  string
		expectedValue string
		expectedTTL   time.Duration
		wantErr       bool
	}{
		{
			name: "A record",
			input: dnsRecordResponse{
				Id: 123,
				dnsRecord: dnsRecord{
					Name: "www",
					Type: "A",
					Data: "192.0.2.1",
					Ttl:  3600,
				},
			},
			expected:      "www",
			expectedType:  "A",
			expectedValue: "192.0.2.1",
			expectedTTL:   3600 * time.Second,
		},
		{
			name: "AAAA record",
			input: dnsRecordResponse{
				Id: 124,
				dnsRecord: dnsRecord{
					Name: "ipv6",
					Type: "AAAA",
					Data: "2001:db8::1",
					Ttl:  7200,
				},
			},
			expected:      "ipv6",
			expectedType:  "AAAA",
			expectedValue: "2001:db8::1",
			expectedTTL:   7200 * time.Second,
		},
		{
			name: "CNAME record",
			input: dnsRecordResponse{
				Id: 125,
				dnsRecord: dnsRecord{
					Name: "alias",
					Type: "CNAME",
					Data: "target.example.com.",
					Ttl:  3600,
				},
			},
			expected:      "alias",
			expectedType:  "CNAME",
			expectedValue: "target.example.com.",
			expectedTTL:   3600 * time.Second,
		},
		{
			name: "MX record",
			input: dnsRecordResponse{
				Id: 126,
				dnsRecord: dnsRecord{
					Name:     "@",
					Type:     "MX",
					Data:     "mail.example.com.",
					Ttl:      3600,
					Priority: uint16Ptr(10),
				},
			},
			expected:      "@",
			expectedType:  "MX",
			expectedValue: "10 mail.example.com.",
			expectedTTL:   3600 * time.Second,
		},
		{
			name: "TXT record",
			input: dnsRecordResponse{
				Id: 127,
				dnsRecord: dnsRecord{
					Name: "txt",
					Type: "TXT",
					Data: "v=spf1 include:_spf.example.com ~all",
					Ttl:  3600,
				},
			},
			expected:      "txt",
			expectedType:  "TXT",
			expectedValue: "v=spf1 include:_spf.example.com ~all",
			expectedTTL:   3600 * time.Second,
		},
		{
			name: "NS record",
			input: dnsRecordResponse{
				Id: 129,
				dnsRecord: dnsRecord{
					Name: "subdomain",
					Type: "NS",
					Data: "ns1.example.com.",
					Ttl:  86400,
				},
			},
			expected:      "subdomain",
			expectedType:  "NS",
			expectedValue: "ns1.example.com.",
			expectedTTL:   86400 * time.Second,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.input.toLibdns(zone)

			if test.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			rr := result.RR()

			if rr.Name != test.expected {
				t.Errorf("Expected name %s, got %s", test.expected, rr.Name)
			}

			if rr.Type != test.expectedType {
				t.Errorf("Expected type %s, got %s", test.expectedType, rr.Type)
			}

			// Only check expectedValue if it's set (some tests don't need it)
			if test.expectedValue != "" && rr.Data != test.expectedValue {
				t.Errorf("Expected value %s, got %s", test.expectedValue, rr.Data)
			}

			if rr.TTL != test.expectedTTL {
				t.Errorf("Expected TTL %v, got %v", test.expectedTTL, rr.TTL)
			}
		})
	}
}

func TestToSimply(t *testing.T) {
	// Helper function to create a uint16 pointer
	uint16Ptr := func(v uint16) *uint16 {
		return &v
	}

	// Create IPv4 and IPv6 addresses for testing
	ipv4 := netip.MustParseAddr("192.0.2.1")
	ipv6 := netip.MustParseAddr("2001:db8::1")

	tests := []struct {
		name             string
		input            libdns.Record
		expectedName     string
		expectedType     string
		expectedData     string
		expectedTtl      int
		expectedPriority *uint16
	}{
		{
			name: "A record",
			input: libdns.Address{
				Name: "www",
				TTL:  3600 * time.Second,
				IP:   ipv4,
			},
			expectedName: "www",
			expectedType: "A",
			expectedData: "192.0.2.1",
			expectedTtl:  3600,
		},
		{
			name: "AAAA record",
			input: libdns.Address{
				Name: "ipv6",
				TTL:  7200 * time.Second,
				IP:   ipv6,
			},
			expectedName: "ipv6",
			expectedType: "AAAA",
			expectedData: "2001:db8::1",
			expectedTtl:  7200,
		},
		{
			name: "CNAME record",
			input: libdns.CNAME{
				Name:   "alias",
				TTL:    3600 * time.Second,
				Target: "target.example.com.",
			},
			expectedName: "alias",
			expectedType: "CNAME",
			expectedData: "target.example.com.",
			expectedTtl:  3600,
		},
		{
			name: "MX record",
			input: libdns.MX{
				Name:       "mail",
				TTL:        3600 * time.Second,
				Preference: 10,
				Target:     "mail.example.com.",
			},
			expectedName:     "mail",
			expectedType:     "MX",
			expectedData:     "mail.example.com.",
			expectedTtl:      3600,
			expectedPriority: uint16Ptr(10),
		},
		{
			name: "NS record",
			input: libdns.NS{
				Name:   "subdomain",
				TTL:    86400 * time.Second,
				Target: "ns1.example.com.",
			},
			expectedName: "subdomain",
			expectedType: "NS",
			expectedData: "ns1.example.com.",
			expectedTtl:  86400,
		},
		{
			name: "SRV record",
			input: libdns.SRV{
				Service:   "sip",
				Transport: "tcp",
				Name:      "example.com.",
				TTL:       3600 * time.Second,
				Priority:  10,
				Weight:    5,
				Port:      5060,
				Target:    "sipserver.example.com.",
			},
			expectedName:     "_sip._tcp.example.com.",
			expectedType:     "SRV",
			expectedData:     "5 5060 sipserver.example.com.",
			expectedTtl:      3600,
			expectedPriority: uint16Ptr(10),
		},
		{
			name: "TXT record",
			input: libdns.TXT{
				Name: "txt",
				TTL:  3600 * time.Second,
				Text: "v=spf1 include:_spf.example.com ~all",
			},
			expectedName: "txt",
			expectedType: "TXT",
			expectedData: "v=spf1 include:_spf.example.com ~all",
			expectedTtl:  3600,
		},
		{
			name: "CAA record",
			input: libdns.CAA{
				Name:  "caa",
				TTL:   3600 * time.Second,
				Flags: 0,
				Tag:   "issue",
				Value: "letsencrypt.org",
			},
			expectedName: "caa",
			expectedType: "CAA",
			expectedData: `0 issue "letsencrypt.org"`,
			expectedTtl:  3600,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := toSimply(test.input)

			if result.Name != test.expectedName {
				t.Errorf("Expected name %s, got %s", test.expectedName, result.Name)
			}

			if result.Type != test.expectedType {
				t.Errorf("Expected type %s, got %s", test.expectedType, result.Type)
			}

			if result.Data != test.expectedData {
				t.Errorf("Expected data %s, got %s", test.expectedData, result.Data)
			}

			if result.Ttl != test.expectedTtl {
				t.Errorf("Expected TTL %d, got %d", test.expectedTtl, result.Ttl)
			}

			// Check priority if expected
			if test.expectedPriority != nil {
				if result.Priority == nil {
					t.Errorf("Expected priority %d, got nil", *test.expectedPriority)
				} else if *result.Priority != *test.expectedPriority {
					t.Errorf("Expected priority %d, got %d", *test.expectedPriority, *result.Priority)
				}
			}
		})
	}
}

func TestRoundTripConversions(t *testing.T) {
	zone := "example.com."

	// Create IPv4 and IPv6 addresses for testing
	ipv4 := netip.MustParseAddr("192.0.2.1")
	ipv6 := netip.MustParseAddr("2001:db8::1")

	tests := []struct {
		name  string
		input libdns.Record
	}{
		{
			name: "A record",
			input: libdns.Address{
				Name: "www",
				TTL:  3600 * time.Second,
				IP:   ipv4,
			},
		},
		{
			name: "AAAA record",
			input: libdns.Address{
				Name: "ipv6",
				TTL:  7200 * time.Second,
				IP:   ipv6,
			},
		},
		{
			name: "CNAME record",
			input: libdns.CNAME{
				Name:   "alias",
				TTL:    3600 * time.Second,
				Target: "target.example.com.",
			},
		},
		{
			name: "MX record",
			input: libdns.MX{
				Name:       "mail",
				TTL:        3600 * time.Second,
				Preference: 10,
				Target:     "mail.example.com.",
			},
		},
		{
			name: "NS record",
			input: libdns.NS{
				Name:   "subdomain",
				TTL:    86400 * time.Second,
				Target: "ns1.example.com.",
			},
		},
		{
			name: "SRV record",
			input: libdns.SRV{
				Service:   "sip",
				Transport: "tcp",
				Name:      "example.com.",
				TTL:       3600 * time.Second,
				Priority:  10,
				Weight:    5,
				Port:      5060,
				Target:    "sipserver.example.com.",
			},
		},
		{
			name: "TXT record",
			input: libdns.TXT{
				Name: "txt",
				TTL:  3600 * time.Second,
				Text: "v=spf1 include:_spf.example.com ~all",
			},
		},
		{
			name: "CAA record",
			input: libdns.CAA{
				Name:  "caa",
				TTL:   3600 * time.Second,
				Flags: 0,
				Tag:   "issue",
				Value: "letsencrypt.org",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Convert libdns -> simply
			simplyRecord := toSimply(test.input)

			// Create a response object from the simply record
			simplyResponse := dnsRecordResponse{
				Id:        12345,
				dnsRecord: simplyRecord,
			}

			// Convert back simply -> libdns
			resultRecord, err := simplyResponse.toLibdns(zone)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify the libdns record matches the original
			inputRR := test.input.RR()
			resultRR := resultRecord.RR()

			// Compare the type
			if resultRR.Type != inputRR.Type {
				t.Errorf("Expected type %s, got %s", inputRR.Type, resultRR.Type)
			}

			// For empty names, the conversion back and forth will result in "@" -> ""
			if inputRR.Name == "" {
				if resultRR.Name != "" {
					t.Errorf("Expected empty name, got %s", resultRR.Name)
				}
			} else if resultRR.Name != inputRR.Name {
				t.Errorf("Expected name %s, got %s", inputRR.Name, resultRR.Name)
			}

			if resultRR.Data != inputRR.Data {
				t.Errorf("Expected data %s, got %s", inputRR.Data, resultRR.Data)
			}

			if resultRR.TTL != inputRR.TTL {
				t.Errorf("Expected TTL %v, got %v", inputRR.TTL, resultRR.TTL)
			}
		})
	}
}
