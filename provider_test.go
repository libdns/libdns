// Integration tests for the netcup provider

package netcup

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/libdns/libdns"
)

var (
	customerNumber = ""
	apiKey         = ""
	apiPassword    = ""
	zone           = ""
	testRecords    = []libdns.Record{
		{
			Type:  "TXT",
			Name:  "test",
			Value: "testval1",
		},
	}
)

func TestMain(m *testing.M) {
	fmt.Println("Loading environment variables to set up provider")
	customerNumber = os.Getenv("LIBDNS_NETCUP_CUSTOMER_NUMBER")
	apiKey = os.Getenv("LIBDNS_NETCUP_API_KEY")
	apiPassword = os.Getenv("LIBDNS_NETCUP_API_PASSWORD")
	zone = os.Getenv("LIBDNS_NETCUP_ZONE")

	os.Exit(m.Run())
}

func setupTestRecords(t *testing.T, p *Provider) []libdns.Record {
	fmt.Println("Appending test records")
	records, err := p.AppendRecords(context.TODO(), zone, testRecords)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
		return nil
	}

	return records
}

func cleanupRecords(t *testing.T, p *Provider, records []libdns.Record) {
	fmt.Println("Cleaning up test records")
	if _, err := p.DeleteRecords(context.TODO(), zone, records); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
}

func TestProvider_GetRecords(t *testing.T) {
	fmt.Println("Test GetRecords")

	p := &Provider{
		CustomerNumber: customerNumber,
		APIKey:         apiKey,
		APIPassword:    apiPassword,
	}

	setupRecords := setupTestRecords(t, p)
	defer cleanupRecords(t, p, setupRecords)

	records, err := p.GetRecords(context.TODO(), zone)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) < len(setupRecords) {
		t.Fatalf("Number of records found should have been at least %v", len(setupRecords))
	}

	// this is actually more a test of AppendRecords
	for _, setupRecord := range setupRecords {
		var foundRecord *libdns.Record
		for _, record := range records {
			if record.ID == setupRecord.ID {
				foundRecord = &setupRecord
			}
		}

		if foundRecord == nil {
			t.Fatalf("Record with ID %v not found", setupRecord.ID)
		}
	}
}

func TestProvider_SetRecords(t *testing.T) {
	fmt.Println("Test SetRecords")

	p := &Provider{
		CustomerNumber: customerNumber,
		APIKey:         apiKey,
		APIPassword:    apiPassword,
	}

	setupRecords := setupTestRecords(t, p)
	defer cleanupRecords(t, p, setupRecords)

	var updateRecords []libdns.Record
	// test, if records without IDs update the correct records
	for _, record := range testRecords {
		updateRecords = append(updateRecords, libdns.Record{
			Type:  record.Type,
			Name:  record.Name,
			Value: record.Value + "edit",
		})
	}
	records, err := p.SetRecords(context.TODO(), zone, updateRecords)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) < len(setupRecords) {
		t.Fatalf("Number of records set should have been at least %v", len(setupRecords))
	}

	for _, setupRecord := range setupRecords {
		var foundRecord *libdns.Record
		for _, record := range records {
			if record.ID == setupRecord.ID {
				foundRecord = &setupRecord
			}
		}

		if foundRecord == nil {
			t.Fatalf("Record with ID %v not found", setupRecord.ID)
		}
	}
}
