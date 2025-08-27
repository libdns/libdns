// Package e2e provides end-to-end testing utilities for libdns provider implementations.
//
// These tests create, modify, and delete DNS records with names like "test-append",
// "test-set", "test-delete", and "test-lifecycle" to validate provider behavior.
//
// For real DNS provider implementations, use dedicated test zones since the tests
// will modify DNS records. The dummy provider uses in-memory storage and is completely safe.
//
// Tests run sequentially (not in parallel) to avoid conflicts when testing real
// DNS providers that interact with external services.
package e2e

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// Provider represents a complete libdns provider implementation for testing.
// It combines all the libdns interfaces that a provider might implement.
type Provider interface {
	libdns.RecordGetter
	libdns.RecordAppender
	libdns.RecordSetter
	libdns.RecordDeleter
	libdns.ZoneLister
}

// TestSuite contains all the configuration needed to run e2e tests.
type TestSuite struct {
	Provider	Provider
	Zone		string
	Timeout		time.Duration
}

// NewTestSuite creates a new test suite with the given provider and zone.
func NewTestSuite(provider Provider, zone string) *TestSuite {
	return &TestSuite{
		Provider:	provider,
		Zone:		zone,
		Timeout:	30 * time.Second,
	}
}

// RunAllTests runs the complete e2e test suite sequentially.
// Tests are run sequentially (not in parallel) because many DNS providers
// interact with external services that cannot safely handle concurrent
// modifications to the same zone.
func (ts *TestSuite) RunAllTests(t *testing.T) {
	// run tests sequentially to avoid conflicts with external DNS services
	t.Run("ListZones", ts.TestListZones)
	t.Run("GetRecords", ts.TestGetRecords)
	t.Run("AppendRecords", ts.TestAppendRecords)
	t.Run("SetRecords", ts.TestSetRecords)
	t.Run("DeleteRecords", ts.TestDeleteRecords)
	t.Run("RecordLifecycle", ts.TestRecordLifecycle)
}

// TestListZones tests the ZoneLister interface.
func (ts *TestSuite) TestListZones(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	zones, err := ts.Provider.ListZones(ctx)
	if err != nil {
		t.Fatalf("ListZones failed: %v", err)
	}

	t.Logf("Found %d zones", len(zones))

	// check that the test zone is in the list
	testZoneFound := false
	for _, zone := range zones {
		if zone.Name == "" {
			t.Error("Zone name should not be empty")
		}
		t.Logf("Zone: %s", zone.Name)
		if zone.Name == ts.Zone {
			testZoneFound = true
		}
	}

	if !testZoneFound {
		t.Errorf("Test zone %s not found in ListZones result", ts.Zone)
	}
}

// TestGetRecords tests the RecordGetter interface.
func (ts *TestSuite) TestGetRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	records, err := ts.Provider.GetRecords(ctx, ts.Zone)
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}

	t.Logf("Found %d records in zone %s", len(records), ts.Zone)
	for _, record := range records {
		rr := record.RR()
		if rr.Name == "" {
			t.Error("Record name should not be empty")
		}
		if rr.Type == "" {
			t.Error("Record type should not be empty")
		}
		t.Logf("Record: %s %s %s %s", rr.Name, rr.TTL, rr.Type, rr.Data)
	}
}

// TestAppendRecords tests the RecordAppender interface.
func (ts *TestSuite) TestAppendRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	testRecords := []libdns.Record{
		libdns.Address{
			Name:	"test-append",
			TTL:	300 * time.Second,
			IP:	netip.MustParseAddr("192.0.2.1"),
		},
		libdns.TXT{
			Name:	"test-append-txt",
			TTL:	300 * time.Second,
			Text:	"test-append-value",
		},
		libdns.CNAME{
			Name:	"test-append-cname",
			TTL:	300 * time.Second,
			Target:	"target.example.com.",
		},
	}

	// append records
	appendedRecords, err := ts.Provider.AppendRecords(ctx, ts.Zone, testRecords)
	if err != nil {
		t.Fatalf("AppendRecords failed: %v", err)
	}

	if len(appendedRecords) != len(testRecords) {
		t.Errorf("Expected %d appended records, got %d", len(testRecords), len(appendedRecords))
	}

	ts.verifyRecordsExist(t, ctx, testRecords)

	ts.cleanupRecords(t, ctx, appendedRecords)
}

