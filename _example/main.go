package main

import (
	"context"
	"fmt"
	"time"

	"github.com/libdns/libdns"
	"github.com/libdns/mythicbeasts"
)

func main() {
	ctx := context.TODO()

	zone := "example.com."

	provider := mythicbeasts.Provider{KeyID: "KEYID_GOES_HERE", Secret: "SECRET_GOES_HERE"}

	// Get Records Test
	records, err := provider.GetRecords(ctx, zone)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	// Append Records Test
	recordsAdded, err := provider.AppendRecords(ctx, zone, []libdns.Record{
		{Type: "A", Name: "test1", Value: "8.8.4.4", TTL: time.Duration(123) * time.Second},
		{Type: "CNAME", Name: "test2", Value: "www.example.com.", TTL: time.Duration(666) * time.Second},
	})
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	// Set Records Test
	recordsSet, err := provider.SetRecords(ctx, zone, []libdns.Record{
		{Type: "A", Name: "test1", Value: "8.8.8.8", TTL: time.Duration(999) * time.Second},
		{Type: "CNAME", Name: "test2", Value: "test2.example.com", TTL: time.Duration(999) * time.Second},
		{Type: "CNAME", Name: "test3", Value: "test3.example.net"},
	})
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	// Delete Records Test
	recordsDeleted, err := provider.DeleteRecords(ctx, zone, []libdns.Record{
		{Type: "A", Name: "test1"},
		{Type: "CNAME", Name: "test2"},
	})
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	fmt.Printf("\nThe following records are available from %s:\n%+v\n", zone, records)
	fmt.Printf("\nThe following records have been added to %s:\n%+v\n", zone, recordsAdded)
	fmt.Printf("\nThe following records have been set on %s:\n%+v\n", zone, recordsSet)
	fmt.Printf("\nThe following records have been deleted on %s:\n%s\n", zone, recordsDeleted)
}
