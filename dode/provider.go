// Package dode implements a DNS record management client compatible
// with the libdns interfaces for do.de. Unfortunately, the do.de API only
// supports creating and removing TXT records for domains starting with `_acme-challenge.`
package dode

import (
	"context"
	"fmt"
	"strings"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with do.de.
type Provider struct {
	// API token for do.de API
	APIToken string `json:"api_token,omitempty"`
}

const notSupportedErrorMsg = "the do.de API only supports creating and removing TXT records for domains starting with '_acme-challenge.'"
const acmeChallenge = "_acme-challenge."

// GetRecords lists all the records in the zone.
//
// This is unsupported by the do.de API.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	return nil, fmt.Errorf(notSupportedErrorMsg)
}

// AppendRecords adds records to the zone. It returns the records that were added.
//
// The do.de API only supports creating TXT records that start with `_acme-challenge.`.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	for _, rec := range records {
		if rec.Type != "TXT" || !strings.HasPrefix(rec.Name, acmeChallenge) {
			return nil, fmt.Errorf(notSupportedErrorMsg)
		}

		name := libdns.AbsoluteName(rec.Name, zone)
		err := p.createACMERecord(ctx, strings.TrimSuffix(name, "."), rec.Value)
		if err != nil {
			return nil, err
		}
	}

	return records, nil

}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
//
// The do.de API only supports creating TXT records that start with `_acme-challenge.`.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return p.AppendRecords(ctx, zone, records)
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
//
// The do.de API only supports deleting TXT records that start with `_acme-challenge.`.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	for _, rec := range records {
		if rec.Type != "TXT" || !strings.HasPrefix(rec.Name, acmeChallenge) {
			return nil, fmt.Errorf(notSupportedErrorMsg)
		}

		err := p.deleteACMERecord(ctx, strings.TrimSuffix(libdns.AbsoluteName(rec.Name, zone), "."))
		if err != nil {
			return nil, err
		}
	}

	return records, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
