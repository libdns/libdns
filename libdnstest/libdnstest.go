// Package libdnstest provides testing utilities for libdns provider implementations.
//
// These tests create, modify, and delete DNS records with names like "test-append",
// "test-set", "test-delete", and "test-lifecycle" to validate provider behavior.
//
// For real DNS provider implementations, use dedicated test zones since the tests
// will modify DNS records. The example provider uses in-memory storage and is completely safe.
//
// Tests run sequentially (not in parallel) to avoid conflicts when testing real
// DNS providers that interact with external services.
//
// # Usage
//
//	suite := libdnstest.NewTestSuite(yourProvider, "example.com.")
//	suite.RunTests(t)
//
// # Provider Without ZoneLister
//
// If your provider doesn't implement ZoneLister, use WrapNoZoneLister:
//
//	provider := YourProvider{...}
//	wrappedProvider := libdnstest.WrapNoZoneLister(provider)
//	suite := libdnstest.NewTestSuite(wrappedProvider, "example.com.")
//	suite.RunTests(t)
//
// # Custom Record Construction
//
// Since libdns.Record is an interface, different providers may return their own
// implementations that cannot be constructed using the standard libdns types.
// The TestSuite.AppendRecordFunc field allows you to provide a custom function
// to create Record instances for AppendRecords tests:
//
//	suite := libdnstest.NewTestSuite(yourProvider, "example.com.")
//	suite.AppendRecordFunc = func(record libdns.Record) libdns.Record {
//		// Return your provider's specific Record implementation
//		return yourProvider.NewRecord(record.RR())
//	}
//
// For Set and Delete operations, the tests automatically retrieve existing records
// from the provider to ensure compatibility with provider-specific Record implementations.
package libdnstest

import (
	"context"
	"errors"
	"net/netip"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// RecordProvider represents a provider that implements the core record management interfaces,
// but not ZoneLister.
type RecordProvider interface {
	libdns.RecordGetter
	libdns.RecordAppender
	libdns.RecordSetter
	libdns.RecordDeleter
}

// Provider represents a libdns provider implementation for testing.
type Provider interface {
	RecordProvider
	libdns.ZoneLister
}

// ErrNotImplemented is the sentinel error returned when a method is not implemented
// used for skipping ZoneLister tests
var ErrNotImplemented = errors.New("not implemented")

// WrapNoZoneLister wraps a provider that doesn't implement ZoneLister,
// adding a stub implementation that returns "not implemented" error.
// This allows providers without zone listing capability to work with the test suite.
func WrapNoZoneLister(provider RecordProvider) Provider {
	return &noZoneListerWrapper{provider: provider}
}

type noZoneListerWrapper struct {
	provider RecordProvider
}

func (w *noZoneListerWrapper) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	return w.provider.GetRecords(ctx, zone)
}

func (w *noZoneListerWrapper) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return w.provider.AppendRecords(ctx, zone, records)
}

func (w *noZoneListerWrapper) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return w.provider.SetRecords(ctx, zone, records)
}

func (w *noZoneListerWrapper) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return w.provider.DeleteRecords(ctx, zone, records)
}

func (w *noZoneListerWrapper) ListZones(ctx context.Context) ([]libdns.Zone, error) {
	return nil, ErrNotImplemented
}

// TestSuite contains all the configuration needed to run e2e tests.
type TestSuite struct {
	provider Provider
	zone     string
	Timeout  time.Duration
	// AppendRecordFunc is an optional function to create Record instances for AppendRecords tests.
	// if nil, the tests will use the default libdns record types.
	// the function receives a Record and should return a Record implementation.
	AppendRecordFunc func(record libdns.Record) libdns.Record
	// SkipRRTypes is a map of DNS record types to skip during testing.
	// keys should be DNS record type strings like "MX", "SRV", "CAA", "NS", "SVCB", "HTTPS".
	// if a type is present in the map with a true value, tests for that record type will be skipped.
	// example: SkipRRTypes: map[string]bool{"MX": true, "SRV": true}
	// "A", "CNAME", "TXT" record types are essential and could not be skipped
	SkipRRTypes map[string]bool
	// ExpectEmptyZone when true, verifies that the zone only contains default records (SOA, NS)
	// after all tests complete. The test will fail if any test record types (A, AAAA, CNAME, TXT,
	// MX, SRV, CAA, SVCB, HTTPS) remain. This is useful to confirm complete cleanup in test zones.
	ExpectEmptyZone bool
}

