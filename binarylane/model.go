package binarylane

import (
	"fmt"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

// Binary Lane carries parts of the RDATA of some record types in dedicated
// fields (priority, port, weight, flags, tag) rather than in "data", while
// libdns expects Data to hold the full unescaped zone-file RDATA for every
// type. These helpers compose it on read and split it again on write.

// toLibdns converts an API record to its libdns representation, returning an
// RR-type-specific struct where libdns recognises the type and an opaque
// libdns.RR otherwise — for example SOA, which Binary Lane exposes but libdns
// does not model.
func toLibdns(rec apiRecord) libdns.Record {
	rr := libdns.RR{
		Name: rec.Name,
		Type: rec.Type,
		Data: rec.Data,
		TTL:  time.Duration(rec.TTL) * time.Second,
	}

	switch rec.Type {
	case "MX":
		rr.Data = fmt.Sprintf("%d %s", rec.Priority, rec.Data)
	case "SRV":
		rr.Data = fmt.Sprintf("%d %d %d %s", rec.Priority, rec.Weight, rec.Port, rec.Data)
	case "CAA":
		rr.Data = fmt.Sprintf("%d %s %q", rec.Flags, rec.Tag, rec.Data)
	}

	// A record this libdns version cannot parse into a typed struct is still
	// worth returning: listing a zone should not lose every record because one
	// is malformed or of an unmodelled type.
	parsed, err := rr.Parse()
	if err != nil {
		parsed = rr
	}

	return withRecordID(parsed, rec.ID)
}

// withRecordID stores the API's record ID in the record's ProviderData field so
// that SetRecords and DeleteRecords can address it without a lookup. The opaque
// libdns.RR has no such field, so unmodelled types are returned unchanged and
// matched by value instead.
func withRecordID(rec libdns.Record, id int64) libdns.Record {
	switch v := rec.(type) {
	case libdns.Address:
		v.ProviderData = id
		return v
	case libdns.CAA:
		v.ProviderData = id
		return v
	case libdns.CNAME:
		v.ProviderData = id
		return v
	case libdns.MX:
		v.ProviderData = id
		return v
	case libdns.NS:
		v.ProviderData = id
		return v
	case libdns.SRV:
		v.ProviderData = id
		return v
	case libdns.TXT:
		v.ProviderData = id
		return v
	default:
		return rec
	}
}

// recordID recovers an ID previously stored by withRecordID. ok is false for
// records the caller constructed itself, which must be matched by value.
func recordID(rec libdns.Record) (id int64, ok bool) {
	var pd any

	switch v := rec.(type) {
	case libdns.Address:
		pd = v.ProviderData
	case libdns.CAA:
		pd = v.ProviderData
	case libdns.CNAME:
		pd = v.ProviderData
	case libdns.MX:
		pd = v.ProviderData
	case libdns.NS:
		pd = v.ProviderData
	case libdns.SRV:
		pd = v.ProviderData
	case libdns.TXT:
		pd = v.ProviderData
	default:
		return 0, false
	}

	id, ok = pd.(int64)

	return id, ok && id != 0
}

// toAPI converts a libdns record into the API's representation, splitting the
// RDATA back out into the fields Binary Lane expects.
func toAPI(rec libdns.Record) (apiRecord, error) {
	rr := rec.RR()

	// TTL is deliberately not sent. Binary Lane enforces a fixed zone TTL and
	// rejects any other value outright:
	//
	//	HTTP 400: The default TTL of 3600 may not be changed.
	//
	// Omitting the field lets the API apply its own default, so a caller
	// asking for a shorter TTL gets a longer one rather than an error. That
	// matters most for ACME: clients set a short TTL on challenge records, and
	// forwarding it would fail every DNS-01 challenge. libdns sanctions this —
	// "some providers may reject or silently increase TTLs ... it is
	// recommended to not rely on the exact TTL value".
	out := apiRecord{
		Type: rr.Type,
		Name: rr.Name,
		Data: rr.Data,
	}

	switch v := rec.(type) {
	case libdns.Address:
		// netip normalises the textual form, which the raw Data may not be.
		out.Data = v.IP.String()
	case libdns.MX:
		out.Priority = int(v.Preference)
		out.Data = v.Target
	case libdns.SRV:
		out.Priority = int(v.Priority)
		out.Weight = int(v.Weight)
		out.Port = int(v.Port)
		out.Data = v.Target
		// libdns splits the service and transport labels out of Name; the API
		// wants the single assembled owner name back.
		out.Name = strings.Trim(fmt.Sprintf("_%s._%s.%s", v.Service, v.Transport, v.Name), ".")
	case libdns.CAA:
		out.Flags = int(v.Flags)
		out.Tag = v.Tag
		out.Data = v.Value
	case libdns.TXT:
		// libdns hands us unquoted text and expects the implementation to do
		// whatever its API needs; Binary Lane stores the value verbatim.
		out.Data = v.Text
	case libdns.RR:
		// An opaque RR of a type the API splits cannot be written safely: the
		// dedicated fields would be left at zero and the whole RDATA sent as
		// the hostname.
		switch rr.Type {
		case "MX", "SRV", "CAA":
			parsed, err := rr.Parse()
			if err != nil {
				return apiRecord{}, fmt.Errorf("binarylane: parsing %s record %q: %w", rr.Type, rr.Name, err)
			}

			return toAPI(parsed)
		}
	}

	if out.Name == "" {
		out.Name = "@"
	}

	return out, nil
}

// sameRRset reports whether an API record belongs to the RRset identified by
// name and type.
func sameRRset(rec apiRecord, name, rrtype string) bool {
	return rec.Name == name && rec.Type == rrtype
}

// matches implements the partial-match rules of libdns.RecordDeleter: Name is
// always significant, while an empty Type, zero TTL or empty Data act as
// wildcards.
func matches(existing apiRecord, want libdns.RR) bool {
	if existing.Name != want.Name {
		return false
	}
	if want.Type != "" && existing.Type != want.Type {
		return false
	}
	if want.TTL != 0 && time.Duration(existing.TTL)*time.Second != want.TTL {
		return false
	}
	if want.Data != "" && toLibdns(existing).RR().Data != want.Data {
		return false
	}

	return true
}

// unFQDN trims the trailing dot from a zone name. The API does not use FQDNs.
func unFQDN(zone string) string {
	return strings.TrimSuffix(zone, ".")
}
