// Package libdns defines core interfaces that should be implemented by DNS
// provider clients. They are small and idiomatic Go interfaces with
// well-defined semantics for the purposes of reading and manipulating
// DNS records using DNS provider APIs.
//
// This documentation uses the definitions for terms from RFC 7719:
//
//	https://datatracker.ietf.org/doc/html/rfc7719
//
// Records are described independently of any particular zone, a convention
// that grants Record structs portability across zones. As such, record names
// are partially qualified, i.e. relative to the zone. For example, an A
// record called "sub" in zone "example.com." represents a fully-qualified
// domain name (FQDN) of "sub.example.com.". Implementations should expect
// that input records conform to this standard, while also ensuring that
// output records do; adjustments to record names may need to be made before
// or after provider API calls, for example, to maintain consistency with
// all other libdns packages. Helper functions are available in this package
// to convert between relative and absolute names.
//
// Although zone names are a required input, libdns does not coerce any
// particular representation of DNS zones; only records. Since zone name and
// records are separate inputs in libdns interfaces, it is up to the caller
// to pair a zone's name with its records in a way that works for them.
//
// All interface implementations must be safe for concurrent/parallel use,
// meaning 1) no data races, and 2) simultaneous method calls must result
// in either both their expected outcomes or an error.
//
// For example, if AppendRecords() is called at the same time and two API
// requests are made to the provider at the same time, the result of both
// requests must be visible after they both complete; if the provider does
// not synchronize the writing of the zone file and one request overwrites
// the other, then the client implementation must take care to synchronize
// on behalf of the incompetent provider. This synchronization need not be
// global; for example: the scope of synchronization might only need to be
// within the same zone, allowing multiple requests at once as long as all
// of them are for different zones. (Exact logic depends on the provider.)
package libdns

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// RecordGetter can get records from a DNS zone.
type RecordGetter interface {
	// GetRecords returns all the records in the DNS zone.
	//
	// DNSSEC-related records are typically not included in the output, but
	// this behavior is implementation-defined. *If* an implementation
	// includes DNSSEC records in the output, this behavior should be
	// documented.
	//
	// Implementations must honor context cancellation and be safe for
	// concurrent use.
	GetRecords(ctx context.Context, zone string) ([]Record, error)
}

// RecordAppender can non-destructively add new records to a DNS zone.
type RecordAppender interface {
	// AppendRecords creates the requested records in the given zone
	// and returns the populated records that were created. It never
	// changes existing records. Therefore, it is invalid to use this
	// method with CNAME-type records.
	//
	// Implementations must honor context cancellation and be safe for
	// concurrent use.
	AppendRecords(ctx context.Context, zone string, recs []Record) ([]Record, error)
}

// RecordSetter can set new or update existing records in a DNS zone.
type RecordSetter interface {
	// SetRecords updates the zone so that the records described in the
	// input are reflected in the output. It may create or overwrite
	// records or -- depending on the record type -- delete records to
	// maintain parity with the input. No other records are affected.
	// It returns the records which were set.
	//
	// For any (name, type) pair in the input, `SetRecords` ensures that the
	// only records in the output zone with that (name, type) pair are those
	// that were provided in the input.
	//
	// In RFC 7719 terms, `SetRecords` appends, modifies, or deletes records
	// in the zone so that for each RRset in the input, the records provided
	// in the input are the only members of their RRset in the output zone.
	//
	// Implementations may decide whether or not to support DNSSEC-related
	// records in calls to `SetRecords`, but should document their decision.
	// Note that the decision to support DNSSEC records in `SetRecords` is
	// independent of the decision to support them in `GetRecords`, so end-users
	// should not blindly call `SetRecords` on the output of `GetRecords`.
	//
	// Implementations must honor context cancellation and be safe for
	// concurrent use.
	//
	// Examples:
	//
	// 1. Original zone:
	//    example.com. 3600 IN A   192.0.2.1
	//    example.com. 3600 IN A   192.0.2.2
	//    example.com. 3600 IN TXT "hello world"
	//
	//    Input:
	//    example.com. 3600 IN A   192.0.2.3
	//
	//    Resultant zone:
	//    example.com. 3600 IN A   192.0.2.3
	//    example.com. 3600 IN TXT "hello world"
	//
	// 2. Original zone:
	//    a.example.com. 3600 IN AAAA 2001:db8::1
	//    a.example.com. 3600 IN AAAA 2001:db8::2
	//    b.example.com. 3600 IN AAAA 2001:db8::3
	//    b.example.com. 3600 IN AAAA 2001:db8::4
	//
	//    Input:
	//    a.example.com. 3600 IN AAAA 2001:db8::1
	//    a.example.com. 3600 IN AAAA 2001:db8::2
	//    a.example.com. 3600 IN AAAA 2001:db8::5
	//
	//    Resultant zone:
	//    a.example.com. 3600 IN AAAA 2001:db8::1
	//    a.example.com. 3600 IN AAAA 2001:db8::2
	//    a.example.com. 3600 IN AAAA 2001:db8::5
	//    b.example.com. 3600 IN AAAA 2001:db8::3
	//    b.example.com. 3600 IN AAAA 2001:db8::4
	SetRecords(ctx context.Context, zone string, recs []Record) ([]Record, error)
}

