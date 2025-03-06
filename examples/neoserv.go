package main

import (
	"context"
	"fmt"
	"os"

	"github.com/libdns/libdns"
	"github.com/libdns/neoserv"
)

// This example demonstrates how to use the Neoserv provider to manage DNS records.
// It requires the following environment variables to be set:
// - NEOSERV_USERNAME: the username for the Neoserv API
// - NEOSERV_PASSWORD: the password for the Neoserv API
// - NEOSERV_ZONE: the zone to manage
// The example will list the existing records, add a new TXT record, and list the added record.
func main() {
	username := os.Getenv("NEOSERV_USERNAME")
	password := os.Getenv("NEOSERV_PASSWORD")
	zone := os.Getenv("NEOSERV_ZONE")
	if username == "" || password == "" || zone == "" {
		fmt.Println("Please set the NEOSERV_USERNAME, NEOSERV_PASSWORD and NEOSERV_ZONE environment variables.")
		os.Exit(1)
	}

	provider := neoserv.Provider{
		Username: username,
		Password: password,
	}

	// Get existing records
	records, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	for _, record := range records {
		fmt.Printf("%s (ID: %s): %s, %s, TTL: %s\n", record.Name, record.ID, record.Type, record.Value, record.TTL.String())
	}

	// Add a new record
	newRecords, err := provider.AppendRecords(context.TODO(), zone, []libdns.Record{
		{
			Type:  "TXT",
			Name:  "test",
			Value: "This is a test",
			TTL:   neoserv.TTL12h, // Neoserv supports specific TTL values
		},
	})
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	fmt.Printf("Added: %v\n", newRecords)
}
