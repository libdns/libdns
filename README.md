NETLIFY for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/CL0Pinette/libdns-netlify)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for Netlify, allowing you to manage DNS records.

## Example
* Create a `.env` file in the `_example/` directory with the following inside:
```
NETLIFY_TOKEN=YOUR_NETLIFY_TOKEN
ZONE=YOUR_ZONE_NAME
```

* Then run `go run main.go`
