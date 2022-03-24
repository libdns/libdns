package acmedns

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/libdns/libdns"
)

var serverURL = "https://auth.acme-dns.io"

type account struct {
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
	FullDomain string `json:"fulldomain,omitempty"`
}

func createDomainConfig(t *testing.T) DomainConfig {
	resp, err := http.PostForm(serverURL+"/register", nil)
	if err != nil {
		t.Fatal("Failed to register an account")
	}
	var acc account
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("Failed to read response body")
	}
	err = json.Unmarshal(body, &acc)
	if err != nil {
		t.Fatal("Failed to unmarshal response")
	}
	if acc.Username == "" || acc.Password == "" || acc.Subdomain == "" || acc.FullDomain == "" {
		t.Fatal("Account struct has empty fields")
	}
	t.Logf("Account domain is %s", acc.FullDomain)
	return DomainConfig{
		Username:   acc.Username,
		Password:   acc.Password,
		Subdomain:  acc.Subdomain,
		FullDomain: acc.FullDomain,
		ServerURL:  serverURL,
	}
}

func makeRecord(recordValue string) libdns.Record {
	return libdns.Record{
		Type:  "TXT",
		Name:  "_acme-challenge",
		Value: recordValue,
	}
}

func TestAppendRecords(t *testing.T) {
	domain1, domain2 := "example.com", "sub.example.com"
	config1, config2 := createDomainConfig(t), createDomainConfig(t)
	value1, value2, value3 :=
		"__validation_token_received_from_the_ca_1__",
		"__validation_token_received_from_the_ca_2__",
		"__validation_token_received_from_the_ca_3__"
	p := Provider{
		Configs: map[string]DomainConfig{
			domain1: config1,
			domain2: config2,
		},
	}

	_, err := p.AppendRecords(
		context.TODO(),
		domain1,
		[]libdns.Record{makeRecord(value1)},
	)
	if err != nil {
		t.Fatal("Failed to append records: ", err)
	}

	_, err = p.AppendRecords(
		context.TODO(),
		domain2,
		[]libdns.Record{makeRecord(value2), makeRecord(value3)},
	)
	if err != nil {
		t.Fatal("Failed to append records: ", err)
	}

	newRecords, err := net.LookupTXT(p.Configs[domain1].FullDomain)
	if err != nil {
		t.Fatal("TXT record lookup failed")
	}
	if len(newRecords) != 1 {
		t.Fatal("Only 1 TXT record expected")
	}
	if newRecords[0] != value1 {
		t.Fatalf("Unexpected TXT record value %s", newRecords[0])
	}

	newRecords, err = net.LookupTXT(p.Configs[domain2].FullDomain)
	if err != nil {
		t.Fatal("TXT record lookup failed")
	}
	if len(newRecords) != 2 {
		t.Fatal("2 TXT records expected")
	}
	if value2 != newRecords[0] && value2 != newRecords[1] {
		t.Fatalf("Expected record %s, not found in %v", value2, newRecords)
	}
	if value3 != newRecords[0] && value3 != newRecords[1] {
		t.Fatalf("Expected record %s, not found in %v", value2, newRecords)
	}
}
