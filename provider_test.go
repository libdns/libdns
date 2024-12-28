package dnsexit

import (
	"context"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/joho/godotenv"
	"github.com/libdns/libdns"
)

var (
	apiKey string
	zone   string
)

var (
	provider Provider
)

// It is best to run these tests against a throwaway domain to prevent mistakes (DNSExit provides free domains). There is no cleanup after each test, so they do not work well as a suite, and are best run individually. Because the getRecords functionality uses Google DNS, there need to be records in the domain, and they need to have replicated to Google's DNS servers before this test will pass. (You can use https://toolbox.googleapps.com/apps/dig/ to verify before running.)
func init() {
	envErr := godotenv.Load()
	if envErr != nil {
		log.Fatalf("Unable to load environment variables from file")
	}
	apiKey = os.Getenv("LIBDNS_DNSEXIT_API_KEY")
	zone = os.Getenv("LIBDNS_DNSEXIT_ZONE")
	debug = os.Getenv("LIBDNS_DNSEXIT_DEBUG") == "TRUE"
	if apiKey == "" {
		log.Fatalf("API key needs to be provided in env var LIBDNS_DNSEXIT_API_KEY")
	}
	if zone == "" {
		log.Fatalf("DNS zone needs to be provided in env var LIBDNS_DNSEXIT_ZONE")
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
	if !(reflect.DeepEqual(records, createdRecords)) {
		t.Errorf("Appended records do not match")
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
}

func TestDeleteRecords(t *testing.T) {
	ctx := context.Background()

	records := []libdns.Record{
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
