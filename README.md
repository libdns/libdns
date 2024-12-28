DNSExit for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/dnsexit)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for DNSExit, allowing you to manage DNS records.

Configuration
=============

[DNSExit API documentation](https://dnsexit.com/dns/dns-api/) details the process of getting an API key.

To run clone the `.env_template` to a file named `.env` and populate with the API key and zone. Note that setting the environment variable 'LIBDNS_DNSEXIT_DEBUG=TRUE' will output the request body, which includes the API key.

Example
=======

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/libdns/dnsexit"
	"github.com/libdns/libdns"
)

func main() {
	key := os.Getenv("LIBDNS_DNSEXIT_API_KEY")
	if key == "" {
		fmt.Println("LIBDNS_DNSEXIT_API_KEY not set")
		return
	}

	zone := os.Getenv("LIBDNS_DNSEXIT_ZONE")
	if zone == "" {
		fmt.Println("LIBDNS_DNSEXIT_ZONE not set")
		return
	}

	p := &dnsexit.Provider{
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

Caveats
=======

The API does not include a GET method, so fetching records is done via Google DNS. There will be some latency.

If an 'A' and 'AAAA' record have the same name, deleting either of them will remove both records.

If multiple record updates are sent in one request, the API may return a code other than 0, to indicate partial success. This is currently judged as a fail and API error message is returned instead of the successfully amended records. 

MX records have mail-zone and mail-server properties, which do not exist in the LibDNS record type, so updating these has not been fully implemented. 'name' can be used instead to specify the mail server, but there is no way to specify the mail-zone. See https://dnsexit.com/dns/dns-api/#example-update-mx

For [Dynamic DNS](https://dnsexit.com/dns/dns-api/#dynamic-ip-update) DNSExit recommend their dedicated GET endpoint, which can set the domain's IP to the one making the request. That is not implemented in this library.
