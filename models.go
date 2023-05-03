package directadmin

import (
	"errors"
	"fmt"
	"github.com/libdns/libdns"
	"strconv"
	"strings"
	"time"
)

type daZone struct {
	Records                  []daRecord `json:"records"`
	Dnssec                   string     `json:"dnssec,omitempty"`
	UserDnssecControl        string     `json:"user_dnssec_control,omitempty"`
	DNSNs                    string     `json:"dns_ns,omitempty"`
	DNSPtr                   string     `json:"dns_ptr,omitempty"`
	DNSSpf                   string     `json:"dns_spf,omitempty"`
	DNSTTL                   string     `json:"dns_ttl,omitempty"`
	DNSAffectPointersDefault string     `json:"DNS_AFFECT_POINTERS_DEFAULT,omitempty"`
	DNSTLSa                  string     `json:"dns_tlsa,omitempty"`
	DNSCaa                   string     `json:"dns_caa,omitempty"`
	AllowDNSUnderscore       string     `json:"allow_dns_underscore,omitempty"`
	FullMxRecords            string     `json:"full_mx_records,omitempty"`
	DefaultTTL               string     `json:"default_ttl,omitempty"`
	AllowTTLOverride         string     `json:"allow_ttl_override,omitempty"`
	TTLIsOverridden          string     `json:"ttl_is_overridden,omitempty"`
	TTL                      string     `json:"ttl,omitempty"`
	TTLValue                 string     `json:"ttl_value,omitempty"`
}

type daRecord struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Combined string `json:"combined"`
	TTL      string `json:"ttl,omitempty"`
}

var ErrUnsupported = errors.New("unsupported record type")

func (r daRecord) libdnsRecord(zone string) (libdns.Record, error) {
	record := libdns.Record{
		ID:   r.Combined,
		Type: r.Type,
		Name: r.Name,
	}

	switch r.Type {
	case "MX":
		splits := strings.Split(r.Value, " ")

		priority, err := strconv.Atoi(splits[0])
		if err != nil {
			return record, fmt.Errorf("failed to parse MX priority for %v: %v", r.Name, err)
		}

		record.Priority = priority
		record.Value = fmt.Sprintf("%v.%v", splits[1], zone)
	case "SRV":
		return record, ErrUnsupported
	case "URI":
		return record, ErrUnsupported
	default:
		record.Value = r.Value
	}

	if len(r.TTL) > 0 {
		ttl, err := strconv.Atoi(r.TTL)
		if err != nil {
			return record, fmt.Errorf("failed to parse TTL for %v: %v", r.Name, err)
		}
		record.TTL = time.Duration(ttl) * time.Second
	}

	return record, nil
}

type daResponse struct {
	Error   string `json:"error,omitempty"`
	Success string `json:"success,omitempty"`
	Result  string `json:"result,omitempty"`
}