// TestSetRecords tests the RecordSetter interface.
// Tests that SetRecords only affects records with matching (name, type) pairs
// and leaves other records untouched.
func (ts *TestSuite) TestSetRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	// create a TXT record that should NOT be affected by SetRecords operations
	preservedRecord := libdns.TXT{
		Name:	"test-set-preserve",
		TTL:	300 * time.Second,
		Text:	"should-not-change",
	}

	preservedRecords, err := ts.Provider.AppendRecords(ctx, ts.Zone, []libdns.Record{preservedRecord})
	if err != nil {
		t.Fatalf("Failed to create preserved record: %v", err)
	}

	initialRecords := []libdns.Record{
		libdns.Address{
			Name:	"test-set",
			TTL:	300 * time.Second,
			IP:	netip.MustParseAddr("192.0.2.1"),
		},
		libdns.Address{
			Name:	"test-set",
			TTL:	300 * time.Second,
			IP:	netip.MustParseAddr("192.0.2.2"),
		},
		libdns.CNAME{
			Name:	"test-set-cname",
			TTL:	300 * time.Second,
			Target:	"initial.example.com.",
		},
	}

	setRecords, err := ts.Provider.SetRecords(ctx, ts.Zone, initialRecords)
	if err != nil {
		t.Fatalf("SetRecords (initial) failed: %v", err)
	}

	if len(setRecords) != len(initialRecords) {
		t.Errorf("Expected %d set records, got %d", len(initialRecords), len(setRecords))
	}

	ts.verifyRecordsExist(t, ctx, []libdns.Record{preservedRecord})

	updatedRecords := []libdns.Record{
		libdns.Address{
			Name:	"test-set",
			TTL:	600 * time.Second,
			IP:	netip.MustParseAddr("192.0.2.3"),
		},
		libdns.CNAME{
			Name:	"test-set-cname",
			TTL:	600 * time.Second,
			Target:	"updated.example.com.",
		},
	}

	setRecords, err = ts.Provider.SetRecords(ctx, ts.Zone, updatedRecords)
	if err != nil {
		t.Fatalf("SetRecords (update) failed: %v", err)
	}

	if len(setRecords) != len(updatedRecords) {
		t.Errorf("Expected %d updated records, got %d", len(updatedRecords), len(setRecords))
	}

	ts.verifyRecordsExist(t, ctx, updatedRecords)

	ts.verifyRecordsNotExist(t, ctx, initialRecords)

	ts.verifyRecordsExist(t, ctx, []libdns.Record{preservedRecord})

	ts.cleanupRecords(t, ctx, setRecords)
	ts.cleanupRecords(t, ctx, preservedRecords)
}

// TestDeleteRecords tests the RecordDeleter interface.
func (ts *TestSuite) TestDeleteRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	testRecords := []libdns.Record{
		libdns.Address{
			Name:	"test-delete",
			TTL:	300 * time.Second,
			IP:	netip.MustParseAddr("192.0.2.1"),
		},
		libdns.TXT{
			Name:	"test-delete-txt",
			TTL:	300 * time.Second,
			Text:	"test-delete-value",
		},
		libdns.CNAME{
			Name:	"test-delete-cname",
			TTL:	300 * time.Second,
			Target:	"target.example.com.",
		},
	}

	// create records
	createdRecords, err := ts.Provider.AppendRecords(ctx, ts.Zone, testRecords)
	if err != nil {
		t.Fatalf("AppendRecords (for delete test) failed: %v", err)
	}

	// delete records
	deletedRecords, err := ts.Provider.DeleteRecords(ctx, ts.Zone, createdRecords)
	if err != nil {
		t.Fatalf("DeleteRecords failed: %v", err)
	}

	if len(deletedRecords) != len(createdRecords) {
		t.Errorf("Expected %d deleted records, got %d", len(createdRecords), len(deletedRecords))
	}

	// verify records were deleted
	ts.verifyRecordsNotExist(t, ctx, deletedRecords)
}