// NewTestSuite creates a new test suite for a libdns provider.
func NewTestSuite(provider Provider, zone string) *TestSuite {
	return &TestSuite{
		provider:         provider,
		zone:             zone,
		Timeout:          30 * time.Second,
		AppendRecordFunc: nil,
	}
}

// RunTests does zone cleanup and runs all tests
func (ts *TestSuite) RunTests(t *testing.T) {
	// validate that essential record types are not skipped
	essentialTypes := []string{"TXT", "A", "CNAME"}
	for _, rrType := range essentialTypes {
		if ts.SkipRRTypes[rrType] {
			t.Fatalf("Cannot skip essential record type %s - it is used in the test framework", rrType)
		}
	}

	if err := ts.AttemptZoneCleanup(); err != nil {
		t.Fatalf("Initial cleanup failed: %v", err)
	}

	t.Run("ListZones", ts.TestListZones)
	t.Run("GetRecords", ts.TestGetRecords)
	t.Run("AppendRecords", ts.TestAppendRecords)
	t.Run("SetRecords", ts.TestSetRecords)
	t.Run("DeleteRecords", ts.TestDeleteRecords)

	// If ExpectEmptyZone is true, verify the zone is clean after all tests
	if ts.ExpectEmptyZone {
		ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
		defer cancel()
		ts.verifyZoneClean(t, ctx)
	}
}

// testRecordTypes defines all DNS record types that the test suite creates and manages.
var testRecordTypes = []string{"A", "AAAA", "CNAME", "TXT", "MX", "SRV", "CAA", "NS", "SVCB", "HTTPS"}

// createRecord creates a Record using the AppendRecordFunc if provided,
// or falling back to the original record.
func (ts *TestSuite) createRecord(record libdns.Record) libdns.Record {
	if ts.AppendRecordFunc != nil {
		return ts.AppendRecordFunc(record)
	}

	return record
}

// filterRecords removes records based on skip flags.
func (ts *TestSuite) filterRecords(records []libdns.Record) []libdns.Record {
	if len(ts.SkipRRTypes) == 0 {
		return records
	}

	var filtered []libdns.Record

	for _, record := range records {
		rrType := record.RR().Type

		// special handling for ServiceBinding which can be SVCB or HTTPS
		if _, ok := record.(libdns.ServiceBinding); ok {
			sb := record.(libdns.ServiceBinding)
			if sb.Scheme == "https" && ts.SkipRRTypes["HTTPS"] {
				continue
			} else if sb.Scheme != "https" && ts.SkipRRTypes["SVCB"] {
				continue
			}
		} else if ts.SkipRRTypes[rrType] {
			continue
		}

		filtered = append(filtered, record)
	}

	return filtered
}

// findRecordsByNameAndType finds all records from the provider that match the given name and type.
// This is used to get provider-specific Record implementations for Set and Delete operations.
func (ts *TestSuite) findRecordsByNameAndType(ctx context.Context, name, recordType string) ([]libdns.Record, error) {
	allRecords, err := ts.provider.GetRecords(ctx, ts.zone)
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
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	zones, err := ts.provider.ListZones(ctx)
	if err != nil {
		// skip test if ZoneLister is not implemented
		if errors.Is(err, ErrNotImplemented) {
			t.Skip("ZoneLister not implemented by provider")
		}
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
		if zone.Name == ts.zone {
			testZoneFound = true
		}
	}

	if !testZoneFound {
		t.Errorf("Test zone %s not found in ListZones result", ts.zone)
	}
}

