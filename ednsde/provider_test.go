package ednsde

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// A realistic ACME key authorization digest: 43 characters, base64url alphabet.
const validValue = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"

// capturedRequest is what the fake eDNS API saw.
type capturedRequest struct {
	Method string
	Token  string
	Body   apiRequest
}

// fakeAPI stands in for https://dns-challenge.edns.de. respond is called for
// each request and writes the canned response; calls counts every request that
// reached the server, which lets tests assert that validation happens *before*
// any HTTP traffic.
type fakeAPI struct {
	server    *httptest.Server
	calls     atomic.Int32
	seen      []capturedRequest
	rawBodies []string
}

func newFakeAPI(t *testing.T, respond func(w http.ResponseWriter, call int)) *fakeAPI {
	t.Helper()
	f := &fakeAPI{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := int(f.calls.Add(1))
		body, _ := io.ReadAll(r.Body)
		var parsed apiRequest
		_ = json.Unmarshal(body, &parsed)
		f.seen = append(f.seen, capturedRequest{
			Method: r.Method,
			Token:  r.Header.Get("X-API-TOKEN"),
			Body:   parsed,
		})
		// Also keep the raw body so tests can assert on field presence/absence.
		f.rawBodies = append(f.rawBodies, string(body))
		respond(w, call)
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeAPI) provider() *Provider {
	return &Provider{APIToken: "TOKEN_ABC", Endpoint: f.server.URL}
}

// success writes a 200 response with the given result code, echoing the request.
func success(resultCode int, result string) func(http.ResponseWriter, int) {
	return func(w http.ResponseWriter, _ int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"status":200,"message":"Success","data":{`+
			`"action":"addChallengeRecord","domain":"winkler.tel",`+
			`"challenge_token":"`+validValue+`",`+
			`"result":"`+result+`","result_code":`+strconv.Itoa(resultCode)+`}}`)
	}
}

// apiError writes a real eDNS error response, where "data" is an empty ARRAY.
func apiErrorResponse(status int, message string) func(http.ResponseWriter, int) {
	return func(w http.ResponseWriter, _ int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"status":`+strconv.Itoa(status)+`,"message":"`+message+`","data":[]}`)
	}
}

