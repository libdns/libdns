package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	easydns "github.com/libdns/easydns"
	"github.com/libdns/libdns"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	token := os.Getenv("EASYDNS_TOKEN")
	if token == "" {
		fmt.Printf("EASYDNS_TOKEN not set\n")
		return
	}
	key := os.Getenv("EASYDNS_KEY")
	if key == "" {
		fmt.Printf("EASYDNS_KEY not set\n")
		return
	}
	url := os.Getenv("EASYDNS_URL")
	if url == "" {
		url = "https://rest.easydns.net"
	}
	zone := os.Getenv("EASYDNS_ZONE")
	if zone == "" {
		fmt.Printf("EASYDNS_ZONE not set\n")
		return
	}
	provider := easydns.Provider{
		APIToken: token,
		APIKey:   key,
		APIUrl:   url,
	}

	records, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		log.Fatalln("ERROR: %s\n", err.Error())
	}

	testName := "_acme-challenge.home"
	hasTestName := false
	var testRecord libdns.Record = libdns.Record{}

	for _, record := range records {
		fmt.Printf("%s (.%s): %s, %s\n", record.Name, zone, record.Value, record.Type)
		if record.Name == fmt.Sprintf("%s.%s", testName, zone) {
			hasTestName = true
			testRecord = record
		}
	}

	if !hasTestName {
		appendedRecords, err := provider.AppendRecords(context.TODO(), zone, []libdns.Record{
			libdns.Record{
				Type:  "TXT",
				Name:  testName + "." + zone,
				TTL:   0,
				Value: "test_add_record_value",
			},
		})

		if err != nil {
			log.Fatalln("ERROR: %s\n", err.Error())
		}

		fmt.Println("appendedRecords")
		fmt.Println(appendedRecords)
	} else if testRecord.Value == "test_add_record_value" {
		testRecord.Value = "test_update_record_value"
		updatedRecords, err := provider.SetRecords(context.TODO(), zone, []libdns.Record{
			testRecord,
		})

		if err != nil {
			log.Fatalln("ERROR: %s\n", err.Error())
		}

		fmt.Println("updatedRecords")
		fmt.Println(updatedRecords)
	} else {
		deleteRecords, err := provider.DeleteRecords(context.TODO(), zone, []libdns.Record{
			testRecord,
		})

		if err != nil {
			log.Fatalln("ERROR: %s\n", err.Error())
		}

		fmt.Println("deleteRecords")
		fmt.Println(deleteRecords)
	}
}
