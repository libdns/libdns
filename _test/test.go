package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/libdns/libdns"
	porkbun "github.comn/Niallfitzy1/libdns-porkbun"
)

func main() {

	apikey := os.Getenv("PORKBUN_API_KEY")
	secretapikey := os.Getenv("PORKBUN_SECRET_API_KEY")
	zone := os.Getenv("ZONE")

	if apikey == "" || secretapikey == "" || zone == "" {
		fmt.Println("All variables must be set in '.env' file")
		return
	}

	provider := porkbun.Provider{
		APIKey:       apikey,
		APISecretKey: secretapikey,
	}

	//Check Authorization
	_, err := provider.CheckCredentials(context.TODO())

	if err != nil {
		log.Fatalln("Credential check failed: %s\n", err.Error())
	}

	//Get records
	records, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		log.Fatalln("Failed to fetch records: %s\n", err.Error())
	}

	log.Println("Records fetched:")
	for _, record := range records {
		fmt.Printf("%s (.%s): %s, %s\n", record.Name, zone, record.Value, record.Type)
	}

	testValue := "test-value"
	updatedTestValue := "updated-test-value"
	ttl := time.Duration(600 * time.Second)
	recordType := "TXT"
	testFullName := "libdns_test_record." + zone

	//Create record
	appendedRecords, err := provider.AppendRecords(context.TODO(), zone, []libdns.Record{
		libdns.Record{
			Type:  recordType,
			Name:  testFullName,
			TTL:   ttl,
			Value: testValue,
		},
	})

	if err != nil {
		log.Fatalln("ERROR: %s\n", err.Error())
	}
	fmt.Printf("Created record: \n%v\n", appendedRecords[0])

	// Update record
	updatedRecords, err := provider.SetRecords(context.TODO(), zone, []libdns.Record{
		libdns.Record{
			Type:  recordType,
			Name:  testFullName,
			TTL:   ttl,
			Value: updatedTestValue,
		},
	})

	if err != nil {
		log.Fatalln("ERROR: %s\n", err.Error())
	}
	fmt.Printf("Updated record: \n%v\n", updatedRecords[0])

	// Delete record
	deleteRecords, err := provider.DeleteRecords(context.TODO(), zone, []libdns.Record{
		libdns.Record{
			Type: recordType,
			Name: testFullName,
		},
	})

	if err != nil {
		log.Fatalln("ERROR: %s\n", err.Error())
	}
	fmt.Printf("Deleted record: \n%v\n", deleteRecords[0])

}
