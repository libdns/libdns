

libdns-loopia for [`libdns`](https://github.com/libdns/libdns)
=======================
[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/loopia)
[![Go](https://github.com/libdns/loopia/actions/workflows/go.yml/badge.svg)](https://github.com/libdns/loopia/actions/workflows/go.yml)

[![Loopia API](https://static.loopia.se/loopiaweb/images/logos/loopia-api-logo.png)](https://www.loopia.se/api/)




This package implements the [libdns interfaces](https://github.com/libdns/libdns) for Loopia, allowing you to manage DNS records.

# Usage
```golang
include (
    loopia "github.com/libdns/loopia"
)
p := &loopia.Provider{
    Username: "youruser@loopiaapi",
    Password: "yourpassword",
}

zone := "example.org"
records, err := p.GetRecords(ctx, zone)
```
For more details check the `_examples` folder in the source.

## Noteworthy
If you are adding or chainging records, like acme/letsencrypt validation, Loopia is somewhat slow to propagate the result.
It might take __up to 15 minutes__. That said, I have seen it come throug in as little as 1,5 minutes.
