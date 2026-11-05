package domainchief

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// fakeAPI is an in-memory stand-in for the Domain Chief API.
type fakeAPI struct {
	t        *testing.T
	mu       sync.Mutex
	records  []apiRecord
	nextID   int
	requests []string
	domains  []apiDomain
	perPage  int
	failures map[string][]int
}

func newFakeAPI(t *testing.T, records ...apiRecord) (*fakeAPI, *Provider) {
	api := &fakeAPI{t: t, records: records, nextID: len(records) + 1, perPage: perPage, failures: map[string][]int{}}

	server := httptest.NewServer(api)
	t.Cleanup(server.Close)

	return api, &Provider{APIToken: "ctt_test", BaseURL: server.URL}
}

func (api *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.mu.Lock()
	defer api.mu.Unlock()

	request := r.Method + " " + r.URL.Path
	api.requests = append(api.requests, request)

	if r.Header.Get("Authorization") != "Bearer ctt_test" {
		api.t.Errorf("unexpected authorization header %q", r.Header.Get("Authorization"))
	}

	if statuses := api.failures[request]; len(statuses) > 0 {
		api.failures[request] = statuses[1:]
		w.WriteHeader(statuses[0])

		code := ""
		if statuses[0] == http.StatusConflict {
			code = "dns.zone_busy"
		}

		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Failure.", "code": code})

		return
	}

	switch {
	case r.URL.Path == "/acme/challenges":
		api.serveChallenge(w, r)
	case r.URL.Path == "/domains" && r.Method == http.MethodGet:
		api.paginate(w, r, api.domains)
	case r.URL.Path == "/domains/example.com/dns/records" && r.Method == http.MethodGet:
		api.paginate(w, r, api.records)
	case r.URL.Path == "/domains/example.com/dns/records" && r.Method == http.MethodPost:
		var input recordInput
		_ = json.NewDecoder(r.Body).Decode(&input)

		record := api.store(fmt.Sprintf("rec%d", api.nextID), input)
		api.nextID++
		api.records = append(api.records, record)

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": record})
	case strings.HasPrefix(r.URL.Path, "/domains/example.com/dns/records/"):
		id := strings.TrimPrefix(r.URL.Path, "/domains/example.com/dns/records/")

		for i, record := range api.records {
			if record.ID != id {
				continue
			}

			if r.Method == http.MethodDelete {
				api.records = append(api.records[:i], api.records[i+1:]...)
				w.WriteHeader(http.StatusNoContent)

				return
			}

			var input recordInput
			_ = json.NewDecoder(r.Body).Decode(&input)

			api.records[i] = api.store(id, input)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": api.records[i]})

			return
		}

		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Unable to find record with that ID."}`))
	default:
		api.t.Errorf("unexpected request %s", request)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (api *fakeAPI) serveChallenge(w http.ResponseWriter, r *http.Request) {
	var name, value string

	if r.Method == http.MethodPost {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		name, value = body["name"], body["value"]
	} else {
		name, value = r.URL.Query().Get("name"), r.URL.Query().Get("value")
	}

	recordName := strings.TrimSuffix(strings.TrimSuffix(name, "."), ".example.com")

	for i, record := range api.records {
		if record.Type == "TXT" && record.Name == recordName && record.Value == value {
			if r.Method == http.MethodDelete {
				api.records = append(api.records[:i], api.records[i+1:]...)
				w.WriteHeader(http.StatusNoContent)

				return
			}

			_ = json.NewEncoder(w).Encode(map[string]any{"data": apiChallenge{ID: record.ID, Name: name, Value: value, TTL: record.TTL}})

			return
		}
	}

	if r.Method == http.MethodDelete {
		w.WriteHeader(http.StatusNoContent)

		return
	}

	record := api.store(fmt.Sprintf("rec%d", api.nextID), recordInput{Type: "TXT", Name: recordName, Content: value, TTL: 60})
	api.nextID++
	api.records = append(api.records, record)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": apiChallenge{ID: record.ID, Name: name, Value: value, TTL: 60}})
}

func (api *fakeAPI) store(id string, input recordInput) apiRecord {
	record := apiRecord{ID: id, Type: input.Type, Name: input.Name, Content: input.Content, Value: input.Content, TTL: input.TTL, Prio: input.Prio}

	if input.Type == "TXT" {
		record.Content = strconv.Quote(input.Content)
	}

	return record
}

func paginate[T any](items []T, page, size int) ([]T, int) {
	lastPage := (len(items) + size - 1) / size
	if lastPage == 0 {
		lastPage = 1
	}

	start := min((page-1)*size, len(items))
	end := min(start+size, len(items))

	return items[start:end], lastPage
}

func (api *fakeAPI) paginate(w http.ResponseWriter, r *http.Request, items any) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))

	var data any
	var lastPage int

	switch typed := items.(type) {
	case []apiRecord:
		data, lastPage = paginate(typed, page, api.perPage)
	case []apiDomain:
		data, lastPage = paginate(typed, page, api.perPage)
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"data": data,
		"meta": map[string]int{"current_page": page, "last_page": lastPage},
	})
}

