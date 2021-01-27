package designate

import (
	"context"
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
	"github.com/libdns/libdns"
	"sync"
)

// Provider implements the libdns interfaces for OpenStack Designate.
type Provider struct {
	dnsClient     *gophercloud.ServiceClient
	AuthOpenStack AuthOpenStack `json:"auth_open_stack"`
	zoneID        string
	mu            sync.Mutex
}

// AuthOpenStack contains credentials for OpenStack Designate.
type AuthOpenStack struct {
	RegionName         string `json:"region_name"`
	TenantID           string `json:"tenant_id"`
	IdentityApiVersion string `json:"identity_api_version"`
	Password           string `json:"password"`
	AuthURL            string `json:"auth_url"`
	Username           string `json:"username"`
	TenantName         string `json:"tenant_name"`
	EndpointType       string `json:"endpoint_type"`
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	err := p.auth()
	if err != nil {
		return nil, fmt.Errorf("cannot authenticate to OpenStack Designate: %v", err)
	}

	err = p.setZone(zone)
	if err != nil {
		return nil, fmt.Errorf("cannot set ZONE: %v", err)
	}

	listOpts := recordsets.ListOpts{
		Status: "ACTIVE",
	}

	allPages, err := recordsets.ListByZone(p.dnsClient, p.zoneID, listOpts).AllPages()
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
	err := p.auth()
	if err != nil {
		return nil, fmt.Errorf("cannot authenticate to OpenStack Designate: %v", err)
	}

	err = p.setZone(zone)
	if err != nil {
		return nil, fmt.Errorf("cannot set ZONE: %v", err)
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
	err := p.auth()
	if err != nil {
		return nil, fmt.Errorf("cannot authenticate to OpenStack Designate: %v", err)
	}

	err = p.setZone(zone)
	if err != nil {
		return nil, fmt.Errorf("cannot set ZONE: %v", err)
	}

	var deletedRecords []libdns.Record

	for _, record := range records {
		recordID, err := p.getRecordID(record.Name, zone)
		if err != nil {
			return nil, err
		}

		if recordID == "" {
			return nil, errors.New("recordID does not exist")
		}

		err = p.deleteRecord(recordID)
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
	err := p.auth()
	if err != nil {
		return nil, fmt.Errorf("cannot authenticate to OpenStack Designate: %v", err)
	}

	err = p.setZone(zone)
	if err != nil {
		return nil, fmt.Errorf("cannot set ZONE: %v", err)
	}

	var setRecords []libdns.Record

	for _, record := range records {
		recordID, err := p.getRecordID(record.Name, zone)
		if err != nil {
			return nil, err
		}

		if recordID == "" {
			return nil, errors.New("recordID does not exist")
		}

		err = p.updateRecord(record, recordID)
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
