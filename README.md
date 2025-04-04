# mijn.host for `libdns`

This package implements the libdns interfaces for the [mijn.host API](https://mijn.host/api/doc)

## Authenticating

To authenticate you need to supply our API key that you can create in your account.

## Example

Here's a minimal example of how to get all your DNS records using this `libdns` provider

```go
package main

import (
        "context"
        "fmt"
        "github.com/libdns/mijnhost"
)

func main() {
        provider := mijnhost.Provider{ApiKey: "api-key-1"}

        records, err  := provider.GetRecords(context.TODO(), "example.com")
        if err != nil {
                fmt.Println(err.Error())
        }

        for _, record := range records {
                fmt.Printf("%s %v %s %s\n", record.Name, record.TTL.Seconds(), record.Type, record.Value)
        }
}
```