func (api *fakeAPI) snapshot() ([]apiRecord, []string) {
	api.mu.Lock()
	defer api.mu.Unlock()

	return append([]apiRecord(nil), api.records...), append([]string(nil), api.requests...)
}

func TestAppendRecordsPublishesACMEChallengesThroughChallengeEndpoint(t *testing.T) {
	api, provider := newFakeAPI(t)

	appended, err := provider.AppendRecords(context.Background(), "example.com.", []libdns.Record{
		libdns.TXT{Name: "_acme-challenge.www", Text: "gfj9Xq-Rg85nM_"},
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []libdns.Record{libdns.TXT{Name: "_acme-challenge.www", TTL: time.Minute, Text: "gfj9Xq-Rg85nM_"}}
	if !reflect.DeepEqual(appended, expected) {
		t.Errorf("unexpected appended records %#v", appended)
	}

	records, requests := api.snapshot()
	if !reflect.DeepEqual(requests, []string{"POST /acme/challenges"}) {
		t.Errorf("unexpected requests %v", requests)
	}

	if len(records) != 1 || records[0].Name != "_acme-challenge.www" {
		t.Errorf("unexpected records %v", records)
	}
}

func TestAppendRecordsUsesRecordEndpointForOtherTXTValuesOnChallengeNames(t *testing.T) {
	api, provider := newFakeAPI(t)

	if _, err := provider.AppendRecords(context.Background(), "example.com.", []libdns.Record{
		libdns.TXT{Name: "_acme-challenge", Text: "not a challenge value"},
	}); err != nil {
		t.Fatal(err)
	}

	if _, requests := api.snapshot(); !reflect.DeepEqual(requests, []string{"POST /domains/example.com/dns/records"}) {
		t.Errorf("unexpected requests %v", requests)
	}
}

func TestAppendRecordsConvertsRecordsToAPIFormat(t *testing.T) {
	api, provider := newFakeAPI(t)

	appended, err := provider.AppendRecords(context.Background(), "example.com.", []libdns.Record{
		libdns.RR{Name: "@", Type: "A", Data: "192.0.2.1", TTL: 120 * time.Second},
		libdns.MX{Name: "@", Preference: 10, Target: "mail.example.net.", TTL: time.Hour},
		libdns.SRV{Service: "sip", Transport: "tcp", Name: "@", Priority: 5, Weight: 10, Port: 5060, Target: "sip.example.net.", TTL: time.Hour},
		libdns.CNAME{Name: "www", Target: "example.net.", TTL: 0},
	})
	if err != nil {
		t.Fatal(err)
	}

	records, _ := api.snapshot()

	expected := []apiRecord{
		{ID: "rec1", Type: "A", Name: "", Content: "192.0.2.1", Value: "192.0.2.1", TTL: 60},
		{ID: "rec2", Type: "MX", Name: "", Content: "mail.example.net", Value: "mail.example.net", TTL: 3600, Prio: 10},
		{ID: "rec3", Type: "SRV", Name: "_sip._tcp", Content: "10 5060 sip.example.net", Value: "10 5060 sip.example.net", TTL: 3600, Prio: 5},
		{ID: "rec4", Type: "CNAME", Name: "www", Content: "example.net", Value: "example.net", TTL: 3600},
	}
	if !reflect.DeepEqual(records, expected) {
		t.Errorf("unexpected records\n got: %#v\nwant: %#v", records, expected)
	}

	if srv, ok := appended[2].(libdns.SRV); !ok || srv.Service != "sip" || srv.Target != "sip.example.net." || srv.Priority != 5 {
		t.Errorf("unexpected appended SRV record %#v", appended[2])
	}
}

func TestAppendRecordsRejectsUnsupportedTypes(t *testing.T) {
	api, provider := newFakeAPI(t)

	_, err := provider.AppendRecords(context.Background(), "example.com.", []libdns.Record{
		libdns.RR{Name: "@", Type: "HTTPS", Data: "1 . alpn=h2"},
	})
	if err == nil || !strings.Contains(err.Error(), `"HTTPS" is not supported`) {
		t.Errorf("expected unsupported type error, got %v", err)
	}

	if _, requests := api.snapshot(); len(requests) != 0 {
		t.Errorf("expected no requests, got %v", requests)
	}
}

func TestGetRecordsReturnsTypedRecordsAcrossPagesAndKeepsUnparsableRecords(t *testing.T) {
	api, provider := newFakeAPI(t,
		apiRecord{ID: "1", Type: "A", Name: "", Content: "192.0.2.1", Value: "192.0.2.1", TTL: 300},
		apiRecord{ID: "2", Type: "TXT", Name: "", Content: `"v=spf1 -all"`, Value: "v=spf1 -all", TTL: 3600},
		apiRecord{ID: "3", Type: "MX", Name: "", Content: "mail.example.net", Value: "mail.example.net", TTL: 3600, Prio: 10},
		apiRecord{ID: "4", Type: "SRV", Name: "_sip._tcp.voice", Content: "10 5060 sip.example.net", Value: "10 5060 sip.example.net", TTL: 3600, Prio: 5},
		apiRecord{ID: "5", Type: "ALIAS", Name: "", Content: "example.net", Value: "example.net", TTL: 3600},
		apiRecord{ID: "6", Type: "SRV", Name: "_sip._tcp", Content: "unexpected", Value: "unexpected", TTL: 3600},
	)
	api.perPage = 2

	records, err := provider.GetRecords(context.Background(), "example.com.")
	if err != nil {
		t.Fatal(err)
	}

	expected := []libdns.Record{
		libdns.Address{Name: "@", TTL: 5 * time.Minute, IP: mustParseAddr(t, "192.0.2.1")},
		libdns.TXT{Name: "@", TTL: time.Hour, Text: "v=spf1 -all"},
		libdns.MX{Name: "@", TTL: time.Hour, Preference: 10, Target: "mail.example.net."},
		libdns.SRV{Service: "sip", Transport: "tcp", Name: "voice", TTL: time.Hour, Priority: 5, Weight: 10, Port: 5060, Target: "sip.example.net."},
		libdns.RR{Name: "@", TTL: time.Hour, Type: "ALIAS", Data: "example.net."},
		libdns.RR{Name: "_sip._tcp", TTL: time.Hour, Type: "SRV", Data: "unexpected"},
	}
	if !reflect.DeepEqual(records, expected) {
		t.Errorf("unexpected records\n got: %#v\nwant: %#v", records, expected)
	}

	if _, requests := api.snapshot(); len(requests) != 3 {
		t.Errorf("expected 3 page requests, got %v", requests)
	}
}

func TestDeleteRecordsRemovesACMEChallengesThroughChallengeEndpoint(t *testing.T) {
	api, provider := newFakeAPI(t,
		apiRecord{ID: "1", Type: "TXT", Name: "_acme-challenge", Content: `"first"`, Value: "first", TTL: 60},
		apiRecord{ID: "2", Type: "TXT", Name: "_acme-challenge", Content: `"second"`, Value: "second", TTL: 60},
	)

	deleted, err := provider.DeleteRecords(context.Background(), "example.com.", []libdns.Record{
		libdns.TXT{Name: "_acme-challenge", Text: "first", TTL: time.Minute},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(deleted) != 1 {
		t.Errorf("unexpected deleted records %v", deleted)
	}

	records, requests := api.snapshot()
	if !reflect.DeepEqual(requests, []string{"DELETE /acme/challenges"}) {
		t.Errorf("unexpected requests %v", requests)
	}

	if len(records) != 1 || records[0].Value != "second" {
		t.Errorf("expected only the second value to remain, got %v", records)
	}
}

func TestDeleteRecordsOnlyDeletesMatchingRecords(t *testing.T) {
	api, provider := newFakeAPI(t,
		apiRecord{ID: "1", Type: "A", Name: "www", Content: "192.0.2.1", Value: "192.0.2.1", TTL: 300},
		apiRecord{ID: "2", Type: "A", Name: "www", Content: "192.0.2.2", Value: "192.0.2.2", TTL: 300},
		apiRecord{ID: "3", Type: "AAAA", Name: "www", Content: "2001:db8::1", Value: "2001:db8::1", TTL: 300},
		apiRecord{ID: "4", Type: "A", Name: "api", Content: "192.0.2.1", Value: "192.0.2.1", TTL: 300},
		apiRecord{ID: "5", Type: "TXT", Name: "www", Content: `"hello"`, Value: "hello", TTL: 300},
	)

	deleted, err := provider.DeleteRecords(context.Background(), "example.com.", []libdns.Record{
		// Exact match.
		libdns.Address{Name: "www", IP: mustParseAddr(t, "192.0.2.1"), TTL: 5 * time.Minute},
		// TTL mismatch, kept.
		libdns.Address{Name: "www", IP: mustParseAddr(t, "192.0.2.2"), TTL: time.Hour},
		// Empty value matches any AAAA record on the name.
		libdns.RR{Name: "www", Type: "AAAA"},
		// Not in the zone.
		libdns.TXT{Name: "www", Text: "other"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(deleted) != 2 {
		t.Errorf("expected 2 deleted records, got %#v", deleted)
	}

	records, _ := api.snapshot()

	var remaining []string
	for _, record := range records {
		remaining = append(remaining, record.ID)
	}

	if !reflect.DeepEqual(remaining, []string{"2", "4", "5"}) {
		t.Errorf("unexpected remaining records %v", remaining)
	}
}

func TestSetRecordsReconcilesEachRRset(t *testing.T) {
	api, provider := newFakeAPI(t,
		apiRecord{ID: "1", Type: "A", Name: "www", Content: "192.0.2.1", Value: "192.0.2.1", TTL: 3600},
		apiRecord{ID: "2", Type: "A", Name: "www", Content: "192.0.2.2", Value: "192.0.2.2", TTL: 3600},
		apiRecord{ID: "3", Type: "A", Name: "www", Content: "192.0.2.3", Value: "192.0.2.3", TTL: 3600},
		apiRecord{ID: "4", Type: "TXT", Name: "www", Content: `"keep"`, Value: "keep", TTL: 3600},
		apiRecord{ID: "5", Type: "CNAME", Name: "blog", Content: "old.example.net", Value: "old.example.net", TTL: 3600},
	)

	set, err := provider.SetRecords(context.Background(), "example.com.", []libdns.Record{
		libdns.Address{Name: "www", IP: mustParseAddr(t, "192.0.2.1"), TTL: time.Hour},
		libdns.Address{Name: "www", IP: mustParseAddr(t, "192.0.2.9"), TTL: time.Hour},
		libdns.CNAME{Name: "blog", Target: "new.example.net.", TTL: 5 * time.Minute},
		libdns.Address{Name: "api", IP: mustParseAddr(t, "192.0.2.10"), TTL: time.Hour},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(set) != 4 {
		t.Errorf("expected 4 set records, got %#v", set)
	}

	records, requests := api.snapshot()

	var state []string
	for _, record := range records {
		state = append(state, fmt.Sprintf("%s %s %s %s %d", record.ID, record.Name, record.Type, record.Value, record.TTL))
	}
	sort.Strings(state)

	expected := []string{
		"1 www A 192.0.2.1 3600",
		"2 www A 192.0.2.9 3600",
		"4 www TXT keep 3600",
		"5 blog CNAME new.example.net 300",
		"rec6 api A 192.0.2.10 3600",
	}
	if !reflect.DeepEqual(state, expected) {
		t.Errorf("unexpected zone\n got: %v\nwant: %v", state, expected)
	}

	expectedRequests := []string{
		"GET /domains/example.com/dns/records",
		"PUT /domains/example.com/dns/records/2",
		"DELETE /domains/example.com/dns/records/3",
		"PUT /domains/example.com/dns/records/5",
		"POST /domains/example.com/dns/records",
	}
	if !reflect.DeepEqual(requests, expectedRequests) {
		t.Errorf("unexpected requests\n got: %v\nwant: %v", requests, expectedRequests)
	}
}

func TestRequestsAreRetriedWhileTheZoneIsBusy(t *testing.T) {
	api, provider := newFakeAPI(t)
	api.failures["POST /acme/challenges"] = []int{http.StatusConflict}

	if _, err := provider.AppendRecords(context.Background(), "example.com.", []libdns.Record{
		libdns.TXT{Name: "_acme-challenge", Text: "token"},
	}); err != nil {
		t.Fatal(err)
	}

	if _, requests := api.snapshot(); len(requests) != 2 {
		t.Errorf("expected 2 requests, got %v", requests)
	}
}

func TestRecordCreationIsNotRetriedAfterServerErrors(t *testing.T) {
	api, provider := newFakeAPI(t)
	api.failures["POST /domains/example.com/dns/records"] = []int{http.StatusBadGateway}

	_, err := provider.AppendRecords(context.Background(), "example.com.", []libdns.Record{
		libdns.Address{Name: "www", IP: mustParseAddr(t, "192.0.2.1"), TTL: time.Hour},
	})

	var apiErr *APIError
	if !errorsAs(err, &apiErr) || apiErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected API error, got %v", err)
	}

	if _, requests := api.snapshot(); len(requests) != 1 {
		t.Errorf("expected 1 request, got %v", requests)
	}
}

func TestErrorsIncludeValidationMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message":"The domain does not use hosted DNS.","code":"dns.not_hosted"}`))
	}))
	defer server.Close()

	var requests atomic.Int32
	provider := &Provider{APIToken: "ctt_test", BaseURL: server.URL, HTTPClient: &http.Client{Transport: countingTransport{&requests}}}

	_, err := provider.GetRecords(context.Background(), "example.com.")
	if err == nil || err.Error() != "domain chief API error (409, dns.not_hosted): The domain does not use hosted DNS." {
		t.Errorf("unexpected error %v", err)
	}

	if requests.Load() != 1 {
		t.Errorf("expected permanent conflicts not to be retried, got %d requests", requests.Load())
	}
}

func TestListZonesReturnsHostedDomains(t *testing.T) {
	api, provider := newFakeAPI(t)
	api.domains = []apiDomain{
		{Domain: "example.com", IsUsingHostedDNS: true},
		{Domain: "example.net", IsUsingHostedDNS: false},
		{Domain: "example.org", IsUsingHostedDNS: true},
	}
	api.perPage = 2

	zones, err := provider.ListZones(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(zones, []libdns.Zone{{Name: "example.com."}, {Name: "example.org."}}) {
		t.Errorf("unexpected zones %v", zones)
	}
}

func TestMissingAPITokenIsReported(t *testing.T) {
	_, err := (&Provider{}).GetRecords(context.Background(), "example.com.")

	if err == nil || !strings.Contains(err.Error(), "missing API token") {
		t.Errorf("expected missing token error, got %v", err)
	}
}

type countingTransport struct{ count *atomic.Int32 }

func (c countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.count.Add(1)

	return http.DefaultTransport.RoundTrip(r)
}

func mustParseAddr(t *testing.T, value string) netip.Addr {
	t.Helper()

	addr, err := netip.ParseAddr(value)
	if err != nil {
		t.Fatal(err)
	}

	return addr
}

func errorsAs(err error, target any) bool {
	return errors.As(err, target)
}
