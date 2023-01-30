package linode

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/libdns/libdns"
	"github.com/linode/linodego"
)

func (p *Provider) init(ctx context.Context) {
	p.client = linodego.NewClient(http.DefaultClient)
}

func (p *Provider) convertRecordType(recordType string) linodego.DomainRecordType {
	switch recordType {
	case "A":
		return linodego.RecordTypeA
	case "AAAA":
		return linodego.RecordTypeAAAA
	case "CNAME":
		return linodego.RecordTypeCNAME
	case "MX":
		return linodego.RecordTypeMX
	case "CAA":
		return linodego.RecordTypeCAA
	case "NS":
		return linodego.RecordTypeNS
	case "TXT":
		return linodego.RecordTypeTXT
	case "PTR":
		return linodego.RecordTypePTR
	case "SRV":
		return linodego.RecordTypeSRV
	}
	return linodego.RecordTypeA
}

func (p *Provider) getDomainIdByZone(ctx context.Context, zone string) (int, error) {
	var done bool
	var page int
	listOptions := linodego.NewListOptions(page, "")
	var domainId int
	for {
		domains, err := p.client.ListDomains(ctx, listOptions)
		if err != nil {
			return 0, fmt.Errorf("could not list domains: %v", err)
		}
		for _, d := range domains {
			if d.Domain == p.Domain {
				domainId = d.ID
				done = true
				break
			}
		}
		if done {
			break
		}
		if !done && listOptions.Pages > page {
			page++
			listOptions.Page = page
		}
		if !done && listOptions.PageOptions.Page == page {
			return 0, fmt.Errorf("could not find the domain provided")
		}
	}

	return domainId, nil
}

func (p *Provider) appendRecord(records []libdns.Record, record *linodego.DomainRecord) []libdns.Record {
	return append(records, *p.convertToLibdns(record))
}

func (p *Provider) createDomainRecord(ctx context.Context, domainID int, record *libdns.Record) (*libdns.Record, error) {
	newRec, err := p.client.CreateDomainRecord(ctx, domainID, linodego.DomainRecordCreateOptions{
		Type:   p.convertRecordType(record.Type),
		Name:   record.Name,
		Target: record.Value,
		TTLSec: int(record.TTL.Seconds()),
	})
	if err != nil {
		return nil, err
	}
	return p.convertToLibdns(newRec), nil
}

func (p *Provider) updateDomainRecord(ctx context.Context, domainID int, record *libdns.Record, remoteRecord *linodego.DomainRecord) (*libdns.Record, error) {
	updatedRec, err := p.client.UpdateDomainRecord(ctx, domainID, remoteRecord.ID, linodego.DomainRecordUpdateOptions{
		Type:   p.convertRecordType(record.Type),
		Name:   record.Name,
		Target: record.Value,
		TTLSec: int(record.TTL.Seconds()),
	})
	if err != nil {
		return nil, err
	}
	return p.convertToLibdns(updatedRec), nil
}

func (p *Provider) deleteDomainRecord(ctx context.Context, domainID int, record *libdns.Record) error {
	recordID, err := strconv.Atoi(record.ID)
	if err != nil {
		return err
	}
	return p.client.DeleteDomainRecord(ctx, domainID, recordID)
}

func (p *Provider) convertToLibdns(record *linodego.DomainRecord) *libdns.Record {
	return &libdns.Record{
		ID:    strconv.Itoa(record.ID),
		Type:  string(record.Type),
		Name:  record.Name,
		Value: record.Target,
		TTL:   time.Duration(record.TTLSec),
	}
}

func (p *Provider) getRemoteRecords(ctx context.Context, domainID int) (map[string]*linodego.DomainRecord, error) {
	// We don't care about paging. Just get all of them.
	remoteRecordList, err := p.client.ListDomainRecords(ctx, domainID, nil)
	if err != nil {
		return nil, fmt.Errorf("could not list domain records: %v", err)
	}

	remoteRecords := make(map[string]*linodego.DomainRecord, len(remoteRecordList))

	for _, record := range remoteRecordList {
		remoteRecords[record.Name] = &record
	}

	return remoteRecords, nil
}
