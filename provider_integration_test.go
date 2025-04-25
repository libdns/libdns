package simplydotcom

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// TestProvider_Integration tests the Provider against the real Simply.com API
// This test will only run if the required environment variables are set, see below.
// It will cretae, modify and delete records in the test zone. It should not affect any records
// in the test zone that are present before tests are run, and it attempt to clean up after itself.
//
// Note that if an error occurs, it may modify the zone so make sure you have a backup before running
// these tests, or run them in a test zone that you don't mind if modified.

// Define environment variable names
const (
	envTestZone    = "SIMPLY_TEST_ZONE"    // The zone for testing, eg. example.com
	envAccountName = "SIMPLY_ACCOUNT_NAME" // The account name for the Simply.com account, eg. S123456
	envAPIKey      = "SIMPLY_API_KEY"      // The API key for the Simply.com account
)

func TestProvider_Integration(t *testing.T) {
	// Check if integration tests are enabled
	apiKey := os.Getenv(envAPIKey)
	accountName := os.Getenv(envAccountName)
	testZone := os.Getenv(envTestZone)

	if apiKey == "" || accountName == "" || testZone == "" {
		t.Skip("Skipping integration tests. To run integration tests, set the following environment variables: " +
			"SIMPLY_API_KEY, SIMPLY_ACCOUNT_NAME, SIMPLY_TEST_ZONE")
	}

	// Ensure test zone ends with a dot
	if !strings.HasSuffix(testZone, ".") {
		testZone = testZone + "."
	}

	// Create a provider with the given credentials
	provider := Provider{
		APIKey:      apiKey,
		AccountName: accountName,
	}

	// Create a context for all operations
	ctx := context.Background()

	// Generate a unique prefix for test records to avoid conflicts
	// and make it easy to identify and clean up test records if needed
	testPrefix := fmt.Sprintf("libdnstest-")

	// Execute the tests
	t.Run("Integration tests", func(t *testing.T) {
		// Test GetRecords
		t.Run("GetRecords", func(t *testing.T) {
			runGetRecordsTest(t, ctx, &provider, testZone, testPrefix)
		})

		// Test AppendRecords with all record types
		t.Run("AppendRecords", func(t *testing.T) {
			runAppendRecordsTest(t, ctx, &provider, testZone, testPrefix)
		})

		// Test SetRecords with all record types
		t.Run("SetRecords", func(t *testing.T) {
			runSetRecordsTest(t, ctx, &provider, testZone, testPrefix)
		})

		// Test DeleteRecords
		t.Run("DeleteRecords", func(t *testing.T) {
			runDeleteRecordsTest(t, ctx, &provider, testZone, testPrefix)
		})
	})
}

// runGetRecordsTest tests the GetRecords method
func runGetRecordsTest(t *testing.T, ctx context.Context, provider *Provider, zone string, prefix string) {
	// Create test records to ensure we have something to retrieve
	testRecords := []libdns.Record{
		libdns.Address{
			Name: prefix + "a",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("198.51.100.10"),
		},
		libdns.TXT{
			Name: prefix + "txt",
			TTL:  300 * time.Second,
			Text: "GetRecords test",
		},
	}

	// Add the test records
	addedRecords, err := provider.AppendRecords(ctx, zone, testRecords)
	if err != nil {
		t.Fatalf("Failed to add test records: %v", err)
	}

	// Clean up at the end
	defer func() {
		_, err := provider.DeleteRecords(ctx, zone, addedRecords)
		if err != nil {
			t.Logf("Warning: Failed to clean up test records: %v", err)
		}
	}()

	// Now test GetRecords
	records, err := provider.GetRecords(ctx, zone)
	if err != nil {
		t.Fatalf("Failed to get records: %v", err)
	}

	// Verify we got records back
	if len(records) == 0 {
		t.Errorf("Expected to get at least some records, but got none")
	}

	t.Logf("Successfully retrieved %d records from zone %s", len(records), zone)

	// Specifically verify our test records were retrieved
	foundCount := 0
	for _, testRecord := range testRecords {
		for _, record := range records {
			if areRecordsEqual(record, testRecord, zone) {
				foundCount++
				break
			}
		}
	}

	if foundCount != len(testRecords) {
		t.Errorf("Expected to find all %d test records, but found only %d",
			len(testRecords), foundCount)
	}
}

