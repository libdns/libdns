package main

import (
	"context"
	"fmt"
	"os"
	"time"

	scaleway "github.com/libdns/digitalocean"
	"github.com/libdns/libdns"
)

func main() {
	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		fmt.Printf("SECRET_KEY not set\n")
		return
	}
	organisationID := os.Getenv("ORGANISATION_ID")
	if organisationID == "" {
		fmt.Printf("ORGANISATION_ID not set\n")
		return
	}
	zone := os.Getenv("ZONE")
	if zone == "" {
		fmt.Printf("ZONE not set\n")
		return
	}
	provider := scaleway.Provider{
		SecretKey:      secretKey,
		OrganizationID: organisationID,
	}

	records, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

	testName := "libdnsx-test"
	testId := ""
	for _, record := range records {
		fmt.Printf("%s (.%s): %s, %s\n", record.Name, zone, record.Value, record.Type)
		if record.Name == (testName + "." + zone + ".") {
			testId = record.ID
		}
	}

	if testId != "" {
		/*
			fmt.Printf("Delete entry for %s (id:%s)\n", testName, testId)
			_, err = provider.DeleteRecords(context.TODO(), zone, []libdns.Record{libdns.Record{
				ID: testId,
			}})
			if err != nil {
				fmt.Printf("ERROR: %s\n", err.Error())
			}
		*/
		// Set only works if we have a record.ID
		fmt.Printf("Replacing entry for %s\n", testName)
		_, err = provider.SetRecords(context.TODO(), zone, []libdns.Record{libdns.Record{
			Type:  "TXT",
			Name:  testName,
			Value: fmt.Sprintf("Replacement test entry created by libdns %s", time.Now()),
			TTL:   time.Duration(30) * time.Second,
			ID:    testId,
		}})
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
	} else {
		fmt.Printf("Creating new entry for %s\n", testName)
		_, err = provider.AppendRecords(context.TODO(), zone, []libdns.Record{libdns.Record{
			Type:  "TXT",
			Name:  testName,
			Value: fmt.Sprintf("This is a test entry created by libdns %s", time.Now()),
			TTL:   time.Duration(30) * time.Second,
		}})
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
	}
}
