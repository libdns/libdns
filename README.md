# IONOS DNS API for `libdns`

This package implements the libdns interfaces for the [IONOS DNS
API (beta)](https://developer.hosting.ionos.de/docs/dns)

## Authenticating

To authenticate you need to supply a IONOS API Key, as described on
https://developer.hosting.ionos.de/docs/getstarted

## Example

Here's a minimal example of how to get all DNS records for zone.

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/libdns/libdns/ionos"
)

func main() {
	token := os.Getenv("LIBDNS_IONOS_TOKEN")
	if token == "" {
		panic("LIBDNS_IONOS_TOKEN not set")
	}

	zone := os.Getenv("LIBDNS_IONOS_ZONE")
	if zone == "" {
		panic("LIBDNS_IONOS_ZONE not set")
	}

	p := &ionos.Provider{
		AuthAPIToken: token,
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Duration(15*time.Second))
	records, err := p.GetRecords(ctx, zone)
	if err != nil {
		panic(err)
	}

	out, _ := json.MarshalIndent(records, "", "  ")
	fmt.Println(string(out))
}
```

## Test

The file `provisioner_test.go` contains an end-to-end test suite, using the
original IONOS API service (i.e. no test doubles - be careful). To run the
tests:

```console
$ export LIBDNS_IONOS_TEST_ZONE=mydomain.org
$ export LIBDNS_IONOS_TEST_TOKEN=aaaaaaaaaaa.bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
$ go  test -v
=== RUN   Test_AppendRecords
--- PASS: Test_AppendRecords (43.01s)
=== RUN   Test_DeleteRecords
--- PASS: Test_DeleteRecords (23.91s)
=== RUN   Test_GetRecords
--- PASS: Test_GetRecords (30.96s)
=== RUN   Test_SetRecords
--- PASS: Test_SetRecords (51.39s)
PASS
ok  	github.com/libdns/ionos	149.277s
```

The tests were taken from the [Hetzner libdns
module](https://github.com/libdns/hetzner) and are not modified.

## Author

original Work (C) Copyright 2020 by matthiasng (based on https://github.com/libdns/hetzner),
this version (C) Copyright 2021 by Jan Delgado.

License: MIT