// generateTestRecords generates a set of test records for all supported record types
func generateTestRecords(prefix string) []libdns.Record {
	return []libdns.Record{
		// A record
		libdns.Address{
			Name: prefix + "a",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("198.51.100.1"),
		},
		// AAAA record
		libdns.Address{
			Name: prefix + "aaaa",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("2001:db8::1"),
		},
		// CNAME record
		libdns.CNAME{
			Name:   prefix + "cname",
			TTL:    300 * time.Second,
			Target: "example.com.",
		},
		// MX record
		libdns.MX{
			Name:       prefix + "mx",
			TTL:        300 * time.Second,
			Preference: 10,
			Target:     "mail.example.com.",
		},
		// TXT record
		libdns.TXT{
			Name: prefix + "txt",
			TTL:  300 * time.Second,
			Text: "v=spf1 -all",
		},
		// SRV record
		libdns.SRV{
			Service:   "minecraft",
			Transport: "tcp",
			Name:      prefix + "srv",
			TTL:       300 * time.Second,
			Priority:  10,
			Weight:    20,
			Port:      443,
			Target:    "service.example.com.",
		},
		// CAA record (if supported)
		libdns.CAA{
			Name:  prefix + "caa",
			TTL:   300 * time.Second,
			Tag:   "issue",
			Value: "letsencrypt.org",
		},
		// NS record
		libdns.NS{
			Name:   prefix + "ns",
			TTL:    300 * time.Second,
			Target: "ns.example.com.",
		},
	}
}

// runAppendRecordsTest tests the AppendRecords method with all record types
func runAppendRecordsTest(t *testing.T, ctx context.Context, provider *Provider, zone string, prefix string) {
	// Generate test records
	recordsToAdd := generateTestRecords(prefix)

	// Add records
	addedRecords, err := provider.AppendRecords(ctx, zone, recordsToAdd)
	if err != nil {
		t.Fatalf("Failed to append records: %v", err)
	}

	// Clean up at the end
	defer func() {
		_, err := provider.DeleteRecords(ctx, zone, addedRecords)
		if err != nil {
			t.Logf("Warning: Failed to clean up test records: %v", err)
		}
	}()

	// Verify all records were added
	if len(addedRecords) != len(recordsToAdd) {
		t.Errorf("Expected %d records to be added, got %d", len(recordsToAdd), len(addedRecords))
	}

	// Verify each record was added correctly
	verifyRecords(t, addedRecords, recordsToAdd, zone)

	// Verify records are retrievable
	allRecords, err := provider.GetRecords(ctx, zone)
	if err != nil {
		t.Fatalf("Failed to get records after append: %v", err)
	}

	// Verify our test records exist in the full record set
	for _, recordToFind := range recordsToAdd {
		found := false
		for _, record := range allRecords {
			if areRecordsEqual(record, recordToFind, zone) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Record %s of type %s not found in zone after append",
				recordToFind.RR().Name, recordToFind.RR().Type)
		}
	}
}

