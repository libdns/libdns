INWX for [`libdns`](https://github.com/libdns/libdns)
=====================================================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/inwx)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [INWX](https://www.inwx.de/en), allowing you to manage DNS records.


Authenticating
==============

To authenticate you need to supply following credentials:

  * Your INWX username
  * Your INWX password
  * A shared secret if you have enabled two-factor authentication


Example
=======

```go
package main

import (
    "context"
    "fmt"

    "github.com/libdns/inwx"
)

func main() {
    provider := &inwx.Provider{
        User: "<username>",
        Pass: "<password>",
        SharedSecret: "<sharedSecret>",
        Test: true,
    }
    zone := "example.com."

    records, err := provider.GetRecords(context.TODO(), zone)

    if err != nil {
        fmt.Printf("Error: %s", err.Error())
        return
    }

    fmt.Println(records)
}

```