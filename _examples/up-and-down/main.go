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
	resAll, err := p.GetRecords(context.TODO(), zone)
	exitOnError(err)
	printRecords("records at start", resAll)

	fmt.Println("appending")
	res, err := p.AppendRecords(context.TODO(), zone,
		[]libdns.Record{
			{Name: "test", Type: "A", Value: "192.168.1.10", TTL: 5 * time.Minute},
			{Name: "test", Type: "A", Value: "192.168.1.20", TTL: 5 * time.Minute},
		})
	exitOnError(err)
	printRecords("back from append", res)
	fmt.Println("Will sleep for a few seconds...")
	time.Sleep(time.Second * 5)

	// change TTL
	res[0].TTL = 15 * time.Minute
	test2 := res[1]
	res, err = p.SetRecords(context.TODO(), zone, []libdns.Record{res[0]})
	exitOnError(err)
	printRecords("back from set", res)
	time.Sleep(time.Second)

	// resAll, err = p.GetRecords(context.TODO(), zone)
	// exitOnError(err)
	// printRecords("after change", resAll)
	// time.Sleep(time.Second)

	// delete first
	res, err = p.DeleteRecords(context.TODO(), zone, res)
	exitOnError(err)
	printRecords("after delete 1", res)

	resAll, err = p.GetRecords(context.TODO(), zone)
	exitOnError(err)
	printRecords("All after delete 1", resAll)
	time.Sleep(time.Second)

	// delete second
	res, err = p.DeleteRecords(context.TODO(), zone, []libdns.Record{test2})
	exitOnError(err)
	printRecords("after delete 2", res)

	// check final result
	resAll, err = p.GetRecords(context.TODO(), zone)
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
