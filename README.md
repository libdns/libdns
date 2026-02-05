Arvancloud for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/TODO:PROVIDER_NAME)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for Arvancloud, allowing you to manage DNS records.

## Authenticating
This package uses the Apikey authentication method.

The provided apikey must have the `DNS administrator` and `DOMAIN administrator` permissions.

![permissions](https://github.com/user-attachments/assets/54d30f0f-72ac-4de6-8718-95c35f1486ec)

## Example Configuration

```golang
p := arvancloud.Provider{
    AuthAPIKey: "apikey",
}
```
