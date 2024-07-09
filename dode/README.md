[do.de][dode] for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/dode)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [do.de][dode], allowing you to manage DNS records.

The [do.de][dode] API only supports creating and removing TXT records for domains starting with `_acme-challenge.`. This means that this package can only be used when issuing a TLS certificate using the [DNS-01 Challenge](https://letsencrypt.org/docs/challenge-types/#dns-01-challenge).

To authenticate with the [do.de][dode] API, you need an API token. You can retrieve by logging into your [do.de][dode] account and navigating to `Domains > Einstellungen >  Let's Encrypt API-Token`.

[dode]: https://do.de
