// Package desec implements a DNS record management client compatible with the libdns interfaces for
// [deSEC].
//
// # Updates are not atomic
//
// The deSEC API doesn't map 1:1 to the libdns API. The main issue with that is that it's not
// possible to update records atomically. The implementation here goes to great lengths to avoid
// interference of multiple concurrent requests, but that only works within a single process.
//
// If multiple processes are modifying a deSEC zone concurrently, care must be taken that the
// different processes operate on different [resource record sets]. Otherwise multiple concurrent
// operations will override one another. The easiest way to protect against that is to use different
// names within the zone for different processes.
//
// If multiple processes operate on the same resource record set, it's possible for two concurrently
// running writes to result in inconsistent records.
//
// # TTL attribute
//
// For a the same reason as above, the  TTL attribute cannot be set on the per record level. If
// multiple different TTLs are specified for different records of the same name and type, one of
// them wins. It's not defined which on that is.
//
// # Large zones (> 500 resource record sets)
//
// deSEC requires the use of pagination for zones with more than 500 RRSets. This is a reasonable
// limit for a general purpose library like libdns and no effort is made to handle zones with more
// than 500 RRSets. Methods that can fail with more than 500 RRSets have a godoc comment explaining
// this.
//
// # Rate Limiting
//
// deSEC applies [rate limiting], this implementation will retry when running into a rate limit
// while observing context cancellation. In practice this means that calls to methods of this
// provider can take multiple seconds and longer. It's therefore very important to set a deadline in
// the context.
//
// [deSEC]: http://desec.io
// [resource record sets]: https://desec.readthedocs.io/en/latest/dns/rrsets.html
// [rate limiting]: https://desec.readthedocs.io/en/latest/rate-limits.html
package desec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/libdns/libdns"
	"golang.org/x/exp/slices"
)

// writeToken is used to synchronize all writes to deSEC to make sure the API here adheres to the
// libdns contract. This is necessary because libdns operates in units of records (zone, name, type,
// value) while deSEC operates in units of record sets (zone, name, type). This makes it necessary
// to perform read-modify cycles that can't be done atomically due to limitations in the deSEC API.
//
// This also can't be a mutex, because the time it takes to perform a write is theoretically
// unbounded due to rate limiting (https://desec.readthedocs.io/en/latest/rate-limits.html) and
// using a mutex here would make it impossible to adhere to context cancellation.
//
// Rate limiting on the deSEC side also means that this is unlikely to become a bottleneck, though
// use cases may exist that cause this single synchronization point to become one.
var writeToken = make(chan struct{}, 1)