// TestGetRecords tests the RecordGetter interface.
func (ts *TestSuite) TestGetRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	records, err := ts.provider.GetRecords(ctx, ts.zone)
	if err != nil {
		t.Fatalf("GetRecords failed: %v", err)
	}

	t.Logf("Found %d records in zone %s", len(records), ts.zone)
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

	t.Cleanup(func() {
		if err := ts.AttemptZoneCleanup(); err != nil {
			t.Logf("Warning: cleanup after AppendRecords failed: %v", err)
		}
	})

	t.Log("Creating test records for append operation")

	targetRecords := []libdns.Record{
		libdns.Address{
			Name: "test-append-address",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.1"),
		},
		libdns.Address{
			Name: "test-append-address",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.2"),
		},
		libdns.Address{
			Name: "test-append-address-ipv6",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("2001:db8::1"),
		},
		libdns.CAA{
			Name:  "test-append-caa",
			TTL:   300 * time.Second,
			Flags: 0,
			Tag:   "issue",
			Value: "letsencrypt.org",
		},
		libdns.CNAME{
			Name:   "test-append-cname",
			TTL:    300 * time.Second,
			Target: "example.com.",
		},
		libdns.ServiceBinding{
			Name:     "test-append-https",
			TTL:      300 * time.Second,
			Scheme:   "https",
			Priority: 1,
			Target:   "test-append-address." + ts.zone,
			Params: libdns.SvcParams{
				"alpn":     {"h2", "h3"},
				"ipv4hint": {"192.0.2.1", "192.0.2.2"},
				"ipv6hint": {"2001:db8::1"},
				"port":     {"443"},
			},
		},
		libdns.MX{
			Name:       "test-append-mx",
			TTL:        300 * time.Second,
			Preference: 10,
			Target:     "mx.example.com.",
		},
		libdns.NS{
			Name:   "test-append-ns",
			TTL:    300 * time.Second,
			Target: "ns1.example.com.",
		},
		libdns.SRV{
			Name:      "test-append-srv",
			Service:   "exampleservice",
			Transport: "tcp",
			TTL:       300 * time.Second,
			Priority:  10,
			Weight:    20,
			Port:      443,
			Target:    "service.example.com.",
		},
		libdns.ServiceBinding{
			Name:     "test-append-svcb",
			TTL:      300 * time.Second,
			Scheme:   "dns",
			Priority: 1,
			Target:   ".",
			Params: libdns.SvcParams{
				"alpn": {"dot"},
			},
		},
		libdns.TXT{
			Name: "test-append-txt",
			TTL:  300 * time.Second,
			Text: "Hello, world!",
		},
	}

	filteredTargetRecords := ts.filterRecords(targetRecords)

	var testRecords []libdns.Record
	for _, record := range filteredTargetRecords {
		testRecords = append(testRecords, ts.createRecord(record))
	}

	t.Logf("Appending %d new records", len(testRecords))
	appendedRecords, err := ts.provider.AppendRecords(ctx, ts.zone, testRecords)
	if err != nil {
		t.Fatalf("AppendRecords failed: %v", err)
	}

	if len(appendedRecords) != len(testRecords) {
		t.Errorf("Expected %d appended records, got %d", len(testRecords), len(appendedRecords))
	}
	t.Logf("Appended %d records successfully", len(appendedRecords))

	t.Log("Verifying appended records exist in zone")
	ts.verifyRecordsExist(t, ctx, testRecords)
}

