package libdns

import (
	"fmt"
	"testing"
)

func ExampleRelativeName() {
	fmt.Println(RelativeName("sub.example.com.", "example.com."))
	// Output: sub
}

func ExampleAbsoluteName() {
	fmt.Println(AbsoluteName("sub", "example.com."))
	// Output: sub.example.com.
}

func TestRelativeName(t *testing.T) {
	for i, test := range []struct {
		fqdn, zone string
		expect     string
	}{
		{
			fqdn:   "",
			zone:   "",
			expect: "",
		},
		{
			fqdn:   "",
			zone:   "example.com",
			expect: "",
		},
		{
			fqdn:   "example.com",
			zone:   "",
			expect: "example.com",
		},
		{
			fqdn:   "sub.example.com",
			zone:   "example.com",
			expect: "sub",
		},
		{
			fqdn:   "foo.bar.example.com",
			zone:   "bar.example.com",
			expect: "foo",
		},
		{
			fqdn:   "foo.bar.example.com",
			zone:   "example.com",
			expect: "foo.bar",
		},
		{
			fqdn:   "foo.bar.example.com.",
			zone:   "example.com.",
			expect: "foo.bar",
		},
		{
			fqdn:   "foo.bar.example.com",
			zone:   "example.com.",
			expect: "foo.bar",
		},
		{
			fqdn:   "foo.bar.example.com.",
			zone:   "example.com",
			expect: "foo.bar",
		},
		{
			fqdn:   "example.com",
			zone:   "example.net",
			expect: "example.com",
		},
	} {
		actual := RelativeName(test.fqdn, test.zone)
		if actual != test.expect {
			t.Errorf("Test %d: FQDN=%s ZONE=%s - expected '%s' but got '%s'",
				i, test.fqdn, test.zone, test.expect, actual)
		}
	}
}

func TestAbsoluteName(t *testing.T) {
	for i, test := range []struct {
		name, zone string
		expect     string
	}{
		{
			name:   "",
			zone:   "example.com",
			expect: "example.com",
		},
		{
			name:   "@",
			zone:   "example.com.",
			expect: "example.com.",
		},
		{
			name:   "www",
			zone:   "example.com.",
			expect: "www.example.com.",
		},
		{
			name:   "www",
			zone:   "example.com.",
			expect: "www.example.com.",
		},
		{
			name:   "www.",
			zone:   "example.com.",
			expect: "www.example.com.",
		},
		{
			name:   "foo.bar",
			zone:   "example.com.",
			expect: "foo.bar.example.com.",
		},
		{
			name:   "foo.bar.",
			zone:   "example.com.",
			expect: "foo.bar.example.com.",
		},
		{
			name:   "foo",
			zone:   "",
			expect: "foo",
		},
	} {
		actual := AbsoluteName(test.name, test.zone)
		if actual != test.expect {
			t.Errorf("Test %d: NAME=%s ZONE=%s - expected '%s' but got '%s'",
				i, test.name, test.zone, test.expect, actual)
		}
	}
}

func TestSRVRecords(t *testing.T) {
	for i, test := range []struct {
		rec Record
		srv SRV
	}{
		{
			rec: Record{
				Type:     "SRV",
				Name:     "_service._proto.name",
				Priority: 15,
				Weight:   30,
				Value:    "5223 example.com",
			},
			srv: SRV{
				Service:  "service",
				Proto:    "proto",
				Name:     "name",
				Priority: 15,
				Weight:   30,
				Port:     5223,
				Target:   "example.com",
			},
		},
		{
			rec: Record{
				Type:     "SRV",
				Name:     "_service._proto.sub.example",
				Priority: 15,
				Weight:   30,
				Value:    "5223 foo",
			},
			srv: SRV{
				Service:  "service",
				Proto:    "proto",
				Name:     "sub.example",
				Priority: 15,
				Weight:   30,
				Port:     5223,
				Target:   "foo",
			},
		},
	} {
		// Record -> SRV
		actualSRV, err := test.rec.ToSRV()
		if err != nil {
			t.Errorf("Test %d: Record -> SRV: Expected no error, but got: %v", i, err)
			continue
		}
		if actualSRV != test.srv {
			t.Errorf("Test %d: Record -> SRV: For record %+v:\nEXPECTED %+v\nGOT      %+v",
				i, test.rec, test.srv, actualSRV)
		}

		// Record -> SRV
		actualRec := test.srv.ToRecord()
		if actualRec != test.rec {
			t.Errorf("Test %d: SRV -> Record: For SRV %+v:\nEXPECTED %+v\nGOT      %+v",
				i, test.srv, test.rec, actualRec)
		}
	}
}
