infomaniak for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/infomaniak)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [infomaniak](https://infomaniak), allowing you to manage DNS records.

## Code example
```go
import "github.com/libdns/infomaniak"
provider := &infomaniak.Provider{
    APIToken:  "YOUR_API_TOKEN"
}
```

## Create Your API Token
Please login to your infomaniak account and then navigate [here](https://manager.infomaniak.com/v3/infomaniak-api) to issue your API access token. The scope of your token has to include "domain".


> :warning: The API for domains is currently not listed in the [infomaniak API reference](https://developer.infomaniak.com/docs/api). Their support told me that the API for domains is "not yet mature". The API calls in this module were implemented based on the [go-acme/lego infomaniak provider](https://github.com/go-acme/lego/tree/master/providers/dns/infomaniak) and the [acmesh-official/acme.sh infomaniak provider](https://github.com/acmesh-official/acme.sh/blob/master/dnsapi/dns_infomaniak.sh). The API could be subject to changes.