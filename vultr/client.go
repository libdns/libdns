package vultr

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/vultr/govultr"
	"github.com/libdns/libdns"
)

type Client struct {
	client *govultr.Client
	mutex	sync.Mutex
}

func (p *Provider) getClient() error {
	if p.client == nil { 
		p.client = govultr.NewClient(nil, p.APIToken)
	}

	return nil
}

func (p *Provider) getDNSEntries(ctx context.Context, domain string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.getClient()

	var records []libdns.Record
	dns_entries, err := p.client.DNSRecord.List(ctx, domain)
	if err != nil {
		return records, err
	}

	for _, entry := range dns_entries {
		record := libdns.Record{
			Name: entry.Name,
			Value: entry.Data,
			Type:	entry.Type,
			TTL:	time.Duration(entry.TTL) * time.Second,
			ID:		strconv.Itoa(entry.RecordID),
		}
		records = append(records, record)
	}

	return records, nil
}

func (p *Provider) addDNSRecord(ctx context.Context, domain string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.getClient()

	err := p.client.DNSRecord.Create(ctx, domain, record.Type, record.Name, record.Value, int(record.TTL.Seconds()), 0)
	if err != nil {
		return record, err
	}

	return record, nil
}

func (p *Provider) removeDNSRecord(ctx context.Context, domain string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.getClient()

	err := p.client.DNSRecord.Delete(ctx, domain, record.ID)
	if err != nil {
		return record, err
	}

	return record, nil
}

func (p *Provider) updateDNSRecord(ctx context.Context, domain string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.getClient()

	id, err := strconv.Atoi(record.ID)
	if err != nil {
		return record, err
	}

	entry := govultr.DNSRecord{
		Name: record.Name,
		Data: record.Value,
		Type: record.Type,
		TTL: int(record.TTL.Seconds()),
		RecordID: id,
	}

	err = p.client.DNSRecord.Update(ctx, domain, &entry)
	if err != nil {
		return record, err
	}

	return record, nil
}