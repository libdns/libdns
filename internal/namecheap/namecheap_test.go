package namecheap_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/libdns/namecheap/internal/namecheap"
)

const (
	setHostsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ApiResponse xmlns="https://api.namecheap.com/xml.response" Status="OK">
  <Errors />
  <RequestedCommand>namecheap.domains.dns.setHosts</RequestedCommand>
  <CommandResponse Type="namecheap.domains.dns.setHosts">
    <DomainDNSSetHostsResult Domain="domain51.com" IsSuccess="true" />
  </CommandResponse>
  <Server>SERVER-NAME</Server>
  <GMTTimeDifference>+5</GMTTimeDifference>
  <ExecutionTime>32.76</ExecutionTime>
</ApiResponse>`

	getHostsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ApiResponse xmlns="http://api.namecheap.com/xml.response" Status="OK">
  <Errors />
  <RequestedCommand>namecheap.domains.dns.getHosts</RequestedCommand>
  <CommandResponse Type="namecheap.domains.dns.getHosts">
    <DomainDNSGetHostsResult Domain="domain.com" IsUsingOurDNS="true">
      <Host HostId="12" Name="@" Type="A" Address="1.2.3.4" MXPref="10" TTL="1800" />
      <Host HostId="14" Name="www" Type="A" Address="122.23.3.7" MXPref="10" TTL="1800" />
    </DomainDNSGetHostsResult>
  </CommandResponse>
  <Server>SERVER-NAME</Server>
  <GMTTimeDifference>+5</GMTTimeDifference>
  <ExecutionTime>32.76</ExecutionTime>
</ApiResponse>`

	emptyHostsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ApiResponse xmlns="http://api.namecheap.com/xml.response" Status="OK">
  <Errors />
  <RequestedCommand>namecheap.domains.dns.getHosts</RequestedCommand>
  <CommandResponse Type="namecheap.domains.dns.getHosts">
    <DomainDNSGetHostsResult Domain="domain.com" IsUsingOurDNS="true" />
  </CommandResponse>
  <Server>SERVER-NAME</Server>
  <GMTTimeDifference>+5</GMTTimeDifference>
  <ExecutionTime>32.76</ExecutionTime>
</ApiResponse>`

	errorResponse = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="ERROR" xmlns="http://api.namecheap.com/xml.response">
  <Errors>
    <Error Number="1010102">Parameter APIKey is missing</Error>
  </Errors>
  <Warnings />
  <RequestedCommand />
  <Server>TEST111</Server>
  <GMTTimeDifference>--1:00</GMTTimeDifference>
  <ExecutionTime>0</ExecutionTime>
</ApiResponse>`
)

func ensureQueryParams(t *testing.T, r *http.Request, expectedQueryParams url.Values) {
	t.Helper()
	if diff := cmp.Diff(expectedQueryParams, r.URL.Query()); diff != "" {
		t.Fatalf("Expected query params does not match received: %s", diff)
	}
}

func toURLValues(values map[string]string) url.Values {
	urlValues := make(url.Values)
	for k, v := range values {
		urlValues[k] = []string{v}
	}
	return urlValues
}

func TestGetHosts(t *testing.T) {
	expectedValues := map[string]string{
		"ApiUser":  "testUser",
		"ApiKey":   "testAPIKey",
		"UserName": "testUser",
		"ClientIp": "localhost",
		"Command":  "namecheap.domains.dns.getHosts",
		"TLD":      "domain",
		"SLD":      "any",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ensureQueryParams(t, r, toURLValues(expectedValues))
		_, err := w.Write([]byte(getHostsResponse))
		if err != nil {
			t.Fatal(err)
		}
	}))
	t.Cleanup(ts.Close)

	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.WithEndpoint(ts.URL), namecheap.WithClientIP("localhost"))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	hosts, err := c.GetHosts(context.TODO(), "any.domain")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	// Test that hosts unmarshal correctly.
	expectedHosts := map[string]namecheap.HostRecord{
		"12": {
			Name:       "@",
			HostID:     "12",
			RecordType: namecheap.A,
			Address:    "1.2.3.4",
			MXPref:     "10",
			TTL:        1800,
		},
		"14": {
			Name:       "www",
			HostID:     "14",
			RecordType: namecheap.A,
			Address:    "122.23.3.7",
			MXPref:     "10",
			TTL:        1800,
		},
	}

	if len(hosts) != len(expectedHosts) {
		t.Fatalf("Length does not match expected. Expected: %d. Got: %d.", len(expectedHosts), len(hosts))
	}

	for _, host := range hosts {
		if host.HostID == "" {
			t.Fatal("Empty HostID")
		}

		if diff := cmp.Diff(host, expectedHosts[host.HostID]); diff != "" {
			t.Fatalf("Host and expected host are not equal. Diff: %s", diff)
		}
	}
}

