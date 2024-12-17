# Katapult for [`libdns`](https://github.com/libdns/libdns)

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/katapult)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [Katapult](https://katapult.io), allowing you to manage DNS records.

## Authentication

To use this package you will need a Katapult API token, see the [Katapult docs](https://docs.katapult.io/docs/dev/key-concepts/authentication) for more information.

## Example

See the `example/main.go` file for a demonstration on how to use the package. The example can be ran by specifying the API token and zone:

```
LIBDNS_KATAPULT_API_TOKEN=your-api-token LIBDNS_KATAPULT_ZONE=your-domain go run ./example
```

The example can be modified to try different functions available on the provider.

## Tests

The tests use the Katapult API, therefore you should create and use a new and empty domain/zone.

You can then set the API token and zone when running the tests:

```
LIBDNS_KATAPULT_API_TOKEN=your-api-token LIBDNS_KATAPULT_ZONE=your-domain go test
```