func acquireWriteToken(ctx context.Context) error {
	select {
	case <-writeToken:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseWriteToken() {
	writeToken <- struct{}{}
}

func init() {
	writeToken <- struct{}{}
}

// Provider facilitates DNS record manipulation with deSEC.
type Provider struct {
	// Token is a token created on https://desec.io/tokens. A basic token without the permission
	// to manage tokens is sufficient.
	Token string `json:"token,omitempty"`
}

// GetRecords lists all the records in the zone.
//
// Caveat: This method will fail if there are more than 500 RRsets in the zone. See package
// documentation for more detail.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	// https://desec.readthedocs.io/en/latest/dns/rrsets.html#retrieving-all-rrsets-in-a-zone
	rrsets, err := p.listRRSets(ctx, zone)
	if err != nil {
		return nil, err
	}
	var records []libdns.Record
	for _, rrset := range rrsets {
		records0, err := libdnsRecords(rrset)
		if err != nil {
			return nil, fmt.Errorf("parsing RRSet: %v", err)
		}
		records = append(records, records0...)
	}
	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	err := acquireWriteToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for inflight requests to finish: %v", err)
	}
	defer releaseWriteToken()

	rrsets := make(map[rrKey]*rrSet)

	// Fetch or create base rrsets to append to
	for _, r := range records {
		key := rrKey{rrSetSubname(r), r.Type}
		if _, ok := rrsets[key]; ok {
			continue
		}

		rrset, err := p.getRRSet(ctx, zone, key)
		switch {
		case errors.Is(err, errNotFound):
			// no RRSet exists, create one
			rrset = rrSet{
				Subname: key.Subname,
				Type:    key.Type,
				Records: nil,
				TTL:     int(r.TTL / time.Second),
			}
		case err != nil:
			return nil, fmt.Errorf("retrieving RRSet: %v", err)
		}
		rrsets[key] = &rrset
	}

	// Merge records into base
	dirty := make(map[rrKey]struct{})
	var ret []libdns.Record
	for _, r := range records {
		key := rrKey{rrSetSubname(r), r.Type}
		rrset := rrsets[key]

		v := rrSetRecord(r)
		if slices.Contains(rrset.Records, v) {
			// Don't modify existing records, if all records in a record set already exist, the
			// record set will not be marked dirty and excluded from the update request.
			continue
		}

		rrset.Records = append(rrset.Records, v)
		ret = append(ret, libdns.Record{
			Name:     r.Name,
			Type:     r.Type,
			Value:    r.Value,
			TTL:      time.Duration(rrset.TTL) * time.Second,
			Priority: r.Priority,
			Weight:   r.Weight,
		})

		// Mark this key as dirty, only dirty keys will result in an update.
		dirty[key] = struct{}{}
	}

	update := make([]rrSet, 0, len(dirty))
	for key := range dirty {
		update = append(update, *rrsets[key])
	}
	if err := p.putRRSets(ctx, zone, update); err != nil {
		return nil, fmt.Errorf("writing RRSets: %v", err)
	}
	return ret, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
//
// Caveat: This method will fail if there are more than 500 RRsets in the zone. See package
// documentation for more detail.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	err := acquireWriteToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for inflight requests to finish: %v", err)
	}
	defer releaseWriteToken()

	// Build the desired state
	rrsets := make(map[rrKey]*rrSet)
	for _, r := range records {
		key := rrKey{rrSetSubname(r), r.Type}
		rrset := rrsets[key]
		if rrset == nil {
			rrset = &rrSet{
				Subname: key.Subname,
				Type:    key.Type,
				Records: nil,
				TTL:     int(r.TTL / time.Second),
			}
			rrsets[key] = rrset
		}
		rrset.Records = append(rrset.Records, rrSetRecord(r))
	}

	// Fetch existing rrSets and compare to desired state
	existing, err := p.listRRSets(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("listing RRSets: %v", err)
	}
	for _, g := range existing {
		key := rrKey{g.Subname, g.Type}
		w := rrsets[key]
		switch {
		case w == nil:
			// rrset exists, but not in the input, delete it by adding it to rrsets and set
			// records to an empty slice to represent the deletion.
			// See https://desec.readthedocs.io/en/latest/dns/rrsets.html#deleting-an-rrset
			w0 := g
			w0.Records = []string{}
			rrsets[key] = &w0
		case g.equal(w):
			// rrset exists and is equal to the one we want; skip it in the update.
			delete(rrsets, key)
		}
	}

	// Generate updates to arrive at desired state.
	update := make([]rrSet, 0, len(rrsets))
	var ret []libdns.Record
	for _, rrset := range rrsets {
		update = append(update, *rrset)

		// Add all records being set here. This ignores records that are being deleted, because
		// those are represented as an rrset without any records.
		records0, err := libdnsRecords(*rrset)
		if err != nil {
			return nil, fmt.Errorf("parsing RRSet: %v", err)
		}
		ret = append(ret, records0...)
	}

	if err := p.putRRSets(ctx, zone, update); err != nil {
		return nil, fmt.Errorf("writing RRSets: %v", err)
	}
	return ret, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	err := acquireWriteToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("waiting for inflight requests to finish: %v", err)
	}
	defer releaseWriteToken()

	rrsets := make(map[rrKey]*rrSet)
	dirty := make(map[rrKey]struct{})
	var ret []libdns.Record

	// Fetch rrsets with records requested for deletion.
	for _, r := range records {
		key := rrKey{rrSetSubname(r), r.Type}
		rrset := rrsets[key]
		if rrset == nil {
			rrset0, err := p.getRRSet(ctx, zone, key)
			switch {
			case errors.Is(err, errNotFound):
				continue
			case err != nil:
				return nil, fmt.Errorf("retrieving RRSet: %v", err)
			}
			rrsets[key] = &rrset0
			rrset = &rrset0
		}

		// Delete the record if it exists and mark the rrset as dirty, only dirty rrsets will be
		// updated.
		v := rrSetRecord(r)
		if i := slices.Index(rrset.Records, v); i >= 0 {
			rrset.Records = slices.Delete(rrset.Records, i, i+1)
			dirty[key] = struct{}{}
			ret = append(ret, r)
		}
	}

	update := make([]rrSet, 0, len(dirty))
	for key := range dirty {
		update = append(update, *rrsets[key])
	}
	if err := p.putRRSets(ctx, zone, update); err != nil {
		return nil, fmt.Errorf("writing RRSets: %v", err)
	}
	return ret, nil
}