func TestGetHostsContextCanceled(t *testing.T) {
	// Testing that the request context gets canceled
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			w.Write([]byte(errorResponse))
		case <-time.After(time.Second):
			t.Fatal("Context was not cancelled in time")
		}
	}))
	t.Cleanup(ts.Close)

	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.WithEndpoint(ts.URL), namecheap.WithClientIP("localhost"))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.GetHosts(ctx, "any.domain"); err == nil {
		t.Fatal("Expected error cancelling context but got none")
	}
}

func TestGetHostsWithExtraDotInDomain(t *testing.T) {
	expectedValues := map[string]string{
		"ApiUser":  "testUser",
		"ApiKey":   "testAPIKey",
		"UserName": "testUser",
		"ClientIp": "localhost",
		"Command":  "namecheap.domains.dns.getHosts",
		"TLD":      "domain",
		"SLD":      "any",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ensureQueryParams(t, r, toURLValues(expectedValues))
		_, err := w.Write([]byte(getHostsResponse))
		if err != nil {
			t.Fatal(err)
		}
	}))
	t.Cleanup(ts.Close)

	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.WithEndpoint(ts.URL), namecheap.WithClientIP("localhost"))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	if _, err := c.GetHosts(context.TODO(), "any.domain."); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

func TestGetHostsWithExtraDotsInTLD(t *testing.T) {
	expectedValues := map[string]string{
		"ApiUser":  "testUser",
		"ApiKey":   "testAPIKey",
		"UserName": "testUser",
		"ClientIp": "localhost",
		"Command":  "namecheap.domains.dns.getHosts",
		"TLD":      "co.uk",
		"SLD":      "any",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ensureQueryParams(t, r, toURLValues(expectedValues))
		_, err := w.Write([]byte(getHostsResponse))
		if err != nil {
			t.Fatal(err)
		}
	}))
	t.Cleanup(ts.Close)

	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.WithEndpoint(ts.URL), namecheap.WithClientIP("localhost"))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	if _, err := c.GetHosts(context.TODO(), "any.co.uk"); err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

func TestSetHosts(t *testing.T) {
	expected := map[string]string{
		"ApiUser":     "testUser",
		"ApiKey":      "testAPIKey",
		"UserName":    "testUser",
		"ClientIp":    "localhost",
		"Command":     "namecheap.domains.dns.setHosts",
		"TLD":         "com",
		"SLD":         "domain",
		"HostName1":   "first_host",
		"RecordType1": string(namecheap.A),
		"TTL1":        "180",
		"HostName2":   "second_host",
		"RecordType2": string(namecheap.A),
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			ensureQueryParams(t, r, toURLValues(expected))
			w.Write([]byte(setHostsResponse))
		case http.MethodGet:
			w.Write([]byte(emptyHostsResponse))
		}
	}))
	t.Cleanup(ts.Close)
	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.WithEndpoint(ts.URL), namecheap.WithClientIP("localhost"))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	hosts := []namecheap.HostRecord{
		{
			Name:       "first_host",
			RecordType: namecheap.A,
			TTL:        uint16(180),
		},
		{
			Name:       "second_host",
			RecordType: namecheap.A,
		},
	}

	_, err = c.SetHosts(context.TODO(), "domain.com", hosts)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

func TestAddHostsNoExisting(t *testing.T) {
	expectedValues := map[string]string{
		"ApiUser":     "testUser",
		"ApiKey":      "testAPIKey",
		"UserName":    "testUser",
		"ClientIp":    "localhost",
		"Command":     "namecheap.domains.dns.setHosts",
		"TLD":         "com",
		"SLD":         "domain",
		"HostName1":   "first_host",
		"RecordType1": string(namecheap.A),
		"TTL1":        "180",
		"HostName2":   "second_host",
		"RecordType2": string(namecheap.A),
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			ensureQueryParams(t, r, toURLValues(expectedValues))
			w.Write([]byte(setHostsResponse))
		case http.MethodGet:
			w.Write([]byte(emptyHostsResponse))
		}
	}))
	t.Cleanup(ts.Close)

	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.WithEndpoint(ts.URL), namecheap.WithClientIP("localhost"))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	newHosts := []namecheap.HostRecord{
		{
			Name:       "first_host",
			RecordType: namecheap.A,
			TTL:        uint16(180),
		},
		{
			Name:       "second_host",
			RecordType: namecheap.A,
		},
	}
	_, err = c.AddHosts(context.TODO(), "domain.com", newHosts)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

