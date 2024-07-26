# LuaDNS for [`libdns`](https://github.com/libdns/libdns)

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/luadns)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [LuaDNS](https://www.luadns.com/api.html), allowing you to manage DNS records.

Usage:

```go
// Init Provider struct.
provider := luadns.Provider{
	Email:  email,
	APIKey: key,
}

// List zone records.
records, err := provider.GetRecords(ctx, zone)
if err != nil {
	log.Fatalln(err)
}

// Set zone records.
records, err = provider.SetRecords(ctx, zone, records)
if err != nil {
	log.Fatalln(err)
}

// Append new records.
records, err = provider.AppendRecords(ctx, zone, []libdns.Record{
	libdns.Record{Name: "_acme-challenge", Type: "TXT", Value: "Hello, world!", TTL: 3600 * time.Second},
})
if err != nil {
	log.Fatalln(err)
}

// Delete a list of records.
_, err = provider.DeleteRecords(ctx, zone, records)
if err != nil {
	log.Fatalln(err)
}
```

For a complete example see [_examples/main.go](_examples/main.go).
