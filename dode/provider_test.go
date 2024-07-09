// Integration tests for the do.de provider

package dode

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/libdns/libdns"
)

var (
	apiToken    = ""
	zone        = ""
	testRecords = []libdns.Record{
		{
			Type:  "TXT",
			Name:  "_acme-challenge.test",
			Value: "foo",
		},
	}
)

func TestMain(m *testing.M) {
	fmt.Println("Loading environment variables to set up provider")
	apiToken = os.Getenv("LIBDNS_DODE_API_TOKEN")
	zone = os.Getenv("LIBDNS_DODE_ZONE")

	os.Exit(m.Run())
}

func TestProvider_AppendDeleteRecords(t *testing.T) {
	p := &Provider{
		APIToken: apiToken,
	}

	_, err := p.AppendRecords(context.TODO(), zone, testRecords)
	if err != nil {
		t.Fatalf("appending record failed: %v", err)
		return
	}

	_, err = p.DeleteRecords(context.TODO(), zone, testRecords)
	if err != nil {
		t.Fatalf("deleting record failed: %v", err)
	}
}