// runSetRecordsTest tests the SetRecords method with all record types
func runSetRecordsTest(t *testing.T, ctx context.Context, provider *Provider, zone string, prefix string) {
	// First add some initial records that we'll modify
	initialRecords := []libdns.Record{
		// Multiple A records for same name
		libdns.Address{
			Name: prefix + "multi-a",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.1"),
		},
		libdns.Address{
			Name: prefix + "multi-a",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.2"),
		},
		// Single AAAA record that will be replaced with multiple
		libdns.Address{
			Name: prefix + "multi-aaaa",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("2001:db8::1"),
		},
		// TXT record that will remain untouched
		libdns.TXT{
			Name: prefix + "unchanged",
			TTL:  300 * time.Second,
			Text: "this record should remain unchanged",
		},
	}

	// Add the initial records
	_, err := provider.AppendRecords(ctx, zone, initialRecords)
	if err != nil {
		t.Fatalf("Failed to add initial records: %v", err)
	}

	// Generate records to set - this will:
	// 1. Replace multiple A records with a single one (multi-a)
	// 2. Replace single AAAA record with multiple ones (multi-aaaa)
	// 3. Leave the TXT record untouched
	recordsToSet := []libdns.Record{
		// Single A record replacing multiple
		libdns.Address{
			Name: prefix + "multi-a",
			TTL:  600 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.3"),
		},
		// Multiple AAAA records replacing single
		libdns.Address{
			Name: prefix + "multi-aaaa",
			TTL:  600 * time.Second,
			IP:   netip.MustParseAddr("2001:db8::2"),
		},
		libdns.Address{
			Name: prefix + "multi-aaaa",
			TTL:  600 * time.Second,
			IP:   netip.MustParseAddr("2001:db8::3"),
		},
	}

	// Set records
	updatedRecords, err := provider.SetRecords(ctx, zone, recordsToSet)
	if err != nil {
		t.Fatalf("Failed to set records: %v", err)
	}

	// Verify via GetRecords
	allRecordsAfterSet, err := provider.GetRecords(ctx, zone)
	if err != nil {
		t.Fatalf("Failed to get records after set: %v", err)
	}

	// Helper to count records of specific name and type
	countRecords := func(records []libdns.Record, name, recordType string) int {
		count := 0
		for _, r := range records {
			rr := r.RR()
			if rr.Name == name && rr.Type == recordType {
				count++
			}
		}
		return count
	}

	// Verify single A record replaced multiple
	aCount := countRecords(allRecordsAfterSet, prefix+"multi-a", "A")
	if aCount != 1 {
		t.Errorf("Expected 1 A record for %s, got %d", prefix+"multi-a", aCount)
	}

	// Verify multiple AAAA records replaced single
	aaaaCount := countRecords(allRecordsAfterSet, prefix+"multi-aaaa", "AAAA")
	if aaaaCount != 2 {
		t.Errorf("Expected 2 AAAA records for %s, got %d", prefix+"multi-aaaa", aaaaCount)
	}

	// Verify TXT record still exists and is unchanged
	found := false
	for _, record := range allRecordsAfterSet {
		rr := record.RR()
		if rr.Name == prefix+"unchanged" && rr.Type == "TXT" {
			found = true
			txt, ok := record.(libdns.TXT)
			if !ok {
				t.Errorf("Expected TXT record but got different type")
				continue
			}
			if txt.Text != "this record should remain unchanged" {
				t.Errorf("TXT record was modified when it should have remained unchanged")
			}
			break
		}
	}
	if !found {
		t.Errorf("TXT record was deleted when it should have remained unchanged")
	}

	// Clean up
	allTestRecords := append([]libdns.Record{}, updatedRecords...)
	for _, record := range allRecordsAfterSet {
		if strings.HasPrefix(record.RR().Name, prefix) {
			allTestRecords = append(allTestRecords, record)
		}
	}
	_, err = provider.DeleteRecords(ctx, zone, allTestRecords)
	if err != nil {
		t.Logf("Warning: Failed to clean up test records: %v", err)
	}
}

// runDeleteRecordsTest tests the DeleteRecords method
func runDeleteRecordsTest(t *testing.T, ctx context.Context, provider *Provider, zone string, prefix string) {
	// First add test records of all types that we'll delete
	recordsToAdd := generateTestRecords(prefix)

	// Add the records
	addedRecords, err := provider.AppendRecords(ctx, zone, recordsToAdd)
	if err != nil {
		t.Fatalf("Failed to add test records: %v", err)
	}

	// Verify records were added successfully
	allRecordsBeforeDelete, err := provider.GetRecords(ctx, zone)
	if err != nil {
		t.Fatalf("Failed to get records after adding: %v", err)
	}

	// Verify all test records exist before deletion
	for _, recordToFind := range recordsToAdd {
		found := false
		for _, record := range allRecordsBeforeDelete {
			if areRecordsEqual(record, recordToFind, zone) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Test record %s of type %s not found before deletion",
				recordToFind.RR().Name, recordToFind.RR().Type)
		}
	}

	// Delete the records
	deletedRecords, err := provider.DeleteRecords(ctx, zone, addedRecords)
	if err != nil {
		t.Fatalf("Failed to delete records: %v", err)
	}

	// Verify all records were deleted
	if len(deletedRecords) != len(addedRecords) {
		t.Errorf("Expected %d records to be deleted, got %d", len(addedRecords), len(deletedRecords))
	}

	// Verify records were actually deleted from the zone
	recordsAfterDelete, err := provider.GetRecords(ctx, zone)
	if err != nil {
		t.Fatalf("Failed to get records after delete: %v", err)
	}

	// Check that none of our test records exist anymore
	for _, record := range recordsAfterDelete {
		if strings.HasPrefix(record.RR().Name, prefix) {
			t.Errorf("Record %s of type %s still exists after deletion",
				record.RR().Name, record.RR().Type)
		}
	}
}

