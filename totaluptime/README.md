Total Uptime for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/totaluptime)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for Total Uptime, allowing you to manage DNS records.

## Authenticating

This package supports basic authentication with Total Uptime's API. It's recommended that you create a role and user specific to API use, so that you can ensure least-privilege access to the account. Listed below are the steps to achieve this using Total Uptime's management portal:

* Settings -> Roles & Security -> Role Management -> Add
  * Name: **API_User_Role**
  * DNS: **Enabled**
    * Information: **Read**
    * Domains: **Full**
  * *(all other access set to "Disabled")*

* Settings -> Users -> Add
    * First Name: **API**
    * Last Name: **User**
    * User Name (Email): **apiuser@mydomain.com**  *(can be anything and does not need to receive mail)*
    * Active: **[ X ]** *(checked)*
    * Role: **API_User_Role**
    * API Account: **[ X ]** *(checked)*

## Example

Here's a minimal example of how to list all DNS records using this `libdns` provider (see `examples/main.go` for more examples)

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/libdns/totaluptime"
)

func main() func main() {
	provider := totaluptime.Provider{
		Username: "USERNAME", // provider API username
		Password: "PASSWORD", // provider API password
	}

	zone := "DNS_ZONE" // zone in provider account
	ctx := context.Background()

	result, err := provider.GetRecords(ctx, zone)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(result)
}
```