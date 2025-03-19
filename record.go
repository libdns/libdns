package libdns

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// Record is any type that can reduce itself to the [RR] struct.
type Record interface {
	RR() (RR, error)
}

// RR represents a DNS Resource Record, which resembles how records are
// represented by DNS servers in zone files (see
// https://en.wikipedia.org/wiki/Domain_Name_System#Resource_records).
//
// The fields in this struct are common to all RRs, with the data field
// being opaque; it has no particular meaning until it is parsed.
type RR struct {
	// The name of the record. It is partially qualified, relative to the zone.
	// For the sake of consistency, use "@" to represent the root of the zone.
	// An empty name typically refers to the last-specified name in the zone
	// file, which is only determinable in specific contexts.
	//
	// (For the following examples, assume the zone is “example.com.”)
	//
	// Examples:
	//   - “www” (for “www.example.com.”)
	//   - “@” (for “example.com.”)
	//   - “subdomain” (for “subdomain.example.com.”)
	//   - “sub.subdomain” (for “sub.subdomain.example.com.”)
	//
	// Invalid:
	//   - “www.example.com.” (fully-qualified)
	//   - “example.net.” (fully-qualified)
	//   - "" (empty)
	//
	// Valid, but probably doesn't do what you want:
	//   - “www.example.net” (refers to “www.example.net.example.com.”)
	Name string

	// The time-to-live of the record. This is represented in the DNS zone file as
	// an unsigned integral number of seconds, but is provided here as a
	// [time.Duration] for ease of use in Go code. Fractions of seconds will be
	// rounded down (truncated). A value of 0 means that the record should not be
	// cached. Some provider implementations may assume a default TTL from 0; to
	// avoid this, set TTL to a sub-second duration.
	//
	// Note that some providers may reject or silently increase TTLs that are below
	// a certain threshold, and that DNS resolvers may choose to ignore your TTL
	// settings, so it is recommended to not rely on the exact TTL value.
	TTL time.Duration

	// The type of the record as an uppercase string. DNS provider packages are
	// encouraged to support as many of the most common record types as possible,
	// especially: A, AAAA, CNAME, TXT, HTTPS, and SRV.
	//
	// Other custom record types may be supported with implementation-defined
	// behavior.
	Type string

	// The data (or "value") of the record. This field should be formatted in the
	// standard zone file syntax (technically, the "RDATA" field as defined by
	// RFC 1034).
	Data string
}

// RR returns itself. This may be the case when trying to parse an RR type
// that is not (yet) supported/implemented by this package.
func (r RR) RR() (RR, error) { return r, nil }

// Parse returns a type-specific structure for this RR, if it is
// a known/supported type. Otherwise, it returns itself.
//
// Callers will typically want to type-assert (or use a type switch on)
// the return value to extract values or manipulate it.
func (r RR) Parse() (Record, error) {
	switch r.Type {
	case "A":
		return r.toA()
	case "AAAA":
		return r.toAAAA()
	case "CNAME":
		return r.toCNAME()
	case "HTTPS":
		return r.toHTTPS()
	case "SRV":
		return r.toSRV()
	case "TXT":
		return r.toTXT()
	default:
		return r, nil
	}
}

