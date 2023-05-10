// desc_test is an integration test for the desec provider, to run it a deSEC token is required.
//
// Run it using
//
//	go test . -token=<deSEC token> -domain=<test domain>
//
// The <test domain> must not exist prior to running the test. This is mainly to protect against
// modifications of domains that are in use already.
package desec_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/libdns/desec"
	"github.com/libdns/libdns"
)

var (
	token  = flag.String("token", "", "deSEC token")
	domain = flag.String("domain", "", "Domain to test with of the form sld.tld, it must not exist prior to running this test")
)

var sortRecords = cmpopts.SortSlices(func(x, y libdns.Record) bool {
	if v := strings.Compare(x.Name, y.Name); v != 0 {
		return v < 0
	}
	if v := strings.Compare(x.Type, y.Type); v != 0 {
		return v < 0
	}
	if v := strings.Compare(x.Value, y.Value); v != 0 {
		return v < 0
	}
	if x.Priority != y.Priority {
		return x.Priority < y.Priority
	}
	if x.Weight != y.Weight {
		return x.Weight < y.Weight
	}
	return false
})

func httpDo(ctx context.Context, t *testing.T, method, url string, in []byte) ([]byte, int) {
	t.Helper()

	var r io.Reader
	if len(in) > 0 {
		r = bytes.NewReader(in)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, r)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Token "+*token)
	req.Header.Set("Accept", "application/json; charset=utf-8")
	if len(in) > 0 {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return body, res.StatusCode
}

func putRRSets(ctx context.Context, t *testing.T, domain, content string) {
	t.Helper()

	url := fmt.Sprintf("https://desec.io/api/v1/domains/%s/rrsets/", url.PathEscape(domain))
	body, status := httpDo(ctx, t, "PUT", url, []byte(content))
	switch status {
	case http.StatusOK:
		// success
	default:
		t.Fatalf("unexpected status code: %d: %v", status, string(body))
		panic("never reached")
	}
}

func domainExists(ctx context.Context, t *testing.T, domain string) bool {
	t.Helper()

	url := fmt.Sprintf("https://desec.io/api/v1/domains/%s/", url.PathEscape(domain))
	body, status := httpDo(ctx, t, "GET", url, nil)
	switch status {
	case http.StatusOK:
		return true
	case http.StatusNotFound:
		return false
	default:
		t.Fatalf("unexpected status code: %d: %v", status, string(body))
		panic("never reached")
	}
}

func createDomain(ctx context.Context, t *testing.T, domain string) {
	t.Helper()

	payload, err := json.Marshal(struct {
		Name string `json:"name"`
	}{
		Name: domain,
	})
	if err != nil {
		t.Fatal(err)
	}

	url := "https://desec.io/api/v1/domains/"
	body, status := httpDo(ctx, t, "POST", url, payload)
	switch status {
	case http.StatusCreated:
		// success
	default:
		t.Fatalf("unexpected status code: %d: %v", status, string(body))
		panic("never reached")
	}
}

func deleteDomain(ctx context.Context, t *testing.T, domain string) {
	t.Helper()

	url := fmt.Sprintf("https://desec.io/api/v1/domains/%s/", url.PathEscape(domain))
	body, status := httpDo(ctx, t, "DELETE", url, nil)
	switch status {
	case http.StatusNoContent:
		// success
	default:
		t.Fatalf("unexpected status code: %d: %v", status, string(body))
		panic("never reached")
	}
}

// setup performs all test setup
//   - skip the test if -domain or -token are not provided
//   - fail if the domain provided by -domain exists already
//   - create the domain and setup the deletion of the domain at the end of the test
//   - ensure that the domain contains only the specified rrsets
//   - returns a context that can be used in the test
func setup(t *testing.T, rrsets string) context.Context {
	t.Helper()

	if *token == "" || *domain == "" {
		t.Skip("skipping integration test; both -token and -domain must be set")
	}

	ctx := context.Background()
	if deadline, ok := t.Deadline(); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		t.Cleanup(cancel)
	}

	if domainExists(ctx, t, *domain) {
		t.Fatalf("domain %q exists, but it must not; either the domain was created outside of this test or something in this test went wrong", *domain)
	}
	createDomain(ctx, t, *domain)
	t.Cleanup(func() { deleteDomain(ctx, t, *domain) })

	// A freshly created domain contains a default NS record. To make sure the domain only has
	// the rrsets specified in the call to setup we need to delete them first
	putRRSets(ctx, t, *domain, `[{"subname": "", "type": "NS", "ttl": 3600, "records": []}]`)
	putRRSets(ctx, t, *domain, rrsets)

	return ctx
}

