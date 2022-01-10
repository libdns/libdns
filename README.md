# netcup for [`libdns`](https://github.com/libdns/libdns)

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/netcup)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for the [netcup DNS API](https://ccp.netcup.net/run/webservice/servers/endpoint.php), allowing you to manage DNS records.

## Configuration

The provider is configured by instantiating the `netcup.Provider` with the customer number, the API key and the API password for the DNS API obtained from netcup ([guide](https://www.netcup-wiki.de/wiki/CCP_API)).
Here is a minimal working example to get all DNS records using environment variables for the credentials:

```go
import (
	"context"
	"fmt"
	"os"

	"github.com/libdns/netcup"
)

func main() {
	provider := netcup.Provider{
		CustomerNumber: os.Getenv("LIBDNS_NETCUP_CUSTOMER_NUMBER"),
		APIKey:         os.Getenv("LIBDNS_NETCUP_API_KEY"),
		APIPassword:    os.Getenv("LIBDNS_NETCUP_API_PASSWORD"),
	}
	ctx := context.TODO()
	zone := os.Getenv("LIBDNS_NETCUP_ZONE")

	records, err := provider.GetRecords(ctx, zone)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for _, record := range records {
		fmt.Printf("%+v\n", record)
	}
}
```
