package domainchief

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// TestIntegration runs against the real Domain Chief API and changes records in the test zone.
//
// Run it with a token that has the `domainchief:dns:read`, `domainchief:dns:write` and `domainchief:acme:write` scopes:
//
//	DOMAINCHIEF_API_TOKEN=ctt_... DOMAINCHIEF_TEST_ZONE=example.com. go test -run TestIntegration -v ./...
func TestIntegration(t *testing.T) {
	token, zone := os.Getenv("DOMAINCHIEF_API_TOKEN"), os.Getenv("DOMAINCHIEF_TEST_ZONE")
	if token == "" || zone == "" {
		t.Skip("set DOMAINCHIEF_API_TOKEN and DOMAINCHIEF_TEST_ZONE to run the integration test")
	}

	provider := &Provider{APIToken: token, BaseURL: os.Getenv("DOMAINCHIEF_API_BASE_URL")}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	suffix := time.Now().Format("20060102150405")

	t.Run("acme challenge", func(t *testing.T) {
		challenge := libdns.TXT{Name: "_acme-challenge.libdns-" + suffix, Text: "libdns-test-" + suffix}

		appended, err := provider.AppendRecords(ctx, zone, []libdns.Record{challenge, challenge})
		if err != nil {
			t.Fatal(err)
		}

		if len(appended) != 2 {
			t.Fatalf("expected 2 appended records, got %v", appended)
		}

		if count := countRecords(t, provider, ctx, zone, challenge); count != 1 {
			t.Errorf("expected the challenge to be published once, found %d", count)
		}

		if _, err := provider.DeleteRecords(ctx, zone, appended[:1]); err != nil {
			t.Fatal(err)
		}

		if count := countRecords(t, provider, ctx, zone, challenge); count != 0 {
			t.Errorf("expected the challenge to be deleted, found %d", count)
		}
	})

	t.Run("record lifecycle", func(t *testing.T) {
		name := "libdns-" + suffix
		first := libdns.Address{Name: name, TTL: 5 * time.Minute, IP: netip.MustParseAddr("192.0.2.1")}
		second := libdns.Address{Name: name, TTL: 5 * time.Minute, IP: netip.MustParseAddr("192.0.2.2")}

		t.Cleanup(func() {
			_, _ = provider.DeleteRecords(context.Background(), zone, []libdns.Record{libdns.RR{Name: name}})
		})

		if _, err := provider.AppendRecords(ctx, zone, []libdns.Record{first}); err != nil {
			t.Fatal(err)
		}

		if count := countRecords(t, provider, ctx, zone, first); count != 1 {
			t.Fatalf("expected the record to be created, found %d", count)
		}

		if _, err := provider.SetRecords(ctx, zone, []libdns.Record{second}); err != nil {
			t.Fatal(err)
		}

		if count := countRecords(t, provider, ctx, zone, first); count != 0 {
			t.Errorf("expected the first value to be replaced, found %d", count)
		}

		if count := countRecords(t, provider, ctx, zone, second); count != 1 {
			t.Errorf("expected the second value to be set, found %d", count)
		}

		deleted, err := provider.DeleteRecords(ctx, zone, []libdns.Record{second})
		if err != nil {
			t.Fatal(err)
		}

		if len(deleted) != 1 {
			t.Errorf("expected 1 deleted record, got %v", deleted)
		}
	})
}

func countRecords(t *testing.T, provider *Provider, ctx context.Context, zone string, wanted libdns.Record) int {
	t.Helper()

	records, err := provider.GetRecords(ctx, zone)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	want := wanted.RR()

	for _, record := range records {
		rr := record.RR()

		if rr.Name == want.Name && rr.Type == want.Type && rr.Data == want.Data {
			count++
		}
	}

	if testing.Verbose() {
		fmt.Printf("found %d %s records for %s\n", count, want.Type, want.Name)
	}

	return count
}
