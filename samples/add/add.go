package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/libdns/libdns"
	"github.com/libdns/metaname"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: ", os.Args[0], "<zone>", "<name>", "<type>", "<value>")
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
	name := os.Args[2]
	rtype := os.Args[3]
	value := os.Args[4]
	added, err := provider.AppendRecords(ctx, zone, []libdns.Record{
		{
			Name:  name,
			TTL:   time.Duration(3600) * time.Second,
			Value: value,
			Type:  rtype,
		},
	})
	if err != nil {
		fmt.Println(err)
	}
	newone := added[0]
	fmt.Println("Reference:", newone.ID)
}
