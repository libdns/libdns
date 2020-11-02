package vultr

import (
	"context"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

// Provider implements the libdns interfaces for Vultr
// Adapted from libdns/digitalocean to work with the Vultr API
type Provider struct {
	Client
	// APIToken is the Vultr API token
	// see https://my.vultr.com/settings/#settingsapi
	APIToken string `json:"auth_token"`
}

// unFQDN trims any trailing "." from fqdn.
func (p *Provider) unFQDN(fqdn string) string {
	return strings.TrimSuffix(fqdn, ".")
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	records, err := p.getDNSEntries(ctx, p.unFQDN(zone))
	if err != nil {
		return nil, err
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var appendedRecords []libdns.Record

	for _, record := range records {
		newRecord, err := p.addDNSRecord(ctx, p.unFQDN(zone), record)
		if err != nil {
			return nil, err
		}
		newRecord.TTL = time.Duration(newRecord.TTL) * time.Second
		appendedRecords = append(appendedRecords, newRecord)
	}

	return appendedRecords, nil
}

// DeleteRecords deletes the records from the zone.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var deletedRecords []libdns.Record

	for _, record := range records {
		deletedRecord, err := p.removeDNSRecord(ctx, p.unFQDN(zone), record)
		if err != nil {
			return nil, err
		}
		deletedRecord.TTL = time.Duration(deletedRecord.TTL) * time.Second
		deletedRecords = append(deletedRecords, deletedRecord)
	}

	return deletedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records
// or creating new ones. It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var setRecords []libdns.Record

	for _, record := range records {
		// TODO: if there is no ID, look up the Name, and fill it in, or call
		//       newRecord, err := p.addDNSEntry(ctx, zone, record)
		setRecord, err := p.updateDNSRecord(ctx, p.unFQDN(zone), record)
		if err != nil {
			return setRecords, err
		}
		setRecord.TTL = time.Duration(setRecord.TTL) * time.Second
		setRecords = append(setRecords, setRecord)
	}

	return setRecords, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
