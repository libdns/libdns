// Package selectelv2 implements a DNS record management client compatible
// with the libdns interfaces for selectel v2.
package selectelv2

import (
	"context"
	"sync"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with <TODO: PROVIDER NAME>.
type Provider struct {
	User string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	AccountId string `json:"account_id,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
	KeystoneToken string
	ZonesCache map[string]string
	once sync.Once
	mutex sync.Mutex
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.uniRecords(recordMethods.get, ctx, zone, nil)
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.uniRecords(recordMethods.append, ctx, zone, records)
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.uniRecords(recordMethods.set, ctx, zone, records)
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.uniRecords(recordMethods.delete, ctx, zone, records)
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