// https://desec.readthedocs.io/en/latest/dns/rrsets.html#rrset-field-reference
type rrSet struct {
	Subname string   `json:"subname"`
	Type    string   `json:"type"`
	Records []string `json:"records"`
	TTL     int      `json:"ttl,omitempty"`
}

// rrKey uniquely identifies an rrSet within a zone.
type rrKey struct {
	Subname string
	Type    string
}

func (rrs *rrSet) equal(other *rrSet) bool {
	return rrs.Subname == other.Subname && rrs.Type == other.Type && rrs.TTL == other.TTL && slices.Equal(rrs.Records, other.Records)
}

// libdnsName returns the rrSet subname converted to libdns conventions.
//
// deSEC represents the zone itself using an empty (or missing) subname, libdns
// uses "@"
func libdnsName(rrs rrSet) string {
	if rrs.Subname == "" {
		return "@"
	}
	return rrs.Subname
}

// rrSetSubname returns the libdns name converted to deSEC conventions.
//
// deSEC represents the zone itself using an empty (or missing) subname, libdns
// uses "@"
func rrSetSubname(r libdns.Record) string {
	// deSEC represents the zone itself using an empty (or missing) subname, libdns
	// uses "@"
	if r.Name == "@" {
		return ""
	}
	return r.Name
}

// libdnsValue returns the value of the i-th record in libdns conventions
//
// libdns provides priority and weight for DNS entries that support it, the list is
// documented in the libdns.Record documentation.
func libdnsValue(rrs rrSet, i int) (prio, weight uint, value string, err error) {
	v := rrs.Records[i]
	var uints []uint
	switch rrs.Type {
	default:
		value = v
	case "HTTPS", "MX":
		uints, value, err = splitUints(v, 1)
		if err != nil {
			err = fmt.Errorf("desec: parsing %v record value %q: %v", rrs.Type, v, err)
			return
		}
		prio = uints[0]
	case "SRV", "URI":
		uints, value, err = splitUints(v, 2)
		if err != nil {
			err = fmt.Errorf("desec: parsing %v record value %q: %v", rrs.Type, v, err)
			return
		}
		prio = uints[0]
		weight = uints[1]
	}
	return
}

func splitUints(s string, n int) ([]uint, string, error) {
	parts := strings.SplitN(s, " ", n+1)
	if len(parts) != n+1 {
		return nil, "", fmt.Errorf("expected %d space separated values, got %d", n+1, len(parts))
	}
	uints := make([]uint, n)
	for i := range uints {
		v, err := strconv.ParseUint(parts[i], 10, strconv.IntSize)
		if err != nil {
			return nil, "", err
		}
		uints[i] = uint(v)
	}
	return uints, parts[n], nil
}

// rrSetRecord returns the libdns record value in deSEC conventions.
//
// libdns provides priority and weight for DNS entries that support it, the list is
// documented in the libdns.Record documentation.
func rrSetRecord(r libdns.Record) string {
	var v string
	switch r.Type {
	default:
		v = r.Value
	case "HTTPS", "MX":
		v = fmt.Sprintf("%d %s", r.Priority, r.Value)
	case "SRV", "URI":
		v = fmt.Sprintf("%d %d %s", r.Priority, r.Weight, r.Value)
	}
	return v
}