// verifyRecords compares the actual records returned by the API with what we expect
func verifyRecords(t *testing.T, actual []libdns.Record, expected []libdns.Record, zone string) {
	t.Helper()

	// First check we have same number of records
	if len(actual) != len(expected) {
		t.Errorf("Record count mismatch: got %d, want %d", len(actual), len(expected))
		return
	}

	// Make copies of slices so we can mark matches
	remainingActual := make([]libdns.Record, len(actual))
	copy(remainingActual, actual)
	remainingExpected := make([]libdns.Record, len(expected))
	copy(remainingExpected, expected)

	// For each expected record, find matching actual record
	for _, expectedRecord := range remainingExpected {
		found := false
		for j, actualRecord := range remainingActual {
			if actualRecord != nil && areRecordsEqual(actualRecord, expectedRecord, zone) {
				// Mark as matched by setting to nil
				remainingActual[j] = nil
				found = true
				break
			}
		}

		if !found {
			expectedRR := expectedRecord.RR()
			t.Errorf("Expected record not found: %s of type %s with data %s",
				expectedRR.Name, expectedRR.Type, expectedRR.Data)
		}
	}

	// Check for any unmatched actual records
	for _, actualRecord := range remainingActual {
		if actualRecord != nil {
			rr := actualRecord.RR()
			t.Errorf("Unexpected record found: %s of type %s with data %s",
				rr.Name, rr.Type, rr.Data)
		}
	}
}

// areRecordsEqual checks if two records are semantically equal
// This is a flexible comparison that ignores minor differences
func areRecordsEqual(actual, expected libdns.Record, zone string) bool {

	aRR := actual.RR()
	bRR := expected.RR()

	// Always compare name and type
	if !strings.EqualFold(libdns.AbsoluteName(aRR.Name, zone), libdns.AbsoluteName(bRR.Name, zone)) || aRR.Type != bRR.Type {
		return false
	}

	// For different record types, compare specific fields
	switch aRR.Type {
	case "A", "AAAA":
		// For address records, compare IP
		aAddr, aOK := actual.(libdns.Address)
		bAddr, bOK := expected.(libdns.Address)
		if !aOK || !bOK {
			return false
		}
		return aAddr.IP == bAddr.IP

	case "CNAME", "NS":
		// For CNAME/NS records, compare target
		aValue := aRR.Data
		bValue := bRR.Data
		return canonicalizeTarget(aValue) == canonicalizeTarget(bValue)

	case "MX":
		// For MX records, compare preference and target
		aMX, aOK := actual.(libdns.MX)
		bMX, bOK := expected.(libdns.MX)
		if !aOK || !bOK {
			return false
		}
		return aMX.Preference == bMX.Preference &&
			canonicalizeTarget(aMX.Target) == canonicalizeTarget(bMX.Target)

	case "TXT":
		// For TXT records, compare value
		aTXT, aOK := actual.(libdns.TXT)
		bTXT, bOK := expected.(libdns.TXT)
		if !aOK || !bOK {
			return false
		}
		return aTXT.Text == bTXT.Text

	case "SRV":
		// For SRV records, compare priority, weight, port, and target
		aSRV, aOK := actual.(libdns.SRV)
		bSRV, bOK := expected.(libdns.SRV)
		if !aOK || !bOK {
			return false
		}
		return aSRV.Priority == bSRV.Priority &&
			aSRV.Weight == bSRV.Weight &&
			aSRV.Port == bSRV.Port &&
			canonicalizeTarget(aSRV.Target) == canonicalizeTarget(bSRV.Target)

	case "CAA":
		// For CAA records, compare tag and value
		aCAA, aOK := actual.(libdns.CAA)
		bCAA, bOK := expected.(libdns.CAA)
		if !aOK || !bOK {
			return false
		}
		return aCAA.Tag == bCAA.Tag && aCAA.Value == bCAA.Value

	default:
		// For other record types, just compare the Data field
		return aRR.Data == bRR.Data
	}
}

// canonicalizeTarget ensures targets are compared consistently
// Some APIs may return targets with/without trailing dots, normalize them
func canonicalizeTarget(target string) string {
	// Ensure trailing dot for absolute domains
	if !strings.HasSuffix(target, ".") {
		return target + "."
	}
	return target
}
