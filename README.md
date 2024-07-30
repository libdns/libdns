Hurricane Electric for [`libdns`](https://github.com/libdns/libdns)
========================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/he)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for Hurricane Electric,
allowing you to manage DNS records.

This package uses the dynamic DNS feature of Hurricane Electric Hosted DNS.

Configuration
=============

To configure a dynamic DNS record in HE, login to the [HE DNS portal](https://dns.he.net/)
and select the domain to configure the record under.

Add a new A/AAAA/TXT record and select the "Enable entry for dynamic dns" option.

Once the record has been created click the Generate a DDNS key 🗘 button and set a key.

Example
=======

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/libdns/he"
	"github.com/libdns/libdns"
)

func main() {
	key := os.Getenv("LIBDNS_HE_KEY")
	if key == "" {
		fmt.Println("LIBDNS_HE_KEY not set")
		return
	}

	zone := os.Getenv("LIBDNS_HE_ZONE")
	if zone == "" {
		fmt.Println("LIBDNS_HE_ZONE not set")
		return
	}

	p := &he.Provider{
		APIKey: key,
	}

	records := []libdns.Record{
		{
			Type:  "A",
			Name:  "test",
			Value: "198.51.100.1",
		},
		{
			Type:  "AAAA",
			Name:  "test",
			Value: "2001:0db8::1",
		},
		{
			Type:  "TXT",
			Name:  "test",
			Value: "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		},
	}

	ctx := context.Background()
	_, err := p.SetRecords(ctx, zone, records)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return
	}
}
```
