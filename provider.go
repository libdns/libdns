// Package netcup implements a DNS record management client compatible
// with the libdns interfaces for netcup.
package netcup

import (
	"context"
	"fmt"
	"sync"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with netcup.
// CustomerNumber, APIKey and APIPassword have to be filled with the respective credentials from netcup.
// The netcup API requires a session ID for all requests, so at the beginning of each method call
// a login is performed to receive the session ID and at the end the session is stopped with a logout.
// The mutex locks concurrent access on all four implemented methods to make sure there is
// no race condition in the netcup zone and record configuration.
type Provider struct {
	CustomerNumber string `json:"customer_number"`
	APIKey         string `json:"api_key"`
	APIPassword    string `json:"api_password"`
	mutex          sync.Mutex
}

const loggingPrefixLibdnsNetcup = "[libdns_netcup]"

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	fmt.Printf("%v Getting records of zone %v\n", loggingPrefixLibdnsNetcup, zone)

	apiSessionID, err := p.login(ctx)
	if err != nil {
		return nil, err
	}
	defer p.logout(ctx, apiSessionID)

	shortZone := unFQDN(zone)

	dnsZone, err := p.infoDNSZone(ctx, shortZone, apiSessionID)
	if err != nil {
		return nil, err
	}

	recordSet, err := p.infoDNSRecords(ctx, shortZone, apiSessionID)
	if err != nil {
		return nil, err
	}

	return toLibdnsRecords(recordSet.DnsRecords, dnsZone.TTL), nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
// netcup records cannot have individual TTLs, there is one TTL for all records in the zone
//
// For each input record, if no ID is given, the first record that matches the host name and type is searched.
// If none is found or the search result doesn't equal the input, a new one is appended.
// For MX records the priority is needed as an additional search parameter.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	fmt.Printf("%v Appending records %+v to zone %v\n", loggingPrefixLibdnsNetcup, records, zone)

	apiSessionID, err := p.login(ctx)
	if err != nil {
		return nil, err
	}
	defer p.logout(ctx, apiSessionID)

	shortZone := unFQDN(zone)

	dnsZone, err := p.infoDNSZone(ctx, shortZone, apiSessionID)
	if err != nil {
		return nil, err
	}

	existingRecordSet, err := p.infoDNSRecords(ctx, shortZone, apiSessionID)
	if err != nil {
		return nil, err
	}

	netcupRecords := toNetcupRecords(records)
	recordsToAppend := getRecordsToAppend(netcupRecords, existingRecordSet.DnsRecords)
	if len(recordsToAppend) == 0 {
		return []libdns.Record{}, nil
	}
	recordSetToAppend := dnsRecordSet{
		DnsRecords: recordsToAppend,
	}
	updatedRecordSet, err := p.updateDNSRecords(ctx, shortZone, recordSetToAppend, apiSessionID)
	if err != nil {
		return nil, err
	}

	// the netcup API always returns all records, so the ones before the update have to be compared to the ones after to return only the appended records
	appendedRecords := difference(updatedRecordSet.DnsRecords, existingRecordSet.DnsRecords)

	return toLibdnsRecords(appendedRecords, dnsZone.TTL), nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
//
// netcup records cannot have individual TTLs, there is one TTL for all records in the zone. So these can not be set.
//
// For each input record, if no ID is given, the first record that matches the host name and type is searched.
// If none is found, the input is appended. If one is found, it is updated accordingly.
// For MX records the priority is needed as an additional search parameter.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	fmt.Printf("%v Setting records %+v for zone %v\n", loggingPrefixLibdnsNetcup, records, zone)

	apiSessionID, err := p.login(ctx)
	if err != nil {
		return nil, err
	}
	defer p.logout(ctx, apiSessionID)

	shortZone := unFQDN(zone)

	dnsZone, err := p.infoDNSZone(ctx, shortZone, apiSessionID)
	if err != nil {
		return nil, err
	}

	existingRecordSet, err := p.infoDNSRecords(ctx, shortZone, apiSessionID)
	if err != nil {
		return nil, err
	}

	netcupRecords := toNetcupRecords(records)
	recordsToSet := getRecordsToSet(netcupRecords, existingRecordSet.DnsRecords)
	if len(recordsToSet) == 0 {
		return []libdns.Record{}, nil
	}
	recordSetToSet := dnsRecordSet{
		DnsRecords: recordsToSet,
	}
	updatedRecordSet, err := p.updateDNSRecords(ctx, shortZone, recordSetToSet, apiSessionID)
	if err != nil {
		return nil, err
	}

	// the netcup API always returns all records, so the ones before the update have to be compared to the ones after to return only the updated records
	updatedRecords := difference(updatedRecordSet.DnsRecords, existingRecordSet.DnsRecords)

	return toLibdnsRecords(updatedRecords, dnsZone.TTL), nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
//
// For each input record, if no ID is given, the first record that matches the host name and type is searched and deleted.
// For MX records the priority is needed as an additional search parameter.
// To be safe, the records to delete should include the IDs (for example from GetRecords)
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	fmt.Printf("%v Deleting records %+v from zone %v\n", loggingPrefixLibdnsNetcup, records, zone)

	apiSessionID, err := p.login(ctx)
	if err != nil {
		return nil, err
	}
	defer p.logout(ctx, apiSessionID)

	shortZone := unFQDN(zone)

	dnsZone, err := p.infoDNSZone(ctx, shortZone, apiSessionID)
	if err != nil {
		return nil, err
	}

	existingRecordSet, err := p.infoDNSRecords(ctx, shortZone, apiSessionID)
	if err != nil {
		return nil, err
	}

	netcupRecords := toNetcupRecords(records)
	recordsToDelete := getRecordsToDelete(netcupRecords, existingRecordSet.DnsRecords)
	if len(recordsToDelete) == 0 {
		return []libdns.Record{}, nil
	}
	recordSetToDelete := dnsRecordSet{
		DnsRecords: recordsToDelete,
	}
	updatedRecordSet, err := p.updateDNSRecords(ctx, shortZone, recordSetToDelete, apiSessionID)
	if err != nil {
		return nil, err
	}

	// the netcup API always returns all records, so the ones before the deletion have to be compared to the ones after to return only the deleted records
	deletedRecords := difference(existingRecordSet.DnsRecords, updatedRecordSet.DnsRecords)

	return toLibdnsRecords(deletedRecords, dnsZone.TTL), nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
