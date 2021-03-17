package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/libdns/libdns"
	"github.com/libdns/metaname"
)

func main() {
	ctx := context.TODO()
	if len(os.Args) < 2 {
		fmt.Println("Usage: ", os.Args[0], "<zone>")
		fmt.Println("This program adds, updates, and deletes specific records in the given zone.")
		fmt.Println("These changes may be destructive to existing data!")
		fmt.Println("Guesswork deletion is checked using a TXT record 'todelete' with value 'this will go away'")
		fmt.Println("Other records created/changed are 'test' and 'additional'.")
		os.Exit(1)
	}
	endpoint := "https://test.metaname.net/api/1.1"
	provider := metaname.Provider{APIKey: os.Getenv("api_key"),
		AccountReference: os.Getenv("account_reference"),
		Endpoint:         endpoint}
	zone := os.Args[1]
	recs, _ := provider.GetRecords(ctx, zone)
	for _, r := range recs {
		fmt.Println("found", r.Name, r.Type, r.Value)
	}
	added, err := provider.AppendRecords(ctx, zone, []libdns.Record{
		{
			Name:  "test",
			TTL:   time.Duration(300) * time.Second,
			Value: "8.8.8.8",
			Type:  "A",
		},
	})
	if err != nil {
		fmt.Println(err)
	}
	newone := added[0]
	fmt.Println("New test record's ID", newone.ID)
	for _, r := range added {
		fmt.Println("added", r.Name, r.Type, r.Value)
	}
	rec := libdns.Record{Name: "todelete", Type: "TXT", Value: "this will go away"}
	deleted, err := provider.DeleteRecords(ctx, zone, []libdns.Record{rec})
	if err != nil {
		fmt.Println(err)
	}
	for _, r := range deleted {
		fmt.Println("deleted", r.Name, r.Type, r.Value)
	}
	updated, err := provider.SetRecords(ctx, zone, []libdns.Record{
		{
			ID:    newone.ID,
			Value: "1.2.3.4",
		},
		{
			Name:  "additional",
			TTL:   3600,
			Value: "google.com.",
			Type:  "CNAME",
		},
	})
	if err != nil {
		fmt.Println(err)
	}
	for _, r := range updated {
		fmt.Println("updated", r.Name, r.Type, r.Value)
	}
}
