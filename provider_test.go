package joohoi_acme_dns

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/libdns/libdns"
)

var (
	clientURL   = "https://auth.acme-dns.io"
	recordValue = "___validation_token_received_from_the_ca___"
	record      = libdns.Record{Type: "TXT", Value: recordValue}
)

type account struct {
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
	FullDomain string `json:"fulldomain,omitempty"`
}

func registerAccount(t *testing.T) account {
	resp, err := http.PostForm(clientURL+"/register", nil)
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
	return acc
}

func TestAppendRecords(t *testing.T) {
	acc := registerAccount(t)
	p := Provider{Username: acc.Username, Password: acc.Password, Subdomain: acc.Subdomain, ClientURL: clientURL}
	_, err := p.AppendRecords(context.TODO(), "any-zone", []libdns.Record{record})
	if err != nil {
		t.Fatal("Failed to append records: %w", err)
	}
	newRecords, err := net.LookupTXT(acc.FullDomain)
	if err != nil {
		t.Fatal("TXT record lookup failed")
	}
	if len(newRecords) != 1 {
		t.Fatal("Only 1 TXT record expected")
	}
	if newRecords[0] != recordValue {
		t.Fatalf("Unexpected TXT record value %s", newRecords[0])
	}
}
