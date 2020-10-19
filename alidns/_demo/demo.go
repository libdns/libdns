package main

import (
	"context"
	"fmt"
	"time"

	al "github.com/libdns/alidns"
	l "github.com/libdns/libdns"
)

func main() {
	provider := al.Provider{
		AccKeyID:     "LTAI4G3UTA4x2XRm8HrgGJ63",       //Modify your AccessKeyId here
		AccKeySecret: "wO0q5OUKPWg8Iuy63VNuxLsdHGSH6d", //Modify your AccessKeySecret here
	}
	zone := "viscrop.top" //Modify your Zone here
	records, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
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
		}
	}

	fmt.Println("Press any Key to update the test entry")
	fmt.Scanln()
	if testID != "" {
		fmt.Println("Replacing entry for ", testName)
		_, err = provider.SetRecords(context.TODO(), zone, []l.Record{l.Record{
			Type:  "TXT",
			Name:  testName,
			Value: fmt.Sprintf("Replacement test entry created by libdns %s", time.Now()),
			TTL:   time.Duration(605) * time.Second,
			ID:    testID,
		}})
		if err != nil {
			fmt.Printf("ERROR: %s\n", err.Error())
		}
	}
	fmt.Println("Press any Key to delete the test entry")
	fmt.Scanln()
	fmt.Println("Deleting the entry for ", testName)
	_, err = provider.DeleteRecords(context.TODO(), zone, records)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}

}
