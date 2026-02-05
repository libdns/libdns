package arvancloud

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

type arDomain struct {
	ID                 string   `json:"id"`
	AccountID          string   `json:"account_id"`
	UserID             string   `json:"user_id"`
	Domain             string   `json:"domain"`
	Name               string   `json:"name"`
	PlanLevel          int      `json:"plan_level"`
	NSKeys             []string `json:"ns_keys"`
	SmartRoutingStatus string   `json:"smart_routing_status"`
	CurrentNs          []string `json:"current_ns"`
	Status             string   `json:"status"`
	Restriction        []string `json:"restriction"`
	Type               string   `json:"type"`
	CNAMETarget        string   `json:"cname_target"`
	CustomCNAME        string   `json:"custom_cname"`
	UseNewWAfEngine    bool     `json:"use_new_waf_engine"`
	Transfer           struct {
		Domain      string    `json:"domain"`
		AccountId   string    `json:"account_id"`
		AccountName string    `json:"account_name"`
		OwnerId     string    `json:"owner_id"`
		OwnerName   string    `json:"owner_name"`
		Time        time.Time `json:"time"`
		Incoming    bool      `json:"Incoming"`
	}
	FingerPrintStatus bool      `json:"fingerprint_status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type arDNSRecord struct {
	ID            string      `json:"id,omitempty"`
	Type          string      `json:"type"`
	Name          string      `json:"name"`
	Value         interface{} `json:"value"`
	TTL           int         `json:"ttl"`
	Cloud         bool        `json:"cloud"`
	IsProtected   bool        `json:"is_protected,omitempty"`
	UpstreamHTTPS string      `json:"upstream_https,omitempty"`
	IPFilterMode  *IPFilter   `json:"ip_filter_mode,omitempty"`
}

// IPFilter defines the IP filtering mode for a record.
type IPFilter struct {
	Count     string `json:"count"`
	GeoFilter string `json:"geo_filter"`
	Order     string `json:"order"`
}

// ARecordValue represents the value structure for A and AAAA records.
// Note: The API expects a slice of these for A/AAAA records.
type ARecordValue struct {
	IP      string `json:"ip"`
	Port    int    `json:"port,omitempty"`
	Weight  int    `json:"weight,omitempty"`
	Country string `json:"country,omitempty"`
}

// TXTRecordValue represents the value structure for TXT records.
type TXTRecordValue struct {
	Text string `json:"text"`
}

// MXRecordValue represents the value structure for MX records.
type MXRecordValue struct {
	Host     string `json:"host"`
	Priority uint16 `json:"priority"`
}

// CNAMERecordValue represents the value structure for CNAME  records.
type CNAMERecordValue struct {
	Host       string `json:"host"`
	HostHeader string `json:"host_header,omitempty"`
	Port       int    `json:"port,omitempty"`
}

// ANAMERecordValue represents the value structure for ANAME records.
type ANAMERecordValue struct {
	Location   string `json:"location"`
	HostHeader string `json:"host_header,omitempty"`
	Port       int    `json:"port,omitempty"`
}

// SRVRecordValue represents the value structure for SRV records.
type SRVRecordValue struct {
	Target   string `json:"target"`
	Port     uint16 `json:"port"`
	Priority uint16 `json:"priority"`
	Weight   uint16 `json:"weight"`
}

// CAARecordValue represents the value structure for CAA records.
type CAARecordValue struct {
	Value string `json:"value"`
	Tag   string `json:"tag"`
}

// NSRecordValue represents the value structure for NS records.
type NSRecordValue struct {
	Host string `json:"host"`
}

// PTRRecordValue represents the value structure for PTR records.
type PTRRecordValue struct {
	Domain string `json:"domain"`
}

// TLSARecordValue represents the value structure for TLSA records.
type TLSARecordValue struct {
	Usage        string `json:"usage"`
	Selector     string `json:"selector"`
	MatchingType string `json:"matching_type"`
	Certificate  string `json:"certificate"`
}

func (r arDNSRecord) toLibDNSRecord(zone string) (libdns.Record, error) {
	name := libdns.RelativeName(r.Name, zone)
	ttl := time.Duration(r.TTL) * time.Second
	switch strings.ToUpper(r.Type) {
	case "A", "AAAA":
		var ip string
		switch v := r.Value.(type) {
		case []any:
			// If it's a slice, grab the first element (if it exists)
			if len(v) > 0 {
				if first, ok := v[0].(map[string]any); ok {
					ip, _ = first["ip"].(string)
				}
			}
		default:
			return libdns.Address{}, fmt.Errorf("unexpected type for A/AAAA value: %T", v)
		}
		addr, err := netip.ParseAddr(ip)
		if err != nil {
			return libdns.Address{}, fmt.Errorf("invalid IP address %q: %v", ip, err)
		}
		return libdns.Address{
			Name: name,
			TTL:  ttl,
			IP:   addr,
		}, nil
	case "CAA":
		var val CAARecordValue
		if err := decodeValue(r.Value, &val); err != nil {
			return nil, err
		}
		return libdns.CAA{
			Name:  name,
			TTL:   ttl,
			Tag:   val.Tag,
			Value: val.Value,
		}, nil
	case "CNAME":
		var val CNAMERecordValue
		if err := decodeValue(r.Value, &val); err != nil {
			return nil, err
		}
		return libdns.CNAME{
			Name:   name,
			TTL:    ttl,
			Target: val.Host,
		}, nil
	case "MX":
		var val MXRecordValue
		if err := decodeValue(r.Value, &val); err != nil {
			return nil, err
		}
		return libdns.MX{
			Name:       name,
			TTL:        ttl,
			Preference: val.Priority,
			Target:     val.Host,
		}, nil
	case "NS":
		var val NSRecordValue
		if err := decodeValue(r.Value, &val); err != nil {
			return nil, err
		}
		return libdns.NS{
			Name:   name,
			TTL:    ttl,
			Target: val.Host,
		}, nil
	case "SRV":
		var val SRVRecordValue
		if err := decodeValue(r.Value, &val); err != nil {
			return nil, err
		}
		return libdns.SRV{
			Name:     name,
			TTL:      ttl,
			Priority: val.Priority,
			Weight:   val.Weight,
			Port:     val.Port,
			Target:   val.Target,
		}, nil
	case "TXT":
		var val TXTRecordValue
		if err := decodeValue(r.Value, &val); err != nil {
			return nil, err
		}
		// unwrap the quotes from the content
		unwrappedContent := unwrapContent(val.Text)
		return libdns.TXT{
			Name: name,
			TTL:  ttl,
			Text: unwrappedContent,
		}, nil
	// 	fallthrough
	default:
		var fields map[string]any
		json.Unmarshal([]byte(r.Value.(string)), &fields)
		var vals []string
		for _, v := range fields {
			vals = append(vals, fmt.Sprintf("%v", v))
		}
		return libdns.RR{
			Name: name,
			TTL:  ttl,
			Type: r.Type,
			Data: strings.Join(vals, " "),
		}.Parse()
	}
}

func arvancloudRecord(r libdns.Record) (arDNSRecord, error) {

	rr := r.RR()
	arRec := arDNSRecord{
		// ID:   r.ID,
		Name: rr.Name,
		Type: strings.ToLower(rr.Type),
		TTL:  int(rr.TTL.Seconds()),
	}
	switch rec := r.(type) {
	case libdns.Address:
		arRec.Value = []ARecordValue{
			{
				IP: rec.IP.String(),
			},
		}
	case libdns.CNAME:
		arRec.Value = CNAMERecordValue{
			Host: rec.Target,
		}
	case libdns.NS:
		arRec.Value = NSRecordValue{
			Host: rec.Target,
		}
	case libdns.CAA:
		arRec.Value = CAARecordValue{
			Tag:   rec.Tag,
			Value: rec.Value,
		}
	case libdns.MX:
		arRec.Value = MXRecordValue{
			Priority: rec.Preference,
			Host:     rec.Target,
		}
	case libdns.SRV:
		arRec.Value = SRVRecordValue{
			Priority: rec.Priority,
			Weight:   rec.Weight,
			Port:     rec.Port,
			Target:   rec.Target,
		}
	case libdns.TXT:
		arRec.Value = TXTRecordValue{
			Text: rec.Text,
		}
	case libdns.Record:
		switch strings.ToUpper(rec.RR().Type) {
		case "A", "AAAA":
			arRec.Value = []ARecordValue{{IP: rec.RR().Data}}
		case "CNAME":
			arRec.Value = CNAMERecordValue{Host: rec.RR().Data}
		case "NS":
			arRec.Value = NSRecordValue{Host: rec.RR().Data}
		case "TXT":
			arRec.Value = TXTRecordValue{Text: rec.RR().Data}
		}
	}
	return arRec, nil
}

func decodeValue(input any, target any) error {
	raw, err := json.Marshal(input) // Turn the map back into JSON bytes
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target) // Unmarshal bytes into your struct
}

type arResponse struct {
	Data    json.RawMessage `json:"data,omitempty"`
	Status  bool            `json:"status,omitempty"`
	Errors  []string        `json:"errors,omitempty"`
	Message string          `json:"message,omitempty"`
	Links   *arLinks        `json:"links,omitempty"`
	Meta    *arMeta         `json:"meta,omitempty"`
}

type arMeta struct {
	CurrentPage int               `json:"current_page"`
	From        int               `json:"from"`
	LastPage    int               `json:"last_page"`
	Path        string            `json:"path"`
	PerPage     int               `json:"per_page"`
	To          int               `json:"to"`
	Total       int               `json:"total"`
	Links       []json.RawMessage `json:"links,omitempty"`
}

type arLinks struct {
	First *string `json:"First"`
	Last  *string `json:"Last"`
	Prev  *string `json:"Prev"`
	Next  *string `json:"Next"`
}
