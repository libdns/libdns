package luadns

import (
	"strconv"
	"strings"
	"time"

	"github.com/libdns/libdns"
	"github.com/luadns/luadns-go"
)

// unFQDN trims any trailing "." from fqdn name.
func unFQDN(fqdn string) string {
	return strings.TrimSuffix(fqdn, ".")
}

// newLibRecord builds a libdns Record from the provider Record.
func newLibRecord(r *luadns.Record, zone string) libdns.Record {
	return libdns.Record{
		ID:    strconv.FormatInt(r.ID, 10),
		Type:  r.Type,
		Name:  libdns.RelativeName(r.Name, zone),
		Value: r.Content,
		TTL:   time.Duration(r.TTL) * time.Second,
	}
}

// newLuaRecord builds a provider Record from the libdns Record.
func newLuaRecord(r libdns.Record, zone string) (*luadns.Record, error) {
	pr := &luadns.Record{
		Type:    r.Type,
		Name:    libdns.AbsoluteName(r.Name, zone),
		Content: r.Value,
		TTL:     uint32(r.TTL.Seconds()),
	}

	if r.ID != "" {
		n, err := strconv.ParseInt(r.ID, 10, 64)
		if err != nil {
			return nil, err
		}
		pr.ID = n
	}

	return pr, nil
}
