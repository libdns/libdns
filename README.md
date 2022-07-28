# OVH DNS for `libdns`

[![godoc reference](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/libdns/ovh)


This package implements the libdns interfaces for the [OVH DNS API](https://docs.ovh.com/gb/en/api/first-steps-with-ovh-api/) using the [OVH API GO SDK](https://github.com/ovh/go-ovh)

## Authenticating

To authenticate you need to create [script credentials](https://github.com/ovh/go-ovh#supported-apis) in your region account and specify these API rights :

For multiple domains :

```
GET /domain/zone/*/record
POST /domain/zone/*/record
GET /domain/zone/*/record/*
PUT /domain/zone/*/record/*
DELETE /domain/zone/*/record/*
GET /domain/zone/*/soa
POST /domain/zone/*/refresh
```

For a single domain or delegation :

```
GET /domain/zone/yourdomain.com/record
POST /domain/zone/yourdomain.com/record
GET /domain/zone/yourdomain.com/record/*
PUT /domain/zone/yourdomain.com/record/*
DELETE /domain/zone/yourdomain.com/record/*
GET /domain/zone/yourdomain.com/soa
POST /domain/zone/yourdomain.com/refresh
```

## Example

Here's a minimal example of how to get all DNS records for zone. See also: [provider_test.go](https://github.com/libdns/ovh/blob/master/provider_test.go)

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/libdns/ovh"
)

func main() {
	endPoint := os.Getenv("LIBDNS_OVH_TEST_ENDPOINT")
	if endPoint == "" {
		fmt.Printf("LIBDNS_OVH_TEST_ENDPOINT not set\n")
		return
	}

	applicationKey := os.Getenv("LIBDNS_OVH_TEST_APPLICATION_KEY")
	if applicationKey == "" {
		fmt.Printf("LIBDNS_OVH_TEST_APPLICATION_KEY not set\n")
		return
	}

	applicationSecret := os.Getenv("LIBDNS_OVH_TEST_APPLICATION_SECRET")
	if applicationSecret == "" {
		fmt.Printf("LIBDNS_OVH_TEST_APPLICATION_SECRET not set\n")
		return
	}

	consumerKey := os.Getenv("LIBDNS_OVH_TEST_CONSUMER_KEY")
	if consumerKey == "" {
		fmt.Printf("LIBDNS_OVH_TEST_CONSUMER_KEY not set\n")
		return
	}

	zone := os.Getenv("LIBDNS_OVH_TEST_ZONE")
	if zone == "" {
		fmt.Printf("LIBDNS_OVH_TEST_ZONE not set\n")
		return
	}

	p := &ovh.Provider{
		Endpoint: endPoint,
		ApplicationKey: applicationKey,
		ApplicationSecret: applicationSecret,
		ConsumerKey: consumerKey,
	}

	records, err := p.GetRecords(context.TODO(), zone)
	if err != nil {
        fmt.Printf("Error: %s", err.Error())
        return
	}

	fmt.Println(records)
}

```