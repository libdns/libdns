deSEC for [`libdns`](https://github.com/libdns/libdns)
======================================================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/desec)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for
[deSEC](https://desec.io).

## Authentication

Authentication performed using a token created on https://desec.io/tokens. A basic token without the
permission to manage tokens is sufficient. For security reasons it's strongly recommended to not use
tokens that allow token management.

## Limitations and Caveats

* Concurrent updates from multiple processes can result in an inconsistent state
* The TTL attribute is not settable per record (zone, name, type, value), only per record set (zone,
  name, type). If different TTL values are specified, it's undefined which one wins.
* Large zones with more than 500 resource record sets only have limited support
* Rate limiting always results in retries if no context deadline is specified

Please refer to the [Go reference](https://pkg.go.dev/github.com/libdns/desec) for
detailed documentation.