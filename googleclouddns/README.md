Google Cloud DNS for `libdns`
=======================

[![godoc reference](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/libdns/googleclouddns)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for [Google Cloud](https://cloud.google.com/).

## Authenticating

The googleclouddns package will authenticate using the supported authentication methods found in the [google-cloud-go library](https://github.com/googleapis/google-cloud-go#authorization):

* the environment variable `GOOGLE_APPLICATION_CREDENTIALS` pointing to a service account file
* `ServiceAccountJSON` (`json:"gcp_application_default"`)
  * The path to a service account JSON file
    
The package also requires the project where the Google Cloud DNS zone exists

* `Project` (`json:"gcp_project"`)
    * The ID of the GCP Project
---

Google Cloud DNS for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/googleclouddns)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for Google Cloud DNS, allowing you to manage DNS records.

## Example

Here's a minimal example of how to get all your DNS records using this `libdns` provider:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/libdns/googleclouddns"
)

// main shows how libdns works with Google Cloud DNS.
//
// In this example, the project where the Cloud DNS zone exists is passed 
// as an environment variable. Auth data is determined through normal 
// Google Cloud Go API sources.
func main() {
	// Create new provider instance
	googleProvider := googleclouddns.Provider{
		Project: os.Getenv("GCP_PROJECT"),
	}
	zone := `example.localhost`

	// List existing records
	fmt.Printf("List existing records\n")
	currentRecords, err := googleProvider.GetRecords(context.TODO(), zone)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	for _, record := range currentRecords {
		fmt.Printf("Exists: %v\n", record)
	}
}
```
Note: The Google Cloud DNS API returns 1-n values for each Google DNS recordset. This is converted to a slice of libdns.Records each with
the same name but unique values.

This also applies to `AppendRecords` and `SetRecords`. If multiple fields are desired for a Google Cloud DNS entry, pass
a slice of libdns.Record entries into those functions and they will be added to the Google DNS record in the order of the
slice.

## Testing
Testing relies on the Google [httpreplay](https://pkg.go.dev/cloud.google.com/go/httpreplay) package. If an updated request to the 
Google API servers is required, you can do the following:

* install the Google Cloud SDK
* generate application default credentials: `gcloud auth application-default login`
* delete the appropriate JSON file in the `replay` directory
* rerun that test step, this will give you a fresh JSON file for that test