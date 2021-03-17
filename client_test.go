package metaname

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/libdns/libdns"
)

var (
	p    Provider
	ctx  context.Context = context.Background()
	zone string
)

// These tests expect the credentials to be in the environment variables api_key and account_reference (same as the
// official API implementations). It also expects the full name of the zone to be in the test_zone variable. If
// everything is working, all records created by the tests are also removed by them, but if it isn't it may be
// necessary to clear records from the control panel before running the tests again (notably CNAMEs cannot be
// recreated).
//
// The Metaname API occasionally returns unexpected failures on good calls, which makes the test suite slightly flaky,
// but it should succeed the overwhelming majority of the time. The test suite is hard-coded to use the test API
// endpoint.
func init() {
	p = Provider{
		APIKey:           os.Getenv("api_key"),
		AccountReference: os.Getenv("account_reference"),
		Endpoint:         "https://test.metaname.net/api/1.1",
	}
	zone = os.Getenv("test_zone")
}

func TestGetRecords(t *testing.T) {
	// Confirm no errors from retrieving records - actual contents
	// used in other tests, where some of the records are known already.
	_, err := p.GetRecords(ctx, zone)
	if err != nil {
		t.Fatal(err)
	}
}

// Helper function to confirm existence of record.
func expectRecord(t *testing.T, name string, rtype string, value string) {
	records, err := p.GetRecords(ctx, zone)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, rec := range records {
		if rec.Name == name && rec.Type == rtype && rec.Value == value {
			found = true
		}
	}
	if !found {
		t.Fatal("expected to find record " + name + " " + rtype + " " + value)
	}
}

// Helper function to confirm non-existence of record.
func expectNoSuchRecord(t *testing.T, name string, rtype string, value string) {
	records, err := p.GetRecords(ctx, zone)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, rec := range records {
		if rec.Name == name && rec.Type == rtype && rec.Value == value {
			found = true
		}
	}
	if found {
		t.Fatal("expected to find no record like " + name + " " + rtype + " " + value)
	}
}

func TestAppendRecords(t *testing.T) {
	// Add a single record
	added, err := p.AppendRecords(ctx, zone, []libdns.Record{
		{Name: "provider-test-1", TTL: 3600, Type: "A", Value: "127.0.0.1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 1 {
		t.Fatal(fmt.Sprintf("expected to add 1 record; added %d", len(added)))
	}
	expectRecord(t, "provider-test-1", "A", "127.0.0.1")
	// Add two records at once
	added, err = p.AppendRecords(ctx, zone, []libdns.Record{
		{Name: "provider-test-2", TTL: 300, Type: "CNAME", Value: "provider-test-1"},
		{Name: "provider-test-3", TTL: 86400, Type: "TXT", Value: "initial stored txt value"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 2 {
		t.Fatal(fmt.Sprintf("expected to add 2 records; added %d", len(added)))
	}
	expectRecord(t, "provider-test-2", "CNAME", "provider-test-1")
	expectRecord(t, "provider-test-3", "TXT", "initial stored txt value")
}

func TestSetRecords(t *testing.T) {
	// Add a single record to modify
	added, err := p.AppendRecords(ctx, zone, []libdns.Record{
		{Name: "provider-test-4", TTL: 3600, Type: "A", Value: "127.0.0.1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := added[0].ID
	// Update the previous record by ID to hold a new value,
	// and simultaneously add a new TXT record.
	added, err = p.SetRecords(ctx, zone, []libdns.Record{
		{Name: "provider-test-4b", Value: "0.0.0.0", ID: ref},
		{Name: "provider-test-5", Type: "TXT", Value: "abcd", TTL: 600},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 2 {
		t.Fatal(fmt.Sprintf("expected to update/add 2 records; updated/added %d", len(added)))
	}
	expectRecord(t, "provider-test-4b", "A", "0.0.0.0")
	expectRecord(t, "provider-test-5", "TXT", "abcd")
}

func TestDeleteRecords(t *testing.T) {
	// Add, then delete, a single record by guesswork
	p.AppendRecords(ctx, zone, []libdns.Record{
		{Name: "provider-test-6", TTL: 7200, Type: "CNAME", Value: "google.com."},
	})
	deleted, err := p.DeleteRecords(ctx, zone, []libdns.Record{
		{
			Name:  "provider-test-6",
			Type:  "CNAME",
			Value: "google.com.",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 1 {
		t.Fatal(fmt.Sprintf("expected to delete 1 record; deleted %d", len(deleted)))
	}
	expectNoSuchRecord(t, "provider-test-6", "CNAME", "google.com.")
	// Delete all provider-test-* records at once, by reference
	records, _ := p.GetRecords(ctx, zone)
	var todelete []libdns.Record
	for _, rec := range records {
		if strings.HasPrefix(rec.Name, "provider-test-") {
			todelete = append(todelete, rec)
		}
	}
	deleted, err = p.DeleteRecords(ctx, zone, todelete)
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != len(todelete) {
		t.Fatal(fmt.Sprintf("expected to have deleted %d records but deleted %d", len(todelete), len(deleted)))
	}
	// Confirm that no test records remain
	records, _ = p.GetRecords(ctx, zone)
	for _, rec := range records {
		if strings.HasPrefix(rec.Name, "provider-test-") {
			t.Fatal(fmt.Sprintf("record %s %s %s should have been deleted already", rec.Name, rec.Type, rec.Value))
		}
	}
}

func TestErrors(t *testing.T) {
	// Check that various error cases from the API don't crash and are relayed.
	_, err := p.GetRecords(ctx, "nosuch-"+zone+"-notTLD")
	if err == nil {
		t.Fatal("expected error from bad zone")
	}
	_, err = p.AppendRecords(ctx, zone, []libdns.Record{
		{
			Name: "provider-test-7",
			// No type, value, TTL is an error
		},
	})
	if err == nil {
		t.Fatal("expected error from append missing record details")
	}
	_, err = p.DeleteRecords(ctx, zone, []libdns.Record{
		{
			Name:  "provider-test-8",
			Value: "@",
		},
	})
	if err == nil {
		t.Fatal("expected error from delete missing record details")
	}
}
