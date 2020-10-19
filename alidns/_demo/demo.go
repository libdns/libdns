package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	al "github.com/libdns/alidns"
	l "github.com/libdns/libdns"
)

func main() {
	accKeyID := strings.TrimSpace(os.Getenv("ACCESS_KEY_ID"))
	accKeySec := strings.TrimSpace(os.Getenv("ACCESS_KEY_SECRET"))
	zone := strings.TrimSpace(os.Args[1])
	if (accKeyID == "") || (accKeySec == "") {
		fmt.Printf("ERROR: %s\n", "ACCESS_KEY_ID or ACCESS_KEY_SECRET missing")
		return
	}
	if zone == "" {
		fmt.Printf("ERROR: %s\n", "First arg zone missing")
		return
	}
	fmt.Printf("Get ACCESS_KEY_ID: %s,ACCESS_KEY_SECRET: %s,ZONE: %s\n", accKeyID, accKeySec, zone)
	provider := al.Provider{
		AccKeyID:     accKeyID,
		AccKeySecret: accKeySec,
	}
	records, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	}
	testName := "_libdns_test"
	testID := ""
	for _, record := range records {
		fmt.Printf("%s (.%s): %s, %s\n", record.Name, zone, record.Value, record.Type)
		if testName == record.Name {
			testID = record.ID
		}
	}

	if testID == "" {
		fmt.Println("Creating new entry for ", testName)
		records, err = provider.AppendRecords(context.TODO(), zone, []l.Record{l.Record{
			Type:  "TXT",
			Name:  testName,
			Value: fmt.Sprintf("This+is a test entry created by libdns %s", time.Now()),
			TTL:   time.Duration(600) * time.Second,
		}})
		if len(records) == 1 {
			testID = records[0].ID
		}
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
			return
		}
	}

	fmt.Println("Press any Key to update the test entry")
	fmt.Scanln()
	if testID != "" {
		fmt.Println("Replacing entry for ", testName)
		records, err = provider.SetRecords(context.TODO(), zone, []l.Record{l.Record{
			Type:  "TXT",
			Name:  testName,
			Value: fmt.Sprintf("Replacement test entry created by libdns %s", time.Now()),
			TTL:   time.Duration(605) * time.Second,
			ID:    testID,
		}})
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
			return
		}
	}
	fmt.Println("Press any Key to delete the test entry")
	fmt.Scanln()
	fmt.Println("Deleting the entry for ", testName)
	_, err = provider.DeleteRecords(context.TODO(), zone, records)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	}

}
