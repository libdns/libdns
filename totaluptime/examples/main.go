package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/libdns/libdns"
	"github.com/libdns/totaluptime"
)

func main() {
	var rec libdns.Record
	var recs []libdns.Record

	provider := totaluptime.Provider{
		Username: "USERNAME", // provider API username
		Password: "PASSWORD", // provider API password
	}
	zone := "DNS_ZONE" // zone in provider account
	ctx := context.Background()

	// GetRecords demonstration -------------------------------------------------------------------
	fmt.Printf("List all records in zone: %s "+strings.Repeat("-", 80), zone)
	pauseForUser()

	result, err := provider.GetRecords(ctx, zone)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(PrettyPrint(result))

	// AppendRecords demonstration ----------------------------------------------------------------
	fmt.Printf("Append new records to zone: %s "+strings.Repeat("-", 80), zone)
	pauseForUser()

	recs = nil
	rec = libdns.Record{
		Type:  "TXT",
		Name:  "test-txt-record",
		Value: "txt record contents",
		TTL:   3600 * time.Second,
	}
	recs = append(recs, rec)

	rec = libdns.Record{
		Type:  "CNAME",
		Name:  "google",
		Value: "www.google.com",
		TTL:   3600 * time.Second,
	}
	recs = append(recs, rec)

	result, err = provider.AppendRecords(ctx, zone, recs)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(PrettyPrint(result))

	// SetRecords demonstration -------------------------------------------------------------------
	fmt.Printf("Set new records to zone (both append and modify): %s "+strings.Repeat("-", 80), zone)
	pauseForUser()

	recs = nil
	rec = libdns.Record{
		Type:  "TXT",
		Name:  "test-txt-record",
		Value: "txt record contents have been updated", //updated
		TTL:   60 * time.Second,                        // updated
	}
	recs = append(recs, rec)

	rec = libdns.Record{
		Type:  "CNAME",
		Name:  "bing", // record is net-new
		Value: "www.bing.com",
		TTL:   3600 * time.Second,
	}
	recs = append(recs, rec)

	result, err = provider.SetRecords(ctx, zone, recs)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(PrettyPrint(result))

	// DeleteRecords demonstration ----------------------------------------------------------------
	fmt.Printf("Delete test records in zone: %s "+strings.Repeat("-", 80), zone)
	pauseForUser()

	recs = nil
	rec = libdns.Record{
		Type:  "TXT",
		Name:  "test-txt-record",
		Value: "txt record contents",
		TTL:   3600 * time.Second,
	}
	recs = append(recs, rec)

	rec = libdns.Record{
		Type:  "CNAME",
		Name:  "google",
		Value: "www.google.com",
		TTL:   3600 * time.Second,
	}
	recs = append(recs, rec)

	rec = libdns.Record{
		Type:  "CNAME",
		Name:  "bing",
		Value: "www.bing.com",
		TTL:   3600 * time.Second,
	}
	recs = append(recs, rec)

	result, err = provider.DeleteRecords(ctx, zone, recs)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(PrettyPrint(result))

	// GetRecords demonstration -------------------------------------------------------------------
	fmt.Printf("List all (original) records in zone: %s "+strings.Repeat("-", 80), zone)
	pauseForUser()

	result, err = provider.GetRecords(ctx, zone)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(PrettyPrint(result))
}

func pauseForUser() {
	fmt.Printf("\nPress [Enter] to continue...\n")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func PrettyPrint(ugly interface{}) string {
	pretty, _ := json.MarshalIndent(ugly, "", "\t")
	return fmt.Sprintln(string(pretty))
}
