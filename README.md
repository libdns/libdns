name.com for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/namedotcom)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for name.com, allowing you to manage DNS records.

## Authenticating

To authenticate you need to supply the following arguments to the provider: 
1. name.com **user name**.
2. name.com **api token**.
3. name.com **api url** ( e.g. https://api.name.com)

## Example

Here's a minimal example illustrating common use cases.

```go
package main

import (
	"context"
	"github.com/libdns/libdns"
	"github.com/libdns/namedotcom"
	"log"
)

func main() {
	ctx := context.TODO()

	zone := "example.com."

	// configure the name.com DNS provider 
	provider := namedotcom.Provider{
		APIToken : "super_secret_token",
		User :     "user",
		APIUrl : "https://api.name.com",
	}

	// list records
	recs, err := provider.GetRecords(ctx, zone)
	if err != nil {
		log.Fatal(err)
	}
	for _, rec := range recs {
		log.Println(rec)
	}

	// create records (AppendRecords is similar)
	newRecs, err = provider.SetRecords(ctx, zone, []libdns.Record{
		Type:  "A",
		Name:  "sub",
		Value: "1.2.3.4",
	})

	// delete records (DeleteRecords() will attempt to find the record ID if not specified)
	deletedRecs, err = provider.DeleteRecords(ctx, zone, []libdns.Record{
		Type:  "A",
		Name:  "sub",
		Value: "1.2.3.4",
	})

}
```
