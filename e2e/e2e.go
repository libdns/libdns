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
//
// # Provider Types
//
// Most libdns providers implement the basic record operations but not ZoneLister.
// This package provides two interfaces and corresponding test functions:
//
// - RecordProvider: For providers that implement basic DNS record operations
// - FullProvider: For providers that also implement ZoneLister
//
// Use the appropriate constructor and test runner:
//
//	// For basic providers (most common)
//	suite := e2e.NewRecordTestSuite(yourProvider, "test-zone.com.")
//	suite.RunRecordTests(t)
//
//	// For providers with ZoneLister support
//	suite := e2e.NewFullTestSuite(yourProvider, "test-zone.com.")
//	suite.RunFullTests(t) // runs ZoneLister test + all record tests
//
// # Custom Record Construction
//
// Since libdns.Record is an interface, different providers may return their own
// implementations that cannot be constructed using the standard libdns types.
// The TestSuite.AppendRecordFunc field allows you to provide a custom function
// to create Record instances for AppendRecords tests:
//
//	suite := e2e.NewRecordTestSuite(yourProvider, "test-zone.com.")
//	suite.AppendRecordFunc = func(rr libdns.RR) libdns.Record {
//		// Return your provider's specific Record implementation
//		return yourProvider.NewRecord(rr)
//	}
//
// For Set and Delete operations, the tests automatically retrieve existing records
// from the provider to ensure compatibility with provider-specific Record implementations.
package e2e

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// RecordProvider represents a basic libdns provider that handles DNS records.
// This is the most common type of provider implementation.
type RecordProvider interface {
	libdns.RecordGetter
	libdns.RecordAppender
	libdns.RecordSetter
	libdns.RecordDeleter
}

// FullProvider represents a complete libdns provider implementation for testing.
// It combines all the libdns interfaces that a provider might implement.
type FullProvider interface {
	RecordProvider
	libdns.ZoneLister
}

// TestSuite contains all the configuration needed to run e2e tests.
type TestSuite struct {
	recordProvider RecordProvider
	fullProvider   FullProvider
	Zone           string
	Timeout        time.Duration
	// AppendRecordFunc is an optional function to create Record instances for AppendRecords tests.
	// if nil, the tests will use the default libdns record types.
	// the function receives an RR and should return a Record implementation.
	AppendRecordFunc func(rr libdns.RR) libdns.Record
}

// NewRecordTestSuite creates a new test suite for a record-only provider.
func NewRecordTestSuite(provider RecordProvider, zone string) *TestSuite {
	return &TestSuite{
		recordProvider:   provider,
		fullProvider:     nil,
		Zone:             zone,
		Timeout:          30 * time.Second,
		AppendRecordFunc: nil,
	}
}

// NewFullTestSuite creates a new test suite for a full provider (with ZoneLister).
func NewFullTestSuite(provider FullProvider, zone string) *TestSuite {
	return &TestSuite{
		recordProvider:   provider,
		fullProvider:     provider,
		Zone:             zone,
		Timeout:          30 * time.Second,
		AppendRecordFunc: nil,
	}
}

// RunRecordTests runs the record-only e2e test suite sequentially.
// This is suitable for providers that implement RecordProvider but not ZoneLister.
func (ts *TestSuite) RunRecordTests(t *testing.T) {
	t.Run("GetRecords", ts.TestGetRecords)
	t.Run("AppendRecords", ts.TestAppendRecords)
	t.Run("SetRecords", ts.TestSetRecords)
	t.Run("DeleteRecords", ts.TestDeleteRecords)
}

// RunFullTests runs the complete e2e test suite sequentially, including ZoneLister tests.
// This is suitable for providers that implement FullProvider.
func (ts *TestSuite) RunFullTests(t *testing.T) {
	t.Run("ListZones", ts.TestListZones)
	ts.RunRecordTests(t)
}

