// Package dummy provides an in-memory implementation of all libdns interfaces for testing.
package dummy

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/libdns/libdns"
)

type Provider struct {
	zones		map[string][]libdns.Record
	availableZones	[]string
}

func New(zones ...string) *Provider {
	if len(zones) == 0 {
		zones = []string{"example.com."}
	}

	p := &Provider{
		zones:		make(map[string][]libdns.Record),
		availableZones:	make([]string, len(zones)),
	}
	copy(p.availableZones, zones)
	for _, zone := range zones {
		p.zones[zone] = []libdns.Record{}
	}
	return p
}

func (p *Provider) ListZones(ctx context.Context) ([]libdns.Zone, error) {
	zones := make([]libdns.Zone, len(p.availableZones))
	for i, zoneName := range p.availableZones {
		zones[i] = libdns.Zone{Name: zoneName}
	}

	return zones, nil
}

func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	records, exists := p.zones[zone]
	if !exists {
		return nil, fmt.Errorf("zone %s not found", zone)
	}

	result := make([]libdns.Record, len(records))
	copy(result, records)
	return result, nil
}

func (p *Provider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	if _, exists := p.zones[zone]; !exists {
		return nil, fmt.Errorf("zone %s not found", zone)
	}

	appendedRecords := make([]libdns.Record, 0, len(recs))
	for _, rec := range recs {
		concreteRecord := p.toConcreteType(rec)
		p.zones[zone] = append(p.zones[zone], concreteRecord)
		appendedRecords = append(appendedRecords, concreteRecord)
	}

	return appendedRecords, nil
}

func (p *Provider) SetRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	if _, exists := p.zones[zone]; !exists {
		return nil, fmt.Errorf("zone %s not found", zone)
	}

	inputGroups := make(map[string][]libdns.Record)
	for _, rec := range recs {
		rr := rec.RR()
		key := fmt.Sprintf("%s:%s", rr.Name, rr.Type)
		inputGroups[key] = append(inputGroups[key], rec)
	}

	var keepRecords []libdns.Record
	for _, existingRec := range p.zones[zone] {
		existingRR := existingRec.RR()
		key := fmt.Sprintf("%s:%s", existingRR.Name, existingRR.Type)
		if _, shouldReplace := inputGroups[key]; !shouldReplace {
			keepRecords = append(keepRecords, existingRec)
		}
	}

	setRecords := make([]libdns.Record, 0, len(recs))
	for _, rec := range recs {
		concreteRecord := p.toConcreteType(rec)
		keepRecords = append(keepRecords, concreteRecord)
		setRecords = append(setRecords, concreteRecord)
	}

	p.zones[zone] = keepRecords
	return setRecords, nil
}

func (p *Provider) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	if _, exists := p.zones[zone]; !exists {
		return nil, fmt.Errorf("zone %s not found", zone)
	}

	var remainingRecords []libdns.Record
	var deletedRecords []libdns.Record

	existingRecords := p.zones[zone]

	for _, existingRec := range existingRecords {
		deleted := false

		for _, deleteRec := range recs {
			if p.recordsMatch(existingRec, deleteRec) {
				deletedRecords = append(deletedRecords, existingRec)
				deleted = true
				break
			}
		}

		if !deleted {
			remainingRecords = append(remainingRecords, existingRec)
		}
	}

	p.zones[zone] = remainingRecords

	return deletedRecords, nil
}

func (p *Provider) toConcreteType(rec libdns.Record) libdns.Record {
	if _, isRR := rec.(libdns.RR); !isRR {
		return rec
	}

	rr := rec.RR()

	switch rr.Type {
	case "A", "AAAA":
		return p.rrToAddress(rr)

	case "TXT":
		return libdns.TXT{
			Name:	rr.Name,
			TTL:	rr.TTL,
			Text:	rr.Data,
		}

	case "CNAME":
		return libdns.CNAME{
			Name:	rr.Name,
			TTL:	rr.TTL,
			Target:	rr.Data,
		}

	case "NS":
		return libdns.NS{Name: rr.Name, TTL: rr.TTL, Target: rr.Data}
	default:
		// for simplicity, complex record types (MX, SRV, CAA, HTTPS, SVCB) return as RR
		return rr
	}
}

func (p *Provider) rrToAddress(rr libdns.RR) libdns.Address {
	return libdns.Address{
		Name:	rr.Name,
		TTL:	rr.TTL,
		IP:	netip.MustParseAddr(rr.Data),	// test data should always be valid
	}
}

func (p *Provider) recordsMatch(existingRec, deleteRec libdns.Record) bool {
	existingRR := existingRec.RR()
	deleteRR := deleteRec.RR()

	return existingRR.Name == deleteRR.Name &&
		(deleteRR.Type == "" || existingRR.Type == deleteRR.Type) &&
		(deleteRR.TTL == 0 || existingRR.TTL == deleteRR.TTL) &&
		(deleteRR.Data == "" || existingRR.Data == deleteRR.Data)
}
