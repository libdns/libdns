EasyDNS for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/easydns)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [EasyDNS](https://easydns.com/), allowing you to manage DNS records.

To manage and create token and key information for your account, navigate to https://cp.easydns.com/manage/security/.

Example usage can be found in the `_example` directory in this repository.

Constraints:
- Minimum TTL allowed 5 mins (300 seconds), a TTL less than 300 will be set to 300
