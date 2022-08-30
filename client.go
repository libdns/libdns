package civo

import (
	"context"
	"github.com/libdns/libdns"
	"strings"
	"sync"
	"time"

	"github.com/civo/civogo"
)

type Client struct {
	client *civogo.Client
	mutex  sync.Mutex
}

func (p *Provider) getClient() error {
	if p.client == nil {
		var err error
		// DNS is region independent, we can use any region
		p.client, err = civogo.NewClient(p.APIToken, "LON1")
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
	var records []libdns.Record
	domain, err := p.client.GetDNSDomain(zone)
	if err != nil {
		return nil, err
	}

	dnsRecords, err := p.client.ListDNSRecords(domain.ID)
	if err != nil {
		return nil, err
	}

	for _, entry := range dnsRecords {
		record := libdns.Record{
			Name:  entry.Name + "." + strings.Trim(zone, ".") + ".",
			Value: entry.Value,
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

	domain, err := p.client.GetDNSDomain(zone)
	if err != nil {
		return record, err
	}

	dnsRecord, err := p.client.CreateDNSRecord(domain.ID, &civogo.DNSRecordConfig{
		Name:  strings.Trim(strings.ReplaceAll(record.Name, zone, ""), "."),
		Value: record.Value,
		Type:  civogo.DNSRecordType(record.Type),
		TTL:   int(record.TTL.Seconds()),
	})
	if err != nil {
		return libdns.Record{}, err
	}
	if err != nil {
		return record, err
	}
	record.ID = dnsRecord.ID
	return record, nil
}

func (p *Provider) removeDNSEntry(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	err := p.getClient()
	if err != nil {
		return record, err
	}

	domain, err := p.client.GetDNSDomain(zone)
	if err != nil {
		return record, err
	}
	dnsRecord, err := p.client.GetDNSRecord(domain.ID, record.ID)
	if err != nil {
		return record, err
	}
	_, err = p.client.DeleteDNSRecord(dnsRecord)
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

	domain, err := p.client.GetDNSDomain(zone)
	if err != nil {
		return record, err
	}
	dnsRecord, err := p.client.GetDNSRecord(domain.ID, record.ID)
	if err != nil {
		return record, err
	}
	_, err = p.client.UpdateDNSRecord(dnsRecord, &civogo.DNSRecordConfig{
		Name:  strings.Trim(strings.ReplaceAll(record.Name, zone, ""), "."),
		Value: record.Value,
		Type:  civogo.DNSRecordType(record.Type),
		TTL:   int(record.TTL.Seconds()),
	})
	if err != nil {
		return record, err
	}

	return record, nil
}
