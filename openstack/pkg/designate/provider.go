package designate

import (
	"context"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
	"github.com/libdns/libdns"
	"github.com/pkg/errors"
)

type Provider struct {
	DNSClient  *gophercloud.ServiceClient
	Request    Request
	DNSOptions DNSOptions
}

type Request struct {
	ZoneID   string
	RecordID string
}

type DNSOptions struct {
	DNSDescription string
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	listOpts := recordsets.ListOpts{
		Status: "ACTIVE",
	}

	allPages, err := recordsets.ListByZone(p.DNSClient, p.Request.ZoneID, listOpts).AllPages()
	if err != nil {
		return nil, errors.Wrap(err, "trying to get list by ZoneID")
	}

	allRRs, err := recordsets.ExtractRecordSets(allPages)
	if err != nil {
		return nil, errors.Wrap(err, "trying to extract records")
	}

	return p.getRecords(allRRs)
}

// AppendRecords adds records to the zone and returns the records that were created.
// Due to technical limitations of the LiveDNS API, it may affect the TTL of similar records
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
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