// TestSetRecords tests the RecordSetter interface.
// Tests that SetRecords only affects records with matching (name, type) pairs
// and leaves other records untouched.
func (ts *TestSuite) TestSetRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	t.Cleanup(func() {
		if err := ts.AttemptZoneCleanup(); err != nil {
			t.Logf("Warning: cleanup after SetRecords failed: %v", err)
		}
	})

	t.Log("Creating preserved record that should not be affected by SetRecords")
	preservedRR := libdns.RR{
		Name: "test-set-preserve",
		TTL:  300 * time.Second,
		Type: "TXT",
		Data: "should-not-change",
	}
	preservedRecord := ts.createRecord(preservedRR)

	_, err := ts.provider.AppendRecords(ctx, ts.zone, []libdns.Record{preservedRecord})
	if err != nil {
		t.Fatalf("Failed to create preserved record: %v", err)
	}
	t.Logf("Created preserved record: %s", preservedRecord.RR().Name)

	initialTargetRecords := []libdns.Record{
		libdns.Address{
			Name: "test-set-address",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.1"),
		},
		libdns.Address{
			Name: "test-set-address",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.2"),
		},
		libdns.CAA{
			Name:  "test-set-caa",
			TTL:   300 * time.Second,
			Flags: 128,
			Tag:   "issue",
			Value: "initial.example.com",
		},
		libdns.CNAME{
			Name:   "test-set-cname",
			TTL:    300 * time.Second,
			Target: "initial.example.com.",
		},
		libdns.MX{
			Name:       "test-set-mx",
			TTL:        300 * time.Second,
			Preference: 10,
			Target:     "initial-mx.example.com.",
		},
		libdns.TXT{
			Name: "test-set-txt",
			TTL:  300 * time.Second,
			Text: "initial value",
		},
	}

	filteredInitialRecords := ts.filterRecords(initialTargetRecords)

	var initialRecords []libdns.Record
	for _, record := range filteredInitialRecords {
		initialRecords = append(initialRecords, ts.createRecord(record))
	}

	t.Logf("Setting initial records: %d records of various types", len(initialRecords))
	setRecords, err := ts.provider.SetRecords(ctx, ts.zone, initialRecords)
	if err != nil {
		t.Fatalf("SetRecords (initial) failed: %v", err)
	}

	if len(setRecords) != len(initialRecords) {
		t.Errorf("Expected %d set records, got %d", len(initialRecords), len(setRecords))
	}
	t.Logf("Set %d initial records successfully", len(setRecords))

	t.Log("Verifying preserved record still exists")
	ts.verifyRecordsExist(t, ctx, []libdns.Record{preservedRecord})

	updatedTargetRecords := []libdns.Record{
		libdns.Address{
			Name: "test-set-address",
			TTL:  600 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.3"),
		},
		libdns.CAA{
			Name:  "test-set-caa",
			TTL:   600 * time.Second,
			Flags: 0,
			Tag:   "issue",
			Value: "updated.example.com",
		},
		libdns.CNAME{
			Name:   "test-set-cname",
			TTL:    600 * time.Second,
			Target: "updated.example.com.",
		},
		libdns.MX{
			Name:       "test-set-mx",
			TTL:        600 * time.Second,
			Preference: 20,
			Target:     "updated-mx.example.com.",
		},
		libdns.TXT{
			Name: "test-set-txt",
			TTL:  600 * time.Second,
			Text: "updated value",
		},
		libdns.SRV{
			Name:      "test-set-srv",
			Service:   "newservice",
			Transport: "tcp",
			TTL:       600 * time.Second,
			Priority:  5,
			Weight:    10,
			Port:      80,
			Target:    "updated.example.com.",
		},
	}

	filteredUpdatedRecords := ts.filterRecords(updatedTargetRecords)

	var updatedRecords []libdns.Record
	for _, record := range filteredUpdatedRecords {
		updatedRecords = append(updatedRecords, ts.createRecord(record))
	}

	t.Logf("Updating records: modifying existing records and adding new ones")
	setRecords, err = ts.provider.SetRecords(ctx, ts.zone, updatedRecords)
	if err != nil {
		t.Fatalf("SetRecords (update) failed: %v", err)
	}

	if len(setRecords) != len(updatedRecords) {
		t.Errorf("Expected %d updated records, got %d", len(updatedRecords), len(setRecords))
	}
	t.Logf("Updated %d records successfully", len(setRecords))

	t.Log("Verifying updated records exist")
	ts.verifyRecordsExist(t, ctx, updatedRecords)

	t.Log("Verifying old records were replaced")
	currentSetAddressRecords, err := ts.findRecordsByNameAndType(ctx, "test-set-address", "A")
	if err != nil {
		t.Fatalf("Failed to find current test-set-address A records: %v", err)
	}

	if len(currentSetAddressRecords) != 1 {
		t.Errorf("Expected 1 address record after update, got %d", len(currentSetAddressRecords))
	}
	initialIP1 := initialTargetRecords[0].(libdns.Address).IP.String()
	initialIP2 := initialTargetRecords[1].(libdns.Address).IP.String()
	updatedIP := updatedTargetRecords[0].(libdns.Address).IP.String()

	for _, current := range currentSetAddressRecords {
		currentRR := current.RR()
		if currentRR.Data == initialIP1 || currentRR.Data == initialIP2 {
			t.Errorf("Old record data still exists: %s", currentRR.Data)
		}
		if currentRR.Data != updatedIP {
			t.Errorf("Expected updated record data %s, got %s", updatedIP, currentRR.Data)
		}
	}

	t.Log("Verifying preserved record was not affected")
	ts.verifyRecordsExist(t, ctx, []libdns.Record{preservedRecord})
}

