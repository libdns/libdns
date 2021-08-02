// libdns implementation for IONOS DNS API.
// libdns uses FQDN, i.e. domain names that terminate with ".". From the
// libdns documentation:
//   For example, an A record called "sub" in zone "example.com." represents
//   a fully-qualified domain name (FQDN) of "sub.example.com."
//
// The IONOS API seems not to use FQDNs.
//
// https://developer.hosting.ionos.de/docs/dns
package ionos

import (
	"context"
	"strings"

	"github.com/libdns/libdns"
)

// Provider implements the libdns interfaces for IONOS
type Provider struct {
	// AuthAPIToken is the IONOS Auth API token -
	// see https://dns.ionos.com/api-docs#section/Authentication/Auth-API-Token
	AuthAPIToken string `json:"auth_api_token"`
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	return getAllRecords(ctx, p.AuthAPIToken, unFQDN(zone))
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var appendedRecords []libdns.Record

	for _, record := range records {
		newRecord, err := createRecord(ctx, p.AuthAPIToken, unFQDN(zone), record)
		if err != nil {
			return nil, err
		}
		appendedRecords = append(appendedRecords, newRecord)
	}
	return appendedRecords, nil
}

// DeleteRecords deletes the records from the zone.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	for _, record := range records {
		err := deleteRecord(ctx, p.AuthAPIToken, unFQDN(zone), record)
		if err != nil {
			return nil, err
		}
	}
	return records, nil
}

// SetRecords sets the records in the zone, either by updating existing records
// or creating new ones. It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var setRecords []libdns.Record

	for _, record := range records {
		setRecord, err := createOrUpdateRecord(ctx, p.AuthAPIToken, unFQDN(zone), record)
		if err != nil {
			return setRecords, err
		}
		setRecords = append(setRecords, setRecord)
	}

	return setRecords, nil
}

// unFQDN trims any trailing "." from fqdn. IONOS's API does not use FQDNs.
func unFQDN(fqdn string) string {
	return strings.TrimSuffix(fqdn, ".")
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
