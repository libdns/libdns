package libdns

import (
	"net/netip"
	"reflect"
	"testing"
	"time"
)

func TestToAddress(t *testing.T) {
	for i, test := range []struct {
		input     RR
		expect    Address
		shouldErr bool
	}{
		{
			input: RR{
				Name: "sub",
				TTL:  5 * time.Minute,
				Type: "A",
				Data: "1.2.3.4",
			},
			expect: Address{
				Name: "sub",
				TTL:  5 * time.Minute,
				IP:   netip.MustParseAddr("1.2.3.4"),
			},
		},
		{
			input: RR{
				Name: "@",
				TTL:  5 * time.Minute,
				Type: "AAAA",
				Data: "2001:db8:3c4d:15:0:d234:3eee::",
			},
			expect: Address{
				Name: "@",
				TTL:  5 * time.Minute,
				IP:   netip.MustParseAddr("2001:db8:3c4d:15:0:d234:3eee::"),
			},
		},
	} {
		actual, err := test.input.toAddress()
		if err == nil && test.shouldErr {
			t.Errorf("Test %d: Expected error, got none", i)
		}
		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error, but got: %v", i, err)
		}
		if !reflect.DeepEqual(actual, test.expect) {
			t.Errorf("Test %d: INPUT=%#v\nEXPECTED: %#v\nACTUAL:   %#v", i, test.input, test.expect, actual)
		}
	}
}

func TestToCAA(t *testing.T) {
	for i, test := range []struct {
		input     RR
		expect    CAA
		shouldErr bool
	}{
		{
			input: RR{
				Name: "@",
				TTL:  5 * time.Minute,
				Type: "CAA",
				Data: `128 issue "letsencrypt.org"`,
			},
			expect: CAA{
				Name:  "@",
				TTL:   5 * time.Minute,
				Flags: 128,
				Tag:   "issue",
				Value: "letsencrypt.org",
			},
		},
		{
			input: RR{
				Name: "@",
				TTL:  1 * time.Hour,
				Type: "CAA",
				Data: `0 issuewild "letsencrypt.org; validationmethods=dns-01; accounturi=https://acme-v02.api.letsencrypt.org/acme/acct/1234567890"`,
			},
			expect: CAA{
				Name:  "@",
				TTL:   1 * time.Hour,
				Flags: 0,
				Tag:   "issuewild",
				Value: "letsencrypt.org; validationmethods=dns-01; accounturi=https://acme-v02.api.letsencrypt.org/acme/acct/1234567890",
			},
		},
	} {
		actual, err := test.input.toCAA()
		if err == nil && test.shouldErr {
			t.Errorf("Test %d: Expected error, got none", i)
		}
		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error, but got: %v", i, err)
		}
		if !reflect.DeepEqual(actual, test.expect) {
			t.Errorf("Test %d: INPUT=%#v\nEXPECTED: %#v\nACTUAL:   %#v", i, test.input, test.expect, actual)
		}
	}
}

func TestToCNAME(t *testing.T) {
	for i, test := range []struct {
		input     RR
		expect    CNAME
		shouldErr bool
	}{
		{
			input: RR{
				Name: "@",
				TTL:  5 * time.Minute,
				Type: "CNAME",
				Data: "example.com.",
			},
			expect: CNAME{
				Name:   "@",
				TTL:    5 * time.Minute,
				Target: "example.com.",
			},
		},
	} {
		actual, err := test.input.toCNAME()
		if err == nil && test.shouldErr {
			t.Errorf("Test %d: Expected error, got none", i)
		}
		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error, but got: %v", i, err)
		}
		if !reflect.DeepEqual(actual, test.expect) {
			t.Errorf("Test %d: INPUT=%#v\nEXPECTED: %#v\nACTUAL:   %#v", i, test.input, test.expect, actual)
		}
	}
}

