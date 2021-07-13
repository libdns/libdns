package main

import (
	"context"

	"github.com/libdns/libdns"

	"github.com/nathanejohnson/pdnsprovider"
)

func main() {
	p := &pdnsprovider.Provider{
		ServerURL: "http://localhost", // required
		ServerID:  "localhost",        // if left empty, defaults to localhost.
		APIToken:  "asdfasdfasdf",     // required
	}

	_, err := p.AppendRecords(context.Background(), "example.org.", []libdns.Record{
		{
			Name: "_acme_whatever",
			Type: "TXT",
			Value: "123456",
		},
	})
	if err != nil {
		panic(err)
	}

}
