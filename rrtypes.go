package libdns

import (
	"fmt"
	"net/netip"
	"time"
)

// Address represents a parsed A-type or AAAA-type record,
// which associates a name with an IPv4 or IPv6 address
// respectively. This is typically how to "point a domain
// to your server."
//
// Since A and AAAA are semantically identical, with the
// exception of the bit length of the IP address in the
// data field, these record types are combined for ease of
// use in Go programs, which supports both address sizes,
// to help simplify code.
type Address struct {
	Name string
	TTL  time.Duration
	IP   netip.Addr
}

func (a Address) RR() RR {
	recType := "A"
	if a.IP.Is6() {
		recType = "AAAA"
	}
	return RR{
		Name: a.Name,
		TTL:  a.TTL,
		Type: recType,
		Data: a.IP.String(),
	}
}

// CAA represents a parsed CAA-type record, which is used to specify which PKIX
// certificate authorities are allowed to issue certificates for a domain. See
// also the [registry of flags and tags].
//
// [registry of flags and tags]: https://www.iana.org/assignments/caa-parameters/caa-parameters.xhtml
type CAA struct {
	Name  string
	TTL   time.Duration
	Flags uint8 // As of March 2025, the only valid values are 0 and 128.
	Tag   string
	Value string
}

func (c CAA) RR() RR {
	return RR{
		Name: c.Name,
		TTL:  c.TTL,
		Type: "CAA",
		Data: fmt.Sprintf(`%d %s %q`, c.Flags, c.Tag, c.Value),
	}
}

// CNAME represents a CNAME-type record, which delegates
// authority to other names.
type CNAME struct {
	Name   string
	TTL    time.Duration
	Target string
}

func (c CNAME) RR() RR {
	return RR{
		Name: c.Name,
		TTL:  c.TTL,
		Type: "CNAME",
		Data: c.Target,
	}
}

// HTTPS represents a parsed HTTPS-type record, which is used
// to provide clients with information for establishing HTTPS
// connections to servers. It may include data about ALPN,
// ECH, IP hints, and more.
type HTTPS struct {
	Name     string
	TTL      time.Duration
	Priority uint16
	Target   string
	Value    SvcParams
}

// RR converts the parsed record data to a generic [Record] struct.
//
// EXPERIMENTAL; subject to change or removal.
func (h HTTPS) RR() RR {
	return RR{
		Name: h.Name,
		TTL:  h.TTL,
		Type: "HTTPS",
		Data: fmt.Sprintf("%d %s %s", h.Priority, h.Target, h.Value),
	}
}

// MX represents a parsed MX-type record, which is used to specify the hostnames
// of the servers that accept mail for a domain.
type MX struct {
	Name       string
	TTL        time.Duration
	Preference uint16 // Lower values indicate that clients should prefer this server
	Target     string // The hostname of the mail server
}

func (m MX) RR() RR {
	return RR{
		Name: m.Name,
		TTL:  m.TTL,
		Type: "MX",
		Data: fmt.Sprintf("%d %s", m.Preference, m.Target),
	}
}

// NS represents a parsed NS-type record, which is used to specify the
// authoritative nameservers for a zone. It is strongly recommended to have at
// least two NS records for redundancy.
//
// Note that the NS records present at the root level of a zone must match those
// delegated to by the parent zone. This means that changing the NS records for
// the root of a registered domain won't have any effect unless you also update
// the NS records with the domain registrar.
//
// Also note that the DNS standards forbid removing the last NS record for a
// zone, so if you want to replace all NS records, you should add the new ones
// before removing the old ones.
type NS struct {
	Name   string
	TTL    time.Duration
	Target string
}

func (n NS) RR() RR {
	return RR{
		Name: n.Name,
		TTL:  n.TTL,
		Type: "NS",
		Data: n.Target,
	}
}

// SRV represents a parsed SRV-type record, which is used to
// manifest services or instances that provide services on a
// network.
//
// The serialization of this record type takes the form:
//
//	_service._proto.name. ttl IN SRV priority weight port target.
type SRV struct {
	Service  string // no leading "_"
	Proto    string // no leading "_"
	Name     string
	TTL      time.Duration
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   string
}

func (s SRV) RR() RR {
	return RR{
		Name: fmt.Sprintf("_%s._%s.%s", s.Service, s.Proto, s.Name),
		TTL:  s.TTL,
		Type: "SRV",
		Data: fmt.Sprintf("%d %d %d %s", s.Priority, s.Weight, s.Port, s.Target),
	}
}

// TXT represents a parsed TXT-type record, which is used to
// add arbitrary text data to a name in a DNS zone. It is often
// used for email integrity (DKIM/SPF), site verification, ACME
// challenges, and more.
type TXT struct {
	Name string
	TTL  time.Duration
	Text string
}

func (t TXT) RR() RR {
	return RR{
		Name: t.Name,
		TTL:  t.TTL,
		Type: "TXT",
		Data: t.Text,
	}
}
