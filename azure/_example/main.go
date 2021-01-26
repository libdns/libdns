package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/libdns/azure"
	"github.com/libdns/libdns"
)

// main shows how libdns works with Azure DNS.
//
// In this example, the information required for authentication is passed as environment variables.
func main() {

	// Create new provider instance
	provider := azure.Provider{
		TenantId:          os.Getenv("AZURE_TENANT_ID"),
		ClientId:          os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret:      os.Getenv("AZURE_CLIENT_SECRET"),
		SubscriptionId:    os.Getenv("AZURE_SUBSCRIPTION_ID"),
		ResourceGroupName: os.Getenv("AZURE_RESOURCE_GROUP_NAME"),
	}
	zone := os.Getenv("AZURE_DNS_ZONE_FQDN")

	// Invoke authentication and store client to instance
	if err := provider.NewClient(); err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	// List existing records
	fmt.Printf("(1) List existing records\n")
	currentRecords, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	for _, record := range currentRecords {
		fmt.Printf("Exists: %v\n", record)
	}

	// Define test records
	testRecords := []libdns.Record{
		libdns.Record{
			Type:  "A",
			Name:  "record-a." + zone,
			Value: "127.0.0.1",
			TTL:   time.Duration(30) * time.Second,
		},
		libdns.Record{
			Type:  "AAAA",
			Name:  "record-aaaa." + zone,
			Value: "::1",
			TTL:   time.Duration(31) * time.Second,
		},
		libdns.Record{
			Type:  "CAA",
			Name:  "record-caa." + zone,
			Value: "0 issue 'ca" + zone + "'",
			TTL:   time.Duration(32) * time.Second,
		},
		libdns.Record{
			Type:  "CNAME",
			Name:  "record-cname." + zone,
			Value: "www." + zone,
			TTL:   time.Duration(33) * time.Second,
		},
		libdns.Record{
			Type:  "MX",
			Name:  "record-mx." + zone,
			Value: "10 mail." + zone,
			TTL:   time.Duration(34) * time.Second,
		},
		// libdns.Record{
		// 	Type:  "NS",
		// 	Name:  zone,
		// 	Value: "ns1.example.com.",
		// 	TTL:   time.Duration(35) * time.Second,
		// },
		libdns.Record{
			Type:  "PTR",
			Name:  "record-ptr." + zone,
			Value: "hoge." + zone,
			TTL:   time.Duration(36) * time.Second,
		},
		// libdns.Record{
		// 	Type:  "SOA",
		// 	Name:  zone,
		// 	Value: "ns1.example.com. hostmaster." + zone + " 1 7200 900 1209600 86400",
		// 	TTL:   time.Duration(37) * time.Second,
		// },
		libdns.Record{
			Type:  "SRV",
			Name:  "record-srv." + zone,
			Value: "1 10 5269 app." + zone,
			TTL:   time.Duration(38) * time.Second,
		},
		libdns.Record{
			Type:  "TXT",
			Name:  "record-txt." + zone,
			Value: "TEST VALUE",
			TTL:   time.Duration(39) * time.Second,
		}}

	// Create new records
	fmt.Printf("(2) Create new records\n")
	createdRecords, err := provider.AppendRecords(context.TODO(), zone, testRecords)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	for _, record := range createdRecords {
		fmt.Printf("Created: %v\n", record)
	}

	// Update new records
	fmt.Printf("(3) Update newly added records\n")
	updatedRecords, err := provider.SetRecords(context.TODO(), zone, testRecords)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	for _, record := range updatedRecords {
		fmt.Printf("Updated: %v\n", record)
	}

	// Delete new records
	fmt.Printf("(4) Delete newly added records\n")
	deletedRecords, err := provider.DeleteRecords(context.TODO(), zone, testRecords)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	for _, record := range deletedRecords {
		fmt.Printf("Deleted: %v\n", record)
	}

}
