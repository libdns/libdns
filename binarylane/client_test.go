package binarylane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// newTestProvider points a Provider at a test server. The handler sees the
// real requests the client builds, so paths, methods, headers and pagination
// are all exercised without credentials.
func newTestProvider(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return &Provider{
		APIToken:   "test-token",
		APIBase:    srv.URL + "/v2/",
		HTTPClient: srv.Client(),
	}
}

func TestClientSendsBearerToken(t *testing.T) {
	var got string

	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(recordsResponse{})
	})

	if _, err := p.listRecords(context.Background(), "example.com"); err != nil {
		t.Fatalf("listRecords() error = %v", err)
	}

	if want := "Bearer test-token"; got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestClientNormalisesAPIBaseWithoutTrailingSlash(t *testing.T) {
	var path string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		json.NewEncoder(w).Encode(recordsResponse{})
	}))
	t.Cleanup(srv.Close)

	// No trailing slash: without normalisation the "v2" segment is discarded
	// when the request path is resolved against it.
	p := &Provider{APIToken: "t", APIBase: srv.URL + "/v2", HTTPClient: srv.Client()}

	if _, err := p.listRecords(context.Background(), "example.com"); err != nil {
		t.Fatalf("listRecords() error = %v", err)
	}

	if want := "/v2/domains/example.com/records"; path != want {
		t.Errorf("request path = %q, want %q", path, want)
	}
}

func TestClientFollowsPagination(t *testing.T) {
	// Two full pages then a short one. per_page records per page, so the client
	// must keep asking until a page comes back short.
	total := perPage*2 + 3

	var pagesServed []string

	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pagesServed = append(pagesServed, r.URL.Query().Get("page"))

		var out recordsResponse
		out.Meta.Total = total

		start := (page - 1) * perPage
		for i := start; i < start+perPage && i < total; i++ {
			out.DomainRecords = append(out.DomainRecords, apiRecord{
				ID: int64(i + 1), Type: "A", Name: fmt.Sprintf("h%d", i), Data: "192.0.2.1", TTL: 60,
			})
		}

		json.NewEncoder(w).Encode(out)
	})

	recs, err := p.listRecords(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("listRecords() error = %v", err)
	}

	if len(recs) != total {
		t.Errorf("got %d records, want %d", len(recs), total)
	}
	if want := "1,2,3"; strings.Join(pagesServed, ",") != want {
		t.Errorf("pages requested = %v, want %s", pagesServed, want)
	}
}

func TestClientStopsOnMiscountedTotal(t *testing.T) {
	// A total larger than the records actually served must not loop forever.
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		var out recordsResponse
		out.Meta.Total = 10_000
		out.DomainRecords = []apiRecord{{ID: 1, Type: "A", Name: "a", Data: "192.0.2.1", TTL: 60}}
		json.NewEncoder(w).Encode(out)
	})

	recs, err := p.listRecords(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("listRecords() error = %v", err)
	}
	if len(recs) != 1 {
		t.Errorf("got %d records, want 1", len(recs))
	}
}

func TestClientRecordPathsAndMethods(t *testing.T) {
	type call struct{ method, path string }

	var calls []call

	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, call{r.Method, r.URL.Path})

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		var in apiRecord
		json.NewDecoder(r.Body).Decode(&in)
		in.ID = 42
		json.NewEncoder(w).Encode(recordResponse{DomainRecord: in})
	})

	ctx := context.Background()
	rec := apiRecord{Type: "TXT", Name: "_acme-challenge", Data: "token", TTL: 120}

	if _, err := p.createRecord(ctx, "example.com", rec); err != nil {
		t.Fatalf("createRecord() error = %v", err)
	}
	if _, err := p.updateRecord(ctx, "example.com", 42, rec); err != nil {
		t.Fatalf("updateRecord() error = %v", err)
	}
	if err := p.deleteRecord(ctx, "example.com", 42); err != nil {
		t.Fatalf("deleteRecord() error = %v", err)
	}

	want := []call{
		{http.MethodPost, "/v2/domains/example.com/records"},
		{http.MethodPut, "/v2/domains/example.com/records/42"},
		{http.MethodDelete, "/v2/domains/example.com/records/42"},
	}

	if len(calls) != len(want) {
		t.Fatalf("got %d calls, want %d: %+v", len(calls), len(want), calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Errorf("call %d = %+v, want %+v", i, calls[i], want[i])
		}
	}
}

