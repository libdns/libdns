# libdns-ednsde

[![Go Reference](https://pkg.go.dev/badge/github.com/libdns/ednsde.svg)](https://pkg.go.dev/github.com/libdns/ednsde)

A [libdns](https://github.com/libdns/libdns) provider for the **eDNS DNS-01
challenge API** (`https://dns-challenge.edns.de`) offered by [edns.de](https://edns.de).

It exists so that [Caddy](https://caddyserver.com), [CertMagic](https://github.com/caddyserver/certmagic)
or any other libdns consumer can solve ACME **DNS-01** challenges for zones
hosted at eDNS — including wildcard certificates. For Caddy, use the module
wrapper: [caddy-dns-ednsde](https://github.com/chris299/caddy-dns-ednsde).

## Scope: challenge records only

The eDNS challenge API is purpose-built for ACME. It can add and remove TXT
challenge records, and it can do nothing else — in particular it has **no
endpoint that lists the records of a zone**.

This provider therefore implements only two libdns interfaces:

| Interface | Implemented | Why |
| --- | --- | --- |
| `libdns.RecordAppender` | yes | `addChallengeRecord` |
| `libdns.RecordDeleter` | yes | `removeChallengeRecord` |
| `libdns.RecordGetter` | **no** | the API cannot list records |
| `libdns.RecordSetter` | **no** | cannot be implemented without reading first |

The two unsupported methods are *absent* rather than present-and-failing. Go
interfaces are structural, so a stub returning "not supported" would still
satisfy `libdns.RecordGetter` and move the failure from compile time into the
middle of a certificate request. `certmagic.DNSProvider` requires only the two
interfaces above, so nothing is missing for Caddy.

This is not a general-purpose DNS management library. It cannot create A
records, it cannot read your zone, and it will refuse anything that is not TXT.

## Usage

```go
import (
    "context"

    "github.com/libdns/libdns"
    ednsde "github.com/libdns/ednsde"
)

provider := &ednsde.Provider{APIToken: os.Getenv("EDNS_TOKEN")}

added, err := provider.AppendRecords(context.Background(), "example.com.", []libdns.Record{
    libdns.TXT{Name: "_acme-challenge", Text: "<43-character ACME digest>"},
})
```

## Getting an access token

1. In the eDNS web interface, go to **SSL-Zertifikate → Automation-API-Verwaltung
   → API-Zugang anlegen** and create a token.
2. Open the zone you want to use it for and select that token on the zone's
   **DNS-01-Challenge** tab.

Step 2 is easy to miss. Without it every request for that zone is answered with
`401`, with the *same* message as an entirely invalid token — the API does not
distinguish the two cases. The error returned by this package says so.

One token can be assigned to several zones.

## Behaviour worth knowing

**Names are passed through verbatim.** The API does not add an `_acme-challenge`
prefix of its own; the libdns record name becomes the API's `subdomain`
parameter unchanged. A record on the zone apex (libdns name `@`) is sent with
the `subdomain` field *omitted* — sending it as an empty string is answered with
`400`.

**TTL is not configurable.** eDNS fixes challenge records at 300 seconds. The
TTL of an input record is ignored, and the returned records report 300s, which
is what is actually in the zone.

**Challenge values must be 10–64 characters without whitespace.** This is
validated before the request goes out, so you get a useful message instead of a
`400`. An ACME key authorization digest is 43 characters and always fits.

**Several values may share one name.** This is what a SAN certificate covering
both `example.com` and `*.example.com` needs, and it works.

**Deleting only affects records this API created.** Records added by hand in the
web interface are reported as not found. Deleting something that is not there is
not an error, but it is not reported as deleted either.

**Propagation is not instant, and removal is much slower than creation.**
Measured against a live zone (`winkler.tel`, nameservers `ns3`/`ns4.edns.de`):

| Operation | Visible on all authoritative nameservers after |
| --- | --- |
| add, on a name that did not exist before | ~23 s |
| add, on a name that already has records | immediately |
| remove | ~307 s |

That removal figure is almost exactly the 300 s record TTL, which suggests eDNS
serves these answers from a cache that expires rather than pushing the change.
It matters less than it looks: ACME validation succeeds as long as *some* TXT
record at the name carries the expected value, so a lingering record from a
previous run does no harm.

**Your zone's SOA minimum matters more than any of these numbers.** A challenge
name does not exist before the first issuance, so the first lookup of it returns
NXDOMAIN — and that negative answer is cached for the SOA minimum. Until it
expires, an ACME client's propagation check cannot find the name's authoritative
servers and will report "not ready", however promptly this package did its job.
Check it, and keep it low:

```sh
dig +short SOA example.com | awk '{print "negative TTL:", $NF}'
```

At 86400 a first issuance can stall for a day. At 300 it costs at most five
minutes, once.

**Retries.** Connection errors, `429` and `5xx` are retried up to three times
with a short backoff, as libdns expects. `4xx` responses are returned
immediately; they will not succeed on a retry.

## Testing

Unit tests run everywhere and need no credentials:

```sh
go test ./...
```

Integration tests talk to the real API and write into a real zone. They clean up
after themselves, including when an assertion fails:

```sh
EDNS_TOKEN=... EDNS_TEST_ZONE=example.com go test -tags integration -v ./...
```

## Licence

MIT