// RecordDeleter can delete records from a DNS zone.
type RecordDeleter interface {
	// `DeleteRecords` deletes the given records from the zone if they exist in
	// the zone and exactly match the input. If the input records do not exist
	// in the zone, they are silently ignored. `DeleteRecords` returns only the
	// the records that were deleted, and does not return any records that were
	// provided in the input but did not exist in the zone.
	//
	// `DeleteRecords` only deletes records from the zone that *exactly* match
	// the input records---that is, the name, type, TTL, and value all must be
	// identical to a record in the zone for it to be deleted.
	//
	// As a special case, you may leave any of the fields `Type`, `TTL`, or
	// `Value` empty ("", 0, and "" respectively). In this case, `DeleteRecords`
	// will delete any records that match the other fields, regardless of the
	// value of the fields that were left empty. Note that this behavior does
	// *not* apply to the `Name` field, which must always be specified.
	//
	// Note that it is semantically invalid to remove the last NS record from a
	// zone, so attempting to do is undefined behavior.
	//
	// Implementations must honor context cancellation and be safe for
	// concurrent use.
	DeleteRecords(ctx context.Context, zone string, recs []Record) ([]Record, error)
}

// ZoneLister can list available DNS zones.
type ZoneLister interface {
	// ListZones returns the list of available DNS zones for use by
	// other libdns methods.
	//
	// Implementations must honor context cancellation and be safe for
	// concurrent use.
	ListZones(ctx context.Context) ([]Zone, error)
}

// Record is a generalized representation of a DNS record.
type Record struct {
	// provider-specific metadata
	ID string

	// general record fields

	// The `Type` field specifies the type of the record. Implementations may
	// or may not support any given record type, and may support additional
	// "private" record types with implementation-defined behavior.
	//
	// Examples: "A", "AAAA", "CNAME", "MX", "TXT"
	Type string

	// The `Name` field specifies the name of the record. It is partially
	// qualified relative to the current zone. This field is called a "Label" by
	// RFC7719. You may use "@" to represent the root of the zone.
	//
	// Examples: "www", "@", "subdomain", "sub.subdomain"
	//
	// Invalid: "www.example.com.", "example.net." (fully-qualified)
	// Invalid: "" (empty)
	//
	// Valid, but probably doesn't do what you want: "www.example.com" (for a
	// zone "example.net.", this refers to "www.example.com.example.net.")
	Name string

	// The `Value` field specifies the value of the record. This field should
	// be formatted in the standard zone file syntax, but should omit any fields
	// that are covered by other fields in this struct.
	//
	// Examples: (A)     "192.0.2.1"
	//           (AAAA)  "2001:db8::1"
	//           (CNAME) "example.com." (Even though the value is traditionally
	//                   called the "target", it is included only in the `Value`
	//                   field here.)
	//           (MX)    "mail.example.com." (Note that this excludes the
	//                   priority field!)
	//	         (TXT)   "Hello, world!"
	//	         (SRV)   "8080 example.com." (Note that this excludes the
	//                   priority and weight fields, but includes the port.
	//                   Also note that the target is included here, and not
	//                   in the `Target` field.)
	//	         (HTTPS) "alpn=h2,h3 port=443" (Note that this excludes the
	//                   priority field and target fields.)
	Value string

	// The `TTL` field specifies the time-to-live of the record. This is
	// represented in the DNS as an unsigned integral number of seconds, but is
	// provided here as a time.Duration. Fractions of seconds will be rounded
	// down (aka truncated). A value of `0` means that the record should not be
	// cached.
	//
	// Note that some providers may reject or silently increase TTLs that are
	// below a certain threshold, and that DNS resolvers may choose to ignore
	// your TTL settings, so it is recommended to not rely on the exact TTL
	// value.
	TTL time.Duration

	// common, type-dependent record fields

	// The `Priority` field specifies the priority of the record. This field is
	// only applicable for certain record types, but is mandatory for those
	// types.
	//
	// Examples: (MX)    10 (Note that this is traditionally called the
	//                   "preference" in the DNS)
	//           (SRV)   10
	//           (URI)   10
	//           (HTTPS) 10
	//           (SVCB)  10
	Priority uint

	// The `Weight` field specifies the weight of the record. This field is
	// only applicable for certain record types, but is mandatory for those
	// types.
	//
	// Examples: (SRV) 20
	//           (URI) 20
	Weight uint

	// The `Target` field specifies the target of the record. This field is
	// only valid for HTTPS and SVCB records, and *not* for SRV, MX, or CNAME
	// records, which store their targets in the `Value` field. This field must
	// be set to the fully-qualified domain name (FQDN) of the target.
	//
	// Examples: (HTTPS) "example.com."
	//           (SVCB)  "example.com."
	Target string
}

