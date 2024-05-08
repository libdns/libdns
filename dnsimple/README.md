# dnsimple for [`libdns`](https://github.com/libdns/libdns)

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/dnsimple)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [dnsimple](https://dnsimple.com), allowing you to manage DNS records.

## Configuration

This provider expects the following configuration:

- `API_ACCESS_TOKEN`: an API key to authenticate calls to the provider, see https://support.dnsimple.com/articles/api-access-token/
- `ACCOUNT_ID` _(optional)_: identifier for the account (only needed if using a user access token), see https://developer.dnsimple.com/v2/accounts/
- `API_URL` _(optional)_: hostname for the API to use (defaults to `api.dnsimple.com`), see https://developer.dnsimple.com/sandbox/