func TestAppendRecords(t *testing.T) {
	tests := []struct {
		name        string
		zone        string
		record      libdns.Record
		respond     func(http.ResponseWriter, int)
		wantCalls   int
		wantSubdom  string
		wantOmitted bool // "subdomain" key must be absent from the JSON body
		wantErr     string
		wantOut     int
	}{
		{
			name:       "record added",
			zone:       "winkler.tel.",
			record:     libdns.TXT{Name: "_acme-challenge", Text: validValue},
			respond:    success(1, "Challenge record added"),
			wantCalls:  1,
			wantSubdom: "_acme-challenge",
			wantOut:    1,
		},
		{
			name:       "already exists is still success",
			zone:       "winkler.tel.",
			record:     libdns.TXT{Name: "_acme-challenge", Text: validValue},
			respond:    success(2, "Challenge record already exists"),
			wantCalls:  1,
			wantSubdom: "_acme-challenge",
			wantOut:    1,
		},
		{
			name:       "multi-label name is passed through verbatim",
			zone:       "winkler.tel.",
			record:     libdns.TXT{Name: "_acme-challenge.vault", Text: validValue},
			respond:    success(1, "Challenge record added"),
			wantCalls:  1,
			wantSubdom: "_acme-challenge.vault",
			wantOut:    1,
		},
		{
			// libdns.RelativeName returns "@" for the zone apex; the eDNS API
			// rejects an empty subdomain with 400, so the key must be omitted.
			name:        "apex omits the subdomain key entirely",
			zone:        "winkler.tel.",
			record:      libdns.TXT{Name: "@", Text: validValue},
			respond:     success(1, "Challenge record added"),
			wantCalls:   1,
			wantOmitted: true,
			wantOut:     1,
		},
		{
			name:    "non-TXT record is rejected before any request",
			zone:    "winkler.tel.",
			record:  libdns.RR{Name: "www", Type: "A", Data: "192.0.2.1"},
			wantErr: "only TXT records",
		},
		{
			name:    "value shorter than 10 characters is rejected",
			zone:    "winkler.tel.",
			record:  libdns.TXT{Name: "_acme-challenge", Text: "tooshort"},
			wantErr: "between 10 and 64",
		},
		{
			name:    "value longer than 64 characters is rejected",
			zone:    "winkler.tel.",
			record:  libdns.TXT{Name: "_acme-challenge", Text: strings.Repeat("x", 65)},
			wantErr: "between 10 and 64",
		},
		{
			name:    "value containing whitespace is rejected",
			zone:    "winkler.tel.",
			record:  libdns.TXT{Name: "_acme-challenge", Text: "abcde 12345xyz"},
			wantErr: "whitespace",
		},
		{
			name:    "label starting with a hyphen is rejected",
			zone:    "winkler.tel.",
			record:  libdns.TXT{Name: "-sub", Text: validValue},
			wantErr: "hyphen",
		},
		{
			name:    "trailing dot in the name is rejected",
			zone:    "winkler.tel.",
			record:  libdns.TXT{Name: "sub.", Text: validValue},
			wantErr: "dot",
		},
		{
			name:    "empty zone is rejected",
			zone:    "",
			record:  libdns.TXT{Name: "_acme-challenge", Text: validValue},
			wantErr: "zone",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			respond := tc.respond
			if respond == nil {
				respond = func(w http.ResponseWriter, _ int) { t.Error("server must not be called") }
			}
			api := newFakeAPI(t, respond)
			p := api.provider()

			out, err := p.AppendRecords(context.Background(), tc.zone, []libdns.Record{tc.record})

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q, got %q", tc.wantErr, err.Error())
				}
				if got := int(api.calls.Load()); got != tc.wantCalls {
					t.Errorf("want %d HTTP calls, got %d", tc.wantCalls, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := int(api.calls.Load()); got != tc.wantCalls {
				t.Fatalf("want %d HTTP calls, got %d", tc.wantCalls, got)
			}
			if len(out) != tc.wantOut {
				t.Fatalf("want %d records back, got %d", tc.wantOut, len(out))
			}

			req := api.seen[0]
			if req.Method != http.MethodPost {
				t.Errorf("want POST, got %s", req.Method)
			}
			if req.Token != "TOKEN_ABC" {
				t.Errorf("want X-API-TOKEN header, got %q", req.Token)
			}
			if req.Body.Action != actionAdd {
				t.Errorf("want action %q, got %q", actionAdd, req.Body.Action)
			}
			if req.Body.Domain != "winkler.tel" {
				t.Errorf("want domain without trailing dot, got %q", req.Body.Domain)
			}
			if req.Body.ChallengeToken != validValue {
				t.Errorf("want challenge_token %q, got %q", validValue, req.Body.ChallengeToken)
			}
			if tc.wantOmitted {
				if strings.Contains(api.rawBodies[0], "subdomain") {
					t.Errorf("want no subdomain key in body, got %s", api.rawBodies[0])
				}
			} else if req.Body.Subdomain != tc.wantSubdom {
				t.Errorf("want subdomain %q, got %q", tc.wantSubdom, req.Body.Subdomain)
			}

			// Returned records must be the concrete libdns.TXT type carrying
			// the TTL that eDNS actually applies.
			txt, ok := out[0].(libdns.TXT)
			if !ok {
				t.Fatalf("want libdns.TXT back, got %T", out[0])
			}
			if txt.TTL != ednsTTL {
				t.Errorf("want TTL %v, got %v", ednsTTL, txt.TTL)
			}
			if txt.Text != validValue {
				t.Errorf("want text %q, got %q", validValue, txt.Text)
			}
		})
	}
}

func TestDeleteRecords(t *testing.T) {
	tests := []struct {
		name      string
		record    libdns.Record
		respond   func(http.ResponseWriter, int)
		wantCalls int
		wantOut   int
		wantErr   string
	}{
		{
			name:      "record removed is returned",
			record:    libdns.TXT{Name: "_acme-challenge", Text: validValue},
			respond:   success(3, "Challenge record removed"),
			wantCalls: 1,
			wantOut:   1,
		},
		{
			name:      "already removed from DNS is returned",
			record:    libdns.TXT{Name: "_acme-challenge", Text: validValue},
			respond:   success(4, "Challenge record has already been removed from DNS"),
			wantCalls: 1,
			wantOut:   1,
		},
		{
			// Nothing was deleted, so libdns says it must not appear in the
			// output -- but it is not an error either, so that cleanup after a
			// failed challenge stays quiet.
			name:      "not found is neither returned nor an error",
			record:    libdns.TXT{Name: "_acme-challenge", Text: validValue},
			respond:   success(5, "Challenge record not found (or not set via API)"),
			wantCalls: 1,
			wantOut:   0,
		},
		{
			// libdns allows an empty value to mean "delete everything matching
			// the name". The eDNS API has no listing endpoint, so we cannot.
			name:    "empty value cannot be supported",
			record:  libdns.TXT{Name: "_acme-challenge", Text: ""},
			wantErr: "cannot delete",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			respond := tc.respond
			if respond == nil {
				respond = func(w http.ResponseWriter, _ int) { t.Error("server must not be called") }
			}
			api := newFakeAPI(t, respond)
			p := api.provider()

			out, err := p.DeleteRecords(context.Background(), "winkler.tel.", []libdns.Record{tc.record})

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := int(api.calls.Load()); got != tc.wantCalls {
				t.Errorf("want %d HTTP calls, got %d", tc.wantCalls, got)
			}
			if len(out) != tc.wantOut {
				t.Fatalf("want %d records back, got %d", tc.wantOut, len(out))
			}
			if api.seen[0].Body.Action != actionRemove {
				t.Errorf("want action %q, got %q", actionRemove, api.seen[0].Body.Action)
			}
		})
	}
}

