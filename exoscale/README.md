Exoscale for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/exoscale)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [Exoscale](https://www.exoscale.com/), allowing you to manage DNS records.

# Configuration

---

This provider expects the following configuration:

## **Required**

- `APIKey`: Exoscale API Key
- `APISecret`: Exoscale API Secret

Here is an example of the minimum permissions you can set for the role associated to your API Key.

```json
{
  "default-service-strategy": "deny",
  "services": {
    "dns": {
      "type": "rules",
      "rules": [
        {
          "expression": "operation == 'list-dns-domains'",
          "action": "allow"
        },
        {
          "expression": "!(resources.dns_domain.unicode_name in [\"example.com\"])",
          "action": "deny"
        },
        {
          "expression": "true",
          "action": "allow"
        }
      ]
    }
  }
}
```

# Testing

---

For testing, set the `TEST_API_KEY`, `TEST_API_SECRET` and `TEST_ZONE` as environment variable.

```bash
$ TEST_API_KEY="EXO..." TEST_API_SECRET="..." TEST_ZONE="example.com" go test -v
=== RUN   Test_AppendRecords
--- PASS: Test_AppendRecords (13.89s)
=== RUN   Test_DeleteRecords
--- PASS: Test_DeleteRecords (7.37s)
=== RUN   Test_GetRecords
--- PASS: Test_GetRecords (7.89s)
=== RUN   Test_SetRecords
--- PASS: Test_SetRecords (14.23s)
PASS
ok  	github.com/libdns/exoscale	43.393s
```
