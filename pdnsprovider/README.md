powerdns provider for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/TODO:PROVIDER_NAME)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for 
[PowerDNS](https://powerdns.com/), allowing you to 
manage 
DNS records.

To configure this, simply specify the server URL and the access token. 


    package main
    
    import (
    "context"
    
        "github.com/libdns/libdns"
    
        "github.com/nathanejohnson/pdnsprovider"
    )
    
    func main() {
        p := &pdnsprovider.Provider{
            ServerURL: "http://localhost", // required
            ServerID:  "localhost",        // if left empty, defaults to localhost.
            APIToken:  "asdfasdfasdf",     // required
        }
    
        _, err := p.AppendRecords(context.Background(), "example.org.", []libdns.Record{
            {
                Name: "_acme_whatever",
                Type: "TXT",
                Value: "123456",
            },
        })
        if err != nil {
            panic(err)
        }
    
    }
