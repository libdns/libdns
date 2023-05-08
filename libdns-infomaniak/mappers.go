package infomaniak

import (
	"time"

	"github.com/libdns/libdns"
)

// ToLibDnsRecord maps a infomaniak dns record to a libdns record
func (ikr *IkRecord) ToLibDnsRecord(zone string) libdns.Record {
	return libdns.Record{
		ID:       ikr.ID,
		Type:     ikr.Type,
		Name:     libdns.RelativeName(ikr.SourceIdn, zone),
		Value:    ikr.Target,
		TTL:      time.Duration(ikr.TtlInSec),
		Priority: int(ikr.Priority),
	}
}

// ToInfomaniakRecord maps a libdns record to a infomaniak dns record
func ToInfomaniakRecord(libdnsRec *libdns.Record, zone string) IkRecord {
	ikRec := IkRecord{
		ID:        libdnsRec.ID,
		Type:      libdnsRec.Type,
		SourceIdn: libdns.AbsoluteName(libdnsRec.Name, zone),
		Target:    libdnsRec.Value,
		Priority:  uint(libdnsRec.Priority),
	}
	if libdnsRec.TTL != 0 {
		ikRec.TtlInSec = uint(libdnsRec.TTL)
	}
	return ikRec
}