// createRecord creates a Record from an RR, using the AppendRecordFunc if provided,
// or falling back to the generic libdns.RR record type.
func (ts *TestSuite) createRecord(rr libdns.RR) libdns.Record {
	if ts.AppendRecordFunc != nil {
		return ts.AppendRecordFunc(rr)
	}

	return libdns.RR{Name: rr.Name, TTL: rr.TTL, Type: rr.Type, Data: rr.Data}
}

// findRecordsByNameAndType finds all records from the provider that match the given name and type.
// This is used to get provider-specific Record implementations for Set and Delete operations.
func (ts *TestSuite) findRecordsByNameAndType(ctx context.Context, name, recordType string) ([]libdns.Record, error) {
	allRecords, err := ts.recordProvider.GetRecords(ctx, ts.Zone)
	if err != nil {
		return nil, err
	}

	var matches []libdns.Record
	for _, record := range allRecords {
		rr := record.RR()
		if rr.Name == name && rr.Type == recordType {
			matches = append(matches, record)
		}
	}

	return matches, nil
}

// TestListZones tests the ZoneLister interface.
func (ts *TestSuite) TestListZones(t *testing.T) {
	if ts.fullProvider == nil {
		t.Skip("ZoneLister not supported by this provider")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	zones, err := ts.fullProvider.ListZones(ctx)
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

	records, err := ts.recordProvider.GetRecords(ctx, ts.Zone)
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

	t.Log("Creating test records for append operation")
	testRRs := []libdns.RR{
		{
			Name: "test-append",
			TTL:  300 * time.Second,
			Type: "A",
			Data: "192.0.2.1",
		},
		{
			Name: "test-append-txt",
			TTL:  300 * time.Second,
			Type: "TXT",
			Data: "test-append-value",
		},
		{
			Name: "test-append-cname",
			TTL:  300 * time.Second,
			Type: "CNAME",
			Data: "target.example.com.",
		},
	}

	var testRecords []libdns.Record
	for _, rr := range testRRs {
		testRecords = append(testRecords, ts.createRecord(rr))
	}

	t.Logf("Appending %d new records", len(testRecords))
	appendedRecords, err := ts.recordProvider.AppendRecords(ctx, ts.Zone, testRecords)
	if err != nil {
		t.Fatalf("AppendRecords failed: %v", err)
	}

	if len(appendedRecords) != len(testRecords) {
		t.Errorf("Expected %d appended records, got %d", len(testRecords), len(appendedRecords))
	}
	t.Logf("Appended %d records successfully", len(appendedRecords))

	t.Log("Verifying appended records exist in zone")
	ts.verifyRecordsExist(t, ctx, testRecords)

	t.Log("Cleaning up appended records")
	ts.cleanupRecords(t, ctx, appendedRecords)
}

// TestSetRecords tests the RecordSetter interface.
// Tests that SetRecords only affects records with matching (name, type) pairs
// and leaves other records untouched.
func (ts *TestSuite) TestSetRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	t.Log("Creating preserved record that should not be affected by SetRecords")
	preservedRR := libdns.RR{
		Name: "test-set-preserve",
		TTL:  300 * time.Second,
		Type: "TXT",
		Data: "should-not-change",
	}
	preservedRecord := ts.createRecord(preservedRR)

	preservedRecords, err := ts.recordProvider.AppendRecords(ctx, ts.Zone, []libdns.Record{preservedRecord})
	if err != nil {
		t.Fatalf("Failed to create preserved record: %v", err)
	}
	t.Logf("Created preserved record: %s", preservedRecord.RR().Name)

	initialRRs := []libdns.RR{
		{
			Name: "test-set",
			TTL:  300 * time.Second,
			Type: "A",
			Data: "192.0.2.1",
		},
		{
			Name: "test-set",
			TTL:  300 * time.Second,
			Type: "A",
			Data: "192.0.2.2",
		},
		{
			Name: "test-set-cname",
			TTL:  300 * time.Second,
			Type: "CNAME",
			Data: "initial.example.com.",
		},
	}

	var initialRecords []libdns.Record
	for _, rr := range initialRRs {
		initialRecords = append(initialRecords, ts.createRecord(rr))
	}

	t.Logf("Setting initial records: 2 A records for 'test-set' and 1 CNAME")
	setRecords, err := ts.recordProvider.SetRecords(ctx, ts.Zone, initialRecords)
	if err != nil {
		t.Fatalf("SetRecords (initial) failed: %v", err)
	}

	if len(setRecords) != len(initialRecords) {
		t.Errorf("Expected %d set records, got %d", len(initialRecords), len(setRecords))
	}
	t.Logf("Set %d initial records successfully", len(setRecords))

	t.Log("Verifying preserved record still exists")
	ts.verifyRecordsExist(t, ctx, []libdns.Record{preservedRecord})

	updatedRRs := []libdns.RR{
		{
			Name: "test-set",
			TTL:  600 * time.Second,
			Type: "A",
			Data: "192.0.2.3",
		},
		{
			Name: "test-set-cname",
			TTL:  600 * time.Second,
			Type: "CNAME",
			Data: "updated.example.com.",
		},
	}

	var updatedRecords []libdns.Record
	for _, rr := range updatedRRs {
		updatedRecords = append(updatedRecords, ts.createRecord(rr))
	}

	t.Logf("Updating records: replacing 2 A records with 1 A record, updating CNAME")
	setRecords, err = ts.recordProvider.SetRecords(ctx, ts.Zone, updatedRecords)
	if err != nil {
		t.Fatalf("SetRecords (update) failed: %v", err)
	}

	if len(setRecords) != len(updatedRecords) {
		t.Errorf("Expected %d updated records, got %d", len(updatedRecords), len(setRecords))
	}
	t.Logf("Updated %d records successfully", len(setRecords))

	t.Log("Verifying updated records exist")
	ts.verifyRecordsExist(t, ctx, updatedRecords)

	// verify the old records no longer exist (by checking they don't match the updated records)
	t.Log("Verifying old records were replaced (SetRecords atomicity)")
	currentTestSetRecords, err := ts.findRecordsByNameAndType(ctx, "test-set", "A")
	if err != nil {
		t.Fatalf("Failed to find current test-set A records: %v", err)
	}

	for _, current := range currentTestSetRecords {
		currentRR := current.RR()
		if currentRR.Data == initialRRs[0].Data || currentRR.Data == initialRRs[1].Data {
			t.Errorf("Old record data still exists: %s", currentRR.Data)
		}
		if currentRR.Data != updatedRRs[0].Data {
			t.Errorf("Expected updated record data %s, got %s", updatedRRs[0].Data, currentRR.Data)
		}
	}

	t.Log("Verifying preserved record was not affected")
	ts.verifyRecordsExist(t, ctx, []libdns.Record{preservedRecord})

	t.Log("Cleaning up test records")
	ts.cleanupRecords(t, ctx, setRecords)
	ts.cleanupRecords(t, ctx, preservedRecords)
}

