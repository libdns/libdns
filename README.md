Leaseweb provider for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/leaseweb)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [Leaseweb](https://leaseweb.com/), allowing you to manage DNS records.

## Usage

Generate an API Key via the [Leaseweb customer portal](https://secure.leaseweb.com/); under Administration -> API Key.

Place API Key in the configuration as `APIKey`.

## Compiling

### Docker

Run:

```
docker run --rm -it -v "$PWD":/usr/local/go/src/leaseweb -w /usr/local/go/src/leaseweb golang:1.16
```

which will drop you in an interactive bash prompt where `go` and friends are available.

For example you can build the code with `go build`.

## Example

```go
package main

import (
	"context"
	"fmt"
	"github.com/libdns/leaseweb"
)

func main() {
	provider := leaseweb.Provider{APIKey: "<LEASEWEB API KEY>"}

	records, err  := provider.GetRecords(context.TODO(), "example.com")
	if err != nil {
		fmt.Println(err.Error())
	}

	for _, record := range records {
		fmt.Printf("%s %v %s %s\n", record.Name, record.TTL.Seconds(), record.Type, record.Value)
	}
}
```
