Metaname provider for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/metaname)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for Metaname, allowing you to manage DNS records.

Create a provider with:

    provider := metaname.Provider{APIKey: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
        AccountReference: "xxxx"}
(use Endpoint: "https://test.metaname.net/api/1.1" for testing)

From there, the four standard methods work. Updating and deleting with a record reference ID retrieved from GetRecords or from a
record in the array returned by AppendRecords and SetRecords works, while "guesswork matching" with only record data works in some
cases.

There are three main limitations in the provider currently:

* Guesswork matching requires a complete match for deletion, or a matching A/AAAA/CNAME type and name for setting.
* Does not currently support priorities, as Metaname treats these as separate fields not represented in the libdns Record type.
* Metaname fails with no message for certain erroneous configurations (e.g. additional record with existing CNAME at same name),
  and these are reported only with Metaname's "Internal error" code.