package domainchief

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

const (
	acmeChallengeLabel = "_acme-challenge"
	defaultTTL         = 3600
)

// acmeChallengeValue matches values the ACME challenge endpoints accept (base64url, as used by DNS-01).
var acmeChallengeValue = regexp.MustCompile(`^[A-Za-z0-9_-]{1,255}$`)

// supportedTypes are the record types that can be managed through the Domain Chief API.
var supportedTypes = map[string]bool{
	"A":     true,
	"AAAA":  true,
	"ALIAS": true,
	"CAA":   true,
	"CNAME": true,
	"MX":    true,
	"NS":    true,
	"SRV":   true,
	"TLSA":  true,
	"TXT":   true,
}

// hostnameTypes are the record types whose content is a single hostname.
var hostnameTypes = map[string]bool{
	"ALIAS": true,
	"CNAME": true,
	"NS":    true,
}

// zoneDomain returns the domain name for a zone, without the trailing dot.
func zoneDomain(zone string) string {
	return strings.ToLower(strings.TrimSuffix(zone, "."))
}

// isACMEChallenge reports whether the record can be published through the ACME challenge endpoints,
// which allows using a token with only the `domainchief:acme:write` scope.
func isACMEChallenge(rr libdns.RR) bool {
	if rr.Type != "TXT" || !acmeChallengeValue.MatchString(rr.Data) {
		return false
	}

	name := strings.ToLower(rr.Name)

	return name == acmeChallengeLabel || strings.HasPrefix(name, acmeChallengeLabel+".")
}

// toInput converts a libdns record to a Domain Chief API record.
func toInput(record libdns.Record) (recordInput, error) {
	rr := record.RR()

	input := recordInput{
		Type:    strings.ToUpper(rr.Type),
		Name:    apiName(rr.Name),
		Content: strings.TrimSpace(rr.Data),
		TTL:     apiTTL(rr.TTL),
	}

	if !supportedTypes[input.Type] {
		return input, fmt.Errorf("domain chief: record type %q is not supported", rr.Type)
	}

	switch {
	case input.Type == "MX":
		fields := strings.Fields(input.Content)
		if len(fields) != 2 {
			return input, fmt.Errorf("domain chief: malformed MX value %q, expected 'preference target'", rr.Data)
		}

		prio, err := strconv.ParseUint(fields[0], 10, 16)
		if err != nil {
			return input, fmt.Errorf("domain chief: invalid MX preference %q: %w", fields[0], err)
		}

		input.Prio = int(prio)
		input.Content = trimHostname(fields[1])
	case input.Type == "SRV":
		fields := strings.Fields(input.Content)
		if len(fields) != 4 {
			return input, fmt.Errorf("domain chief: malformed SRV value %q, expected 'priority weight port target'", rr.Data)
		}

		prio, err := strconv.ParseUint(fields[0], 10, 16)
		if err != nil {
			return input, fmt.Errorf("domain chief: invalid SRV priority %q: %w", fields[0], err)
		}

		input.Prio = int(prio)
		input.Content = fmt.Sprintf("%s %s %s", fields[1], fields[2], trimHostname(fields[3]))
	case hostnameTypes[input.Type]:
		input.Content = trimHostname(input.Content)
	case input.Type == "TXT":
		// TXT values are sent unquoted, the API adds the zone file quoting.
		input.Content = rr.Data
	}

	return input, nil
}

// toLibdns converts a Domain Chief API record to a libdns record.
//
// Records that cannot be parsed into a libdns struct type are returned as libdns.RR instead of failing.
func toLibdns(record apiRecord) libdns.Record {
	rr := libdns.RR{
		Name: libdnsName(record.Name),
		TTL:  time.Duration(record.TTL) * time.Second,
		Type: record.Type,
		Data: record.Content,
	}

	switch {
	case record.Type == "TXT":
		rr.Data = record.Value
	case record.Type == "MX":
		rr.Data = fmt.Sprintf("%d %s", record.Prio, fqdnHostname(record.Content))
	case record.Type == "SRV":
		fields := strings.Fields(record.Content)
		if len(fields) == 3 {
			rr.Data = fmt.Sprintf("%d %s %s %s", record.Prio, fields[0], fields[1], fqdnHostname(fields[2]))
		}
	case hostnameTypes[record.Type]:
		rr.Data = fqdnHostname(record.Content)
	}

	parsed, err := rr.Parse()
	if err != nil {
		return rr
	}

	return parsed
}

// matches reports whether an existing record matches the record to delete.
//
// Following the libdns semantics, an empty type, zero TTL or empty value matches any value.
func matches(existing apiRecord, record libdns.Record) bool {
	rr := record.RR()

	if !strings.EqualFold(apiName(rr.Name), existing.Name) {
		return false
	}

	if rr.Type != "" && !strings.EqualFold(rr.Type, existing.Type) {
		return false
	}

	if rr.TTL != 0 && apiTTL(rr.TTL) != existing.TTL {
		return false
	}

	if rr.Data == "" {
		return true
	}

	input, err := toInput(record)
	if err != nil {
		return false
	}

	return sameContent(existing, input)
}

// sameContent reports whether an existing record holds the same data as the input.
func sameContent(existing apiRecord, input recordInput) bool {
	if !strings.EqualFold(existing.Type, input.Type) {
		return false
	}

	switch input.Type {
	case "TXT":
		return existing.Value == input.Content
	case "MX":
		return existing.Prio == input.Prio && strings.EqualFold(trimHostname(existing.Content), input.Content)
	case "SRV":
		return existing.Prio == input.Prio && strings.EqualFold(normalizeFields(existing.Content), normalizeFields(input.Content))
	default:
		if hostnameTypes[input.Type] {
			return strings.EqualFold(trimHostname(existing.Content), input.Content)
		}

		return strings.EqualFold(normalizeFields(existing.Content), normalizeFields(input.Content))
	}
}

// apiName converts a libdns relative name to a Domain Chief record name, which is empty for the zone apex.
func apiName(name string) string {
	name = strings.ToLower(strings.TrimSuffix(name, "."))

	if name == "@" {
		return ""
	}

	return name
}

// libdnsName converts a Domain Chief record name to a libdns relative name.
func libdnsName(name string) string {
	if name == "" {
		return "@"
	}

	return name
}

// supportedTTLs are the TTLs the API stores, other values are rounded to the closest one.
var supportedTTLs = []int{60, 300, 900, 1800, 3600, 21600, 43200, 86400, 172800, 259200, 604800, 1209600, 2592000}

// apiTTL converts a libdns TTL to the TTL the API stores.
//
// A zero TTL uses the default TTL, other values are rounded to the closest supported TTL the same way the API does.
func apiTTL(ttl time.Duration) int {
	if ttl == 0 {
		return defaultTTL
	}

	seconds := int(ttl.Seconds())
	closest := supportedTTLs[0]

	for _, supported := range supportedTTLs {
		if abs(supported-seconds) < abs(closest-seconds) {
			closest = supported
		}
	}

	return closest
}

func abs(value int) int {
	if value < 0 {
		return -value
	}

	return value
}

func trimHostname(hostname string) string {
	if hostname == "." {
		return hostname
	}

	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(hostname), "."))
}

func fqdnHostname(hostname string) string {
	if hostname == "" || strings.HasSuffix(hostname, ".") {
		return hostname
	}

	return hostname + "."
}

func normalizeFields(content string) string {
	fields := strings.Fields(content)

	if len(fields) > 0 {
		fields[len(fields)-1] = trimHostname(fields[len(fields)-1])
	}

	return strings.Join(fields, " ")
}
