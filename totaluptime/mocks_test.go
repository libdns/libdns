package totaluptime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func AssertStrings(t testing.TB, got, want string) {
	t.Helper()

	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func AssertStringContains(t testing.TB, got, want string) {
	t.Helper()

	if !strings.Contains(got, want) {
		t.Errorf("got %q want %q", got, want)
	}
}

func AssertInts(t testing.TB, got, want int) {
	t.Helper()

	if got != want {
		t.Errorf("got %v want %v", got, want)
	}
}

func AssertNil(t testing.TB, got interface{}) {
	t.Helper()

	if (got != nil) && (got != 0) && (got != "") && (!reflect.ValueOf(got).IsNil()) {
		t.Errorf("expected <nil> but got %v\n", got)
	}
}

func PrettyPrint(ugly interface{}) string {
	pretty, _ := json.MarshalIndent(ugly, "", "\t")
	return fmt.Sprintln(string(pretty))
}

func setupMockServer() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var respBody string
		w.Header().Add("Content-Type", "application/json")

		switch r.URL.Path {
		case "/All": // lookup all domains
			respBody = `{
				"rows": [
				  {
					"domainName": "testdomain.com",
					"ID": "test-domain-id"
				  }
				]
			  }`

		case "/test-domain-id/AllRecords": // lookup all records in domain
			respBody = `{
				"ARecord": {
				  "rows": [
					{
					  "aHostName": "blue-a",
					  "ID": "test-a-id",
					  "aIPAddress": "111.111.111.111",
					  "aTTL": "3600"
					}
				  ]
				},
				"CNAMERecord": {
				  "rows": [
					{
					  "cnameName": "green-cname",
					  "ID": "test-cname-id",
					  "cnameAliasFor": "test-alias-domain.com.",
					  "cnameTTL": "3600"
					}
				  ]
				},
				"MXRecord": {
				  "rows": [
					{
					  "mxDomainName": "@",
					  "ID": "test-mx-id",
					  "mxMailServer": "test-mx-mailserver.com.",
					  "mxTTL": "3600"
					}
				  ]
				},
				"NSRecord": {
				  "rows": [
					{
					  "nsHostName": "@",
					  "ID": "test-ns-id",
					  "nsName": "test-ns-name.com.",
					  "nsTTL": "28800"
					}
				  ]
				},
				"TXTRecord": {
				  "rows": [
					{
					  "txtHostName": "red-txt",
					  "ID": "test-txt-id",
					  "txtText": "test-txt-text-value",
					  "txtTTL": "60"
					}
				  ]
				}
			  }`

		default: // successful transaction response
			respBody = `{
				"StatusCode": null,
				"Type": null,
					"message": "Record added successfully.",
					"status": "Success",
				"data": null,
				"id": "489b7146-9f77-4adc-9e72-33bcceab3010",
				"options": null
			}`
		}

		// 200 status code returned by default
		w.Write([]byte(respBody))
	}))

	APIbase = server.URL
}

func setupMockServerInvalidJSON() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var respBody string
		w.Header().Add("Content-Type", "application/json")

		switch r.URL.Path {
		default:
			// json cannot be unmarshal'd
			respBody = `this can't be unmarshal'd`
		}

		// 200 status code returned by default
		w.Write([]byte(respBody))
	}))

	APIbase = server.URL
}

func setupMockServerFailedTransaction() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var respBody string
		w.Header().Add("Content-Type", "application/json")

		switch r.URL.Path {
		case "/All": // lookup all domains
			respBody = `{
					"rows": [
					  {
						"domainName": "testdomain.com",
						"ID": "test-domain-id"
					  }
					]
				  }`

		case "/test-domain-id/AllRecords": // lookup all records in domain
			respBody = `{
					"ARecord": {
					  "rows": [
						{
						  "aHostName": "blue-a",
						  "ID": "test-a-id",
						  "aIPAddress": "111.111.111.111",
						  "aTTL": "3600"
						}
					  ]
					},
					"CNAMERecord": {
					  "rows": [
						{
						  "cnameName": "green-cname",
						  "ID": "test-cname-id",
						  "cnameAliasFor": "test-alias-domain.com.",
						  "cnameTTL": "3600"
						}
					  ]
					},
					"MXRecord": {
					  "rows": [
						{
						  "mxDomainName": "@",
						  "ID": "test-mx-id",
						  "mxMailServer": "test-mx-mailserver.com.",
						  "mxTTL": "3600"
						}
					  ]
					},
					"NSRecord": {
					  "rows": [
						{
						  "nsHostName": "@",
						  "ID": "test-ns-id",
						  "nsName": "test-ns-name.com.",
						  "nsTTL": "28800"
						}
					  ]
					},
					"TXTRecord": {
					  "rows": [
						{
						  "txtHostName": "red-txt",
						  "ID": "test-txt-id",
						  "txtText": "test-txt-text-value",
						  "txtTTL": "60"
						}
					  ]
					}
				  }`

		default: // failed transaction response
			respBody = `{
				"StatusCode": null,
				"Type": null,
				"message": "Invalid Port. ",
				"status": "Failed to validate",
				"data": null,
				"id": "",
				"options": null
			}`
		}

		// 200 status code returned by default
		w.Write([]byte(respBody))
	}))

	APIbase = server.URL
}
