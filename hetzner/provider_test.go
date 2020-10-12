package hetzner_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/libdns/hetzner"
	"github.com/libdns/libdns"
)

var (
	envToken    = ""
	envZone     = ""
	testRecords = []libdns.Record{
		{
			Type:  "TXT",
			Name:  "test1",
			Value: "test1",
			TTL:   time.Duration(120 * time.Second),
		}, {
			Type:  "TXT",
			Name:  "test2",
			Value: "test2",
			TTL:   time.Duration(120 * time.Second),
		}, {
			Type:  "TXT",
			Name:  "test3",
			Value: "test3",
			TTL:   time.Duration(120 * time.Second),
		},
	}
)

type testRecordsCleanup = func()

func setupTestRecords(t *testing.T, p *hetzner.Provider) ([]libdns.Record, testRecordsCleanup) {
	records, err := p.AppendRecords(context.TODO(), envZone, testRecords)
	if err != nil {
		t.Fatal(err)
		return nil, func() {}
	}

	return records, func() {
		cleanupRecords(t, p, records)
	}
}

func cleanupRecords(t *testing.T, p *hetzner.Provider, r []libdns.Record) {
	_, err := p.DeleteRecords(context.TODO(), envZone, r)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}

func TestMain(m *testing.M) {
	envToken = os.Getenv("LIBDNS_HETZNER_TEST_TOKEN")
	envZone = os.Getenv("LIBDNS_HETZNER_TEST_ZONE")

	if len(envToken) == 0 || len(envZone) == 0 {
		fmt.Println(`Please notice that this test runs agains the public Hetzner DNS Api, so you sould
never run the test with a zone, used in production.
To run this test, you have to specify 'LIBDNS_HETZNER_TEST_TOKEN' and 'LIBDNS_HETZNER_TEST_ZONE'.
Example: "LIBDNS_HETZNER_TEST_TOKEN="123" LIBDNS_HETZNER_TEST_ZONE="my-domain.com" go test ./... -v`)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func Test_AppendRecords(t *testing.T) {
	p := &hetzner.Provider{
		AuthAPIToken: envToken,
	}

	records, err := p.AppendRecords(context.TODO(), envZone, testRecords)
	if err != nil {
		t.Fatal(err)
	}

	if len(testRecords) != len(records) {
		t.Fatalf("len(testRecords) != len(records) => %d != %d", len(testRecords), len(records))
	}

	for i := 0; i < len(records); i++ {
		if len(records[i].ID) == 0 {
			t.Fatalf("len(records[%d].ID) == 0", i)
		}

		if records[i].Type != testRecords[i].Type {
			t.Fatalf("records[%d].Type != testRecords[%d].Type => %s != %s", i, i, records[i].Type, testRecords[i].Type)
		}
		if records[i].Name != testRecords[i].Name {
			t.Fatalf("records[%d].Name != testRecords[%d].Name => %s != %s", i, i, records[i].Name, testRecords[i].Name)
		}
		if records[i].Value != testRecords[i].Value {
			t.Fatalf("records[%d].Value != testRecords[%d].Value => %s != %s", i, i, records[i].Value, testRecords[i].Value)
		}
		if records[i].TTL != testRecords[i].TTL {
			t.Fatalf("records[%d].TTL != testRecords[%d].TTL => %v != %v", i, i, records[i].TTL, testRecords[i].TTL)
		}
	}

	// cleanup
	_, err = p.DeleteRecords(context.TODO(), envZone, records)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_DeleteRecords(t *testing.T) {
	p := &hetzner.Provider{
		AuthAPIToken: envToken,
	}

	testRecords, cleanupFunc := setupTestRecords(t, p)
	defer cleanupFunc()

	records, err := p.GetRecords(context.TODO(), envZone)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) < len(testRecords) {
		t.Fatalf("len(records) < len(testRecords) => %d < %d", len(records), len(testRecords))
	}

	for _, testRecord := range testRecords {
		var foundRecord *libdns.Record
		for _, record := range records {
			if testRecord.ID == record.ID {
				foundRecord = &testRecord
			}
		}

		if foundRecord == nil {
			t.Fatalf("Record not found => %s", testRecord.ID)
		}
	}
}

func Test_GetRecords(t *testing.T) {
	p := &hetzner.Provider{
		AuthAPIToken: envToken,
	}

	testRecords, cleanupFunc := setupTestRecords(t, p)
	defer cleanupFunc()

	records, err := p.GetRecords(context.TODO(), envZone)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) < len(testRecords) {
		t.Fatalf("len(records) < len(testRecords) => %d < %d", len(records), len(testRecords))
	}

	for _, testRecord := range testRecords {
		var foundRecord *libdns.Record
		for _, record := range records {
			if testRecord.ID == record.ID {
				foundRecord = &testRecord
			}
		}

		if foundRecord == nil {
			t.Fatalf("Record not found => %s", testRecord.ID)
		}
	}
}

func Test_SetRecords(t *testing.T) {
	p := &hetzner.Provider{
		AuthAPIToken: envToken,
	}

	existingRecords, _ := setupTestRecords(t, p)
	newTestRecords := []libdns.Record{
		{
			Type:  "TXT",
			Name:  "new_test1",
			Value: "new_test1",
			TTL:   time.Duration(120 * time.Second),
		},
		{
			Type:  "TXT",
			Name:  "new_test2",
			Value: "new_test2",
			TTL:   time.Duration(120 * time.Second),
		},
	}

	allRecords := append(existingRecords, newTestRecords...)
	allRecords[0].Value = "new_value"

	records, err := p.SetRecords(context.TODO(), envZone, allRecords)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupRecords(t, p, records)

	if len(records) != len(allRecords) {
		t.Fatalf("len(records) != len(allRecords) => %d != %d", len(records), len(allRecords))
	}

	if records[0].Value != "new_value" {
		t.Fatalf(`records[0].Value != "new_value" => %s != "new_value"`, records[0].Value)
	}
}
