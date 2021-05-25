package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/libdns/libdns"
	"github.com/libdns/loopia"
)

func exitOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func main() {
	user := os.Getenv("LOOPIA_USER")
	password := os.Getenv("LOOPIA_PASSWORD")
	zone := os.Getenv("ZONE")
	if zone == "" {
		fmt.Fprintf(os.Stderr, "ZONE not set\n")
		os.Exit(1)
	}

	if user == "" {
		exitOnError(fmt.Errorf("user is not set"))
	}

	if password == "" {
		exitOnError(fmt.Errorf("password is not set"))
	}

	fmt.Printf("zone: %s, user: %s\n", zone, user)
	p := &loopia.Provider{
		Username: user,
		Password: password,
	}
	ctx := context.TODO()
	fmt.Println("appending")
	res, err := p.AppendRecords(ctx, zone,
		[]libdns.Record{
			{Name: "_acme-challenge.test", Type: "TXT", Value: "Zgu7tw287LB-LpXyTHYLeROag9-4CLHnM77zvTEvH6o"},
		})
	exitOnError(err)
	printRecords("after append", res)

	fmt.Println("Will sleep for a few seconds...")
	time.Sleep(time.Second * 5)

	resAll, err := p.GetRecords(ctx, zone)
	exitOnError(err)
	printRecords("after all", resAll)

	// delete first
	res, err = p.DeleteRecords(ctx, zone, res)
	exitOnError(err)
	printRecords("after delete", res)

	// check final result
	resAll, err = p.GetRecords(ctx, zone)
	exitOnError(err)
	printRecords("after all", resAll)

	fmt.Println("Done!")
}

func printRecords(title string, records []libdns.Record) {
	fmt.Println(title)
	for i, r := range records {
		fmt.Printf("  [%d] %+v\n", i, r)
	}
}
