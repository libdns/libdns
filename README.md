Golang client NIC.ru for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/TODO:PROVIDER_NAME)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for NIC.ru, allowing you to manage DNS records.

TODO: Show how to configure and use. Explain any caveats.

Example:
```
package nicrudns

import (
	"context"
	"fmt"
	"github.com/libdns/libdns"
	"github.com/pkg/errors"
	"time"
)

var (
	clientID         string
	secretID         string
	username         string
	password         string
	zoneName         string
	nicruServiceName string
	cachePath        string
)

func ExampleLibdnsProvider() error {
	provider := Provider{
		OAuth2ClientID:   clientID,
		OAuth2SecretID:   secretID,
		Username:         username,
		Password:         password,
		NicRuServiceName: nicruServiceName,
		CachePath:        cachePath,
	}
	ctx := context.TODO()
	var records = []libdns.Record{
		{
			Type:  `A`,
			Name:  `www`,
			Value: `1.2.3.4`,
			TTL:   time.Hour,
		},
	}
	if records, err := provider.AppendRecords(ctx, zoneName, records); err != nil {
		return errors.Wrap(err, `append records error`)
	} else {
		for _, record := range records {
			fmt.Println(record.Name, record.TTL, record.TTL, record.Value)
		}
		return nil
	}
}

func ExampleNicruClient() error {
	config := &Config{
		Credentials: &Credentials{
			OAuth2ClientID: clientID,
			OAuth2SecretID: secretID,
			Username:       username,
			Password:       password,
		},
		ZoneName:       zoneName,
		DnsServiceName: nicruServiceName,
		CachePath:      cachePath,
	}
	client := NewClient(config)
	var names = []string{`www`}
	if response, err := client.AddA(names, `1.2.3.4`, `3600`); err != nil {
		return errors.Wrap(err, `add records error`)
	} else {
		for _, rr := range response.Data.Zone[0].Rr {
			fmt.Println(rr.Name, rr.Type, rr.Ttl, rr.A.String())
		}
		return nil
	}
}

```