// TestDeleteRecords tests the RecordDeleter interface.
func (ts *TestSuite) TestDeleteRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	t.Log("Creating test records for deletion")
	testRRs := []libdns.RR{
		{
			Name: "test-delete",
			TTL:  300 * time.Second,
			Type: "A",
			Data: "192.0.2.1",
		},
		{
			Name: "test-delete-txt",
			TTL:  300 * time.Second,
			Type: "TXT",
			Data: "test-delete-value",
		},
		{
			Name: "test-delete-cname",
			TTL:  300 * time.Second,
			Type: "CNAME",
			Data: "target.example.com.",
		},
	}

	var testRecords []libdns.Record
	for _, rr := range testRRs {
		testRecords = append(testRecords, ts.createRecord(rr))
	}

	t.Logf("Creating %d records to be deleted later", len(testRecords))
	createdRecords, err := ts.recordProvider.AppendRecords(ctx, ts.Zone, testRecords)
	if err != nil {
		t.Fatalf("AppendRecords (for delete test) failed: %v", err)
	}
	t.Logf("Created %d records successfully", len(createdRecords))

	t.Log("Deleting the created records")
	deletedRecords, err := ts.recordProvider.DeleteRecords(ctx, ts.Zone, createdRecords)
	if err != nil {
		t.Fatalf("DeleteRecords failed: %v", err)
	}

	if len(deletedRecords) != len(createdRecords) {
		t.Errorf("Expected %d deleted records, got %d", len(createdRecords), len(deletedRecords))
	}
	t.Logf("Deleted %d records successfully", len(deletedRecords))

	t.Log("Verifying records were actually deleted")
	ts.verifyRecordsNotExist(t, ctx, deletedRecords)
}

