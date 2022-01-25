package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"

	godaddy "github.com/artknight/libdns-godaddy"
	"github.com/libdns/libdns"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	token := os.Getenv("GODADDY_TOKEN")
	if token == "" {
		fmt.Printf("GODADDY_TOKEN not set\n")
		return
	}
	zone := os.Getenv("ZONE")
	if zone == "" {
		fmt.Printf("ZONE not set\n")
		return
	}
	provider := godaddy.Provider{
		APIToken: token,
	}

	records, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		log.Fatalln("ERROR: %s\n", err.Error())
	}

	testName := "_acme-challenge.home"
	hasTestName := false

	for _, record := range records {
		fmt.Printf("%s (.%s): %s, %s\n", record.Name, zone, record.Value, record.Type)
		if record.Name == testName {
			hasTestName = true
		}
	}

	if !hasTestName {
		appendedRecords, err := provider.AppendRecords(context.TODO(), zone, []libdns.Record{
			libdns.Record{
				Type:  "TXT",
				Name:  testName + "." + zone,
				TTL:   0,
				Value: "20HnRk5p6rZd7TXhiMoVEYSjt5OpetC6mdovlTfJ4As",
			},
		})

		if err != nil {
			log.Fatalln("ERROR: %s\n", err.Error())
		}

		fmt.Println("appendedRecords")
		fmt.Println(appendedRecords)
	} else {
		deleteRecords, err := provider.DeleteRecords(context.TODO(), zone, []libdns.Record{
			libdns.Record{
				Type: "TXT",
				Name: testName,
			},
		})

		if err != nil {
			log.Fatalln("ERROR: %s\n", err.Error())
		}

		fmt.Println("deleteRecords")
		fmt.Println(deleteRecords)
	}
}