// TestRecordLifecycle tests a complete record lifecycle: create, update, delete.
func (ts *TestSuite) TestRecordLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	recordName := "test-lifecycle"

	// step 1: Create record
	createRecord := libdns.Address{
		Name:	recordName,
		TTL:	300 * time.Second,
		IP:	netip.MustParseAddr("192.0.2.10"),
	}

	createdRecords, err := ts.Provider.AppendRecords(ctx, ts.Zone, []libdns.Record{createRecord})
	if err != nil {
		t.Fatalf("Lifecycle create failed: %v", err)
	}

	t.Logf("Created record: %s", createdRecords[0].RR().Data)

	// step 2: Update record using SetRecords
	updateRecord := libdns.Address{
		Name:	recordName,
		TTL:	600 * time.Second,
		IP:	netip.MustParseAddr("192.0.2.20"),
	}

	updatedRecords, err := ts.Provider.SetRecords(ctx, ts.Zone, []libdns.Record{updateRecord})
	if err != nil {
		t.Fatalf("Lifecycle update failed: %v", err)
	}

	t.Logf("Updated record: %s", updatedRecords[0].RR().Data)

	// verify update
	ts.verifyRecordsExist(t, ctx, []libdns.Record{updateRecord})

	// step 3: Delete record
	deletedRecords, err := ts.Provider.DeleteRecords(ctx, ts.Zone, updatedRecords)
	if err != nil {
		t.Fatalf("Lifecycle delete failed: %v", err)
	}

	t.Logf("Deleted %d records", len(deletedRecords))

	// verify deletion
	ts.verifyRecordsNotExist(t, ctx, deletedRecords)
}

// verifyRecordsExist checks that all given records exist in the zone.
func (ts *TestSuite) verifyRecordsExist(t *testing.T, ctx context.Context, expectedRecords []libdns.Record) {
	allRecords, err := ts.Provider.GetRecords(ctx, ts.Zone)
	if err != nil {
		t.Fatalf("GetRecords (verify exist) failed: %v", err)
	}

	for _, expected := range expectedRecords {
		found := false
		expectedRR := expected.RR()

		for _, actual := range allRecords {
			actualRR := actual.RR()
			if ts.recordsMatch(expectedRR, actualRR) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected record not found: %s %s %s", expectedRR.Name, expectedRR.Type, expectedRR.Data)
		}
	}
}

// verifyRecordsNotExist checks that none of the given records exist in the zone.
func (ts *TestSuite) verifyRecordsNotExist(t *testing.T, ctx context.Context, unexpectedRecords []libdns.Record) {
	allRecords, err := ts.Provider.GetRecords(ctx, ts.Zone)
	if err != nil {
		t.Fatalf("GetRecords (verify not exist) failed: %v", err)
	}

	for _, unexpected := range unexpectedRecords {
		unexpectedRR := unexpected.RR()

		for _, actual := range allRecords {
			actualRR := actual.RR()
			if ts.recordsMatch(unexpectedRR, actualRR) {
				t.Errorf("Unexpected record found: %s %s %s", actualRR.Name, actualRR.Type, actualRR.Data)
			}
		}
	}
}

// recordsMatch compares two RR records for equality (ignoring TTL for flexibility).
func (ts *TestSuite) recordsMatch(a, b libdns.RR) bool {
	return a.Name == b.Name && a.Type == b.Type && a.Data == b.Data
}

// cleanupRecords attempts to clean up test records (best effort).
func (ts *TestSuite) cleanupRecords(t *testing.T, ctx context.Context, records []libdns.Record) {
	if len(records) == 0 {
		return
	}

	_, err := ts.Provider.DeleteRecords(ctx, ts.Zone, records)
	if err != nil {
		t.Logf("Warning: cleanup failed: %v", err)
	}
}