func TestToSVCB(t *testing.T) {
	for i, test := range []struct {
		input     RR
		expect    ServiceBinding
		shouldErr bool
	}{
		{
			input: RR{
				Name: "@",
				TTL:  5 * time.Minute,
				Type: "HTTPS",
				Data: `1 . key=value1,value2 ech="foobar"`,
			},
			expect: ServiceBinding{
				Name:     "@",
				TTL:      5 * time.Minute,
				Scheme:   "https",
				Priority: 1,
				Target:   ".",
				Params: SvcParams{
					"key": []string{"value1", "value2"},
					"ech": []string{"foobar"},
				},
			},
		},
		{
			input: RR{
				Name: "_8443._https.test",
				TTL:  1 * time.Hour,
				Type: "HTTPS",
				Data: "0 example.com.",
			},
			expect: ServiceBinding{
				Name:          "test",
				Scheme:        "https",
				URLSchemePort: 8443,
				TTL:           1 * time.Hour,
				Priority:      0,
				Target:        "example.com.",
				Params:        SvcParams{},
			},
		},
		{
			input: RR{
				Name: "_dns.example.com.",
				TTL:  1 * time.Second,
				Type: "SVCB",
				Data: "2 example.org. alpn=dot",
			},
			expect: ServiceBinding{
				Name:     "example.com.",
				Scheme:   "dns",
				TTL:      1 * time.Second,
				Priority: 2,
				Target:   "example.org.",
				Params: SvcParams{
					"alpn": []string{"dot"},
				},
			},
		},
		{
			input: RR{
				Name: "_853._dns.example.com.",
				TTL:  1 * time.Second,
				Type: "SVCB",
				Data: "1 . port=53",
			},
			expect: ServiceBinding{
				Name:          "example.com.",
				Scheme:        "dns",
				URLSchemePort: 853,
				TTL:           1 * time.Second,
				Priority:      1,
				Target:        ".",
				Params: SvcParams{
					"port": []string{"53"},
				},
			},
		},
		{
			input: RR{
				Name: "_1234._examplescheme",
				TTL:  1 * time.Hour,
				Type: "SVCB",
				Data: "0 example.com.",
			},
			expect: ServiceBinding{
				Name:          "@",
				Scheme:        "examplescheme",
				URLSchemePort: 1234,
				TTL:           1 * time.Hour,
				Priority:      0,
				Target:        "example.com.",
				Params:        SvcParams{},
			},
		},
		{
			input: RR{
				Name: "_examplescheme",
				TTL:  1 * time.Hour,
				Type: "SVCB",
				Data: "0 example.com.",
			},
			expect: ServiceBinding{
				Name:     "@",
				Scheme:   "examplescheme",
				TTL:      1 * time.Hour,
				Priority: 0,
				Target:   "example.com.",
				Params:   SvcParams{},
			},
		},
		{
			input: RR{
				Name: "_examplescheme.@",
				TTL:  1 * time.Hour,
				Type: "SVCB",
				Data: "0 example.com.",
			},
			expect: ServiceBinding{
				Name:     "@",
				Scheme:   "examplescheme",
				TTL:      1 * time.Hour,
				Priority: 0,
				Target:   "example.com.",
				Params:   SvcParams{},
			},
		},
		{
			input: RR{
				Name: "_1234._examplescheme.@",
				TTL:  1 * time.Hour,
				Type: "SVCB",
				Data: "0 example.com.",
			},
			expect: ServiceBinding{
				Name:          "@",
				Scheme:        "examplescheme",
				URLSchemePort: 1234,
				TTL:           1 * time.Hour,
				Priority:      0,
				Target:        "example.com.",
				Params:        SvcParams{},
			},
		},
	} {
		actual, err := test.input.toServiceBinding()
		if err == nil && test.shouldErr {
			t.Errorf("Test %d: Expected error, got none", i)
		}
		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error, but got: %v", i, err)
		}
		if !reflect.DeepEqual(actual, test.expect) {
			t.Errorf("Test %d: INPUT=%#v\nEXPECTED: %+v\nACTUAL:   %+v", i, test.input, test.expect, actual)
		}
	}
}

func TestToMX(t *testing.T) {
	for i, test := range []struct {
		input     RR
		expect    MX
		shouldErr bool
	}{
		{
			input: RR{
				Name: "@",
				TTL:  5 * time.Minute,
				Type: "MX",
				Data: "10 example.com.",
			},
			expect: MX{
				Name:       "@",
				TTL:        5 * time.Minute,
				Preference: 10,
				Target:     "example.com.",
			},
		},
	} {
		actual, err := test.input.toMX()
		if err == nil && test.shouldErr {
			t.Errorf("Test %d: Expected error, got none", i)
		}
		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error, but got: %v", i, err)
		}
		if !reflect.DeepEqual(actual, test.expect) {
			t.Errorf("Test %d: INPUT=%#v\nEXPECTED: %#v\nACTUAL:   %#v", i, test.input, test.expect, actual)
		}
	}
}