// verifyRecordsExist checks that all given records exist in the zone.
func (ts *TestSuite) verifyRecordsExist(t *testing.T, ctx context.Context, expectedRecords []libdns.Record) {
	allRecords, err := ts.recordProvider.GetRecords(ctx, ts.Zone)
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
			ts.logAllRecords(t, allRecords)
		}
	}
}

// verifyRecordsNotExist checks that none of the given records exist in the zone.
func (ts *TestSuite) verifyRecordsNotExist(t *testing.T, ctx context.Context, unexpectedRecords []libdns.Record) {
	allRecords, err := ts.recordProvider.GetRecords(ctx, ts.Zone)
	if err != nil {
		t.Fatalf("GetRecords (verify not exist) failed: %v", err)
	}

	foundAny := false
	for _, unexpected := range unexpectedRecords {
		unexpectedRR := unexpected.RR()

		for _, actual := range allRecords {
			actualRR := actual.RR()
			if ts.recordsMatch(unexpectedRR, actualRR) {
				t.Errorf("Unexpected record found: %s %s %s", actualRR.Name, actualRR.Type, actualRR.Data)
				foundAny = true
			}
		}
	}

	if foundAny {
		ts.logAllRecords(t, allRecords)
	}
}

// recordsMatch compares two RR records for equality
func (ts *TestSuite) recordsMatch(a, b libdns.RR) bool {
	return a.Name == b.Name && a.Type == b.Type && a.Data == b.Data && a.TTL == b.TTL
}

// logAllRecords logs all records in the zone for debugging purposes.
func (ts *TestSuite) logAllRecords(t *testing.T, allRecords []libdns.Record) {
	t.Logf("Debug: Records present in zone:")
	for _, actual := range allRecords {
		actualRR := actual.RR()
		t.Logf("  - %s %s %s %s", actualRR.Name, actualRR.TTL, actualRR.Type, actualRR.Data)
	}
}

// cleanupRecords attempts to clean up test records (best effort).
func (ts *TestSuite) cleanupRecords(t *testing.T, ctx context.Context, records []libdns.Record) {
	if len(records) == 0 {
		return
	}

	_, err := ts.recordProvider.DeleteRecords(ctx, ts.Zone, records)
	if err != nil {
		t.Logf("Warning: cleanup failed: %v", err)
	}
}

// AttemptZoneCleanup deletes records with names starting with "test-" from the zone.
// This method is useful for cleaning up after test runs or preparing for fresh tests.
// Only deletes A, CNAME, and TXT record types that match the test name pattern.
func (ts *TestSuite) AttemptZoneCleanup() error {
	// Record types used by the e2e tests
	testRecordTypes := []string{"A", "CNAME", "TXT"}
	
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	allRecords, err := ts.recordProvider.GetRecords(ctx, ts.Zone)
	if err != nil {
		return err
	}

	var testRecords []libdns.Record
	for _, record := range allRecords {
		rr := record.RR()
		if len(rr.Name) >= 5 && rr.Name[:5] == "test-" && slices.Contains(testRecordTypes, rr.Type) {
			testRecords = append(testRecords, record)
		}
	}

	if len(testRecords) == 0 {
		return nil
	}

	_, err = ts.recordProvider.DeleteRecords(ctx, ts.Zone, testRecords)
	return err
}
