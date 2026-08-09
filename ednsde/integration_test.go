//go:build integration

// These tests talk to the real eDNS challenge API and write into a real zone.
// They are excluded from normal builds; run them with:
//
//	EDNS_TOKEN=... EDNS_TEST_ZONE=winkler.tel go test -tags integration -v ./...
//
// Every record they create is removed again, including when an assertion fails.
package ednsde

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// testNamePrefix makes any record these tests leave behind recognisable in the
// zone. There is no listing endpoint, so a human reading the zone is the only
// way a leftover is ever found.
const testNamePrefix = "_acme-challenge.libdns-ednsde-test"

// uniqueName returns a record name used by exactly one test in exactly one run.
//
// Tests must not share a name. eDNS confirms a deletion immediately but keeps
// serving the old contents of that name for about five minutes, so a test that
// wrote to a name another test had just emptied would not see its own records
// and would fail on a timeout that has nothing to do with what it is testing.
func uniqueName(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generating a unique record name: %v", err)
	}
	return fmt.Sprintf("%s-%x", testNamePrefix, buf)
}

// propagationTimeout bounds how long we wait for a record to show up on all
// authoritative nameservers. Measured against winkler.tel, a newly created name
// took about 28 seconds, so this leaves generous headroom.
const propagationTimeout = 2 * time.Minute

func testProvider(t *testing.T) (*Provider, string) {
	t.Helper()
	token := os.Getenv("EDNS_TOKEN")
	zone := os.Getenv("EDNS_TEST_ZONE")
	if token == "" || zone == "" {
		t.Skip("set EDNS_TOKEN and EDNS_TEST_ZONE to run integration tests")
	}
	return &Provider{APIToken: token}, zone
}

// randomValue returns a 43-character value in the base64url alphabet, the same
// shape as a real ACME key authorization digest.
func randomValue(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generating a random value: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// authoritativeResolvers returns one resolver per authoritative nameserver of
// the zone, so that propagation is observed at the source rather than through
// a cache.
func authoritativeResolvers(t *testing.T, zone string) map[string]*net.Resolver {
	t.Helper()
	nss, err := net.LookupNS(zone)
	if err != nil {
		t.Fatalf("looking up the nameservers of %s: %v", zone, err)
	}
	if len(nss) == 0 {
		t.Fatalf("%s has no nameservers", zone)
	}

	resolvers := make(map[string]*net.Resolver, len(nss))
	for _, ns := range nss {
		server := net.JoinHostPort(ns.Host, "53")
		resolvers[ns.Host] = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, server)
			},
		}
	}
	return resolvers
}

// waitForTXT blocks until every authoritative nameserver reports the expected
// presence of value at fqdn.
func waitForTXT(t *testing.T, resolvers map[string]*net.Resolver, fqdn, value string, wantPresent bool) {
	t.Helper()
	deadline := time.Now().Add(propagationTimeout)
	start := time.Now()

	for {
		agreed := 0
		for _, r := range resolvers {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			values, err := r.LookupTXT(ctx, fqdn)
			cancel()
			// NXDOMAIN is a legitimate "not there (yet)" answer.
			if err != nil && wantPresent {
				continue
			}
			if slices.Contains(values, value) == wantPresent {
				agreed++
			}
		}
		if agreed == len(resolvers) {
			t.Logf("all %d nameservers agree that present=%v after %s",
				len(resolvers), wantPresent, time.Since(start).Round(time.Second))
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d of %d nameservers reported present=%v for %s within %s",
				agreed, len(resolvers), wantPresent, fqdn, propagationTimeout)
		}
		time.Sleep(5 * time.Second)
	}
}

