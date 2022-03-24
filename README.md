Joohoi's ACME-DNS for [`libdns`](https://github.com/libdns/libdns)
=======================

[![Go Reference](https://pkg.go.dev/badge/test.svg)](https://pkg.go.dev/github.com/libdns/TODO:joohoi_acme_dns)

This package implements the [libdns interfaces](https://github.com/libdns/libdns) for Joohoi's ACME-DNS.

Since ACME-DNS is a simplified DNS server that only allows setting TXT values,
this `libdns` provider only implements `RecordAppender` and `RecordDeleter` interfaces.
`Provider` has a field `Configs` which is a _domain -> ACME-DNS account config_ mapping.
The structure of these configurations directly matches the JSON file used by
[acme-dns-client](https://github.com/acme-dns/acme-dns-client).

ACME-DNS is only meant to be used for completing ACME DNS challenges. A possible way to use this library:

1. Install [acme-dns-client](https://github.com/acme-dns/acme-dns-client).

2. Create an ACME-DNS account / register your domain using `acme-dns-client`.
  For a quick test, you can run

  `sudo acme-dns-client register -d your.test.domain.example.com --dangerous`

  This will create an account on public ACME-DNS server `auth.acme-dns.io`. The client
  saves the configuration as `/etc/acmedns/clientstorage.json`.

  In the process, `acme-dns-client` will ask you to create a DNS CNAME record from
  `_acme-challenge.your.test.domain.example.com` to a target provided by ACME-DNS server.

3. Deserialize the config file into a Provider and use it to create `_acme-challenge`
  records:

  ```go
	content, err := ioutil.ReadFile("configs.json")
	if err != nil {
		log.Fatalf("Failed to read file: %s", err)
	}
	var configs map[Domain]DomainConfig
	err = json.Unmarshal(content, &configs)
	if err != nil {
		log.Fatalf("Failed to unmarshall json: %s", err)
	}
	p := Provider{Configs: configs}
	records, err := p.AppendRecords(
		context.TODO(),
		"your.test.domain.example.com",
		[]libdns.Record{
			{
				Type:  "TXT",
				Name:  "_acme-challenge",
				Value: "___validation_token_received_from_the_ca___",
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to append records: %s", err)
	}
	fmt.Printf("Created records: %v\n", records)
  ```

For more information about Joohoi's ACME-DNS and the motivation for it, see:

* https://github.com/joohoi/acme-dns
* https://www.eff.org/deeplinks/2018/02/technical-deep-dive-securing-automation-acme-dns-challenge-validation