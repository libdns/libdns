package ovh_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/libdns/ovh"
	"github.com/libdns/libdns"
)

var (
	endPoint 			= ""
	applicationKey  	= ""
	applicationSecret   = ""
	consumerKey 		= ""
	zone 				= ""
	ttl      			= time.Duration(120 * time.Second)
)

func TestMain(m *testing.M) {
	endPoint = os.Getenv("LIBDNS_OVH_TEST_ENDPOINT")
	applicationKey = os.Getenv("LIBDNS_OVH_TEST_APPLICATION_KEY")
	applicationSecret = os.Getenv("LIBDNS_OVH_TEST_APPLICATION_SECRET")
	consumerKey = os.Getenv("LIBDNS_OVH_TEST_CONSUMER_KEY")
	zone = os.Getenv("LIBDNS_OVH_TEST_ZONE")
	if len(endPoint) == 0 || len(applicationKey) == 0 || len(applicationSecret) == 0 || len(consumerKey) == 0 || len(zone) == 0 {
		fmt.Println(`Please notice that this test runs agains the public OVH DNS API, so you sould never run the test with a zone used in production. To run this test, you have to specify the environment variables specified in provider_test.go`)
		os.Exit(1)
	}
	
	os.Exit(m.Run())
}

type testRecordsCleanup = func()

func setupTestRecords(t *testing.T, p *ovh.Provider) ([]libdns.Record, testRecordsCleanup) {
	testRecords := []libdns.Record{
		{
			Type:  "TXT",
			Name:  "test1",
			Value: "test1",
			TTL:   ttl,
		}, {
			Type:  "TXT",
			Name:  "test2",
			Value: "test2",
			TTL:   ttl,
		}, {
			Type:  "TXT",
			Name:  "test3",
			Value: "test3",
			TTL:   ttl,
		},
	}

	records, err := p.AppendRecords(context.TODO(), zone, testRecords)
	if err != nil {
		t.Fatal(err)
		return nil, func() {}
	}

	return records, func() {
		cleanupRecords(t, p, records)
	}
}

func cleanupRecords(t *testing.T, p *ovh.Provider, r []libdns.Record) {
	_, err := p.DeleteRecords(context.TODO(), zone, r)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}

func Test_GetRecords(t *testing.T) {
	p := &ovh.Provider{
		Endpoint: endPoint,
		ApplicationKey: applicationKey,
		ApplicationSecret: applicationSecret,
		ConsumerKey: consumerKey,
	}

	testRecords, cleanupFunc := setupTestRecords(t, p)
	defer cleanupFunc()

	records, err := p.GetRecords(context.TODO(), zone)
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

func Test_DeleteRecords(t *testing.T) {
	p := &ovh.Provider{
		Endpoint: endPoint,
		ApplicationKey: applicationKey,
		ApplicationSecret: applicationSecret,
		ConsumerKey: consumerKey,
	}

	testRecords, cleanupFunc := setupTestRecords(t, p)
	defer cleanupFunc()

	records, err := p.GetRecords(context.TODO(), zone)
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
	p := &ovh.Provider{
		Endpoint: endPoint,
		ApplicationKey: applicationKey,
		ApplicationSecret: applicationSecret,
		ConsumerKey: consumerKey,
	}

	existingRecords, _ := setupTestRecords(t, p)
	newTestRecords := []libdns.Record{
		{
			Type:  "TXT",
			Name:  "new_test1",
			Value: "new_test1",
			TTL:   ttl,
		},
		{
			Type:  "TXT",
			Name:  "new_test2",
			Value: "new_test2",
			TTL:   ttl,
		},
	}

	allRecords := append(existingRecords, newTestRecords...)
	allRecords[0].Value = "new_value"

	records, err := p.SetRecords(context.TODO(), zone, allRecords)
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

func Test_AppendRecords(t *testing.T) {
	p := &ovh.Provider{
		Endpoint: endPoint,
		ApplicationKey: applicationKey,
		ApplicationSecret: applicationSecret,
		ConsumerKey: consumerKey,
	}

	testCases := []struct {
		records  []libdns.Record
		expected []libdns.Record
	}{
		{
			// multiple records
			records: []libdns.Record{
				{Type: "TXT", Name: "test_1", Value: "test_1", TTL: ttl},
				{Type: "TXT", Name: "test_2", Value: "test_2", TTL: ttl},
				{Type: "TXT", Name: "test_3", Value: "test_3", TTL: ttl},
			},
			expected: []libdns.Record{
				{Type: "TXT", Name: "test_1", Value: "test_1", TTL: ttl},
				{Type: "TXT", Name: "test_2", Value: "test_2", TTL: ttl},
				{Type: "TXT", Name: "test_3", Value: "test_3", TTL: ttl},
			},
		},
		{
			// relative name
			records: []libdns.Record{
				{Type: "TXT", Name: "123.test", Value: "123", TTL: ttl},
			},
			expected: []libdns.Record{
				{Type: "TXT", Name: "123.test", Value: "123", TTL: ttl},
			},
		},
		{
			// (fqdn) sans trailing dot
			records: []libdns.Record{
				{Type: "TXT", Name: fmt.Sprintf("123.test.%s", strings.TrimSuffix(zone, ".")), Value: "test", TTL: ttl},
			},
			expected: []libdns.Record{
				{Type: "TXT", Name: "123.test", Value: "test", TTL: ttl},
			},
		},
		{
			// fqdn with trailing dot
			records: []libdns.Record{
				{Type: "TXT", Name: fmt.Sprintf("123.test.%s.", strings.TrimSuffix(zone, ".")), Value: "test", TTL: ttl},
			},
			expected: []libdns.Record{
				{Type: "TXT", Name: "123.test", Value: "test", TTL: ttl},
			},
		},
	}

	for _, c := range testCases {
		func() {
			result, err := p.AppendRecords(context.TODO(), zone+".", c.records)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanupRecords(t, p, result)

			if len(result) != len(c.records) {
				t.Fatalf("len(resilt) != len(c.records) => %d != %d", len(c.records), len(result))
			}

			for k, r := range result {
				if len(result[k].ID) == 0 {
					t.Fatalf("len(result[%d].ID) == 0", k)
				}
				if r.Type != c.expected[k].Type {
					t.Fatalf("r.Type != c.expected[%d].Type => %s != %s", k, r.Type, c.expected[k].Type)
				}
				if r.Name != c.expected[k].Name {
					t.Fatalf("r.Name != c.expected[%d].Name => %s != %s", k, r.Name, c.expected[k].Name)
				}
				if r.Value != c.expected[k].Value {
					t.Fatalf("r.Value != c.expected[%d].Value => %s != %s", k, r.Value, c.expected[k].Value)
				}
				if r.TTL != c.expected[k].TTL {
					t.Fatalf("r.TTL != c.expected[%d].TTL => %s != %s", k, r.TTL, c.expected[k].TTL)
				}
			}
		}()
	}
}

