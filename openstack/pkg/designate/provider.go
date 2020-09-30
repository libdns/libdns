package designate

import (
	"context"
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
	"github.com/libdns/libdns"
)

type Provider struct {
	DNSClient      *gophercloud.ServiceClient
	AuthOpenStack  AuthOpenStack
	ZoneID         string
	RecordID       string
	DNSDescription string
}

type AuthOpenStack struct {
	RegionName         string
	TenantID           string
	IdentityApiVersion string
	Password           string
	AuthURL            string
	Username           string
	TenantName         string
	EndpointType       string
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	err := p.auth(zone)
	if err != nil {
		return nil, fmt.Errorf("cannot authenticate to OpenStack Designate: %v", err)
	}

	listOpts := recordsets.ListOpts{
		Status: "ACTIVE",
	}

	allPages, err := recordsets.ListByZone(p.DNSClient, p.ZoneID, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("trying to get list by ZoneID: %v", err)
	}

	allRRs, err := recordsets.ExtractRecordSets(allPages)
	if err != nil {
		return nil, fmt.Errorf("trying to extract records: %v", err)
	}

	return p.getRecords(allRRs)
}

// AppendRecords adds records to the zone and returns the records that were created.
// Due to technical limitations of the LiveDNS API, it may affect the TTL of similar records
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	err := p.auth(zone)
	if err != nil {
		return nil, fmt.Errorf("trying to run run")
	}

	var appendedRecords []libdns.Record

	for _, record := range records {
		err := p.createRecord(record, zone)
		if err != nil {
			return nil, err
		}
		appendedRecords = append(appendedRecords, record)
	}

	return appendedRecords, nil
}

// DeleteRecords deletes records from the zone and returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	err := p.auth(zone)
	if err != nil {
		return nil, fmt.Errorf("trying to run run")
	}

	var deletedRecords []libdns.Record

	for _, record := range records {
		recordID, err := p.getRecordID(record.Name, zone)
		if err != nil {
			return nil, err
		}

		p.setRecordID(recordID)

		if recordID == "" {
			return nil, errors.New("recordID does not exist")
		}

		err = p.deleteRecord(record, zone)
		if err != nil {
			return nil, err
		}
		deletedRecords = append(deletedRecords, record)
	}

	return deletedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones, and returns the recordsthat were updated.
// Due to technical limitations of the LiveDNS API, it may affect the TTL of similar records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	err := p.auth(zone)
	if err != nil {
		return nil, fmt.Errorf("trying to run run")
	}

	var setRecords []libdns.Record

	for _, record := range records {
		recordID, err := p.getRecordID(record.Name, zone)
		if err != nil {
			return nil, err
		}

		p.setRecordID(recordID)

		if recordID == "" {
			return nil, errors.New("recordID does not exist")
		}

		err = p.updateRecord(record, zone)
		if err != nil {
			return setRecords, err
		}
		setRecords = append(setRecords, record)
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
