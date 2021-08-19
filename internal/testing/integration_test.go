package testing

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/libdns/libdns"
	"github.com/libdns/namecheap"
)

var (
	apiKey      = flag.String("api-key", "", "Namecheap API key.")
	apiUser     = flag.String("username", "", "Namecheap API username.")
	apiEndpoint = flag.String("endpoint", "https://api.sandbox.namecheap.com/xml.response", "Namecheap API endpoint.")
	domain      = flag.String("domain", "", "Domain to test with of the form sld.tld <testing.com>")
)

func TestIntegration(t *testing.T) {
	p := &namecheap.Provider{
		APIKey:      *apiKey,
		User:        *apiUser,
		APIEndpoint: *apiEndpoint,
	}

	newRecords := []libdns.Record{
		{
			Type:  "A",
			Name:  "@",
			Value: "168.91.162.103",
			TTL:   time.Second * 1799,
		},
	}

	t.Log("Appending Records")

	addedRecords, err := p.AppendRecords(context.TODO(), *domain, newRecords)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Records appended: %#v", addedRecords)

	records, err := p.GetRecords(context.TODO(), *domain)
	if err != nil {
		t.Fatal(err)
	}

	// IDs are not returned by append. Maybe they should be?
	ignoreIDField := cmpopts.IgnoreFields(libdns.Record{}, "ID")
	if diff := cmp.Diff(addedRecords, records, ignoreIDField); diff != "" {
		t.Fatalf("Added records not equal to fetched records. Diff: %s", diff)
	}

	firstRecord := records[0]
	_, err = p.DeleteRecords(context.TODO(), *domain, []libdns.Record{firstRecord})
	if err != nil {
		t.Fatal(err)
	}

	records, err = p.GetRecords(context.TODO(), *domain)
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 0 {
		t.Fatalf("Expected 0 but got: %v", len(records))
	}
}
