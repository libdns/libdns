Domain Chief for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/github.com/libdns/domainchief.svg)](https://pkg.go.dev/github.com/libdns/domainchief)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [Domain Chief](https://domain.chief.app?ref=libdns-domainchief) hosted DNS, allowing you to manage DNS records and solve ACME DNS-01 challenges.

## Configuration

```go
import "github.com/libdns/domainchief"

provider := &domainchief.Provider{
	APIToken: "ctt_...",
}
```

| Field       | JSON        | Description                                                              |
|-------------|-------------|--------------------------------------------------------------------------|
| `APIToken`  | `api_token` | A Domain Chief API token (required).                                     |
| `BaseURL`   | `base_url`  | Overrides the API base URL, defaults to `https://domain.chief.app/api/v1`. |
| `HTTPClient`| -           | Overrides the HTTP client used for API requests.                          |

The zone is the domain name in Domain Chief, for example `example.com.`. The domain must use hosted DNS.

## API token scopes

Create an API token [in your Chief Tools account](https://account.chief.app/team/select/token/create?scopes=domainchief:acme:write&ref=libdns-domainchief) for the team that owns your domains. The scopes it needs depend on how you use the provider:

| Use                                     | Scopes                                              |
|-----------------------------------------|-----------------------------------------------------|
| ACME DNS-01 challenges only             | `domainchief:acme:write`                            |
| Reading and changing any DNS record     | `domainchief:dns:read`, `domainchief:dns:write`     |
| `ListZones`                             | `domainchief:domains:read`                          |

The `domainchief:acme:write` scope only allows publishing and deleting TXT records on `_acme-challenge` names. It cannot read your domains or change any other record, so it is the recommended scope when you only need certificates. Anyone holding the token can still complete DNS-01 challenges and obtain certificates for the domains in your team, so treat it as a secret.

## Behavior

- `AppendRecords` and `DeleteRecords` send TXT records on `_acme-challenge` names holding an ACME challenge value (base64url characters) to the dedicated ACME challenge endpoints. Those keep other values on the same name, so concurrent challenges for the same name (for example `example.com` and `*.example.com`) work. Challenge values that are not deleted are removed automatically after 24 hours. Because the challenge endpoint does not report whether a value existed, `DeleteRecords` always returns challenge records as deleted.
- All other records use the DNS record endpoints.
- Supported record types are `A`, `AAAA`, `ALIAS`, `CAA`, `CNAME`, `MX`, `NS`, `SRV`, `TLSA` and `TXT`. `ALIAS` and `TLSA` records are returned as `libdns.RR`.
- TTLs are rounded to the closest TTL Domain Chief supports (`60`, `300`, `900`, `1800`, `3600`, `21600`, `43200`, `86400`, `172800`, `259200`, `604800`, `1209600` or `2592000` seconds). A TTL of `0` uses `3600` seconds.
- `SetRecords` is not atomic. For every name and type in the input it keeps records with the same value, updates other existing records to hold the remaining values and deletes leftover records. When an error is returned some of the changes may have been applied.
- Requests are retried for a few seconds when another hosted DNS write for the same domain is still running or when the API is rate limiting requests.
- DNS changes are published to the hosted DNS nameservers asynchronously. They are usually visible within a minute.

## Testing

The unit tests run against a local fake API:

```bash
go test ./...
```

The integration test changes records in a real zone. It needs a token with the `domainchief:acme:write`, `domainchief:dns:read` and `domainchief:dns:write` scopes:

```bash
DOMAINCHIEF_API_TOKEN=ctt_... DOMAINCHIEF_TEST_ZONE=example.com. go test -run TestIntegration -v ./...
```

The [shared libdns test suite](libdnstest) runs from the `libdnstest` directory with the same environment variables.
