# Azure DNS for `libdns`

This package implements the libdns interfaces for the [Azure DNS API](https://docs.microsoft.com/en-us/rest/api/dns/).

## Authenticating

This package supports authentication using the **Client Credentials** (Azure AD Application ID and Secret) through [azure-sdk-for-go](https://github.com/Azure/azure-sdk-for-go).

You will need to create a service principal using [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli) or [Azure Portal](https://docs.microsoft.com/en-us/azure/active-directory/develop/howto-create-service-principal-portal), and assign the **DNS Zone Contributor** role to the service principal for the DNS zones that you want to manage.

Then keep the following information to authenticate:

* `AZURE_TENANT_ID`
	* [Azure Active Directory] > [Properties] > [Tenant ID]
* `AZURE_CLIENT_ID`
	* [Azure Active Directory] > [App registrations] > Your Application > [Application ID]
* `AZURE_CLIENT_SECRET`
	* [Azure Active Directory] > [App registrations] > Your Application > [Certificates & secrets] > [Client secrets] > [Value]
* `AZURE_SUBSCRIPTION_ID`
	* [DNS zones] > Your Zone > [Subscription ID]
* `AZURE_RESOURCE_GROUP_NAME`
	* [DNS zones] > Your Zone > [Resource group]

## Example

Here's a minimal example of how to get all your DNS records using this `libdns` provider (see `_example/main.go`)

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/libdns/azure"
	"github.com/libdns/libdns"
)

// main shows how libdns works with Azure DNS.
//
// To make this example work, you have to speficy some required environment variables:
// AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_SUBSCRIPTION_ID,
// AZURE_RESOURCE_GROUP_NAME, AZURE_DNS_ZONE_FQDN
func main() {

	// Create new provider instance
	provider := azure.Provider{
		TenantId:          os.Getenv("AZURE_TENANT_ID"),
		ClientId:          os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret:      os.Getenv("AZURE_CLIENT_SECRET"),
		SubscriptionId:    os.Getenv("AZURE_SUBSCRIPTION_ID"),
		ResourceGroupName: os.Getenv("AZURE_RESOURCE_GROUP_NAME"),
	}
	zone := os.Getenv("AZURE_DNS_ZONE_FQDN")

	// Invoke authentication and store client to instance
	if err := provider.NewClient(); err != nil {
		fmt.Printf("%v\n", err)
		return
	}

	// List existing records
	fmt.Printf("List existing records\n")
	currentRecords, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
	for _, record := range currentRecords {
		fmt.Printf("Exists: %v\n", record)
	}
}
```
