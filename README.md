DNS Made Easy for [libdns](https://github.com/libdns/libdns)
=======================
[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/john-k/libdns-dnsmadeeasy
)

A [DNS Made Easy APIv2](https://api-docs.dnsmadeeasy.com) client for [libdns](https://github.com/libdns/libdns) using the [dnsmadeeasy module](https://github.com/john-k/dnsmadeeasy)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [DNS Made Easy](https://dnsmadeeasy.com), allowing you to manage DNS records.

# Configuration
This provider expects the following configuration:
 * APIKey - a DNS Made Easy API Key (from Account Information page)
 * SecretKey - a DNS Made Easy Secret Key (from Account Information page)
 * BaseURL - one of dnsmadeeasy.Sandbox or dnsmadeeasy.Prod

# Notes
This project was authored to support the needs of [Caddy Server](https://caddyserver.com/)
