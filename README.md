Neoserv for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/neoserv)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [Neoserv](https://moj.neoserv.si), allowing you to manage DNS records.

## Installation

```bash
go get github.com/libdns/neoserv
```

## Usage

You can check out a minimal example of using this provider in the [examples](./examples) directory.

Run it with:

```bash
NEOSERV_USERNAME=your@email.com NEOSERV_PASSWORD=your_password NEOSERV_ZONE=your.domain go run ./examples/neoserv.go
```

## Supported TTL Values

Neoserv only supports specific TTL values. The following are the supported TTL values:

- 1 minute
- 5 minutes
- 15 minutes
- 30 minutes
- 1 hour
- 6 hours
- 12 hours
- 24 hours (1 day)
- 2 days
- 7 days
- 14 days
- 30 days

By default, if an unsupported TTL is provided, the provider will use the closest supported value that is greater than or equal to the provided value. If you want to treat unsupported TTL values as errors, set `UnsupportedTTLisError` to `true` when creating the provider:

```go
provider := neoserv.Provider{
	Username:              "your-neoserv-email",
	Password:              "your-neoserv-password",
	UnsupportedTTLisError: true,
}
```

