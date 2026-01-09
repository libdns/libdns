package bluecat

import (
	"context"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// TestProvider tests the provider against a live Bluecat instance
// Set environment variables to run these tests:
// - BLUECAT_SERVER_URL: Bluecat server URL (e.g., https://bluecat.example.com)
// - BLUECAT_USERNAME: API username
// - BLUECAT_PASSWORD: API password
// - BLUECAT_TEST_ZONE: Test zone name (e.g., example.com.)
func TestProvider(t *testing.T) {
	serverURL := os.Getenv("BLUECAT_SERVER_URL")
	username := os.Getenv("BLUECAT_USERNAME")
	password := os.Getenv("BLUECAT_PASSWORD")
	testZone := os.Getenv("BLUECAT_TEST_ZONE")

	if serverURL == "" || username == "" || password == "" || testZone == "" {
		t.Skip("Skipping live tests: environment variables not set")
	}

	provider := &Provider{
		ServerURL: serverURL,
		Username:  username,
		Password:  password,
	}

	ctx := context.Background()

	// Test GetRecords
	t.Run("GetRecords", func(t *testing.T) {
		records, err := provider.GetRecords(ctx, testZone)
		if err != nil {
			t.Fatalf("GetRecords failed: %v", err)
		}
		t.Logf("Found %d records in zone %s", len(records), testZone)
	})

	// Test AppendRecords
	t.Run("AppendRecords", func(t *testing.T) {
		testRecords := []libdns.Record{
			libdns.Address{
				Name: "libdns-test",
				TTL:  300 * time.Second,
				IP:   netip.MustParseAddr("192.0.2.1"),
			},
			libdns.TXT{
				Name: "libdns-test-txt",
				TTL:  300 * time.Second,
				Text: "libdns test record",
			},
		}

		created, err := provider.AppendRecords(ctx, testZone, testRecords)
		if err != nil {
			t.Fatalf("AppendRecords failed: %v", err)
		}

		if len(created) != len(testRecords) {
			t.Errorf("Expected %d records created, got %d", len(testRecords), len(created))
		}

		t.Logf("Created %d records", len(created))

		// Clean up
		defer func() {
			_, _ = provider.DeleteRecords(ctx, testZone, created)
		}()
	})

	// Test SetRecords
	t.Run("SetRecords", func(t *testing.T) {
		testRecords := []libdns.Record{
			libdns.Address{
				Name: "libdns-test-set",
				TTL:  300 * time.Second,
				IP:   netip.MustParseAddr("192.0.2.2"),
			},
		}

		updated, err := provider.SetRecords(ctx, testZone, testRecords)
		if err != nil {
			t.Fatalf("SetRecords failed: %v", err)
		}

		if len(updated) != len(testRecords) {
			t.Errorf("Expected %d records updated, got %d", len(testRecords), len(updated))
		}

		t.Logf("Set %d records", len(updated))

		// Clean up
		defer func() {
			_, _ = provider.DeleteRecords(ctx, testZone, updated)
		}()
	})

	// Test DeleteRecords
	t.Run("DeleteRecords", func(t *testing.T) {
		// First create a record to delete
		testRecords := []libdns.Record{
			libdns.Address{
				Name: "libdns-test-delete",
				TTL:  300 * time.Second,
				IP:   netip.MustParseAddr("192.0.2.3"),
			},
		}

		created, err := provider.AppendRecords(ctx, testZone, testRecords)
		if err != nil {
			t.Fatalf("Failed to create test record: %v", err)
		}

		// Now delete it
		deleted, err := provider.DeleteRecords(ctx, testZone, created)
		if err != nil {
			t.Fatalf("DeleteRecords failed: %v", err)
		}

		if len(deleted) != len(created) {
			t.Errorf("Expected %d records deleted, got %d", len(created), len(deleted))
		}

		t.Logf("Deleted %d records", len(deleted))
	})
}

func TestMatchesRecord(t *testing.T) {
	tests := []struct {
		name     string
		input    libdns.RR
		existing libdns.RR
		want     bool
	}{
		{
			name: "exact match",
			input: libdns.RR{
				Name: "test",
				Type: "A",
				TTL:  300 * time.Second,
				Data: "192.0.2.1",
			},
			existing: libdns.RR{
				Name: "test",
				Type: "A",
				TTL:  300 * time.Second,
				Data: "192.0.2.1",
			},
			want: true,
		},
		{
			name: "different name",
			input: libdns.RR{
				Name: "test1",
				Type: "A",
				TTL:  300 * time.Second,
				Data: "192.0.2.1",
			},
			existing: libdns.RR{
				Name: "test2",
				Type: "A",
				TTL:  300 * time.Second,
				Data: "192.0.2.1",
			},
			want: false,
		},
		{
			name: "empty type in input (wildcard)",
			input: libdns.RR{
				Name: "test",
				Type: "",
				TTL:  300 * time.Second,
				Data: "192.0.2.1",
			},
			existing: libdns.RR{
				Name: "test",
				Type: "A",
				TTL:  300 * time.Second,
				Data: "192.0.2.1",
			},
			want: true,
		},
		{
			name: "empty TTL in input (wildcard)",
			input: libdns.RR{
				Name: "test",
				Type: "A",
				TTL:  0,
				Data: "192.0.2.1",
			},
			existing: libdns.RR{
				Name: "test",
				Type: "A",
				TTL:  300 * time.Second,
				Data: "192.0.2.1",
			},
			want: true,
		},
		{
			name: "empty data in input (wildcard)",
			input: libdns.RR{
				Name: "test",
				Type: "A",
				TTL:  300 * time.Second,
				Data: "",
			},
			existing: libdns.RR{
				Name: "test",
				Type: "A",
				TTL:  300 * time.Second,
				Data: "192.0.2.1",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesRecord(tt.input, tt.existing)
			if got != tt.want {
				t.Errorf("matchesRecord() = %v, want %v", got, tt.want)
			}
		})
	}
}
