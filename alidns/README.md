# AliDNS for `libdns`

This package implements the libdns interfaces for the [AliDNS API](https://help.aliyun.com/knowledge_detail/39863.html) (which has a nice Go SDK implementation [here](https://github.com/aliyun/alibaba-cloud-sdk-go))

## Authenticating

To authenticate you need to supply our AccessKeyId and AccessKeySecret to the Provider.

## Example

Here's a minimal example of how to get all your DNS records using this `libdns` provider

```go
package main

import (
        "context"
        "fmt"
        "github.com/libdns/alidns"
)

func main() {
        provider := alidns.Provider{
               AccKeyID: "<AccessKeyId form your aliyun console>",
               AccKeySecret: "<AccessKeySecret form your aliyun console>",
        }

        records, err  := provider.GetRecords(context.TODO(), "example.com")
        if err != nil {
                fmt.Println(err.Error())
        }

        for _, record := range records {
                fmt.Printf("%s %v %s %s\n", record.Name, record.TTL.Seconds(), record.Type, record.Value)
        }
}
```
For complete demo check [_demo/demo.go](_demo/demo.go)