func (r RR) toA() (A, error) {
	if expectedType := "A"; r.Type != expectedType {
		return A{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}

	ip, err := netip.ParseAddr(r.Data)
	if err != nil {
		return A{}, fmt.Errorf("invalid IP address %q: %v", r.Data, err)
	}
	if !ip.Is4() {
		return A{}, fmt.Errorf("value is not IPv4: %s (parsed=%s, bitlen=%d)", r.Data, ip, ip.BitLen())
	}

	return A{
		Name: r.Name,
		IP:   ip,
		TTL:  r.TTL,
	}, nil
}

func (r RR) toAAAA() (AAAA, error) {
	if expectedType := "AAAA"; r.Type != expectedType {
		return AAAA{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}

	ip, err := netip.ParseAddr(r.Data)
	if err != nil {
		return AAAA{}, fmt.Errorf("invalid IP address %q: %v", r.Data, err)
	}
	if !ip.Is6() {
		return AAAA{}, fmt.Errorf("value is not IPv6: %s (parsed=%s, bitlen=%d)", r.Data, ip, ip.BitLen())
	}

	return AAAA{
		Name: r.Name,
		IP:   ip,
		TTL:  r.TTL,
	}, nil
}

func (r RR) toCNAME() (CNAME, error) {
	if expectedType := "CNAME"; r.Type != expectedType {
		return CNAME{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}
	return CNAME{
		Name:   r.Name,
		TTL:    r.TTL,
		Target: r.Data,
	}, nil
}

func (r RR) toHTTPS() (HTTPS, error) {
	if expectedType := "HTTPS"; r.Type != expectedType {
		return HTTPS{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}

	parts := strings.SplitN(r.Data, " ", 3)
	if expectedLen := 3; len(parts) != expectedLen {
		return HTTPS{}, fmt.Errorf("malformed HTTPS value; expected %d fields in the form 'priority target SvcParams'", expectedLen)
	}

	priority, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 16)
	if err != nil {
		return HTTPS{}, fmt.Errorf("invalid priority %s: %v", parts[0], err)
	}
	target := parts[1]

	svcParams, err := ParseSvcParams(parts[2])
	if err != nil {
		return HTTPS{}, fmt.Errorf("invalid SvcParams: %w", err)
	}

	return HTTPS{
		Name:     r.Name,
		TTL:      r.TTL,
		Priority: uint16(priority),
		Target:   target,
		Value:    svcParams,
	}, nil
}

func (r RR) toSRV() (SRV, error) {
	if expectedType := "SRV"; r.Type != expectedType {
		return SRV{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}

	fields := strings.Fields(r.Data)
	if expectedLen := 4; len(fields) != expectedLen {
		return SRV{}, fmt.Errorf("malformed SRV value; expected %d fields in the form 'priority weight port target'", expectedLen)
	}

	priority, err := strconv.ParseUint(fields[0], 10, 16)
	if err != nil {
		return SRV{}, fmt.Errorf("invalid priority %s: %v", fields[0], err)
	}
	weight, err := strconv.ParseUint(fields[1], 10, 16)
	if err != nil {
		return SRV{}, fmt.Errorf("invalid weight %s: %v", fields[0], err)
	}
	port, err := strconv.ParseUint(fields[2], 10, 16)
	if err != nil {
		return SRV{}, fmt.Errorf("invalid port %s: %v", fields[0], err)
	}
	target := fields[3]

	parts := strings.SplitN(r.Name, ".", 3)
	if len(parts) < 3 {
		return SRV{}, fmt.Errorf("name %v does not contain enough fields; expected format: '_service._proto.name'", r.Name)
	}

	return SRV{
		Service:  strings.TrimPrefix(parts[0], "_"),
		Proto:    strings.TrimPrefix(parts[1], "_"),
		Name:     parts[2],
		TTL:      r.TTL,
		Priority: uint16(priority),
		Weight:   uint16(weight),
		Port:     uint16(port),
		Target:   target,
	}, nil
}

func (r RR) toTXT() (TXT, error) {
	if expectedType := "TXT"; r.Type != expectedType {
		return TXT{}, fmt.Errorf("record type not %s: %s", expectedType, r.Type)
	}
	return TXT{
		Name: r.Name,
		TTL:  r.TTL,
		Text: r.Data,
	}, nil
}

// SvcParams represents SvcParamKey=SvcParamValue pairs as described in
// RFC 9460 section 2.1. See https://www.rfc-editor.org/rfc/rfc9460#presentation.
type SvcParams map[string][]string

// String serializes svcParams into zone presentation format described by RFC 9460.
func (params SvcParams) String() string {
	var sb strings.Builder
	for key, vals := range params {
		if sb.Len() > 0 {
			sb.WriteRune(' ')
		}
		sb.WriteString(key)
		var hasVal, needsQuotes bool
		for _, val := range vals {
			if len(val) > 0 {
				hasVal = true
			}
			if strings.ContainsAny(val, `" `) {
				needsQuotes = true
			}
			if hasVal && needsQuotes {
				break
			}
		}
		if hasVal {
			sb.WriteRune('=')
		}
		if needsQuotes {
			sb.WriteRune('"')
		}
		for i, val := range vals {
			if i > 0 {
				sb.WriteRune(',')
			}
			val = strings.ReplaceAll(val, `"`, `\"`)
			val = strings.ReplaceAll(val, `,`, `\,`)
			sb.WriteString(val)
		}
		if needsQuotes {
			sb.WriteRune('"')
		}
	}
	return sb.String()
}

// ParseSvcParams parses a SvcParams string described by RFC 9460 into a structured type.
func ParseSvcParams(input string) (SvcParams, error) {
	if len(input) > 4096 {
		return nil, fmt.Errorf("input too long: %d", len(input))
	}

	params := make(SvcParams)
	input = strings.TrimSpace(input) + " "

	for cursor := 0; cursor < len(input); cursor++ {
		var key, rawVal string

	keyValPair:
		for i := cursor; i < len(input); i++ {
			switch input[i] {
			case '=':
				key = strings.ToLower(strings.TrimSpace(input[cursor:i]))
				i++
				cursor = i

				var quoted bool
				if input[cursor] == '"' {
					quoted = true
					i++
					cursor = i
				}

				var escaped bool

				for j := cursor; j < len(input); j++ {
					switch input[j] {
					case '"':
						if !quoted {
							return nil, fmt.Errorf("illegal DQUOTE at position %d", j)
						}
						if !escaped {
							// end of quoted value
							rawVal = input[cursor:j]
							j++
							cursor = j
							break keyValPair
						}
					case '\\':
						escaped = true
					case ' ', '\t', '\n', '\r':
						if !quoted {
							// end of unquoted value
							rawVal = input[cursor:j]
							cursor = j
							break keyValPair
						}
					default:
						escaped = false
					}
				}

			case ' ', '\t', '\n', '\r':
				// key with no value (flag)
				key = input[cursor:i]
				params[key] = []string{}
				cursor = i
				break keyValPair
			}
		}

		if rawVal == "" {
			continue
		}

		var sb strings.Builder

		var escape int // start of escape sequence (after \, so 0 is never a valid start)
		for i := 0; i < len(rawVal); i++ {
			ch := rawVal[i]
			if escape > 0 {
				// validate escape sequence
				// (RFC 9460 Appendix A)
				// escaped:   "\" ( non-digit / dec-octet )
				// non-digit: "%x21-2F / %x3A-7E"
				// dec-octet: "0-255 as a 3-digit decimal number"
				if ch >= '0' && ch <= '9' {
					// advance to end of decimal octet, which must be 3 digits
					i += 2
					if i > len(rawVal) {
						return nil, fmt.Errorf("value ends with incomplete escape sequence: %s", rawVal[escape:])
					}
					decOctet, err := strconv.Atoi(rawVal[escape : i+1])
					if err != nil {
						return nil, err
					}
					if decOctet < 0 || decOctet > 255 {
						return nil, fmt.Errorf("invalid decimal octet in escape sequence: %s (%d)", rawVal[escape:i], decOctet)
					}
					sb.WriteRune(rune(decOctet))
					escape = 0
					continue
				} else if (ch < 0x21 || ch > 0x2F) && (ch < 0x3A && ch > 0x7E) {
					return nil, fmt.Errorf("illegal escape sequence %s", rawVal[escape:i])
				}
			}
			switch ch {
			case ';', '(', ')':
				// RFC 9460 Appendix A:
				// > contiguous  = 1*( non-special / escaped )
				// > non-special is VCHAR minus DQUOTE, ";", "(", ")", and "\".
				return nil, fmt.Errorf("illegal character in value %q at position %d: %s", rawVal, i, string(ch))
			case '\\':
				escape = i + 1
			default:
				sb.WriteByte(ch)
				escape = 0
			}
		}

		params[key] = strings.Split(sb.String(), ",")
	}

	return params, nil
}
