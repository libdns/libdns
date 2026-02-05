// Package arvancloud implements a DNS record management client compatible
// with the libdns interfaces for ArvanCloud.
package arvancloud

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with ArvanCloud.
type Provider struct {
	// AuthAPIKey is the API token for ArvanCloud.
	// It can be obtained from the ArvanCloud user panel.
	AuthAPIKey string `json:"auth_api_key,omitempty"`
	client     *client
	mu         sync.Mutex
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mu.Lock()
	if p.client == nil {
		p.client = newClient(p.AuthAPIKey)
	}
	p.mu.Unlock()
	zone = strings.TrimSuffix(zone, `.`)
	arvanRecords, err := p.client.getRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	records := make([]libdns.Record, 0, len(arvanRecords))
	for _, ar := range arvanRecords {
		libRecord, err := ar.toLibDNSRecord(zone)
		if err != nil {
			return nil, err
		}
		records = append(records, libRecord)
	}
	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	if p.client == nil {
		p.client = newClient(p.AuthAPIKey)
	}
	p.mu.Unlock()
	zone = strings.TrimSuffix(zone, `.`)
	var addedRecords []libdns.Record
	for _, r := range records {

		result, err := p.client.createRecord(ctx, zone, r)
		if err != nil {
			return nil, err
		}
		libRecord, err := result.toLibDNSRecord(zone)
		if err != nil {
			return nil, fmt.Errorf("parsing Arvancloud DNS record %+v: %v", r, err)
		}
		addedRecords = append(addedRecords, libRecord)
	}

	return addedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	if p.client == nil {
		p.client = newClient(p.AuthAPIKey)
	}
	p.mu.Unlock()
	zone = strings.TrimSuffix(zone, `.`)
	var results []libdns.Record
	existingRecords, err := p.client.getRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	for _, r := range records {
		// check for existing records
		existing := p.findExistingRecords(existingRecords, r, zone)
		if len(existingRecords) > 0 && len(existing) == 0 {
			// the record doesn't exist, create it
			result, err := p.client.createRecord(ctx, zone, r)
			if err != nil {
				return nil, err
			}
			libRecord, err := result.toLibDNSRecord(zone)
			if err != nil {
				return nil, fmt.Errorf("parsing Arvancloud DNS record %+v: %v", r, err)
			}
			results = append(results, libRecord)
			continue
		}

		if len(existing) > 1 {
			return nil, fmt.Errorf("unexpectedly found more than 1 record for %v", r)
		}

		arRec, err := arvancloudRecord(r)
		if err != nil {
			return nil, err
		}
		result, err := p.client.updateRecord(ctx, zone, existing[0].ID, arRec)
		if err != nil {
			return nil, err
		}
		libRecord, err := result.toLibDNSRecord(zone)
		if err != nil {
			return nil, fmt.Errorf("parsing Arvancloud DNS record %+v: %v", r, err)

		}
		results = append(results, libRecord)
	}

	return results, nil
}

// DeleteRecords deletes the specified records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mu.Lock()
	if p.client == nil {
		p.client = newClient(p.AuthAPIKey)
	}
	p.mu.Unlock()
	zone = strings.TrimSuffix(zone, `.`)
	existingRecords, err := p.client.getRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	var deletedRecords []libdns.Record

	for _, r := range records {
		existing := p.findExistingRecords(existingRecords, r, zone)
		if existing == nil {
			continue
		}

		for _, arRec := range existing {
			_, err := p.client.deleteRecord(ctx, zone, arRec.ID)
			if err != nil {
				return nil, err
			}
			deletedRecords = append(deletedRecords, r)
		}
	}

	return deletedRecords, nil
}

func (p *Provider) ListZones(ctx context.Context) ([]libdns.Zone, error) {

	p.mu.Lock()
	if p.client == nil {
		p.client = newClient(p.AuthAPIKey)
	}
	p.mu.Unlock()

	arDomains, err := p.client.getDomains(ctx)
	if err != nil {
		return nil, err
	}

	return arDomains, nil
}
// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
	_ libdns.ZoneLister  	= (*Provider)(nil)

)