// TestDeleteRecords tests the RecordDeleter interface.
func (ts *TestSuite) TestDeleteRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	t.Cleanup(func() {
		if err := ts.AttemptZoneCleanup(); err != nil {
			t.Logf("Warning: cleanup after DeleteRecords failed: %v", err)
		}
	})

	t.Log("Creating test records for deletion")

	targetRecords := []libdns.Record{
		libdns.Address{
			Name: "test-delete-address",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.1"),
		},
		libdns.Address{
			Name: "test-delete-address-ipv6",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("2001:db8::1"),
		},
		libdns.CAA{
			Name:  "test-delete-caa",
			TTL:   300 * time.Second,
			Flags: 0,
			Tag:   "issue",
			Value: "ca.example.com",
		},
		libdns.CNAME{
			Name:   "test-delete-cname",
			TTL:    300 * time.Second,
			Target: "test-delete-target." + ts.zone,
		},
		libdns.SRV{
			Service:   "service",
			Transport: "tcp",
			Name:      "test-delete-srv",
			TTL:       300 * time.Second,
			Priority:  10,
			Weight:    20,
			Port:      80,
			Target:    "test-delete-srv-target." + ts.zone,
		},
		libdns.MX{
			Name:       "test-delete-mx",
			TTL:        300 * time.Second,
			Preference: 10,
			Target:     "test-delete-mx-target." + ts.zone,
		},
		libdns.NS{
			Name:   "test-delete-ns",
			TTL:    300 * time.Second,
			Target: "test-delete-ns-target." + ts.zone,
		},
		libdns.TXT{
			Name: "test-delete-txt",
			TTL:  300 * time.Second,
			Text: "test-delete-value",
		},
		libdns.ServiceBinding{
			Scheme:        "https",
			URLSchemePort: 443,
			Name:          "test-delete-https",
			TTL:           300 * time.Second,
			Priority:      1,
			Target:        "test-delete-https-target." + ts.zone,
			Params:        libdns.SvcParams{"alpn": {"h2", "h3"}},
		},
		libdns.ServiceBinding{
			Scheme:   "service",
			Name:     "test-delete-svcb",
			TTL:      300 * time.Second,
			Priority: 1,
			Target:   "test-delete-svcb-target." + ts.zone,
			Params:   libdns.SvcParams{"port": {"443"}},
		},
	}

	filteredTargetRecords := ts.filterRecords(targetRecords)

	var testRecords []libdns.Record
	for _, record := range filteredTargetRecords {
		testRecords = append(testRecords, ts.createRecord(record))
	}

	t.Logf("Creating %d records to be deleted later", len(testRecords))
	createdRecords, err := ts.provider.AppendRecords(ctx, ts.zone, testRecords)
	if err != nil {
		t.Fatalf("AppendRecords (for delete test) failed: %v", err)
	}
	t.Logf("Created %d records successfully", len(createdRecords))

	t.Log("Deleting the created records")
	deletedRecords, err := ts.provider.DeleteRecords(ctx, ts.zone, createdRecords)
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

// recordsMatch compares two RR structs by parsing and normalizing them.
// This ensures consistent comparison by round-tripping through the libdns parsing logic.
func (ts *TestSuite) recordsMatch(t *testing.T, expected, actual libdns.RR) bool {
	expectedRecord, err := expected.Parse()
	if err != nil {
		t.Fatalf("Failed to parse expected record %s %s %s: %v", expected.Name, expected.Type, expected.Data, err)
	}

	actualRecord, err := actual.Parse()
	if err != nil {
		t.Fatalf("Failed to parse actual record %s %s %s: %v", actual.Name, actual.Type, actual.Data, err)
	}

	normalizedExpected := expectedRecord.RR()
	normalizedActual := actualRecord.RR()

	return normalizedExpected == normalizedActual
}

