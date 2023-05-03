// Package directadmin implements a DNS record management client compatible
// with the libdns interfaces for DirectAdmin.
package directadmin

import (
	"context"
	"sync"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with DirectAdmin.
type Provider struct {
	// ServerURL should be the hostname (with port if necessary) of the DirectAdmin instance
	// you are trying to use
	ServerURL string `json:"host,omitempty"`

	// User should be the DirectAdmin username that the Login Key is created under
	User string `json:"user,omitempty"`

	// LoginKey is used for authentication
	//
	// The key will need two permissions:
	//
	// `CMD_API_SHOW_DOMAINS`
	//
	// `CMD_API_DNS_CONTROL`
	//
	// Unless you are only using `GetRecords()`, in which case `CMD_API_DNS_CONTROL`
	// can be omitted
	LoginKey string `json:"login_key,omitempty"`

	// InsecureRequests is an optional parameter used to ignore SSL related errors on the
	// DirectAdmin host
	InsecureRequests bool `json:"insecure_requests,omitempty"`

	// Debug - can set this to stdout or stderr to dump
	// debugging information about the API interaction with
	// powerdns.  This will dump your auth token in plain text
	// so be careful.
	Debug string `json:"debug,omitempty"`

	mutex sync.Mutex
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	records, err := p.getZoneRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var created []libdns.Record
	for _, rec := range records {
		result, err := p.appendZoneRecord(ctx, zone, rec)
		if err != nil {
			return nil, err
		}
		created = append(created, result)
	}

	return created, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var updated []libdns.Record
	for _, rec := range records {
		result, err := p.setZoneRecord(ctx, zone, rec)
		if err != nil {
			return nil, err
		}
		updated = append(updated, result)
	}

	return updated, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var deleted []libdns.Record
	for _, rec := range records {
		result, err := p.deleteZoneRecord(ctx, zone, rec)
		if err != nil {
			return nil, err
		}
		deleted = append(deleted, result)
	}

	return deleted, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
