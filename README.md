Bluecat for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/bluecat)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for Bluecat Address Manager, allowing you to manage DNS records.

## Configuration

To use this provider, you need:
- Bluecat Address Manager API endpoint (URL)
- Username
- Password

## Example Usage

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/libdns/bluecat"
	"github.com/libdns/libdns"
)

func main() {
	provider := &bluecat.Provider{
		ServerURL: "https://bluecat.example.com",
		Username:  "api_user",
		Password:  "api_password",
		// Optional: specify configuration and view names
		// ConfigurationName: "config",
		// ViewName: "view",
	}

	zone := "example.com."
	ctx := context.Background()

	// Get existing records
	records, err := provider.GetRecords(ctx, zone)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Found %d records\n", len(records))

	// Append a new A record
	newRecords := []libdns.Record{
		libdns.Address{
			Name: "test",
			TTL:  300 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.1"),
		},
	}
	
	created, err := provider.AppendRecords(ctx, zone, newRecords)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created %d records\n", len(created))
}
```

## Caveats

- The provider requires Bluecat Address Manager 9.5.0 or later with the RESTful v2 API enabled
- Authentication sessions are automatically managed and tokens are cached for efficiency
- The provider supports A, AAAA, CNAME, TXT, MX, NS, and SRV record types
- Record names should be relative to the zone (e.g., "www" for "www.example.com" in zone "example.com.")
