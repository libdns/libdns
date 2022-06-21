package rfc2136

import (
	"context"
	"fmt"
	"github.com/libdns/libdns"
	"github.com/miekg/dns"
	"time"
)

type Provider struct {
	KeyName string `json:"key_name,omitempty"`
	KeyAlg  string `json:"key_alg,omitempty"`
	Key     string `json:"key,omitempty"`
	Server  string `json:"server,omitempty"`
}

func (p *Provider) keyNameFQDN() string {
	return dns.Fqdn(p.KeyName)
}

func (p *Provider) client() *dns.Client {
	return &dns.Client{
		TsigSecret: map[string]string{p.keyNameFQDN(): p.Key},
		Net:        "tcp",
	}
}

func (p *Provider) setTsig(msg *dns.Msg) {
	msg.SetTsig(p.keyNameFQDN(), dns.Fqdn(p.KeyAlg), 300, time.Now().Unix())
}

func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	zone = dns.Fqdn(zone)

	conn, err := p.client().DialContext(ctx, p.Server)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	tn := dns.Transfer{
		Conn: conn,
	}
	tn.TsigSecret = map[string]string{p.keyNameFQDN(): p.Key}

	msg := dns.Msg{}
	msg.SetAxfr(zone)
	p.setTsig(&msg)

	res, err := tn.In(&msg, p.Server)
	if err != nil {
		return nil, fmt.Errorf("start zone transfer: %w", err)
	}

	records := make([]libdns.Record, 0)
	for e := range res {
		if e.Error != nil {
			return nil, fmt.Errorf("zone transfer: %w", e.Error)
		}

		for _, rr := range e.RR {
			records = append(records, recordFromRR(rr, zone))
		}
	}

	return records, nil
}

func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return p.SetRecords(ctx, zone, records)
}

func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	zone = dns.Fqdn(zone)

	msg := dns.Msg{}
	msg.SetUpdate(zone)
	for _, rec := range records {
		rr, err := recordToRR(rec, zone)
		if err != nil {
			return nil, fmt.Errorf("invalid record %s: %w", rec.Name, err)
		}
		msg.Insert([]dns.RR{rr})
	}
	p.setTsig(&msg)

	_, _, err := p.client().ExchangeContext(ctx, &msg, p.Server)
	if err != nil {
		return nil, err
	}

	return records, nil
}

func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	zone = dns.Fqdn(zone)

	msg := dns.Msg{}
	msg.SetUpdate(zone)
	for _, rec := range records {
		rr, err := recordToRR(rec, zone)
		if err != nil {
			return nil, fmt.Errorf("invalid record %s: %w", rec.Name, err)
		}
		msg.Remove([]dns.RR{rr})
	}
	p.setTsig(&msg)

	_, _, err := p.client().ExchangeContext(ctx, &msg, p.Server)
	if err != nil {
		return nil, err
	}

	return records, nil
}

var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
