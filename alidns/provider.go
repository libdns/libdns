package alidns

import (
	"context"

	"github.com/libdns/libdns"
)

// Provider implements the libdns interfaces for Alicloud.
type Provider struct {
	client       Client
	AccKeyID     string `json:"access_key_id"`
	AccKeySecret string `json:"access_key_secret"`
	RegionID     string `json:"region_id,omitempty"`
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	var rls []libdns.Record
	for _, rec := range recs {
		ar := alidnsRecord(rec)
		ar.DName = zone
		rid, err := p.addDomainRecord(ctx, ar)
		if err != nil {
			return nil, err
		}
		ar.RecID = rid
		rls = append(rls, ar.LibdnsRecord())
	}
	return rls, nil
}

// DeleteRecords deletes the records from the zone. If a record does not have an ID,
// it will be looked up. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	var rls []libdns.Record
	for _, rec := range recs {
		ar := alidnsRecord(rec)
		if len(ar.RecID) == 0 {
			r0, err := p.queryDomainRecord(ctx, ar.Rr, zone)
			ar.RecID = r0.RecID
			if err != nil {
				return nil, err
			}
		}
		_, err := p.delDomainRecord(ctx, ar)
		if err != nil {
			return nil, err
		}
		rls = append(rls, ar.LibdnsRecord())
	}
	return rls, nil
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	var rls []libdns.Record
	recs, err := p.queryDomainRecords(ctx, zone)
	if err != nil {
		return nil, err
	}
	for _, rec := range recs {
		rls = append(rls, rec.LibdnsRecord())
	}
	return rls, nil
}

// SetRecords sets the records in the zone, either by updating existing records
// or creating new ones. It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	var rls []libdns.Record
	for _, rec := range recs {
		ar := alidnsRecord(rec)
		if len(ar.RecID) == 0 {
			r0, err := p.queryDomainRecord(ctx, ar.Rr, zone)
			if err != nil {
				ar.RecID, err = p.addDomainRecord(ctx, ar)
			} else {
				ar.RecID = r0.RecID
			}
		}
		_, err := p.setDomainRecord(ctx, ar)
		if err != nil {
			return nil, err
		}
		rls = append(rls, ar.LibdnsRecord())
	}
	return rls, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
