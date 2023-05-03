**DEVELOPER INSTRUCTIONS:**

This repo is a template for developers to use when creating new [libdns](https://github.com/libdns/libdns) provider implementations.

Be sure to update:

- The package name
- The Go module name in go.mod
- The latest `libdns/libdns` version in go.mod
- All comments and documentation, including README below and godocs
- License (must be compatible with Apache/MIT)
- All "TODO:"s is in the code
- All methods that currently do nothing

Remove this section from the readme before publishing.

---

DirectAdmin for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/TODO:PROVIDER_NAME)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for DirectAdmin, allowing you to manage DNS records.

TODO: Show how to configure and use. Explain any caveats.


## Authenticating

This package supports API **[Login Keys](https://docs.directadmin.com/directadmin/customizing-workflow/api-all-about.html#creating-a-login-key)** for authentication.

You will need to create a login key with the following settings:

- **Key Type** 
  - Key
- **Key Name**
  - Select Descriptive Name
- **Expires On**
  - You can set this to whatever you need, but "Never" is likely the best option`
- **Clear Key**
  - Unchecked
- **Allow HTM**
  - Unchecked
- **Commands**
  - `CMD_API_SHOW_DOMAINS`
  - `CMD_API_DNS_CONTROL`

The `CMD_API_SHOW_DOMAINS` permission is needed to get the zone ID, the `CMD_API_DNS_CONTROL` permission is obviously necessary to edit the DNS records.

If you're only using the `GetRecords()` method, you can remove the `CMD_API_DNS_CONTROL` permission to guarantee no changes will be made.