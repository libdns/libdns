# Simply.com for `libdns`

This package implements the [libdns](https://github.com/libdns/libdns) interfaces for [Simply.com](https://www.simply.com/).

## Usage

```go
import "github.com/libdns/simplydotcom"

provider := simplydotcom.Provider{
	AccountName: "S123456", 
	APIKey: "your-api-key"
}
```

## Configuration

The provider requires two pieces of information:

- `account_name`: Your Simply.com account name (e.g. S123456)
- `api_key`: Your Simply.com API key

The following optional configuration is supported:

- `base_url`: The base URL of the Simply.com API. Defaults to `https://api.simply.com/2/`.
- `max_retries`: The maximum number of retries to perform if being rate limited. Defaults to 3. 

You can get your API key from your Simply.com account dashboard.

## Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"time"

	"github.com/libdns/libdns"
	"github.com/libdns/simplydotcom"
)

func main() {
	// Initialize the provider
	provider := simplydotcom.Provider{
		AccountName: "S123456",
		APIKey:      "your-api-key",
	}

	// Create new records
	records, err := provider.AppendRecords(context.Background(), "example.com", []libdns.Record{
		// A record
		&libdns.Address{
			Name: "test",
			TTL:  3600 * time.Second,
			IP:   netip.MustParseAddr("1.2.3.4"),
		},
		// TXT record
		&libdns.TXT{
			Name: "verification",
			TTL:  3600 * time.Second,
			Text: "verify-site=abcdefghijklmnopqrstuvwxyz",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created %d records\n", len(records))
}
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
