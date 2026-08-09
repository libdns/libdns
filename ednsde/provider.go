// Package ednsde implements a [libdns] provider for the eDNS DNS-01 challenge
// API offered by edns.de at https://dns-challenge.edns.de.
//
// The API is purpose-built for ACME DNS-01 challenges: it can add and remove
// TXT challenge records and does nothing else. In particular it has no endpoint
// that lists the records of a zone, so [libdns.RecordGetter] cannot be
// implemented honestly and [libdns.RecordSetter] cannot be implemented at all.
// Those methods are therefore deliberately absent rather than present-and-
// failing, so that misuse is caught by the compiler instead of during a
// certificate request. Only [libdns.RecordAppender] and [libdns.RecordDeleter]
// are provided, which is exactly what certmagic and Caddy require.
//
// The access token must be assigned to the zone on that zone's
// "DNS-01-Challenge" tab in the eDNS web interface, otherwise every request for
// it is answered with 401.
//
// [libdns]: https://github.com/libdns/libdns
package ednsde

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/libdns/libdns"
)

// The provider implements exactly the two interfaces certmagic needs. The
// absence of RecordGetter and RecordSetter is intentional; see the package doc.
var (
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)

// defaultHTTPClient is shared by all Providers that do not bring their own.
var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

// Provider manages ACME challenge records through the eDNS challenge API.
//
// A Provider is safe for concurrent use. Its fields are read-only once
// constructed and must not be modified afterwards.
type Provider struct {
	// APIToken is the eDNS access token, sent as the X-API-TOKEN header. It
	// must be assigned to every zone this Provider is used for.
	APIToken string `json:"api_token,omitempty"`

	// Endpoint overrides the API base URL. It exists so tests can point the
	// Provider at a stub server; leave it empty to use [DefaultEndpoint].
	//
	// It is deliberately excluded from JSON, and the Caddy module does not
	// parse it either: a redirectable API URL in the certificate path is
	// configuration surface that buys nobody anything.
	Endpoint string `json:"-"`

	// HTTPClient overrides the HTTP client used for API calls.
	HTTPClient *http.Client `json:"-"`
}

func (p *Provider) endpoint() string {
	if p.Endpoint != "" {
		return strings.TrimRight(p.Endpoint, "/")
	}
	return DefaultEndpoint
}

func (p *Provider) httpClient() *http.Client {
	if p.HTTPClient != nil {
		return p.HTTPClient
	}
	return defaultHTTPClient
}

// AppendRecords creates the given challenge records in the zone and returns
// those that are now present.
//
// Only TXT records are accepted. The TTL of the input is ignored: eDNS fixes
// challenge records at 300 seconds and offers no way to change that, so the
// returned records carry that value rather than the requested one.
//
// A record that already exists is reported as created, because the resulting
// zone state is the one that was asked for.
func (p *Provider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	return p.apply(ctx, actionAdd, zone, recs)
}

// DeleteRecords removes the given challenge records from the zone and returns
// those that were actually deleted.
//
// Records that no longer exist are silently skipped: they are left out of the
// returned slice, as libdns requires, but are not treated as an error, so that
// cleaning up after a failed challenge stays quiet.
//
// Unlike the general libdns contract, the value of a record may not be left
// empty. Deleting "everything with this name" would require reading the zone
// first, and the eDNS challenge API cannot list records. eDNS also only removes
// records that were created through this same API; records added by hand in the
// web interface are reported as not found.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	return p.apply(ctx, actionRemove, zone, recs)
}

// apply runs one action over all records, one API call each. On the first
// failure it returns the records handled so far together with the error; the
// eDNS API has no transactions, so a partial result is the honest outcome.
func (p *Provider) apply(ctx context.Context, action, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	domain, err := normalizeZone(zone)
	if err != nil {
		return nil, err
	}

	out := make([]libdns.Record, 0, len(recs))
	for _, rec := range recs {
		name, value, err := challengeFrom(rec, action)
		if err != nil {
			return out, err
		}
		subdomain, err := subdomainFor(name)
		if err != nil {
			return out, err
		}

		data, err := p.do(ctx, apiRequest{
			Action:         action,
			Domain:         domain,
			Subdomain:      subdomain,
			ChallengeToken: value,
		})
		if err != nil {
			return out, err
		}

		applied, err := interpret(action, data)
		if err != nil {
			return out, err
		}
		if applied {
			out = append(out, libdns.TXT{Name: name, TTL: ednsTTL, Text: value})
		}
	}
	return out, nil
}

// interpret turns a result code into "did the zone end up as requested".
func interpret(action string, data *apiData) (bool, error) {
	switch action {
	case actionAdd:
		switch data.ResultCode {
		case resultAdded, resultAlreadyExists:
			return true, nil
		}
	case actionRemove:
		switch data.ResultCode {
		case resultRemoved, resultAlreadyRemoved:
			return true, nil
		case resultNotFound:
			return false, nil
		}
	}
	return false, fmt.Errorf("edns api: unexpected result_code %d (%q) for %s",
		data.ResultCode, data.Result, action)
}

// normalizeZone turns a libdns zone into the "domain" the API expects: libdns
// zones are fully qualified and carry a trailing dot, eDNS wants them without.
func normalizeZone(zone string) (string, error) {
	domain := strings.TrimSuffix(strings.TrimSpace(zone), ".")
	if domain == "" {
		return "", fmt.Errorf("zone must not be empty")
	}
	return domain, nil
}

// challengeFrom extracts the record name and challenge value, rejecting
// anything the eDNS challenge API cannot express.
func challengeFrom(rec libdns.Record, action string) (name, value string, err error) {
	rr := rec.RR()
	if rr.Type != "TXT" {
		return "", "", fmt.Errorf(
			"the eDNS challenge API serves only TXT records, got %q for name %q", rr.Type, rr.Name)
	}
	if action == actionRemove && rr.Data == "" {
		return "", "", fmt.Errorf(
			"cannot delete record %q without its exact value: the eDNS challenge API cannot list records", rr.Name)
	}
	if err := validateValue(rr.Data); err != nil {
		return "", "", err
	}
	return rr.Name, rr.Data, nil
}

func validateValue(value string) error {
	if n := len(value); n < minValueLen || n > maxValueLen {
		return fmt.Errorf("challenge value must be between %d and %d characters, got %d",
			minValueLen, maxValueLen, n)
	}
	if strings.ContainsFunc(value, unicode.IsSpace) {
		return fmt.Errorf("challenge value must not contain whitespace")
	}
	return nil
}

// subdomainFor maps a libdns record name to the API's "subdomain" parameter.
// The zone apex -- which libdns.RelativeName reports as "@" -- has no
// subdomain at all, and must be sent as an absent field rather than an empty
// one, which the API rejects with 400.
func subdomainFor(name string) (string, error) {
	if name == "" || name == "@" {
		return "", nil
	}
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
		return "", fmt.Errorf("record name %q must not begin or end with a dot", name)
	}
	for _, label := range strings.Split(name, ".") {
		switch {
		case label == "":
			return "", fmt.Errorf("record name %q contains an empty label", name)
		case len(label) > 63:
			return "", fmt.Errorf("label %q of record name %q is longer than 63 characters", label, name)
		case strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-"):
			return "", fmt.Errorf("label %q of record name %q must not begin or end with a hyphen", label, name)
		case strings.ContainsFunc(label, unicode.IsSpace):
			return "", fmt.Errorf("record name %q must not contain whitespace", name)
		}
	}
	return name, nil
}