func TestToNS(t *testing.T) {
	for i, test := range []struct {
		input     RR
		expect    NS
		shouldErr bool
	}{
		{
			input: RR{
				Name: "@",
				TTL:  5 * time.Minute,
				Type: "NS",
				Data: "example.com.",
			},
			expect: NS{
				Name:   "@",
				TTL:    5 * time.Minute,
				Target: "example.com.",
			},
		},
	} {
		actual, err := test.input.toNS()
		if err == nil && test.shouldErr {
			t.Errorf("Test %d: Expected error, got none", i)
		}
		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error, but got: %v", i, err)
		}
		if !reflect.DeepEqual(actual, test.expect) {
			t.Errorf("Test %d: INPUT=%#v\nEXPECTED: %#v\nACTUAL:   %#v", i, test.input, test.expect, actual)
		}
	}
}

func TestToSRV(t *testing.T) {
	for i, test := range []struct {
		input     RR
		expect    SRV
		shouldErr bool
	}{
		{
			input: RR{
				Name: "_service._proto.name",
				TTL:  5 * time.Minute,
				Type: "SRV",
				Data: "1 2 1234 example.com",
			},
			expect: SRV{
				Service:   "service",
				Transport: "proto",
				Name:      "name",
				TTL:       5 * time.Minute,
				Priority:  1,
				Weight:    2,
				Port:      1234,
				Target:    "example.com",
			},
		},
		{
			input: RR{
				Name: "_service._proto",
				TTL:  5 * time.Minute,
				Type: "SRV",
				Data: "1 2 1234 example.com",
			},
			expect: SRV{
				Service:   "service",
				Transport: "proto",
				Name:      "@",
				TTL:       5 * time.Minute,
				Priority:  1,
				Weight:    2,
				Port:      1234,
				Target:    "example.com",
			},
		},
	} {
		actual, err := test.input.toSRV()
		if err == nil && test.shouldErr {
			t.Errorf("Test %d: Expected error, got none", i)
		}
		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error, but got: %v", i, err)
		}
		if !reflect.DeepEqual(actual, test.expect) {
			t.Errorf("Test %d: INPUT=%#v\nEXPECTED: %+v\nACTUAL:   %+v", i, test.input, test.expect, actual)
		}
	}
}

func TestToTXT(t *testing.T) {
	for i, test := range []struct {
		input     RR
		expect    TXT
		shouldErr bool
	}{
		{
			input: RR{
				Name: "_acme_challenge",
				TTL:  5 * time.Minute,
				Type: "TXT",
				Data: "foobar",
			},
			expect: TXT{
				Name: "_acme_challenge",
				TTL:  5 * time.Minute,
				Text: "foobar",
			},
		},
	} {
		actual, err := test.input.toTXT()
		if err == nil && test.shouldErr {
			t.Errorf("Test %d: Expected error, got none", i)
		}
		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error, but got: %v", i, err)
		}
		if !reflect.DeepEqual(actual, test.expect) {
			t.Errorf("Test %d: INPUT=%#v\nEXPECTED: %#v\nACTUAL:   %#v", i, test.input, test.expect, actual)
		}
	}
}

func TestParseSvcParams(t *testing.T) {
	for i, test := range []struct {
		input     string
		expect    SvcParams
		shouldErr bool
	}{
		{
			input:  "",
			expect: SvcParams{},
		},
		{
			input: `alpn="h2,h3" no-default-alpn ipv6hint=2001:db8::1 port=443`,
			expect: SvcParams{
				"alpn":            {"h2", "h3"},
				"no-default-alpn": {},
				"ipv6hint":        {"2001:db8::1"},
				"port":            {"443"},
			},
		},
		{
			input: `key=value quoted="some string" flag`,
			expect: SvcParams{
				"key":    {"value"},
				"quoted": {"some string"},
				"flag":   {},
			},
		},
		{
			input: `key="nested \"quoted\" value,foobar"`,
			expect: SvcParams{
				"key": {`nested "quoted" value`, "foobar"},
			},
		},
		{
			input: `alpn=h3,h2 tls-supported-groups=29,23 no-default-alpn ech="foobar"`,
			expect: SvcParams{
				"alpn":                 {"h3", "h2"},
				"tls-supported-groups": {"29", "23"},
				"no-default-alpn":      {},
				"ech":                  {"foobar"},
			},
		},
		{
			input: `escape=\097`,
			expect: SvcParams{
				"escape": {"a"},
			},
		},
		{
			input: `escapes=\097\098c`,
			expect: SvcParams{
				"escapes": {"abc"},
			},
		},
	} {
		actual, err := ParseSvcParams(test.input)
		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error, but got: %v (input=%q)", i, err, test.input)
			continue
		} else if err == nil && test.shouldErr {
			t.Errorf("Test %d: Expected an error, but got no error (input=%q)", i, test.input)
			continue
		}
		if !reflect.DeepEqual(test.expect, actual) {
			t.Errorf("Test %d: Expected %v, got %v (input=%q)", i, test.expect, actual, test.input)
			continue
		}
	}
}