func TestGetRecords(t *testing.T) {
	ctx := setup(t, `[
		{"subname": "", "type": "NS", "ttl": 3600, "records": []},
		{"subname": "", "type": "A", "ttl": 3601, "records": ["127.0.0.3"]},
		{"subname": "www", "type": "A", "ttl": 3600, "records": ["127.0.0.1", "127.0.0.2"]},
		{"subname": "www", "type": "HTTPS", "ttl": 3600, "records": ["1 . alpn=\"h2\""]},
		{"subname": "", "type": "MX", "ttl": 3600, "records": ["0 mx0.example.com.", "10 mx1.example.com."]},
		{"subname": "_sip._tcp", "type": "SRV", "ttl": 3600, "records": ["1 100 5061 sip.example.com."]},
		{"subname": "_ftp._tcp", "type": "URI", "ttl": 3600, "records": ["1 2 \"ftp://example.com/arst\""]},
		{"subname": "", "type": "TXT", "ttl": 3600, "records": ["\"hello dns!\""]}
	]`)

	p := &desec.Provider{
		Token: *token,
	}

	want := []libdns.Record{
		{
			Type:  "A",
			Name:  "@",
			Value: `127.0.0.3`,
			TTL:   time.Second * 3601,
		},
		{
			Type:  "TXT",
			Name:  "@",
			Value: `"hello dns!"`,
			TTL:   time.Second * 3600,
		},
		{
			Type:  "A",
			Name:  "www",
			Value: `127.0.0.1`,
			TTL:   3600 * time.Second,
		},
		{
			Type:     "HTTPS",
			Name:     "www",
			Value:    `. alpn=h2`,
			TTL:      3600 * time.Second,
			Priority: 1,
		},
		{
			Type:  "A",
			Name:  "www",
			Value: `127.0.0.2`,
			TTL:   3600 * time.Second,
		},
		{
			Type:     "MX",
			Name:     "@",
			Value:    `mx0.example.com.`,
			TTL:      3600 * time.Second,
			Priority: 0,
		},
		{
			Type:     "MX",
			Name:     "@",
			Value:    `mx1.example.com.`,
			TTL:      3600 * time.Second,
			Priority: 10,
		},
		{
			Type:     "SRV",
			Name:     "_sip._tcp",
			Value:    `5061 sip.example.com.`,
			TTL:      3600 * time.Second,
			Priority: 1,
			Weight:   100,
		},
		{
			Type:     "URI",
			Name:     "_ftp._tcp",
			Value:    `"ftp://example.com/arst"`,
			TTL:      3600 * time.Second,
			Priority: 1,
			Weight:   2,
		},
	}

	got, err := p.GetRecords(ctx, *domain)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(want, got, sortRecords); diff != "" {
		t.Fatalf("p.GetRecords() unexpected diff [-want +got]: %s", diff)
	}
}

func TestSetRecords(t *testing.T) {
	ctx := setup(t, `[
		{"subname": "www", "type": "A", "ttl": 3600, "records": ["127.0.1.1", "127.0.1.2"]},
		{"subname": "", "type": "TXT", "ttl": 3600, "records": ["\"will be overridden\""]},
		{"subname": "www", "type": "HTTPS", "ttl": 3600, "records": ["1 . alpn=\"h2\""]},
		{"subname": "_sip._tcp", "type": "SRV", "ttl": 3600, "records": ["1 100 5061 sip.example.com."]},
		{"subname": "_ftp._tcp", "type": "URI", "ttl": 3600, "records": ["1 2 \"ftp://example.com/arst\""]},
		{"subname": "", "type": "MX", "ttl": 3600, "records": ["0 mx0.example.com.", "10 mx1.example.com."]}
	]`)

	p := &desec.Provider{
		Token: *token,
	}

	records := []libdns.Record{
		{
			Type:  "A",
			Name:  "@",
			Value: `127.0.0.3`,
			TTL:   time.Second * 3601,
		},
		{
			Type:  "TXT",
			Name:  "@",
			Value: `"hello dns!"`,
			TTL:   time.Second * 3600,
		},
		{
			Type:  "A",
			Name:  "www",
			Value: `127.0.0.1`,
			TTL:   3600 * time.Second,
		},
		{
			Type:     "HTTPS",
			Name:     "www",
			Value:    `. alpn=h2`,
			TTL:      3600 * time.Second,
			Priority: 1,
		},
		{
			Type:  "A",
			Name:  "www",
			Value: `127.0.0.2`,
			TTL:   3600 * time.Second,
		},
		{
			Type:     "MX",
			Name:     "@",
			Value:    `mx0.example.com.`,
			TTL:      3600 * time.Second,
			Priority: 0,
		},
		{
			Type:     "MX",
			Name:     "@",
			Value:    `mx1.example.com.`,
			TTL:      3600 * time.Second,
			Priority: 10,
		},
		{
			Type:     "SRV",
			Name:     "_sip._tcp",
			Value:    `5061 sip.example.com.`,
			TTL:      3600 * time.Second,
			Priority: 1,
			Weight:   100,
		},
		{
			Type:     "URI",
			Name:     "_ftp._tcp",
			Value:    `"ftp://example.com/arst"`,
			TTL:      3600 * time.Second,
			Priority: 1,
			Weight:   2,
		},
	}

	created, err := p.SetRecords(ctx, *domain, records)
	if err != nil {
		t.Fatal(err)
	}

	// The records that already existed are not returned by SetRecords.
	wantCreated := []libdns.Record{
		{
			Type:  "A",
			Name:  "@",
			Value: `127.0.0.3`,
			TTL:   time.Second * 3601,
		},
		{
			Type:  "TXT",
			Name:  "@",
			Value: `"hello dns!"`,
			TTL:   time.Second * 3600,
		},
		{
			Type:  "A",
			Name:  "www",
			Value: `127.0.0.1`,
			TTL:   3600 * time.Second,
		},
		{
			Type:  "A",
			Name:  "www",
			Value: `127.0.0.2`,
			TTL:   3600 * time.Second,
		},
	}
	if diff := cmp.Diff(wantCreated, created, sortRecords); diff != "" {
		t.Fatalf("p.SetRecords() unexpected diff [-want +got]: %s", diff)
	}

	got, err := p.GetRecords(ctx, *domain)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(records, got, sortRecords); diff != "" {
		t.Fatalf("p.GetRecords() unexpected diff [-want +got]: %s", diff)
	}
}

