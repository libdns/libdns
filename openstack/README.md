# OpenStack Designate for `libdns`

[![godoc reference](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/libdns/openstack)


This package implements the libdns interfaces for the [OpenStack Designate API](https://docs.openstack.org/api-ref/dns/) (using the Go implementation from: github.com/gophercloud/gophercloud/openstack)

## Authenticating

To authenticate you need to supply a OpenStack API credentials and zone name on which you want to operate.

## Credentials needed to authenticate to the OpenStack Designate API
```bash
export OS_REGION_NAME=""
export OS_TENANT_ID=""
export OS_IDENTITY_API_VERSION=2
export OS_PASSWORD=""
export OS_AUTH_URL=""
export OS_USERNAME=""
export OS_TENANT_NAME=""
export OS_ENDPOINT_TYPE=""
```
## Example

Here's a minimal example of how to get all your DNS records using this `libdns` provider (see `examples/main.go`)

```go
package main

import (
	"context"
	"fmt"
	"github.com/libdns/libdns"
	"github.com/libdns/openstack/pkg/designate"
	"log"
	"os"
	"time"
)

func main() {
    // specify proper ZONE name.
    // example: bar.example.com.
	zone := os.Getenv("ZONE")
	if zone == "" {
		fmt.Printf("ZONE not set\n")
		return
	}

    // call designate.New with your zone name
	provider, err := designate.New(zone)
	if err != nil {
		log.Fatal(err)
	}

    // GET records
	records, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
	}
	fmt.Println("Records", records)

    // CREATE records
	testName := "foo-libdns."
	add, err := provider.AppendRecords(context.TODO(), zone, []libdns.Record{libdns.Record{
		Type:  "TXT",
		Name:  testName,
		Value: fmt.Sprintf("Replacement test entry created by libdns %s", time.Now()),
		TTL:   time.Duration(600) * time.Second,
	}})

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("ADDED", add)


    // UPDATE records
	testName = "foo-libdns."
	edit, err := provider.SetRecords(context.TODO(), zone, []libdns.Record{libdns.Record{
		Type:  "TXT",
		Name:  testName,
		Value: fmt.Sprintf("SET1 test entry created by libdns %s", time.Now()),
		TTL:   time.Duration(600) * time.Second,
	}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("UPDATED", edit)

    // DELETE records
	testName = "foo-libdns."
	del, err := provider.DeleteRecords(context.TODO(), zone, []libdns.Record{libdns.Record{
		Type:  "TXT",
		Name:  testName,
	}})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("DELETED", del)
}
```
