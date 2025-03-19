package libdns

import (
	"fmt"
	"net/netip"
	"time"
)

// A represents a parsed A-type record, which associates
// a name with an IPv4 address. This is typically how
// to "point a domain to your server" (with IPv4).
type A struct {
	Name string
	TTL  time.Duration
	IP   netip.Addr
}

func (a A) RR() (RR, error) {
	return RR{
		Name: a.Name,
		TTL:  a.TTL,
		Type: "A",
		Data: a.IP.String(),
	}, nil
}

// AAAA represents a parsed AAAA-type record, which associates
// a name with an IPv6 address. This is typically how
// to "point a domain to your server" (with IPv6).
type AAAA struct {
	Name string
	TTL  time.Duration
	IP   netip.Addr
}

func (aaaa AAAA) RR() (RR, error) {
	return RR{
		Name: aaaa.Name,
		TTL:  aaaa.TTL,
		Type: "AAAA",
		Data: aaaa.IP.String(),
	}, nil
}

// CNAME represents a CNAME-type record, which delegates
// authority to other names.
type CNAME struct {
	Name   string
	TTL    time.Duration
	Target string
}

func (c CNAME) RR() (RR, error) {
	return RR{
		Name: c.Name,
		TTL:  c.TTL,
		Type: "CNAME",
		Data: c.Target,
	}, nil
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
func (h HTTPS) RR() (RR, error) {
	return RR{
		Name: h.Name,
		TTL:  h.TTL,
		Type: "HTTPS",
		Data: fmt.Sprintf("%d %s %s", h.Priority, h.Target, h.Value),
	}, nil
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

func (s SRV) RR() (RR, error) {
	return RR{
		Name: fmt.Sprintf("_%s._%s.%s", s.Service, s.Proto, s.Name),
		TTL:  s.TTL,
		Type: "SRV",
		Data: fmt.Sprintf("%d %d %d %s", s.Priority, s.Weight, s.Port, s.Target),
	}, nil
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

func (t TXT) RR() (RR, error) {
	return RR{
		Name: t.Name,
		TTL:  t.TTL,
		Type: "TXT",
		Data: t.Text,
	}, nil
}
