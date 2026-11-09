package binarylane_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/libdns/binarylane"
	"github.com/libdns/libdns"
)

// TestLive exercises the full record lifecycle against the real API. It is
// skipped unless both of these are set:
//
//	BINARYLANE_API_TOKEN   a Binary Lane API token
//	BINARYLANE_TEST_ZONE   a zone on that account the test may modify
//
// It only ever touches records under a unique `libdns-test-<timestamp>` label
// and removes them again, so it will not disturb other records in the zone.
//
//	BINARYLANE_API_TOKEN=... BINARYLANE_TEST_ZONE=example.com go test -run TestLive -v
func TestLive(t *testing.T) {
	token := os.Getenv("BINARYLANE_API_TOKEN")
	zone := os.Getenv("BINARYLANE_TEST_ZONE")

	if token == "" || zone == "" {
		t.Skip("set BINARYLANE_API_TOKEN and BINARYLANE_TEST_ZONE to run the live test")
	}

	p := &binarylane.Provider{APIToken: token}
	ctx := context.Background()

	label := fmt.Sprintf("libdns-test-%d", time.Now().Unix())

	// Remove anything this test leaves behind, however it exits.
	t.Cleanup(func() {
		leftovers, err := p.GetRecords(context.Background(), zone)
		if err != nil {
			t.Logf("cleanup: listing records: %v", err)
			return
		}

		var ours []libdns.Record
		for _, rec := range leftovers {
			if rec.RR().Name == label {
				ours = append(ours, rec)
			}
		}
		if len(ours) == 0 {
			return
		}

		if _, err := p.DeleteRecords(context.Background(), zone, ours); err != nil {
			t.Logf("cleanup: deleting %d record(s) named %q: %v", len(ours), label, err)
		} else {
			t.Logf("cleanup: removed %d leftover record(s)", len(ours))
		}
	})

	t.Run("ListZones includes the test zone", func(t *testing.T) {
		zones, err := p.ListZones(ctx)
		if err != nil {
			t.Fatalf("ListZones() error = %v", err)
		}

		want := libdns.AbsoluteName(zone, ".")
		for _, z := range zones {
			if z.Name == want {
				return
			}
		}

		t.Errorf("ListZones() did not include %q; got %v", want, zones)
	})

	t.Run("AppendRecords creates a record", func(t *testing.T) {
		created, err := p.AppendRecords(ctx, zone, []libdns.Record{
			libdns.TXT{Name: label, TTL: time.Minute, Text: "first"},
		})
		if err != nil {
			t.Fatalf("AppendRecords() error = %v", err)
		}
		if len(created) != 1 {
			t.Fatalf("AppendRecords() returned %d records, want 1", len(created))
		}

		if got := findTXT(t, p, ctx, zone, label); len(got) != 1 || got[0] != "first" {
			t.Errorf("after append, TXT values = %v, want [first]", got)
		}
	})

	t.Run("SetRecords replaces the RRset", func(t *testing.T) {
		if _, err := p.SetRecords(ctx, zone, []libdns.Record{
			libdns.TXT{Name: label, TTL: time.Minute, Text: "second"},
		}); err != nil {
			t.Fatalf("SetRecords() error = %v", err)
		}

		if got := findTXT(t, p, ctx, zone, label); len(got) != 1 || got[0] != "second" {
			t.Errorf("after set, TXT values = %v, want [second]", got)
		}
	})

	t.Run("SetRecords grows and shrinks the RRset", func(t *testing.T) {
		if _, err := p.SetRecords(ctx, zone, []libdns.Record{
			libdns.TXT{Name: label, TTL: time.Minute, Text: "a"},
			libdns.TXT{Name: label, TTL: time.Minute, Text: "b"},
		}); err != nil {
			t.Fatalf("SetRecords() growing error = %v", err)
		}
		if got := findTXT(t, p, ctx, zone, label); len(got) != 2 {
			t.Fatalf("after growing, got %d values %v, want 2", len(got), got)
		}

		if _, err := p.SetRecords(ctx, zone, []libdns.Record{
			libdns.TXT{Name: label, TTL: time.Minute, Text: "only"},
		}); err != nil {
			t.Fatalf("SetRecords() shrinking error = %v", err)
		}

		got := findTXT(t, p, ctx, zone, label)
		if len(got) != 1 || got[0] != "only" {
			t.Errorf("after shrinking, TXT values = %v, want [only] — surplus not deleted", got)
		}
	})

	t.Run("DeleteRecords removes the record", func(t *testing.T) {
		all, err := p.GetRecords(ctx, zone)
		if err != nil {
			t.Fatalf("GetRecords() error = %v", err)
		}

		var ours []libdns.Record
		for _, rec := range all {
			if rec.RR().Name == label {
				ours = append(ours, rec)
			}
		}

		deleted, err := p.DeleteRecords(ctx, zone, ours)
		if err != nil {
			t.Fatalf("DeleteRecords() error = %v", err)
		}
		if len(deleted) != len(ours) {
			t.Errorf("DeleteRecords() reported %d deleted, want %d", len(deleted), len(ours))
		}

		if got := findTXT(t, p, ctx, zone, label); len(got) != 0 {
			t.Errorf("after delete, TXT values = %v, want none", got)
		}
	})
}

func findTXT(t *testing.T, p *binarylane.Provider, ctx context.Context, zone, name string) []string {
	t.Helper()

	recs, err := p.GetRecords(ctx, zone)
	if err != nil {
		t.Fatalf("GetRecords() error = %v", err)
	}

	var values []string
	for _, rec := range recs {
		if txt, ok := rec.(libdns.TXT); ok && txt.Name == name {
			values = append(values, txt.Text)
		}
	}

	return values
}
