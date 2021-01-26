# Azure DNS for `libdns`

This package implements the libdns interfaces for the [Azure DNS API](https://docs.microsoft.com/en-us/rest/api/dns/).

## Authenticating

This package supports authentication using the **Client Credentials** (Azure AD Application ID and Secret) through [azure-sdk-for-go](https://github.com/Azure/azure-sdk-for-go).

You will need to create a service principal using [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/create-an-azure-service-principal-azure-cli) or [Azure Portal](https://docs.microsoft.com/en-us/azure/active-directory/develop/howto-create-service-principal-portal), and assign the **DNS Zone Contributor** role to the service principal for the DNS zones that you want to manage.

Then keep the following information to pass to the `Provider` struct fields for authentication:

* `TenantId` (`json:"tenant_id"`)
	* [Azure Active Directory] > [Properties] > [Tenant ID]
* `ClientId` (`json:"client_id"`)
	* [Azure Active Directory] > [App registrations] > Your Application > [Application ID]
* `ClientSecret` (`json:"client_secret"`)
	* [Azure Active Directory] > [App registrations] > Your Application > [Certificates & secrets] > [Client secrets] > [Value]
* `SubscriptionId` (`json:"subscription_id"`)
	* [DNS zones] > Your Zone > [Subscription ID]
* `ResourceGroupName` (`json:"resource_group_name"`)
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
// In this example, the information required for authentication is passed as environment variables.
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
