// Package binarylane implements the libdns interfaces for Binary Lane
// (https://www.binarylane.com.au), an Australian cloud provider whose DNS is
// managed through the Domains endpoints of its public API.
package binarylane

import (
	"context"
	"net/http"
	"sync"

	"github.com/libdns/libdns"
)

// Provider implements the libdns interfaces for Binary Lane.
type Provider struct {
	// APIToken is a Binary Lane API token, created under Account → API in the
	// Binary Lane control panel. Required.
	APIToken string `json:"api_token,omitempty"`

	// APIBase optionally overrides the API endpoint, including the version
	// segment. A trailing slash is added if absent. Defaults to
	// DefaultAPIBase.
	APIBase string `json:"api_base,omitempty"`

	// HTTPClient optionally supplies the client used for API calls, for
	// callers that need their own transport, proxy or timeout. A client with a
	// default timeout is created on first use when this is nil.
	HTTPClient *http.Client `json:"-"`

	// mu guards HTTPClient's lazy creation and serialises the
	// read-modify-write cycles in SetRecords and DeleteRecords, which the API
	// cannot perform atomically.
	mu sync.Mutex
}

// GetRecords lists all the records in the zone.
//
// Records of types libdns does not model — Binary Lane also serves SOA — are
// returned as opaque libdns.RR values.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	api, err := p.listRecords(ctx, unFQDN(zone))
	if err != nil {
		return nil, err
	}

	recs := make([]libdns.Record, 0, len(api))
	for _, rec := range api {
		recs = append(recs, toLibdns(rec))
	}

	return recs, nil
}

// AppendRecords creates the given records in the zone and returns them. It
// never modifies an existing record.
//
// Binary Lane offers no batch endpoint, so this is not atomic: if a later
// record fails, earlier ones remain created.
func (p *Provider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	zone = unFQDN(zone)

	created := make([]libdns.Record, 0, len(recs))
	for _, rec := range recs {
		req, err := toAPI(rec)
		if err != nil {
			return created, err
		}

		out, err := p.createRecord(ctx, zone, req)
		if err != nil {
			return created, err
		}

		created = append(created, toLibdns(out))
	}

	return created, nil
}

// SetRecords makes the zone match the input: for every (name, type) pair
// present in recs, the records given are afterwards the only members of that
// RRset. Other RRsets are untouched.
//
// Existing records are reused where an RRset shrinks or grows, so an unchanged
// record keeps its ID. Binary Lane offers no batch endpoint, so this is not
// atomic.
func (p *Provider) SetRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	zone = unFQDN(zone)

	existing, err := p.listRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	// Group the input into RRsets, keeping the order they arrived in so the
	// returned slice is predictable.
	type key struct{ name, rrtype string }

	var order []key
	groups := map[key][]apiRecord{}

	for _, rec := range recs {
		req, err := toAPI(rec)
		if err != nil {
			return nil, err
		}

		k := key{req.Name, req.Type}
		if _, seen := groups[k]; !seen {
			order = append(order, k)
		}

		groups[k] = append(groups[k], req)
	}

	var result []libdns.Record

	for _, k := range order {
		want := groups[k]

		var have []apiRecord
		for _, rec := range existing {
			if sameRRset(rec, k.name, k.rrtype) {
				have = append(have, rec)
			}
		}

		for i, req := range want {
			var (
				out apiRecord
				err error
			)

			if i < len(have) {
				out, err = p.updateRecord(ctx, zone, have[i].ID, req)
			} else {
				out, err = p.createRecord(ctx, zone, req)
			}
			if err != nil {
				return result, err
			}

			result = append(result, toLibdns(out))
		}

		// Anything left over in the RRset is surplus to the input.
		for _, surplus := range have[min(len(want), len(have)):] {
			if err := p.deleteRecord(ctx, zone, surplus.ID); err != nil {
				return result, err
			}
		}
	}

	return result, nil
}

// DeleteRecords deletes records from the zone, returning those actually
// deleted. Records that do not exist are silently ignored.
//
// A record carrying an ID from GetRecords is deleted directly. Otherwise it is
// matched by value, where an empty Type, zero TTL or empty Data act as
// wildcards and Name is always significant.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	zone = unFQDN(zone)

	existing, err := p.listRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	live := make(map[int64]apiRecord, len(existing))
	for _, rec := range existing {
		live[rec.ID] = rec
	}

	var deleted []libdns.Record

	for _, rec := range recs {
		var targets []apiRecord

		if id, ok := recordID(rec); ok {
			if found, present := live[id]; present {
				targets = []apiRecord{found}
			}
		} else {
			for _, candidate := range existing {
				if _, present := live[candidate.ID]; !present {
					continue
				}
				if matches(candidate, rec.RR()) {
					targets = append(targets, candidate)
				}
			}
		}

		for _, target := range targets {
			if err := p.deleteRecord(ctx, zone, target.ID); err != nil {
				return deleted, err
			}

			delete(live, target.ID)
			deleted = append(deleted, toLibdns(target))
		}
	}

	return deleted, nil
}

// ListZones returns the domains available to the API token.
func (p *Provider) ListZones(ctx context.Context) ([]libdns.Zone, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	names, err := p.listDomains(ctx)
	if err != nil {
		return nil, err
	}

	zones := make([]libdns.Zone, 0, len(names))
	for _, name := range names {
		zones = append(zones, libdns.Zone{Name: libdns.AbsoluteName(name, ".")})
	}

	return zones, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
	_ libdns.ZoneLister     = (*Provider)(nil)
)
