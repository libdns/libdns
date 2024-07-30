package he

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/libdns/libdns"
)

var (
	apiKey = os.Getenv("LIBDNS_HE_KEY")
	zone   = os.Getenv("LIBDNS_HE_ZONE")
)

var (
	provider Provider
)

func init() {
	if apiKey == "" {
		log.Fatalf("API key needs to be provided in env var LIBDNS_HE_KEY")
	}
	if zone == "" {
		log.Fatalf("DNS zone needs to be provided in env var LIBDNS_HE_ZONE")
	}
	if zone[len(zone)-1:] != "." {
		// Zone names come from caddy with trailing period
		zone += "."
	}
	provider = Provider{APIKey: apiKey}
}

func TestAppendRecords(t *testing.T) {
	ctx := context.Background()

	records := []libdns.Record{
		{
			Type:  "A",
			Name:  "test001",
			Value: "192.0.2.1",
		},
		{
			Type:  "AAAA",
			Name:  "test001",
			Value: "2001:0db8:2::1",
		},
		{
			Type:  "TXT",
			Name:  "test001",
			Value: "ZYXWVUTSRQPONMLKJIHGFEDCBA",
		},
	}

	createdRecords, err := provider.AppendRecords(ctx, zone, records)
	if err != nil {
		t.Errorf("%v", err)
	}

	if len(records) != len(createdRecords) {
		t.Errorf("Number of appended records does not match number of records")
	}
}

func TestGetRecords(t *testing.T) {
	ctx := context.Background()

	records, err := provider.GetRecords(ctx, zone)
	if err != nil {
		t.Errorf("%v", err)
	}

	if len(records) == 0 {
		t.Errorf("No records")
	}
}

func TestSetRecords(t *testing.T) {
	ctx := context.Background()

	goodRecords := []libdns.Record{
		{
			Type:  "A",
			Name:  "test001",
			Value: "198.51.100.1",
		},
		{
			Type:  "AAAA",
			Name:  "test001",
			Value: "2001:0db8::1",
		},
		{
			Type:  "TXT",
			Name:  "test001",
			Value: "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		},
		{
			Type:  "A",
			Name:  "test002",
			Value: "198.51.100.2",
		},
		{
			Type:  "A",
			Name:  "test003",
			Value: "198.51.100.3",
		},
	}

	createdRecords, err := provider.SetRecords(ctx, zone, goodRecords)
	if err != nil {
		t.Fatalf("adding records failed: %v", err)
	}

	if len(goodRecords) != len(createdRecords) {
		t.Fatalf("Number of added records does not match number of records")
	}

	badRecords := []libdns.Record{
		{
			Type:  "CNAME",
			Name:  "test000",
			Value: "example.org",
		},
	}

	_, err = provider.SetRecords(ctx, zone, badRecords)
	if err == nil {
		t.Fatalf("unsupported records should return error")
	}
}

func TestDeleteRecords(t *testing.T) {
	ctx := context.Background()

	records := []libdns.Record{
		{
			Type: "A",
			Name: "test001",
		},
		{
			Type: "AAAA",
			Name: "test001",
		},
		{
			Type: "TXT",
			Name: "test001",
		},
	}

	deletedRecords, err := provider.DeleteRecords(ctx, zone, records)
	if err != nil {
		t.Errorf("deleting records failed: %v", err)
	}

	if len(records) != len(deletedRecords) {
		t.Errorf("Number of deleted records does not match number of records")
	}
}
