Joohoi's ACME-DNS for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/TODO:joohoi_acme_dns)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for Joohoi's ACME-DNS.

Since ACME-DNS is a simplified DNS server that only allows setting TXT values,
this `libdns` provider only implements `RecordAppender` and `RecordDeleter` interfaces.
Moreover, DNS zone and record name is ignored by `AppendRecords` method and `DeleteRecords`
always returns an error. This is because ACME-DNS is only meant to be used for ACME DNS
challenges, usually in this way:

1. You would register an account at an ACME-DNS server instance
   (for example, public instance [https://auth.acme-dns.io](https://auth.acme-dns.io)
   or a self-hosted version) by using register endpoint:

   `curl -X POST https://auth.acme-dns.io`

2. This would return a JSON with your account credentials: username, password, subdomain, full domain
  managed by ACME-DNS. Then you would create a CNAME record pointing
  from `_acme-challenge.<YOUR_DOMAIN>` to
  the domain provided by ACME-DNS.

3. After these steps, you can use the credentials generated at step 1 to
  use ACME-DNS `libdns` provider to create DNS challenge records necessary
  for getting a certificate for `<YOUR DOMAIN>` (usually done by, e.g. `certmagic`).

For more information about Joohoi's ACME-DNS and the motivation for it, see:

* https://github.com/joohoi/acme-dns
* https://www.eff.org/deeplinks/2018/02/technical-deep-dive-securing-automation-acme-dns-challenge-validation