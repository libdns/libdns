package rfc2136

import (
	"fmt"
	"github.com/libdns/libdns"
	"github.com/miekg/dns"
	"strings"
	"time"
)

func recordToRR(rec libdns.Record, zone string) (dns.RR, error) {
	str := fmt.Sprintf(`%s %d IN %s %s`, rec.Name,
		int(rec.TTL.Seconds()), rec.Type, rec.Value)
	zp := dns.NewZoneParser(strings.NewReader(str), zone, "")
	rr, _ := zp.Next()
	return rr, zp.Err()
}

func recordFromRR(rr dns.RR, zone string) libdns.Record {
	hdr := rr.Header()
	return libdns.Record{
		Type:  dns.TypeToString[hdr.Rrtype],
		Name:  libdns.RelativeName(hdr.Name, zone),
		TTL:   time.Duration(hdr.Ttl) * time.Second,
		Value: strings.TrimPrefix(rr.String(), hdr.String()),
	}
}