func TestClientSendsZeroValuedRDATAFields(t *testing.T) {
	// Priority 0 on an MX and Flags 0 on a CAA are meaningful, so they must be
	// present in the request body rather than omitted.
	var body map[string]any

	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(recordResponse{})
	})

	_, err := p.createRecord(context.Background(), "example.com",
		apiRecord{Type: "MX", Name: "@", Data: "mail.example.com.", Priority: 0, TTL: 3600})
	if err != nil {
		t.Fatalf("createRecord() error = %v", err)
	}

	for _, field := range []string{"priority", "port", "weight", "flags"} {
		if _, present := body[field]; !present {
			t.Errorf("request body omitted %q; zero is a meaningful value", field)
		}
	}
}

func TestClientSurfacesAPIErrors(t *testing.T) {
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"status":422,"title":"Validation failed","detail":"bad record","errors":{"data":["is invalid"]}}`)
	})

	_, err := p.listRecords(context.Background(), "example.com")
	if err == nil {
		t.Fatal("listRecords() returned no error for a 422")
	}

	for _, want := range []string{"422", "Validation failed", "bad record", "is invalid"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
}

func TestClientSurfacesNonJSONErrors(t *testing.T) {
	// A proxy or gateway may return HTML. The status must still surface.
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, "<html>502 Bad Gateway</html>")
	})

	err := p.deleteRecord(context.Background(), "example.com", 1)
	if err == nil {
		t.Fatal("deleteRecord() returned no error for a 502")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("error %q does not mention the status", err.Error())
	}
}

func TestClientRequiresAToken(t *testing.T) {
	p := &Provider{}
	if _, err := p.listRecords(context.Background(), "example.com"); err == nil {
		t.Error("listRecords() with no APIToken returned no error")
	}
}

func TestListZonesPaginatesAndAbsolutises(t *testing.T) {
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		var out domainsResponse
		out.Meta.Total = 2
		out.Domains = []struct {
			Name string `json:"name"`
		}{{Name: "example.com"}, {Name: "example.net"}}
		json.NewEncoder(w).Encode(out)
	})

	zones, err := p.ListZones(context.Background())
	if err != nil {
		t.Fatalf("ListZones() error = %v", err)
	}

	if len(zones) != 2 {
		t.Fatalf("got %d zones, want 2", len(zones))
	}
	for _, z := range zones {
		if !strings.HasSuffix(z.Name, ".") {
			t.Errorf("zone %q is not absolute", z.Name)
		}
	}
}

func TestConcurrentUseIsRaceFree(t *testing.T) {
	// The wiki requires every exported method to be safe for concurrent use.
	// This exists mainly to give `go test -race` something to find: it drives
	// all five interface methods from several goroutines at once, including the
	// lazy HTTPClient creation on first use.
	p := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case strings.HasSuffix(r.URL.Path, "/records") && r.Method == http.MethodGet:
			var out recordsResponse
			out.Meta.Total = 1
			out.DomainRecords = []apiRecord{{ID: 1, Type: "TXT", Name: "x", Data: "v", TTL: 60}}
			json.NewEncoder(w).Encode(out)
		case r.URL.Path == "/v2/domains":
			json.NewEncoder(w).Encode(domainsResponse{})
		default:
			var in apiRecord
			json.NewDecoder(r.Body).Decode(&in)
			in.ID = 7
			json.NewEncoder(w).Encode(recordResponse{DomainRecord: in})
		}
	})

	// Created lazily on first use, so clear it to exercise that path too.
	p.HTTPClient = nil

	ctx := context.Background()
	rec := libdns.TXT{Name: "x", TTL: time.Minute, Text: "v"}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if _, err := p.GetRecords(ctx, "example.com"); err != nil {
				t.Error(err)
			}
			if _, err := p.AppendRecords(ctx, "example.com", []libdns.Record{rec}); err != nil {
				t.Error(err)
			}
			if _, err := p.SetRecords(ctx, "example.com", []libdns.Record{rec}); err != nil {
				t.Error(err)
			}
			if _, err := p.DeleteRecords(ctx, "example.com", []libdns.Record{rec}); err != nil {
				t.Error(err)
			}
			if _, err := p.ListZones(ctx); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}