func TestAPIErrors(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		message   string
		wantCalls int
		wantHint  string
	}{
		{
			name:      "401 does not retry and explains both causes",
			status:    http.StatusUnauthorized,
			message:   "Invalid token or token not assigned to domain/challenge",
			wantCalls: 1,
			wantHint:  "assigned",
		},
		{
			name:      "400 does not retry",
			status:    http.StatusBadRequest,
			message:   "Missing or invalid parameter: domain",
			wantCalls: 1,
		},
		{
			name:      "405 does not retry",
			status:    http.StatusMethodNotAllowed,
			message:   "Method Not Allowed",
			wantCalls: 1,
		},
		{
			name:      "500 is retried up to three times",
			status:    http.StatusInternalServerError,
			message:   "Internal Server Error",
			wantCalls: maxAttempts,
		},
		{
			name:      "429 is retried up to three times",
			status:    http.StatusTooManyRequests,
			message:   "Too Many Requests",
			wantCalls: maxAttempts,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			api := newFakeAPI(t, apiErrorResponse(tc.status, tc.message))
			p := api.provider()

			_, err := p.AppendRecords(context.Background(), "winkler.tel.",
				[]libdns.Record{libdns.TXT{Name: "_acme-challenge", Text: validValue}})

			if err == nil {
				t.Fatal("want an error, got nil")
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("want *APIError, got %T: %v", err, err)
			}
			if apiErr.StatusCode != tc.status {
				t.Errorf("want status %d, got %d", tc.status, apiErr.StatusCode)
			}
			if !strings.Contains(err.Error(), tc.message) {
				t.Errorf("want the API message in the error, got %q", err.Error())
			}
			if tc.wantHint != "" && !strings.Contains(err.Error(), tc.wantHint) {
				t.Errorf("want a hint containing %q, got %q", tc.wantHint, err.Error())
			}
			if got := int(api.calls.Load()); got != tc.wantCalls {
				t.Errorf("want %d HTTP calls, got %d", tc.wantCalls, got)
			}
		})
	}
}

func TestRetryRecovers(t *testing.T) {
	api := newFakeAPI(t, func(w http.ResponseWriter, call int) {
		if call == 1 {
			apiErrorResponse(http.StatusInternalServerError, "Internal Server Error")(w, call)
			return
		}
		success(1, "Challenge record added")(w, call)
	})
	p := api.provider()

	out, err := p.AppendRecords(context.Background(), "winkler.tel.",
		[]libdns.Record{libdns.TXT{Name: "_acme-challenge", Text: validValue}})
	if err != nil {
		t.Fatalf("want the retry to succeed, got %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("want 1 record, got %d", len(out))
	}
	if got := int(api.calls.Load()); got != 2 {
		t.Errorf("want 2 HTTP calls, got %d", got)
	}
}

func TestContextCancellation(t *testing.T) {
	api := newFakeAPI(t, func(w http.ResponseWriter, _ int) {
		time.Sleep(2 * time.Second)
		success(1, "Challenge record added")(w, 1)
	})
	p := api.provider()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := p.AppendRecords(ctx, "winkler.tel.",
		[]libdns.Record{libdns.TXT{Name: "_acme-challenge", Text: validValue}})
	if err == nil {
		t.Fatal("want a context error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("want context.DeadlineExceeded, got %v", err)
	}
}

func TestDefaultEndpointIsUsedWhenUnset(t *testing.T) {
	p := &Provider{APIToken: "x"}
	if got := p.endpoint(); got != DefaultEndpoint {
		t.Errorf("want %q, got %q", DefaultEndpoint, got)
	}
}

// The provider must satisfy exactly the two interfaces certmagic needs, and no
// more -- see docs/adr/0001. This is a compile-time assertion; the negative
// half of the claim is enforced by the absence of the methods.
var (
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
