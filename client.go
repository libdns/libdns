package scaleway

import (
	"context"
	"strings"
	"sync"
	"time"

	domain "github.com/scaleway/scaleway-sdk-go/api/domain/v2beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type Client struct {
	client *scw.Client
	mutex  sync.Mutex
}

func (p *Provider) getClient() error {
	if p.client == nil {
		var err error
		p.client, err = scw.NewClient(
			scw.WithAuth("SCWXXXXXXXXXXXXXXXXX", p.SecretKey),
			scw.WithDefaultOrganizationID(p.OrganizationID),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) getDNSEntries(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	err := p.getClient()
	if err != nil {
		return nil, err
	}

	domainAPI := domain.NewAPI(p.client)
	var records []libdns.Record

	zoneRecords, err := domainAPI.ListDNSZoneRecords(&domain.ListDNSZoneRecordsRequest{
		DNSZone: zone,
	})
	if err != nil {
		return records, err
	}

	for _, entry := range zoneRecords.Records {
		record := libdns.Record{
			Name:  entry.Name + "." + strings.Trim(zone, ".") + ".",
			Value: entry.Data,
			Type:  string(entry.Type),
			TTL:   time.Duration(entry.TTL) * time.Second,
			ID:    entry.ID,
		}
		records = append(records, record)
	}
	return records, nil
}

func (p *Provider) addDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	err := p.getClient()
	if err != nil {
		return record, err
	}

	domainAPI := domain.NewAPI(p.client)
	records, err := domainAPI.UpdateDNSZoneRecords(&domain.UpdateDNSZoneRecordsRequest{
		DNSZone: zone,
		Changes: []*domain.RecordChange{
			{
				Add: &domain.RecordChangeAdd{
					Records: []*domain.Record{
						{
							Name: strings.Trim(strings.ReplaceAll(record.Name, zone, ""), "."),
							Data: record.Value,
							Type: domain.RecordType(record.Type),
							TTL:  uint32(record.TTL.Seconds()),
						},
					},
				},
			},
		},
	})
	if err != nil {
		return record, err
	}
	record.ID = records.Records[0].ID
	return record, nil
}

func (p *Provider) removeDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	err := p.getClient()
	if err != nil {
		return record, err
	}
	domainAPI := domain.NewAPI(p.client)
	_, err = domainAPI.UpdateDNSZoneRecords(&domain.UpdateDNSZoneRecordsRequest{
		DNSZone: zone,
		Changes: []*domain.RecordChange{
			{
				Delete: &domain.RecordChangeDelete{
					ID: &record.ID,
				},
			},
		},
	})
	if err != nil {
		return record, err
	}
	return record, nil
}

func (p *Provider) updateDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	err := p.getClient()
	if err != nil {
		return record, err
	}
	domainAPI := domain.NewAPI(p.client)
	_, err = domainAPI.UpdateDNSZoneRecords(&domain.UpdateDNSZoneRecordsRequest{
		DNSZone: zone,
		Changes: []*domain.RecordChange{
			{
				Set: &domain.RecordChangeSet{
					ID: &record.ID,
					Records: []*domain.Record{
						{
							Name: strings.Trim(strings.ReplaceAll(record.Name, zone, ""), "."),
							Data: record.Value,
							Type: domain.RecordType(record.Type),
							TTL:  uint32(record.TTL.Seconds()),
						},
					},
				},
			},
		},
	})
	if err != nil {
		return record, err
	}
	return record, nil
}
