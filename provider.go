// Package dnsexit implements a DNS record management client compatible
// with the libdns interfaces for DNSExit.
package dnsexit

import (
	"context"
	"sync"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with DNSExit.
type Provider struct {
	APIKey string `json:"api_key,omitempty"`
	mutex  sync.Mutex
}

// GetRecords lists all the records in the zone.
// NOTE: DNSExit API does not facilitate this, so Google DNS is used.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	libRecords, err := p.getDomain(ctx, zone)
	if err != nil {
		return nil, err
	}

	return libRecords, nil
}

// AppendRecords adds records to the zone. It returns the records that were added. This function will fail if a record with the same name already exists.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return p.amendRecords(zone, records, Append)
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return p.amendRecords(zone, records, Set)
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return p.amendRecords(zone, records, Delete)
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