func TestIntegrationRoundTrip(t *testing.T) {
	p, zone := testProvider(t)
	ctx := context.Background()
	name := uniqueName(t)
	value := randomValue(t)
	rec := libdns.TXT{Name: name, Text: value}
	fqdn := fmt.Sprintf("%s.%s", name, zone)

	// Cleanup runs even if an assertion below fails, so the zone never keeps a
	// leftover record.
	t.Cleanup(func() {
		if _, err := p.DeleteRecords(context.Background(), zone, []libdns.Record{rec}); err != nil {
			t.Errorf("cleanup failed, %s may still hold a record: %v", fqdn, err)
		}
	})

	added, err := p.AppendRecords(ctx, zone, []libdns.Record{rec})
	if err != nil {
		t.Fatalf("AppendRecords: %v", err)
	}
	if len(added) != 1 {
		t.Fatalf("want 1 record back, got %d", len(added))
	}
	txt, ok := added[0].(libdns.TXT)
	if !ok {
		t.Fatalf("want libdns.TXT back, got %T", added[0])
	}
	if txt.TTL != ednsTTL {
		t.Errorf("want the TTL eDNS actually applies (%v), got %v", ednsTTL, txt.TTL)
	}

	resolvers := authoritativeResolvers(t, zone)
	waitForTXT(t, resolvers, fqdn, value, true)

	// Adding the same value again must be reported as success without creating
	// a second record.
	again, err := p.AppendRecords(ctx, zone, []libdns.Record{rec})
	if err != nil {
		t.Fatalf("re-adding the same record must succeed: %v", err)
	}
	if len(again) != 1 {
		t.Errorf("want the existing record reported back, got %d", len(again))
	}

	removed, err := p.DeleteRecords(ctx, zone, []libdns.Record{rec})
	if err != nil {
		t.Fatalf("DeleteRecords: %v", err)
	}
	if len(removed) != 1 {
		t.Fatalf("want 1 record deleted, got %d", len(removed))
	}

	// Deleting again finds nothing: no error, and nothing reported as deleted.
	none, err := p.DeleteRecords(ctx, zone, []libdns.Record{rec})
	if err != nil {
		t.Fatalf("deleting an absent record must not fail: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("want nothing reported as deleted, got %d", len(none))
	}

	// Deliberately no wait for the record to disappear from DNS. eDNS confirms
	// the deletion synchronously (result code 3, then 5), but takes minutes to
	// reflect it on the nameservers -- far longer than it takes to publish one.
	// That lag is eDNS's behaviour, not this provider's, and waiting for it
	// would make the test slow and flaky without testing anything we control.
	// It is harmless for ACME: validation succeeds as long as *some* TXT record
	// at the name carries the expected value.
}

// TestIntegrationTwoValuesShareOneName covers the layout a SAN certificate for
// both "example.com" and "*.example.com" produces: two different challenge
// values on the very same record name.
func TestIntegrationTwoValuesShareOneName(t *testing.T) {
	p, zone := testProvider(t)
	ctx := context.Background()

	name := uniqueName(t)
	first := libdns.TXT{Name: name, Text: randomValue(t)}
	second := libdns.TXT{Name: name, Text: randomValue(t)}
	recs := []libdns.Record{first, second}

	t.Cleanup(func() {
		if _, err := p.DeleteRecords(context.Background(), zone, recs); err != nil {
			t.Errorf("cleanup failed: %v", err)
		}
	})

	added, err := p.AppendRecords(ctx, zone, recs)
	if err != nil {
		t.Fatalf("AppendRecords: %v", err)
	}
	if len(added) != 2 {
		t.Fatalf("want 2 records back, got %d", len(added))
	}

	fqdn := fmt.Sprintf("%s.%s", name, zone)
	resolvers := authoritativeResolvers(t, zone)
	waitForTXT(t, resolvers, fqdn, first.Text, true)
	waitForTXT(t, resolvers, fqdn, second.Text, true)

	removed, err := p.DeleteRecords(ctx, zone, recs)
	if err != nil {
		t.Fatalf("DeleteRecords: %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("want 2 records deleted, got %d", len(removed))
	}
}

// TestIntegrationRejectsForeignZone documents that eDNS answers a zone the
// token is not assigned to with 401, indistinguishable from a bad token.
func TestIntegrationRejectsForeignZone(t *testing.T) {
	p, _ := testProvider(t)

	_, err := p.AppendRecords(context.Background(), "example.com.",
		[]libdns.Record{libdns.TXT{Name: uniqueName(t), Text: randomValue(t)}})
	if err == nil {
		t.Fatal("want an error for a zone the token is not assigned to")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("want HTTP 401, got %d", apiErr.StatusCode)
	}
}