// Zone is a generalized representation of a DNS zone.
type Zone struct {
	Name string
}

// ToSRV parses the record into a SRV struct with fully-parsed, literal values.
//
// EXPERIMENTAL; subject to change or removal.
func (r Record) ToSRV() (SRV, error) {
	if r.Type != "SRV" {
		return SRV{}, fmt.Errorf("record type not SRV: %s", r.Type)
	}

	fields := strings.Fields(r.Value)
	if len(fields) != 2 {
		return SRV{}, fmt.Errorf("malformed SRV value; expected: '<port> <target>'")
	}

	port, err := strconv.Atoi(fields[0])
	if err != nil {
		return SRV{}, fmt.Errorf("invalid port %s: %v", fields[0], err)
	}
	if port < 0 {
		return SRV{}, fmt.Errorf("port cannot be < 0: %d", port)
	}

	parts := strings.SplitN(r.Name, ".", 3)
	if len(parts) < 3 {
		return SRV{}, fmt.Errorf("name %v does not contain enough fields; expected format: '_service._proto.name'", r.Name)
	}

	return SRV{
		Service:  strings.TrimPrefix(parts[0], "_"),
		Proto:    strings.TrimPrefix(parts[1], "_"),
		Name:     parts[2],
		Priority: r.Priority,
		Weight:   r.Weight,
		Port:     uint(port),
		Target:   fields[1],
	}, nil
}

// SRV contains all the parsed data of an SRV record.
//
// EXPERIMENTAL; subject to change or removal.
type SRV struct {
	Service  string // no leading "_"
	Proto    string // no leading "_"
	Name     string
	Priority uint
	Weight   uint
	Port     uint
	Target   string
}

// ToRecord converts the parsed SRV data to a Record struct.
//
// EXPERIMENTAL; subject to change or removal.
func (s SRV) ToRecord() Record {
	return Record{
		Type:     "SRV",
		Name:     fmt.Sprintf("_%s._%s.%s", s.Service, s.Proto, s.Name),
		Priority: s.Priority,
		Weight:   s.Weight,
		Value:    fmt.Sprintf("%d %s", s.Port, s.Target),
	}
}

// RelativeName makes fqdn relative to zone. For example, for a FQDN of
// "sub.example.com" and a zone of "example.com", it returns "sub".
//
// If fqdn is the same as zone (and both are non-empty), "@" is returned.
//
// If fqdn cannot be expressed relative to zone, the input fqdn is returned.
func RelativeName(fqdn, zone string) string {
	// liberally ignore trailing dots on both fqdn and zone, because
	// the relative name won't have a trailing dot anyway; I assume
	// this won't be problematic...?
	// (initially implemented because Cloudflare returns "fully-
	// qualified" domains in their records without a trailing dot,
	// but the input zone typically has a trailing dot)
	rel := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(fqdn, "."), strings.TrimSuffix(zone, ".")), ".")
	if rel == "" && fqdn != "" && zone != "" {
		return "@"
	}
	return rel
}

// AbsoluteName makes name into a fully-qualified domain name (FQDN) by
// prepending it to zone and tidying up the dots. For example, an input
// of name "sub" and zone "example.com." will return "sub.example.com.".
//
// Using `"@"` as the name is the recommended way to represent the root of the
// zone; however, unlike the `Record` struct, using the empty string `""` for
// the name _is_ permitted here, and will be identically to `"@"`.
func AbsoluteName(name, zone string) string {
	if zone == "" {
		return strings.Trim(name, ".")
	}
	if name == "" || name == "@" {
		return zone
	}
	if !strings.HasSuffix(name, ".") {
		name += "."
	}
	return name + zone
}