func TestAddHostsWithExisting(t *testing.T) {
	expectedValues := map[string]string{
		"ApiUser":     "testUser",
		"ApiKey":      "testAPIKey",
		"UserName":    "testUser",
		"ClientIp":    "localhost",
		"Command":     "namecheap.domains.dns.setHosts",
		"TLD":         "com",
		"SLD":         "domain",
		"Address1":    "1.2.3.4",
		"MXPref1":     "10",
		"HostName1":   "@",
		"RecordType1": string(namecheap.A),
		"TTL1":        "1800",
		"Address2":    "122.23.3.7",
		"MXPref2":     "10",
		"HostName2":   "www",
		"RecordType2": string(namecheap.A),
		"TTL2":        "1800",
		"HostName3":   "third_host",
		"RecordType3": string(namecheap.A),
		"TTL3":        "180",
		"HostName4":   "fourth_host",
		"RecordType4": string(namecheap.A),
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			ensureQueryParams(t, r, toURLValues(expectedValues))
			w.Write([]byte(setHostsResponse))
		case http.MethodGet:
			w.Write([]byte(getHostsResponse))
		}
	}))
	t.Cleanup(ts.Close)

	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.WithEndpoint(ts.URL), namecheap.WithClientIP("localhost"))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	newHosts := []namecheap.HostRecord{
		{
			Name:       "third_host",
			RecordType: namecheap.A,
			TTL:        uint16(180),
		},
		{
			Name:       "fourth_host",
			RecordType: namecheap.A,
		},
	}
	_, err = c.AddHosts(context.TODO(), "domain.com", newHosts)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
}

func TestGetHostsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(errorResponse))
	}))
	t.Cleanup(ts.Close)

	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.WithEndpoint(ts.URL), namecheap.WithClientIP("localhost"))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	_, err = c.GetHosts(context.TODO(), "any.domain")
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
}

func TestBadURL(t *testing.T) {
	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.WithEndpoint("any"), namecheap.WithClientIP("localhost"))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	_, err = c.GetHosts(context.TODO(), "com")
	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	_, err = c.GetHosts(context.TODO(), "")
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
}

func TestAutoDiscoverIP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "getHosts") {
			if got := r.URL.Query().Get("ClientIp"); got != "127.0.0.1" {
				t.Fatalf("Expected: %s\tGot: %s", "127.0.0.1", got)
			}
		}
		w.Write([]byte("127.0.0.1"))
	}))
	t.Cleanup(ts.Close)

	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.AutoDiscoverPublicIP(), namecheap.WithDiscoveryAddress(ts.URL), namecheap.WithEndpoint(ts.URL))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	c.GetHosts(context.TODO(), "any.domain")
}

func TestDeleteHostsWithExisting(t *testing.T) {
	expectedValues := map[string]string{
		"ApiUser":     "testUser",
		"ApiKey":      "testAPIKey",
		"UserName":    "testUser",
		"ClientIp":    "localhost",
		"Command":     "namecheap.domains.dns.setHosts",
		"TLD":         "com",
		"SLD":         "domain",
		"Address1":    "1.2.3.4",
		"MXPref1":     "10",
		"HostName1":   "@",
		"RecordType1": string(namecheap.A),
		"TTL1":        "1800",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			ensureQueryParams(t, r, toURLValues(expectedValues))
			w.Write([]byte(setHostsResponse))
		case http.MethodGet:
			w.Write([]byte(getHostsResponse))
		}
	}))
	t.Cleanup(ts.Close)

	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.WithEndpoint(ts.URL), namecheap.WithClientIP("localhost"))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	hostsToDelete := []namecheap.HostRecord{
		{
			HostID: "14",
		},
	}
	hosts, err := c.DeleteHosts(context.TODO(), "domain.com", hostsToDelete)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if len(hosts) != 1 {
		t.Fatalf("Expected 1 host. Got: %v", len(hosts))
	}
}

func TestDeleteHostsNoExisting(t *testing.T) {
	expectedValues := map[string]string{
		"ApiUser":     "testUser",
		"ApiKey":      "testAPIKey",
		"UserName":    "testUser",
		"ClientIp":    "localhost",
		"Command":     "namecheap.domains.dns.setHosts",
		"TLD":         "com",
		"SLD":         "domain",
		"Address1":    "1.2.3.4",
		"MXPref1":     "10",
		"HostName1":   "@",
		"RecordType1": string(namecheap.A),
		"TTL1":        "1800",
		"Address2":    "122.23.3.7",
		"MXPref2":     "10",
		"HostName2":   "www",
		"RecordType2": string(namecheap.A),
		"TTL2":        "1800",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			ensureQueryParams(t, r, toURLValues(expectedValues))
			w.Write([]byte(setHostsResponse))
		case http.MethodGet:
			w.Write([]byte(getHostsResponse))
		}
	}))
	t.Cleanup(ts.Close)

	c, err := namecheap.NewClient("testAPIKey", "testUser", namecheap.WithEndpoint(ts.URL), namecheap.WithClientIP("localhost"))
	if err != nil {
		t.Fatalf("Error creating NewClient. Err: %s", err)
	}

	hostsToDelete := []namecheap.HostRecord{
		{
			HostID: "nonexistanthost",
		},
	}
	hosts, err := c.DeleteHosts(context.TODO(), "domain.com", hostsToDelete)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("Expected 2 host. Got: %v", len(hosts))
	}
}
