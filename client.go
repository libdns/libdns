package ovh

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/libdns/libdns"
	"github.com/ovh/go-ovh/ovh"
)

// Client is an abstraction of OvhClient
type Client struct {
	ovhClient 	*ovh.Client
	mutex       sync.Mutex
}

// Ovh zone record implementation
type OvhDomainZoneRecord struct {
	ID  	  int64  `json:"id,omitempty"`
	Zone      string `json:"zone,omitempty"`
	SubDomain string `json:"subDomain"`
	FieldType string `json:"fieldType,omitempty"`
	Target    string `json:"target"`
	TTL       int64  `json:"ttl"`
}

// Ovh zone soa implementation
type OvhDomainZoneSOA struct {
	Server       string  `json:"server"`
	Email        string  `json:"email"`
	Serial       int64  `json:"serial"`
	Refresh      int64  `json:"refresh"`
	NxDomainTTL  int64  `json:"nxDomainTtl"`
	Expire       int64  `json:"expire"`
	TTL       	 int64  `json:"ttl"`
}

// setupClient invokes authentication and store client to the provider instance.
func (p *Provider) setupClient() error {
	if p.client.ovhClient == nil {
		client, err := ovh.NewClient(p.Endpoint, p.ApplicationKey, p.ApplicationSecret, p.ConsumerKey)

		if err != nil {
			return err
		}

		p.client.ovhClient = client
	}

	return nil
}

// getRecords gets all records in specified zone on Ovh DNS.
// TTL as 0 for any record correspond to the default TTL for the zone
func (p *Provider) getRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.client.mutex.Lock()
	defer p.client.mutex.Unlock()

	if err := p.setupClient(); err != nil {
		return nil, err
	}

	var dzSOA OvhDomainZoneSOA
	if err := p.client.ovhClient.GetWithContext(ctx, fmt.Sprintf("/domain/zone/%s/soa", zone), &dzSOA); err != nil {
		return nil, err
	}

	var idRecords []int64
	if err := p.client.ovhClient.GetWithContext(ctx, fmt.Sprintf("/domain/zone/%s/record", zone), &idRecords); err != nil {
		return nil, err
	}

	var records []libdns.Record
	for _, idr := range idRecords {
		var dzr OvhDomainZoneRecord
		if err := p.client.ovhClient.GetWithContext(ctx, fmt.Sprintf("/domain/zone/%s/record/%d", zone, idr), &dzr); err != nil {
			return nil, err
		}

		if dzr.TTL == 0 {
			dzr.TTL = dzSOA.TTL
		}

		record := libdns.Record{
			ID: strconv.FormatInt(dzr.ID, 10),
			Type: dzr.FieldType,
			Name: dzr.SubDomain,
			Value: strings.TrimRight(strings.TrimLeft(dzr.Target, "\""), "\""),
			TTL: time.Duration(dzr.TTL) * time.Second,
		}
		records = append(records, record)
	}

	return records, nil
}

// createRecord creates a new record in the specified zone.
func (p *Provider) createRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.client.mutex.Lock()
	defer p.client.mutex.Unlock()

	if err := p.setupClient(); err != nil {
		return libdns.Record{}, err
	}

	var nzr OvhDomainZoneRecord
	if err := p.client.ovhClient.PostWithContext(ctx, fmt.Sprintf("/domain/zone/%s/record", zone), &OvhDomainZoneRecord{FieldType: record.Type, SubDomain: normalizeRecordName(record.Name, zone), Target: record.Value, TTL: int64(record.TTL.Seconds())}, &nzr); err != nil {
		return libdns.Record{}, err
	}

	createdRecord := libdns.Record{
		ID: strconv.FormatInt(nzr.ID, 10),
		Type: nzr.FieldType,
		Name: nzr.SubDomain,
		Value: strings.TrimRight(strings.TrimLeft(nzr.Target, "\""), "\""),
		TTL: time.Duration(nzr.TTL) * time.Second,
	}

	return createdRecord, nil
}

// createOrUpdateRecord creates or updates a record, either by updating existing record or creating new one.
func (p *Provider) createOrUpdateRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	if len(record.ID) == 0 {
		return p.createRecord(ctx, zone, record)
	}

	return p.updateRecord(ctx, zone, record)
}

// updateRecord updates a record
func (p *Provider) updateRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.client.mutex.Lock()
	defer p.client.mutex.Unlock()

	if err := p.setupClient(); err != nil {
		return libdns.Record{}, err
	}

	var nzr OvhDomainZoneRecord
	if err := p.client.ovhClient.PutWithContext(ctx, fmt.Sprintf("/domain/zone/%s/record/%s", zone, record.ID), &OvhDomainZoneRecord{SubDomain: record.Name, Target: record.Value, TTL: int64(record.TTL.Seconds())}, &nzr); err != nil {
		return libdns.Record{}, err
	}

	var uzr OvhDomainZoneRecord
	if err := p.client.ovhClient.GetWithContext(ctx, fmt.Sprintf("/domain/zone/%s/record/%s", zone, record.ID), &uzr); err != nil {
		return libdns.Record{}, err
	}
	
	updatedRecord := libdns.Record{
		ID: strconv.FormatInt(uzr.ID, 10),
		Type: uzr.FieldType,
		Name: uzr.SubDomain,
		Value: strings.TrimRight(strings.TrimLeft(uzr.Target, "\""), "\""),
		TTL: time.Duration(uzr.TTL) * time.Second,
	}

	return updatedRecord, nil
}

// deleteRecord deletes an existing record.
// Regardless of the value of the record, if the name and type match, the record will be deleted.
func (p *Provider) deleteRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.client.mutex.Lock()
	defer p.client.mutex.Unlock()

	if err := p.setupClient(); err != nil {
		return libdns.Record{}, err
	}

	if err := p.client.ovhClient.DeleteWithContext(ctx, fmt.Sprintf("/domain/zone/%s/record/%s", zone, record.ID), nil); err != nil {
		return libdns.Record{}, err
	}
	
	return record, nil	
}

// refresh trigger a reload of the DNS zone.
// It must be called after appending, setting or deleting any record
func (p *Provider) refresh(ctx context.Context, zone string) (error) { 
	p.client.mutex.Lock()
	defer p.client.mutex.Unlock()

	if err := p.setupClient(); err != nil {
		return err
	}

	if err := p.client.ovhClient.PostWithContext(ctx, fmt.Sprintf("/domain/zone/%s/refresh", zone), nil, nil); err != nil {
		return err
	}

	return nil
}

// unFQDN trims any trailing "." from fqdn. OVH's API does not use FQDNs.
func unFQDN(fqdn string) string {
	return strings.TrimSuffix(fqdn, ".")
}

// normalizeRecordName remove absolute record name
func normalizeRecordName(recordName string, zone string) string {
	normalized := unFQDN(recordName)
	normalized = strings.TrimSuffix(normalized, unFQDN(zone))
	return unFQDN(normalized)
}