// libdnsRecords returns the libdns.Records corresponding to a given rrSet.
func libdnsRecords(rrs rrSet) ([]libdns.Record, error) {
	records := make([]libdns.Record, 0, len(rrs.Records))
	name := libdnsName(rrs)
	ttl := time.Duration(rrs.TTL) * time.Second
	for i := range rrs.Records {
		prio, weight, value, err := libdnsValue(rrs, i)
		if err != nil {
			return nil, err
		}

		records = append(records, libdns.Record{
			Type:     rrs.Type,
			Name:     name,
			Value:    value,
			TTL:      ttl,
			Priority: prio,
			Weight:   weight,
		})
	}
	return records, nil
}

type statusError struct {
	code   int
	header http.Header
	body   []byte
}

func (err *statusError) Error() string {
	return fmt.Sprintf("unexpected status code %d: %v", err.code, string(err.body))
}

var errNotFound = errors.New("not found")

func (p *Provider) httpDo0(ctx context.Context, method, url string, in []byte) ([]byte, error) {
	var r io.Reader
	if len(in) > 0 {
		r = bytes.NewReader(in)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, r)
	if err != nil {
		return nil, fmt.Errorf("creating request: %v", err)
	}
	req.Header.Set("Authorization", "Token "+p.Token)
	req.Header.Set("Accept", "application/json; charset=utf-8")
	if len(in) > 0 {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	switch res.StatusCode {
	case http.StatusOK:
		return body, nil
	default:
		return nil, &statusError{code: res.StatusCode, header: res.Header, body: body}
	}
}

func (p *Provider) httpDo(ctx context.Context, method, url string, in []byte) ([]byte, error) {
	for {
		out, err := p.httpDo0(ctx, method, url, in)
		if s := (*statusError)(nil); errors.As(err, &s) && s.code == http.StatusTooManyRequests {
			// rate limited, wait until the next request can be send
			retryAfterHeader := s.header.Get("Retry-After")
			retryAfter, err := strconv.Atoi(retryAfterHeader)
			if err != nil {
				return nil, fmt.Errorf("parsing Retry-After header %q: %v", retryAfterHeader, err)
			}
			select {
			case <-time.After(time.Duration(retryAfter) * time.Second):
			case <-ctx.Done():
				return nil, fmt.Errorf("waiting for cooldown to end: %v", ctx.Err())
			}
			continue // try again
		}
		return out, err
	}
}

func (p *Provider) getRRSet(ctx context.Context, zone string, key rrKey) (rrSet, error) {
	// https://desec.readthedocs.io/en/latest/dns/rrsets.html#retrieving-a-specific-rrset
	subname := key.Subname
	if subname == "" {
		subname = "@"
	}
	url := fmt.Sprintf("https://desec.io/api/v1/domains/%s/rrsets/%s/%s", url.PathEscape(zone), url.PathEscape(subname), url.PathEscape(key.Type))
	outb, err := p.httpDo(ctx, "GET", url, nil)
	if err != nil {
		if status, ok := err.(*statusError); ok {
			if status.code == http.StatusNotFound {
				return rrSet{}, errNotFound
			}
		}
		return rrSet{}, err
	}

	var out rrSet
	if err := json.Unmarshal(outb, &out); err != nil {
		return rrSet{}, fmt.Errorf("decoding json: %v", err)
	}
	return out, nil
}

func (p *Provider) listRRSets(ctx context.Context, zome string) ([]rrSet, error) {
	// https://desec.readthedocs.io/en/latest/dns/rrsets.html#retrieving-all-rrsets-in-a-zone
	url := fmt.Sprintf("https://desec.io/api/v1/domains/%s/rrsets/", url.PathEscape(zome))
	buf, err := p.httpDo(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var out []rrSet
	if err := json.Unmarshal(buf, &out); err != nil {
		return nil, fmt.Errorf("decoding json: %v", err)
	}
	return out, nil
}

func (p *Provider) putRRSets(ctx context.Context, zone string, rrs []rrSet) error {
	if len(rrs) == 0 {
		return nil
	}

	// https://desec.readthedocs.io/en/latest/dns/rrsets.html#bulk-modification-of-rrsets
	url := fmt.Sprintf("https://desec.io/api/v1/domains/%s/rrsets/", url.PathEscape(zone))

	var buf []byte
	var err error
	buf, err = json.Marshal(rrs)
	if err != nil {
		return fmt.Errorf("encoding json: %v", err)
	}

	_, err = p.httpDo(ctx, "PUT", url, buf)
	if err != nil {
		return err
	}
	return nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
