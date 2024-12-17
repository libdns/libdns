package main

import (
	"context"
	"fmt"
	"os"

	"github.com/libdns/katapult"
)

var (
	envToken = os.Getenv("LIBDNS_KATAPULT_API_TOKEN")
	envZone  = os.Getenv("LIBDNS_KATAPULT_ZONE")
)

func main() {
	p := katapult.Provider{APIToken: envToken}

	ret, err := p.GetRecords(context.TODO(), envZone)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Result:", ret)
}
