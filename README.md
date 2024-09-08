# Selectel DNS v2 for [libdns](https://github.com/libdns/libdns)

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/selectelv2)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [Selectel DNS v2 API](https://developers.selectel.ru/docs/cloud-services/dns_api/dns_api_actual/), allowing you to manage DNS records.

## Authorize

To authorize you need to use Selectel [Authorization](https://developers.selectel.ru/docs/control-panel/authorization/#%D1%82%D0%BE%D0%BA%D0%B5%D0%BD-keystone).

## Example

Minimal working example of getting DNS zone records.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/libdns/libdns/selectelv2"
)

func main() {

	provider = selectelv2.Provider{
		User:        os.Getenv("SELECTEL_USER"),
		Password:    os.Getenv("SELECTEL_PASSWORD"),
		AccountId:   os.Getenv("SELECTEL_ACCOUNT_ID"),
		ProjectName: os.Getenv("SELECTEL_PROJECT_NAME"),
		ZonesCache:  make(map[string]string),
	}
	zone = os.Getenv("SELECTEL_ZONE")
	ctx = context.Background()

	records, err := provider.GetRecords(ctx, zone)
	if err != nil {
        fmt.Printf("Error: %s", err)
        return
	}

	fmt.Println(records)
}

```

See also: [provider_test.go](https://github.com/libdns/selectelv2/blob/master/provider_test.go)

Always yours [@jjazzme](https://github.com/jjazzme)
