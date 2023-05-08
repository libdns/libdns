package infomaniak

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

// RoundTripFunc to mock transport layer
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip to allow to use RoundTripFunc as transport layer
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// newHttpTestClient returns *http.Client with transport replaced to avoid making real calls
func newHttpTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

// newTestClient returns new client that returns the given answer for an http call
func newTestClient(resultData string, cachedDomains *[]IkDomain) *Client {
	httpClient := newHttpTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"result":"success", "data":%s}`, resultData))),
			Header:     make(http.Header),
		}
	})
	return &Client{HttpClient: httpClient, domains: cachedDomains}
}

// anIdResponse returns the given string as the id of a record in form of an http response
func anIdResponse(id string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString(fmt.Sprintf(`{"result":"success", "data":"%s"}`, id))),
		Header:     make(http.Header),
	}
}

func Test_GetDomainForZone_ReturnsDomainForZone(t *testing.T) {
	domainName := "example.com"
	id := 1893
	client := newTestClient(fmt.Sprintf(`[ { "id":%d, "customer_name":"%s" } ]`, id, domainName), nil)
	domain, err := client.getDomainForZone(context.TODO(), "subdomain."+domainName)

	if err != nil {
		t.Fatal(err)
	}

	if domain.ID != id {
		t.Fatalf("Expected domain ID %d, got %d", id, domain.ID)
	}
}

func Test_GetDomainForZone_ReturnsErrorIfDomainForZoneNotFound(t *testing.T) {
	domainName := "example.com"
	client := newTestClient(fmt.Sprintf(`[ { "id":10, "customer_name":"%s" } ]`, domainName), nil)
	domain, err := client.getDomainForZone(context.TODO(), "subdomain.test.com")

	if err == nil {
		t.Fatalf("Expected error because no domain matched but got %#v", domain)
	}
}

func Test_GetDnsRecordsForZone_OnlyReturnsRecordsForSpecifiedZone(t *testing.T) {
	domainName := "example.com"
	zone := "subzone." + domainName
	recForZone := IkRecord{ID: "1893", SourceIdn: zone}

	jsonString1, err := json.Marshal(recForZone)
	if err != nil {
		t.Fatal(err)
	}
	jsonString2, err := json.Marshal(IkRecord{ID: "335", SourceIdn: domainName})
	if err != nil {
		t.Fatal(err)
	}

	client := newTestClient(fmt.Sprintf(`[ %s, %s ]`, jsonString1, jsonString2), &[]IkDomain{{Name: "example.com", ID: 100}})

	recsForZone, err := client.GetDnsRecordsForZone(context.TODO(), zone)
	if err != nil {
		t.Fatal(err)
	}

	if len(recsForZone) != 1 {
		t.Fatalf("Expected %d records, got %d", 1, len(recsForZone))
	}

	if recsForZone[0].ID != recForZone.ID {
		t.Fatalf("Expected records with ID %s, got %s", recForZone.ID, recsForZone[0].ID)
	}
}

func Test_CreateOrUpdateRecord_UpdatesExistingRecord(t *testing.T) {
	id := "984"
	httpClient := newHttpTestClient(func(req *http.Request) *http.Response {
		if req.Method != http.MethodPut {
			t.Fatalf("Expected http method %s, got %s", http.MethodPut, req.Method)
		}
		return anIdResponse("985")
	})

	client := Client{domains: &[]IkDomain{{Name: "example.com", ID: 100}}, HttpClient: httpClient}
	rec, _ := client.CreateOrUpdateRecord(context.TODO(), "example.com", IkRecord{ID: id})
	if rec.ID != id {
		t.Fatal("ID of already existing record was updated")
	}
}

func Test_CreateOrUpdateRecord_CreatesNewRecord(t *testing.T) {
	id := "445"
	httpClient := newHttpTestClient(func(req *http.Request) *http.Response {
		if req.Method != http.MethodPost {
			t.Fatalf("Expected http method %s, got %s", http.MethodPost, req.Method)
		}
		return anIdResponse(id)
	})

	client := Client{domains: &[]IkDomain{{Name: "example.com", ID: 100}}, HttpClient: httpClient}
	rec, _ := client.CreateOrUpdateRecord(context.TODO(), "example.com", IkRecord{ID: ""})
	if rec.ID != id {
		t.Fatalf("Expected ID to be %s, got %s", id, rec.ID)
	}
}