func TestAppendRecords(t *testing.T) {
	ctx := setup(t, `[
		{"subname": "www", "type": "A", "ttl": 3600, "records": ["127.0.0.1"]},
		{"subname": "", "type": "TXT", "ttl": 3600, "records": ["\"hello dns!\""]}
	]`)

	p := &desec.Provider{
		Token: *token,
	}

	append := []libdns.Record{
		{
			Type:  "A",
			Name:  "@",
			Value: `127.0.0.3`,
			TTL:   time.Second * 3601,
		},
		{
			Type:  "TXT",
			Name:  "@",
			Value: `"hello dns!"`,
			TTL:   time.Second * 3600,
		},
		{
			Type:  "A",
			Name:  "www",
			Value: `127.0.0.2`,
			TTL:   3600 * time.Second,
		},
	}

	added, err := p.AppendRecords(ctx, *domain, append)
	if err != nil {
		t.Fatal(err)
	}

	wantAdded := []libdns.Record{
		{
			Type:  "A",
			Name:  "@",
			Value: `127.0.0.3`,
			TTL:   time.Second * 3601,
		},
		{
			Type:  "A",
			Name:  "www",
			Value: `127.0.0.2`,
			TTL:   3600 * time.Second,
		},
	}
	if diff := cmp.Diff(added, wantAdded, sortRecords); diff != "" {
		t.Fatalf("p.SetRecords() unexpected diff [-want +got]: %s", diff)
	}

	got, err := p.GetRecords(ctx, *domain)
	if err != nil {
		t.Fatal(err)
	}

	want := []libdns.Record{
		{
			Type:  "A",
			Name:  "@",
			Value: `127.0.0.3`,
			TTL:   time.Second * 3601,
		},
		{
			Type:  "TXT",
			Name:  "@",
			Value: `"hello dns!"`,
			TTL:   time.Second * 3600,
		},
		{
			Type:  "A",
			Name:  "www",
			Value: `127.0.0.1`,
			TTL:   3600 * time.Second,
		},
		{
			Type:  "A",
			Name:  "www",
			Value: `127.0.0.2`,
			TTL:   3600 * time.Second,
		},
	}
	if diff := cmp.Diff(want, got, sortRecords); diff != "" {
		t.Fatalf("p.GetRecords() unexpected diff [-want +got]: %s", diff)
	}
}

func TestDeleteRecords(t *testing.T) {
	ctx := setup(t, `[
		{"subname": "www", "type": "A", "ttl": 3600, "records": ["127.0.0.1"]},
		{"subname": "", "type": "TXT", "ttl": 3600, "records": ["\"hello dns!\""]}
	]`)

	p := &desec.Provider{
		Token: *token,
	}

	delete := []libdns.Record{
		{
			Type:  "A",
			Name:  "@",
			Value: `127.0.0.3`,
			TTL:   time.Second * 3601,
		},
		{
			Type:  "TXT",
			Name:  "@",
			Value: `"hello dns!"`,
			TTL:   time.Second * 3600,
		},
	}

	deleted, err := p.DeleteRecords(ctx, *domain, delete)
	if err != nil {
		t.Fatal(err)
	}

	wantDeleted := []libdns.Record{
		{
			Type:  "TXT",
			Name:  "@",
			Value: `"hello dns!"`,
			TTL:   time.Second * 3600,
		},
	}
	if diff := cmp.Diff(deleted, wantDeleted, sortRecords); diff != "" {
		t.Fatalf("p.SetRecords() unexpected diff [-want +got]: %s", diff)
	}

	got, err := p.GetRecords(ctx, *domain)
	if err != nil {
		t.Fatal(err)
	}

	want := []libdns.Record{
		{
			Type:  "A",
			Name:  "www",
			Value: `127.0.0.1`,
			TTL:   3600 * time.Second,
		},
	}
	if diff := cmp.Diff(want, got, sortRecords); diff != "" {
		t.Fatalf("p.GetRecords() unexpected diff [-want +got]: %s", diff)
	}
}
