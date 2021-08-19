// Package namecheap implements a DNS record management client compatible
// with the libdns interfaces for namecheap.
package namecheap

import (
	"context"
	"sync"
	"time"

	"github.com/libdns/libdns"

	"github.com/libdns/namecheap/internal/namecheap"
)

func parseIntoHostRecord(record libdns.Record) namecheap.HostRecord {
	return namecheap.HostRecord{
		HostID:     record.ID,
		RecordType: namecheap.RecordType(record.Type),
		Name:       record.Name,
		TTL:        uint16(record.TTL.Seconds()),
		Address:    record.Value,
	}
}

func parseFromHostRecord(hostRecord namecheap.HostRecord) libdns.Record {
	return libdns.Record{
		ID:    hostRecord.HostID,
		Type:  string(hostRecord.RecordType),
		Name:  hostRecord.Name,
		TTL:   time.Duration(hostRecord.TTL) * time.Second,
		Value: hostRecord.Address,
	}
}

// Provider facilitates DNS record manipulation with namecheap.
// The libdns methods that return updated structs do not have
// their ID fields set since this information is not returned
// by the namecheap API.
type Provider struct {
	// APIKey is your namecheap API key.
	// See: https://www.namecheap.com/support/api/intro/
	// for more details.
	APIKey string `json:"api_key,omitempty"`

	// User is your namecheap API user. This can be the same as your username.
	User string `json:"user,omitempty"`

	// APIEndpoint to use. If testing, you can use the "sandbox" endpoint
	// instead of the production one.
	APIEndpoint string `json:"api_endpoint,omitempty"`

	// ClientIP is the IP address of the requesting client.
	// If this is not set, a discovery service will be
	// used to determine the public ip of the machine.
	// You must first whitelist your IP in the namecheap console
	// before using the API.
	ClientIP string `json:"client_IP,omitempty`

	mu sync.Mutex
}

// getClient inititializes a new namecheap client.
func (p *Provider) getClient() (*namecheap.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	options := []namecheap.ClientOption{}
	if p.APIEndpoint != "" {
		options = append(options, namecheap.WithEndpoint(p.APIEndpoint))
	}

	if p.ClientIP == "" {
		options = append(options, namecheap.AutoDiscoverPublicIP())
	}

	client, err := namecheap.NewClient(p.APIKey, p.User, options...)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// GetRecords lists all the records in the zone.
// This method does return records with the ID field set.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	client, err := p.getClient()
	if err != nil {
		return nil, err
	}

	hostRecords, err := client.GetHosts(ctx, zone)
	if err != nil {
		return nil, err
	}

	var records []libdns.Record
	for _, hr := range hostRecords {
		records = append(records, parseFromHostRecord(hr))
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
// Note that the records returned do NOT have their IDs set as the namecheap
// API does not return this info.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var hostRecords []namecheap.HostRecord
	for _, r := range records {
		hostRecords = append(hostRecords, parseIntoHostRecord(r))
	}

	client, err := p.getClient()
	if err != nil {
		return nil, err
	}

	_, err = client.AddHosts(ctx, zone, hostRecords)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records. // Note that the records returned do NOT have their IDs set as the namecheap
// API does not return this info.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var hostRecords []namecheap.HostRecord
	for _, r := range records {
		hostRecords = append(hostRecords, parseIntoHostRecord(r))
	}

	client, err := p.getClient()
	if err != nil {
		return nil, err
	}

	_, err = client.SetHosts(ctx, zone, hostRecords)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
// Note that the records returned do NOT have their IDs set as the namecheap
// API does not return this info.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var hostRecords []namecheap.HostRecord
	for _, r := range records {
		hostRecords = append(hostRecords, parseIntoHostRecord(r))
	}

	client, err := p.getClient()
	if err != nil {
		return nil, err
	}

	_, err = client.DeleteHosts(ctx, zone, hostRecords)
	if err != nil {
		return nil, err
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
