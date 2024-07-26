// Use the following command line to run this example:
//
//	go run _examples/main.go -email your_email -key your_api_key -zone your_zone.com
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/libdns/libdns"
	"github.com/libdns/luadns"
)

var email string
var key string
var url string
var zone string

func main() {
	flag.StringVar(&email, "email", "joe@example.com", "your email address")
	flag.StringVar(&key, "key", "", "your API key")
	flag.StringVar(&zone, "zone", "example.org.", "zone name")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	provider := luadns.Provider{
		Email:  email,
		APIKey: key,
	}

	ctx := context.Background()
	records, err := provider.GetRecords(ctx, zone)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("===> Running GetRecords ...")
	for _, r := range records {
		fmt.Println(r)
	}

	fmt.Println("===> Running SetRecords ...")
	records, err = provider.SetRecords(ctx, zone, records)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("===> Running AppendRecords ...")
	records, err = provider.AppendRecords(ctx, zone, []libdns.Record{
		libdns.Record{Name: "_acme-challenge", Type: "TXT", Value: "Hello, world!", TTL: 3600 * time.Second},
	})
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("===> Running DeleteRecords ...")
	_, err = provider.DeleteRecords(ctx, zone, records)
	if err != nil {
		log.Fatalln(err)
	}
}
