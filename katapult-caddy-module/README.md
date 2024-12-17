# Katapult module for Caddy

This package contains a DNS provider module for [Caddy](https://github.com/caddyserver/caddy). It can be used to manage DNS records with [Katapult](https://katapult.io).

## Caddy module name

```
dns.providers.katapult
```

## Config examples

To use this module for the ACME DNS challenge, [configure the ACME issuer in your Caddy JSON](https://caddyserver.com/docs/json/apps/tls/automation/policies/issuer/acme/) like so:

```json
{
  "module": "acme",
  "challenges": {
    "dns": {
      "provider": {
        "name": "katapult",
        "api_token": "YOUR_KATAPULT_API_TOKEN"
      }
    }
  }
}
```

or with the Caddyfile:

```
# globally
{
	acme_dns katapult ...
}
```

```
# one site
tls {
	dns katapult ...
}
```
