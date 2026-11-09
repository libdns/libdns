# Binary Lane for `libdns`

[![godoc reference](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/libdns/binarylane)

This package implements the [libdns](https://github.com/libdns/libdns) interfaces
for [Binary Lane](https://www.binarylane.com.au), an Australian cloud provider
whose DNS is managed through the Domains endpoints of its
[public API](https://api.binarylane.com.au/reference/).

It has no dependencies beyond `libdns` itself.

## Authenticating

Create an API token in the Binary Lane control panel under **Account → API**,
and put it in `APIToken`.

## Example

```go
provider := binarylane.Provider{
	APIToken: os.Getenv("BINARYLANE_API_TOKEN"),
}

records, err := provider.GetRecords(context.TODO(), "example.com.")
```

## Configuration

| Field | JSON | Purpose |
| --- | --- | --- |
| `APIToken` | `api_token` | API token. Required. |
| `APIBase` | `api_base` | Overrides the API endpoint, including the version segment. A trailing slash is added if absent. Defaults to `DefaultAPIBase` (`https://api.binarylane.com.au/v2/`). |
| `HTTPClient` | — | Supplies your own `*http.Client` for a custom transport, proxy or timeout. One with a 30s timeout is created on first use when nil. |

No environment variables are read.

## Supported record types

`A`, `AAAA`, `CAA`, `CNAME`, `MX`, `NS`, `SRV` and `TXT`.

Binary Lane keeps part of the RDATA of some types in dedicated API fields
(`priority`, `port`, `weight`, `flags`, `tag`) while libdns expects `Data` to
hold complete zone-file RDATA, so this package composes it on read and splits
it again on write. `TestRoundTrip` pins that behaviour for every affected type.

Binary Lane also serves `SOA`, which libdns does not model. Those records are
returned as an opaque `libdns.RR` rather than being dropped from a listing, and
are not writable.

Binary Lane does not support `SVCB` or `HTTPS` records.

## TTL is fixed at 3600

Binary Lane enforces a single TTL for the whole zone and rejects any other
value with `HTTP 400: The default TTL of 3600 may not be changed.`

This package therefore **never sends a TTL**, letting the API apply its own.
A record's `TTL` field is ignored on write, and reads always report 3600.

This is deliberate rather than an oversight. ACME clients set a short TTL on
`_acme-challenge` records, so forwarding the requested value would fail every
DNS-01 challenge. libdns anticipates exactly this: "some providers may reject
or silently increase TTLs that are below a certain threshold ... it is
recommended to not rely on the exact TTL value."

One consequence worth knowing: `DeleteRecords` matches on TTL when a non-zero
one is given, so a record you constructed with a short TTL will not match the
3600 the zone actually holds. Leave `TTL` zero to match regardless, or delete
records obtained from `GetRecords`.

## Behaviour notes

- **Record identity.** `GetRecords` stores the provider's record ID in each
  record's `ProviderData` field. `DeleteRecords` uses it when present and falls
  back to matching by value otherwise, following the `libdns.RecordDeleter`
  rules: an empty `Type`, zero `TTL` or empty `Data` act as wildcards, and
  `Name` is always significant.
- **Atomicity.** Binary Lane has no batch endpoint, so `AppendRecords`,
  `SetRecords` and `DeleteRecords` are **not** atomic: if a call fails partway,
  earlier changes remain applied.
- **`SetRecords`** reuses existing records within an RRset where it can, so a
  record whose value is unchanged keeps its ID.
- **Concurrency.** All exported methods are safe for concurrent use.

## Testing

`go test ./...` runs offline. The API layer is covered against an
`httptest` server — authentication, pagination, request paths and methods,
zero-valued RDATA fields, and error decoding — and the conversion layer is
covered directly.

The live test runs the full record lifecycle against a real zone -- ListZones,
AppendRecords, SetRecords growing and shrinking an RRset, and DeleteRecords. It
is skipped unless both variables are set, and only touches records under a
unique `libdns-test-<timestamp>` label, which it removes afterwards:

```
BINARYLANE_API_TOKEN=... BINARYLANE_TEST_ZONE=example.com go test -run TestLive -v
```

`go test -race ./...` covers concurrent use of every interface method.

`libdns`'s own `libdnstest` conformance suite is not wired in yet: it is
currently only on `libdns` master, and pinning an unreleased commit would put
this package ahead of the released API it targets.
