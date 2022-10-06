njalla for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/njalla)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for njalla, allowing you to manage DNS records.

Example
======================
```golang
package main

import (
	"context"
	"fmt"

    "github.com/libdns/libdns/njalla"
)

func main() {
	p := Provider{
		APIToken: "TOKEN",
	}

	records, err := p.GetRecords(context.Background(), "domain.tld")
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		return
	}

	fmt.Println(records)
}
```

