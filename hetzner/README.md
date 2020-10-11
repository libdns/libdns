# Hetzner DNS for `libdns`

[![godoc reference](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/libdns/hetzner)


This package implements the libdns interfaces for the [Hetzner DNS API](https://dns.hetzner.com/api-docs)

## Authenticating

To authenticate you need to supply a Hetzner [Auth-API-Token](https://dns.hetzner.com/api-docs#section/Authentication/Auth-API-Token).

## Example

Here's a minimal example of how to get all DNS records for zone. See also: [provider_test.go](https://github.com/libdns/hetzner/blob/master/provider_test.go)

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/libdns/libdns/hetzner"
)

func main() {
	token := os.Getenv("LIBDNS_HETZNER_TOKEN")
	if token == "" {
		fmt.Printf("LIBDNS_HETZNER_TOKEN not set\n")
		return
	}

	zone := os.Getenv("LIBDNS_HETZNER_ZONE")
	if token == "" {
		fmt.Printf("LIBDNS_HETZNER_ZONE not set\n")
		return
	}

	p := &hetzner.Provider{
		AuthAPIToken: token,
	}

	records, err := p.GetRecords(context.WithTimeout(context.Background(), time.Duration(15*time.Second)), zone)
	if err != nil {
        fmt.Printf("Error: %s", err.Error())
        return
	}

	fmt.Println(records)
}

```

