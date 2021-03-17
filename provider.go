// Package metaname implements a DNS record management client compatible
// with the libdns interfaces for Metaname.
package metaname

import (
	"context"
	"sync"
	"time"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with Metaname
type Provider struct {
	APIKey           string `json:"api_key,omitempty"`
	AccountReference string `json:"account_reference,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`

	mutex sync.Mutex
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	metanameRecords, err := p.dns_zone(ctx, zone)
	if err != nil {
		return nil, err
	}

	var libRecords []libdns.Record
	for _, rec := range metanameRecords {
		rec := libdns.Record{
			ID:    rec.Reference,
			Type:  rec.Type,
			Name:  rec.Name,
			TTL:   time.Duration(rec.Ttl) * time.Second,
			Value: rec.Data,
		}

		libRecords = append(libRecords, rec)
	}

	return libRecords, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var added []libdns.Record
	for _, rec := range records {
		mrec := metanameRR{
			Name: rec.Name,
			Type: rec.Type,
			Ttl:  int(rec.TTL.Seconds()),
			Data: rec.Value,
		}
		ref, err := p.create_dns_record(ctx, zone, mrec)
		if err != nil {
			return nil, err
		}
		if ref != "" {
			rec.ID = ref
			added = append(added, rec)
		}
	}
	return added, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var updated []libdns.Record
	var existing []libdns.Record
	var err error
	for _, rec := range records {
		mrec := metanameRR{
			Name: rec.Name,
			Type: rec.Type,
			Ttl:  int(rec.TTL.Seconds()),
			Data: rec.Value,
		}
		if rec.ID != "" {
			err := p.update_dns_record(ctx, zone, rec.ID, mrec)
			if err != nil {
				return updated, err
			}
			updated = append(updated, rec)
		} else {
			if existing == nil {
				existing, err = p.GetRecords(ctx, zone)
				if err != nil {
					return updated, err
				}
			}
			replaced := false
			// When only the new record data is given, find an existing record with the same name and type, provided it
			// is a CNAME/A/AAAA record. For all other records, a new record will be created below.
			for _, cur := range existing {
				if cur.Name == rec.Name && cur.Type == rec.Type && (cur.Type == "CNAME" || cur.Type == "A" || cur.Type == "AAAA") {
					newrec := metanameRR{Name: rec.Name, Type: rec.Type, Data: rec.Value, Ttl: int(rec.TTL.Seconds())}
					err := p.update_dns_record(ctx, zone, cur.ID, newrec)
					if err != nil {
						return updated, err
					}
					rec.ID = cur.ID
					updated = append(updated, rec)
					replaced = true
				}
			}
			if !replaced {
				ref, err := p.create_dns_record(ctx, zone, mrec)
				if err != nil {
					return updated, err
				}
				if ref != "" {
					rec.ID = ref
					updated = append(updated, rec)
				}
			}
		}
	}
	return updated, nil

}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var deleted []libdns.Record
	var existing []libdns.Record
	var err error
	for _, rec := range records {
		if rec.ID != "" {
			r, err := p.delete_dns_record(ctx, zone, rec.ID)
			if err != nil {
				return deleted, err
			}
			if r {
				deleted = append(deleted, rec)
			}
		} else {
			if existing == nil {
				existing, err = p.GetRecords(ctx, zone)
				if err != nil {
					return deleted, err
				}
			}
			// When only record data was provided to delete, match only if name, type, and value match
			// exactly (ignoring TTL).
			for _, cur := range existing {
				if cur.Name == rec.Name && cur.Type == rec.Type && cur.Value == rec.Value {
					r, err := p.delete_dns_record(ctx, zone, cur.ID)
					if err != nil {
						return deleted, err
					}
					if r {
						deleted = append(deleted, rec)
					}
				}
			}
		}
	}
	return deleted, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
