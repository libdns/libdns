# [Domainname.shop](https://domene.shop) DNS for [`libdns`](https://github.com/libdns/libdns)

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/domainnameshop)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [Domainname.shop](https://domene.shop) DNS.

[Domainname.shop API reference](https://api.domeneshop.no/docs/)

> Side note: This provider has ended up with a few different names so just a bit of clarification.  
> The company name is Domeneshop AS (Norwegian company).  
> The main name in Norwegian is "Domeneshop" while in most English context they go by "Domainnameshop".  
> Some known URLs are ``domeneshop.no``, ``domene.shop``, ``domainnameshop.com``, ``domainname.shop``.  
> Package has used the name ``domainnameshop`` as reference.


## Authentication
You will need a API token and API secret to use this module.  
You can get tokens and secrets from the admin panel here:  
https://domene.shop/admin?view=api

## Example
This is a minimal example, but you can also check [provider_test.go](provider_test.go) for more usage. 

````go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/libdns/domainnameshop"
)

func main() {

	token := os.Getenv("LIBDNS_DOMAINNAMESHOP_TOKEN")
	if token == "" {
		fmt.Printf("LIBDNS_DOMAINNAMESHOP_TEST_TOKEN not set\n")
		return
	}

	secret := os.Getenv("LIBDNS_DOMAINNAMESHOP_SECRET")
	if secret == "" {
		fmt.Printf("LIBDNS_DOMAINNAMESHOP_SECRET not set\n")
		return
	}

	zone := os.Getenv("LIBDNS_DOMAINNAMESHOP_ZONE")
	if zone == "" {
		fmt.Printf("LIBDNS_DOMAINNAMESHOP_ZONE not set\n")
		return
	}

	p := &domainnameshop.Provider{
		APIToken:  token,
		APISecret: secret,
	}

	ctx, ctxcancel := context.WithTimeout(context.Background(), time.Duration(15*time.Second))
	records, err := p.GetRecords(ctx, zone)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		return
	}
	_ = ctxcancel

	fmt.Println(records)
}
````

