package katapult

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

var (
	envToken = ""
	envZone  = ""
)

func provisionRecords(t *testing.T, provider *Provider, zone string, recordsToProvision []libdns.Record, recordsToTest []libdns.Record) []libdns.Record {
	t.Helper()

	provisionedRecords, err := provider.AppendRecords(context.Background(), zone, recordsToProvision)
	if err != nil {
		t.Fatalf("failed to create records: %v", err)
	}

	if len(provisionedRecords) > 0 {
		// Replace placeholder IDs of any existing records to test with a real ID from the provision
		for i, record := range recordsToTest {
			if record.ID != "" {
				recordsToTest[i].ID = provisionedRecords[0].ID
			}
		}
	}

	return provisionedRecords
}

func cleanupRecords(t *testing.T, provider *Provider, records []libdns.Record) {
	t.Helper()

	_, err := provider.DeleteRecords(context.Background(), envZone, records)
	if err != nil {
		t.Fatalf("failed to delete records: %v", err)
	}
}

func assertErrorIs(t *testing.T, actual, expected error) {
	t.Helper()

	if expected == nil {
		if actual != nil {
			t.Fatalf("expected no error, but got: %v", actual)
		}
	} else {
		if !errors.Is(actual, expected) {
			t.Fatalf("expected error %v, but got: %v", expected, actual)
		}
	}
}

func assertRecordListsEqual(t *testing.T, actual, expected []libdns.Record) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf("expected %d records, but got %d", len(expected), len(actual))
	}

	for i := range expected {
		if actual[i].ID == "" {
			t.Fatalf("expected record ID to be present, but got nil")
		}
		if actual[i].Type != expected[i].Type {
			t.Fatalf("expected record Type %s, but got %s", expected[i].Type, actual[i].Type)
		}
		if actual[i].Name != expected[i].Name {
			t.Fatalf("expected record Name %s, but got %s", expected[i].Name, actual[i].Name)
		}
		if actual[i].Value != expected[i].Value {
			t.Fatalf("expected record Value %s, but got %s", expected[i].Value, actual[i].Value)
		}
		if actual[i].TTL != expected[i].TTL {
			t.Fatalf("expected record TTL %v, but got %v", expected[i].TTL, actual[i].TTL)
		}
		if actual[i].Priority != expected[i].Priority {
			t.Fatalf("expected record Priority %d, but got %d", expected[i].Priority, actual[i].Priority)
		}
	}
}

func TestGetRecords(t *testing.T) {
	input := map[string]struct {
		zone               string
		recordsToProvision []libdns.Record
		expectedRecords    []libdns.Record
		expectedErr        error
	}{
		"Success": {
			zone: envZone + ".",
			recordsToProvision: []libdns.Record{
				{
					Type:  "A",
					Name:  "example.com",
					Value: "127.0.0.1",
					TTL:   time.Duration(300) * time.Second,
				},
			},
			expectedRecords: []libdns.Record{
				{
					Type:     "A",
					Name:     "example.com",
					Value:    "127.0.0.1",
					TTL:      time.Duration(300) * time.Second,
					Priority: 0,
				},
			},
		},
		"API Error": {
			zone:        "_",
			expectedErr: errUnexpectedStatusCode,
		},
	}

	for name, data := range input {
		t.Run(name, func(t *testing.T) {
			provider := &Provider{APIToken: envToken}
			resultFromProvision := provisionRecords(t, provider, data.zone, data.recordsToProvision, []libdns.Record{})
			defer cleanupRecords(t, provider, resultFromProvision)

			records, err := provider.GetRecords(context.Background(), data.zone)
			assertErrorIs(t, err, data.expectedErr)
			assertRecordListsEqual(t, records, data.expectedRecords)
		})
	}
}

func TestAppendRecords(t *testing.T) {
	input := map[string]struct {
		zone            string
		recordsToAdd    []libdns.Record
		expectedRecords []libdns.Record
		expectedErr     error
	}{
		"Success": {
			zone: envZone + ".",
			recordsToAdd: []libdns.Record{
				{
					Type:  "CNAME",
					Name:  "test",
					Value: "test",
				},
			},
			expectedRecords: []libdns.Record{
				{
					Type:  "CNAME",
					Name:  "test",
					Value: "test." + envZone,
					TTL:   time.Duration(0) * time.Second,
				},
			},
		},
		"API Error": {
			zone: "_",
			recordsToAdd: []libdns.Record{
				{
					Type:  "CNAME",
					Name:  "test",
					Value: "test",
					TTL:   time.Duration(300) * time.Second,
				},
			},
			expectedErr: errUnexpectedStatusCode,
		},
	}

	for name, data := range input {
		t.Run(name, func(t *testing.T) {
			provider := &Provider{APIToken: envToken}
			addedRecords, err := provider.AppendRecords(context.Background(), data.zone, data.recordsToAdd)
			defer cleanupRecords(t, provider, addedRecords)
			assertErrorIs(t, err, data.expectedErr)
			assertRecordListsEqual(t, addedRecords, data.expectedRecords)
		})
	}
}