// verifyRecordsExist checks that all given records exist in the zone.
func (ts *TestSuite) verifyRecordsExist(t *testing.T, ctx context.Context, expectedRecords []libdns.Record) {
	allRecords, err := ts.provider.GetRecords(ctx, ts.zone)
	if err != nil {
		t.Fatalf("GetRecords (verify exist) failed: %v", err)
	}

	for _, expected := range expectedRecords {
		found := false
		expectedRR := expected.RR()

		for _, actual := range allRecords {
			actualRR := actual.RR()
			if ts.recordsMatch(t, expectedRR, actualRR) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected record not found: %+v", expectedRR)
			ts.logAllRecords(t, allRecords)
		}
	}
}

// verifyRecordsNotExist checks that none of the given records exist in the zone.
func (ts *TestSuite) verifyRecordsNotExist(t *testing.T, ctx context.Context, unexpectedRecords []libdns.Record) {
	allRecords, err := ts.provider.GetRecords(ctx, ts.zone)
	if err != nil {
		t.Fatalf("GetRecords (verify not exist) failed: %v", err)
	}

	for _, unexpected := range unexpectedRecords {
		unexpectedRR := unexpected.RR()

		for _, actual := range allRecords {
			actualRR := actual.RR()
			if ts.recordsMatch(t, unexpectedRR, actualRR) {
				t.Errorf("Unexpected record found: %s %s %s", actualRR.Name, actualRR.Type, actualRR.Data)
			}
		}
	}
}

// verifyZoneClean checks that the zone only contains default/expected records.
// By default, zones typically only contain SOA and NS records.
// This verifies that no test record types exist in the zone (NS records are allowed as they're zone defaults).
func (ts *TestSuite) verifyZoneClean(t *testing.T, ctx context.Context) {
	allRecords, err := ts.provider.GetRecords(ctx, ts.zone)
	if err != nil {
		t.Fatalf("GetRecords (verify zone clean) failed: %v", err)
	}

	var unexpectedRecords []libdns.RR
	for _, record := range allRecords {
		rr := record.RR()
		// check if this is one of the record types we use in tests, besides NS
		if rr.Type != "NS" && slices.Contains(testRecordTypes, rr.Type) {
			unexpectedRecords = append(unexpectedRecords, rr)
		}
	}

	if len(unexpectedRecords) > 0 {
		t.Errorf("Found unexpected records in zone that should be clean:")
		for _, rr := range unexpectedRecords {
			t.Errorf("  - %s %s %s %s", rr.Name, rr.TTL, rr.Type, rr.Data)
		}
	}
}

// logAllRecords logs all records in the zone for debugging purposes.
func (ts *TestSuite) logAllRecords(t *testing.T, allRecords []libdns.Record) {
	t.Logf("Debug: Records present in zone:")
	for _, actual := range allRecords {
		actualRR := actual.RR()
		t.Logf("  - %s %s %s %s", actualRR.Name, actualRR.TTL, actualRR.Type, actualRR.Data)
	}
}

// AttemptZoneCleanup deletes records with names starting with "test-" from the zone.
// This method is useful for cleaning up after test runs or preparing for fresh tests.
// Deletes all record types that match the test name pattern.
func (ts *TestSuite) AttemptZoneCleanup() error {
	// filter out skipped types
	var activeRecordTypes []string
	for _, rrType := range testRecordTypes {
		if !ts.SkipRRTypes[rrType] {
			activeRecordTypes = append(activeRecordTypes, rrType)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), ts.Timeout)
	defer cancel()

	allRecords, err := ts.provider.GetRecords(ctx, ts.zone)
	if err != nil {
		return err
	}

	var testRecords []libdns.Record
	for _, record := range allRecords {
		rr := record.RR()
		// check if this is a test record
		// for SRV/SVCB: strip up to two _xxx. prefixes then check for test-
		nameToCheck := rr.Name
		if rr.Type == "SRV" || rr.Type == "SVCB" {
			// Remove up to two "_xxx." prefixes
			nameToCheck = regexp.MustCompile(`^(_[^.]+\.){0,2}`).ReplaceAllString(nameToCheck, "")
		}

		if strings.HasPrefix(nameToCheck, "test-") && slices.Contains(activeRecordTypes, rr.Type) {
			testRecords = append(testRecords, record)
		}
	}

	if len(testRecords) == 0 {
		return nil
	}

	_, err = ts.provider.DeleteRecords(ctx, ts.zone, testRecords)
	return err
}