func TestSvcParamsString(t *testing.T) {
	// this test relies on the parser also working
	// because we can't just compare string outputs
	// since map iteration is unordered
	for i, test := range []SvcParams{
		{},
		{
			"alpn":            {"h2", "h3"},
			"no-default-alpn": {},
			"ipv6hint":        {"2001:db8::1"},
			"port":            {"443"},
		},
		{
			"key":    {"value"},
			"quoted": {"some string"},
			"flag":   {},
		},
		{
			"key": {`nested "quoted" value`, "foobar"},
		},
		{
			"alpn":                 {"h3", "h2"},
			"tls-supported-groups": {"29", "23"},
			"no-default-alpn":      {},
			"ech":                  {"foobar"},
		},
	} {
		combined := test.String()
		parsed, err := ParseSvcParams(combined)
		if err != nil {
			t.Errorf("Test %d: Expected no error, but got: %v (input=%q)", i, err, test)
			continue
		}
		if len(parsed) != len(test) {
			t.Errorf("Test %d: Expected %d keys, but got %d", i, len(test), len(parsed))
			continue
		}
		for key, expectedVals := range test {
			if expected, actual := len(expectedVals), len(parsed[key]); expected != actual {
				t.Errorf("Test %d: Expected key %s to have %d values, but had %d", i, key, expected, actual)
				continue
			}
			for j, expected := range expectedVals {
				if actual := parsed[key][j]; actual != expected {
					t.Errorf("Test %d key %q value %d: Expected '%s' but got '%s'", i, key, j, expected, actual)
					continue
				}
			}
		}
		if !reflect.DeepEqual(parsed, test) {
			t.Errorf("Test %d: Expected %#v, got %#v", i, test, combined)
			continue
		}
	}
}

func TestRelativeRRNames(t *testing.T) {
	for _, test := range []struct {
		input  Record
		expect string
	}{
		{
			input: ServiceBinding{
				Name:          "@",
				Scheme:        "examplescheme",
				URLSchemePort: 1234,
				TTL:           1 * time.Hour,
				Priority:      1,
				Target:        ".",
				Params:        SvcParams{},
			},
			expect: "_1234._examplescheme",
		},
		{
			input: SRV{
				Name:      "@",
				Service:   "exampleservice",
				Transport: "tcp",
				TTL:       1 * time.Hour,
				Priority:  1,
				Weight:    2,
				Target:    ".",
			},
			expect: "_exampleservice._tcp",
		},
		{
			input: ServiceBinding{
				Name:          "test",
				Scheme:        "examplescheme",
				URLSchemePort: 1234,
				TTL:           1 * time.Hour,
				Priority:      1,
				Target:        ".",
				Params:        SvcParams{},
			},
			expect: "_1234._examplescheme.test",
		},
		{
			input: SRV{
				Name:      "test",
				Service:   "exampleservice",
				Transport: "tcp",
				TTL:       1 * time.Hour,
				Priority:  1,
				Weight:    2,
				Target:    ".",
			},
			expect: "_exampleservice._tcp.test",
		},
	} {
		rr := test.input.RR()
		if rr.Name != test.expect {
			t.Errorf("Expected %q, got %q", test.expect, rr.Name)
		}
	}
}

func TestRRDataZeroValues(t *testing.T) {
	for _, test := range []Record{
		Address{
			Name: "example.com",
		},
		CAA{
			Name: "example.com",
		},
		CNAME{
			Name: "example.com",
		},
		MX{
			Name: "example.com",
		},
		NS{
			Name: "example.com",
		},
		SRV{
			Name:      "example.com",
			Transport: "tcp",
			Service:   "exampleservice",
		},
		ServiceBinding{
			Name:   "example.com",
			Scheme: "https",
		},
		TXT{
			Name: "example.com",
		},
	} {
		rr := test.RR()
		if rr.Data != "" {
			t.Errorf("%s: Expected empty Data, got '%s'", rr.Type, rr.Data)
		}
	}
}
