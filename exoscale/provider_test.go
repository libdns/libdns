package exoscale

import (
	"context"
	"os"
	"testing"
	"time"

// 	"github.com/libdns/exoscale"
	"github.com/libdns/libdns"
)

var (
	APIKey = os.Getenv("TEST_API_KEY")
	APISecret = os.Getenv("TEST_API_SECRET")
	zone = os.Getenv("TEST_ZONE")

	ttl = time.Duration(1 * time.Hour)
)

type testRecordsCleanup = func()

func TestMain(m *testing.M) {
	if len(APIKey) == 0 || len(APISecret) == 0 {
		panic("APIKey and APISecret must be set using environment variables")
	}

	os.Exit(m.Run())
}

func setupTestRecords(t *testing.T, ctx context.Context, p *Provider) ([]libdns.Record, testRecordsCleanup) {
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
	records, err := p.AppendRecords(context.Background(), zone, testRecords)
	if err != nil {
		t.Fatal(err)
		return nil, func() {}
	}

	return records, func() {
		cleanupRecords(t, ctx, p, records)
	}
}

func cleanupRecords(t *testing.T, ctx context.Context, p *Provider, r []libdns.Record) {
	_, err := p.DeleteRecords(ctx, zone, r)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}

func Test_AppendRecords(t *testing.T) {
	p := &Provider{
		APIKey: APIKey,
		APISecret: APISecret,
	}
	ctx := context.Background()

	testCases := []struct {
		records  []libdns.Record
		expected []libdns.Record
	}{
		{
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
			records: []libdns.Record{
				{Type: "TXT", Name: "123.test", Value: "123", TTL: ttl},
			},
			expected: []libdns.Record{
				{Type: "TXT", Name: "123.test", Value: "123", TTL: ttl},
			},
		},
		{
			records: []libdns.Record{
				{Type: "TXT", Name: "123.test", Value: "test", TTL: ttl},
			},
			expected: []libdns.Record{
				{Type: "TXT", Name: "123.test", Value: "test", TTL: ttl},
			},
		},
		{
			records: []libdns.Record{
				{Type: "TXT", Name: "abc.test", Value: "test", TTL: ttl},
			},
			expected: []libdns.Record{
				{Type: "TXT", Name: "abc.test", Value: "test", TTL: ttl},
			},
		},
	}

	for _, c := range testCases {
		func() {
			result, err := p.AppendRecords(ctx, zone+".", c.records)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanupRecords(t, ctx, p, result)

			if len(result) != len(c.records) {
				t.Fatalf("len(result) != len(c.records) => %d != %d", len(result), len(c.records))
			}

			for k, r := range result {
				if len(result[k].ID) == 0 {
					t.Fatalf("len(result[%d].ID) == 0", k)
				}
				if r.Type != c.expected[k].Type {
					t.Fatalf("r.Type != c.exptected[%d].Type => %s != %s", k, r.Type, c.expected[k].Type)
				}
				if r.Name != c.expected[k].Name {
					t.Fatalf("r.Name != c.exptected[%d].Name => %s != %s", k, r.Name, c.expected[k].Name)
				}
				if r.Value != c.expected[k].Value {
					t.Fatalf("r.Value != c.exptected[%d].Value => %s != %s", k, r.Value, c.expected[k].Value)
				}
				if r.TTL != c.expected[k].TTL {
					t.Fatalf("r.TTL != c.exptected[%d].TTL => %s != %s", k, r.TTL, c.expected[k].TTL)
				}
			}
		}()
	}
}

func Test_DeleteRecords(t *testing.T) {
	p := &Provider{
		APIKey: APIKey,
		APISecret: APISecret,
	}
	ctx := context.Background()

	testRecords, cleanupFunc := setupTestRecords(t, ctx, p)
	defer cleanupFunc()

	records, err := p.GetRecords(ctx, zone)
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
	p := &Provider{
		APIKey: APIKey,
		APISecret: APISecret,
	}
	ctx := context.Background()

	testRecords, cleanupFunc := setupTestRecords(t, ctx, p)
	defer cleanupFunc()

	records, err := p.GetRecords(ctx, zone)
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
	p := &Provider{
		APIKey: APIKey,
		APISecret: APISecret,
	}
	ctx := context.Background()

	existingRecords, _ := setupTestRecords(t, ctx, p)

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

	records, err := p.SetRecords(ctx, zone, allRecords)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupRecords(t, ctx, p, records)

	if len(records) != len(allRecords) {
		t.Fatalf("len(records) != len(allRecords) => %d != %d", len(records), len(allRecords))
	}

	updated := false
	for _, r := range records {
		if r.Value == "new_value" {
			updated = true
		}
	}
	if !updated {
		t.Fatalf("Did not update value on existing record")
	}
}