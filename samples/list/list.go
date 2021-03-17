package main

import (
	"context"
	"fmt"
	"os"

	"github.com/libdns/metaname"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ", os.Args[0], "<zone>")
		os.Exit(1)
	}
	ctx := context.TODO()
	endpoint := "https://test.metaname.net/api/1.1"
	val, ok := os.LookupEnv("api_endpoint")
	if ok {
		endpoint = val
	}
	provider := metaname.Provider{APIKey: os.Getenv("api_key"),
		AccountReference: os.Getenv("account_reference"),
		Endpoint:         endpoint}
	zone := os.Args[1]
	recs, err := provider.GetRecords(ctx, zone)
	if err != nil {
		fmt.Println(err)
	}
	for _, r := range recs {
		fmt.Println(r.ID, r.Name, r.Type, r.Value)
	}

}