func TestSetRecords(t *testing.T) {
	input := map[string]struct {
		zone               string
		recordsToProvision []libdns.Record
		recordsToSet       []libdns.Record
		expectedRecords    []libdns.Record
		expectedErr        error
	}{
		"Success": {
			zone: envZone + ".",
			recordsToProvision: []libdns.Record{
				{
					Type:  "A",
					Name:  "testrecord.com",
					Value: "0.0.0.0",
					TTL:   time.Duration(300) * time.Second,
				},
			},
			recordsToSet: []libdns.Record{
				{
					ID:    "existingrecord", // here we test changing the type and value of the existing record
					Type:  "TXT",
					Name:  "testrecord.com",
					Value: "hello",
					TTL:   time.Duration(300) * time.Second,
				},
				{
					Type:  "A",
					Name:  "newrecord.com",
					Value: "0.0.0.0",
					TTL:   time.Duration(300) * time.Second,
				},
			},
			expectedRecords: []libdns.Record{
				{
					Type:  "TXT",
					Name:  "testrecord.com",
					Value: "hello",
					TTL:   time.Duration(300) * time.Second,
				},
				{
					Type:  "A",
					Name:  "newrecord.com",
					Value: "0.0.0.0",
					TTL:   time.Duration(300) * time.Second,
				},
			},
		},
		"API Error": {
			zone: "_",
			recordsToSet: []libdns.Record{
				{
					Type:  "A",
					Name:  "newrecord.com",
					Value: "0.0.0.0",
					TTL:   time.Duration(300) * time.Second,
				},
			},
			expectedErr: errUnexpectedStatusCode,
		},
	}

	for name, data := range input {
		t.Run(name, func(t *testing.T) {
			provider := &Provider{APIToken: envToken}
			provisionRecords(t, provider, data.zone, data.recordsToProvision, data.recordsToSet)

			updatedRecords, err := provider.SetRecords(context.Background(), data.zone, data.recordsToSet)
			defer cleanupRecords(t, provider, updatedRecords)
			assertErrorIs(t, err, data.expectedErr)
			assertRecordListsEqual(t, updatedRecords, data.expectedRecords)
		})
	}
}

func TestDeleteRecords(t *testing.T) {
	input := map[string]struct {
		zone               string
		recordsToProvision []libdns.Record
		recordsToDelete    []libdns.Record
		expectedRecords    []libdns.Record
		expectedErr        error
	}{
		"Success": {
			zone: envZone + ".",
			recordsToProvision: []libdns.Record{
				{
					Type:  "A",
					Name:  "testrecord.com",
					Value: "0.0.0.0",
					TTL:   time.Duration(300) * time.Second,
				},
			},
			recordsToDelete: []libdns.Record{
				{
					ID:   "existingrecord",
					Type: "A",
					Name: "testrecord.com",
				},
			},
			expectedRecords: []libdns.Record{
				{
					Type: "A",
					Name: "testrecord.com",
				},
			},
		},
		"API Error": {
			zone: "_",
			recordsToDelete: []libdns.Record{
				{
					ID:   "existingrecord",
					Type: "A",
					Name: "example.com",
				},
			},
			expectedErr: errUnexpectedStatusCode,
		},
	}

	for name, data := range input {
		t.Run(name, func(t *testing.T) {
			provider := &Provider{APIToken: envToken}
			provisionRecords(t, provider, data.zone, data.recordsToProvision, data.recordsToDelete)

			deletedRecords, err := provider.DeleteRecords(context.Background(), data.zone, data.recordsToDelete)
			assertErrorIs(t, err, data.expectedErr)
			assertRecordListsEqual(t, deletedRecords, data.expectedRecords)
		})
	}
}

func TestMain(m *testing.M) {
	envToken = os.Getenv("LIBDNS_KATAPULT_API_TOKEN")
	envZone = os.Getenv("LIBDNS_KATAPULT_ZONE")

	if len(envToken) == 0 || len(envZone) == 0 {
		fmt.Println(`Please note that these tests use the Katapult API.
		You should create a new and empty domain/zone to avoid modifying any production data.
		Specify 'LIBDNS_KATAPULT_API_TOKEN' and 'LIBDNS_KATAPULT_ZONE' to continue.`)
		os.Exit(1)
	}

	os.Exit(m.Run())
}
