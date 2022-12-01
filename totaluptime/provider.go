// Package totaluptime implements a DNS record management client compatible
// with the libdns interfaces for Total Uptime.
// based on https://api.totaluptime.com/api-docs/index.html#!/Cloud32DNS32Calls
package totaluptime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with Total Uptime.
type Provider struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// GetRecords lists all (supported type) records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	domain := getDomain(zone)
	domainID := p.getDomainID(domain)

	if domainID == "" {
		return nil, fmt.Errorf("record lookup cannot proceed: unknown domain: %s", domain)
	}

	// accumulate all records in zone, keeping only libdns.Record data
	var records []libdns.Record

	// configure http request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, APIbase+"/"+domainID+"/AllRecords", nil)
	if err != nil {
		return nil, err
	}

	var providerRecords TotalUptimeRecords
	err = p.performAPIcall(req, &providerRecords)
	if err != nil {
		return nil, err
	}

	// A records
	for _, val := range providerRecords.ARecord.Rows {
		ttl, err := strconv.Atoi(val.ATTL)
		if err != nil {
			return nil, err
		}

		record := libdns.Record{
			ID:    val.ID,
			Type:  "A",
			Name:  val.AHostName,
			Value: val.AIPAddress,
			TTL:   time.Duration(ttl) * time.Second,
		}
		records = append(records, record)
	}

	// CNAME records
	for _, val := range providerRecords.CNAMERecord.Rows {
		ttl, err := strconv.Atoi(val.CnameTTL)
		if err != nil {
			return nil, err
		}

		record := libdns.Record{
			ID:    val.ID,
			Type:  "CNAME",
			Name:  val.CnameName,
			Value: val.CnameAliasFor,
			TTL:   time.Duration(ttl) * time.Second,
		}
		records = append(records, record)
	}

	// MX records
	for _, val := range providerRecords.MXRecord.Rows {
		ttl, err := strconv.Atoi(val.MxTTL)
		if err != nil {
			return nil, err
		}

		record := libdns.Record{
			ID:    val.ID,
			Type:  "MX",
			Name:  val.MxDomainName,
			Value: val.MxMailServer,
			TTL:   time.Duration(ttl) * time.Second,
		}
		records = append(records, record)
	}

	// NS records
	for _, val := range providerRecords.NSRecord.Rows {
		ttl, err := strconv.Atoi(val.NsTTL)
		if err != nil {
			return nil, err
		}

		record := libdns.Record{
			ID:    val.ID,
			Type:  "NS",
			Name:  val.NsHostName,
			Value: val.NsName,
			TTL:   time.Duration(ttl) * time.Second,
		}
		records = append(records, record)
	}

	// SRV records (research needed)
	// 	// TODO: possible compiler or static-check error (Go v1.19.3):
	// 	// issue 1) claims Weight is not part of this struct
	// 	// issue 2) claims Priority is an int (should be uint)

	// TXT records
	for _, val := range providerRecords.TXTRecord.Rows {
		ttl, err := strconv.Atoi(val.TxtTTL)
		if err != nil {
			return nil, err
		}

		record := libdns.Record{
			ID:    val.ID,
			Type:  "TXT",
			Name:  val.TxtHostName,
			Value: val.TxtText,
			TTL:   time.Duration(ttl) * time.Second,
		}
		records = append(records, record)
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	domain := getDomain(zone)
	domainID := p.getDomainID(domain)
	var successAppended []libdns.Record

	if domainID == "" {
		return nil, fmt.Errorf("record append cannot proceed: unknown zone: %s", zone)
	}

	// append each record individually
	for _, rec := range records {
		// identify record type
		typeName, err := convertRecordTypeToProvider(rec.Type)
		if err != nil {
			continue // skip to next record
		}

		// create JSON payload for HTTP post
		payload := buildPayload(rec)
		var buf bytes.Buffer
		err = json.NewEncoder(&buf).Encode(payload)
		if err != nil {
			continue
		}

		// configure http request
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, APIbase+"/"+domainID+"/"+typeName, &buf)
		if err != nil {
			return nil, err
		}

		var response providerResponse
		err = p.performAPIcall(req, &response)
		if err != nil {
			return nil, err
		}

		if response.Status == "Success" {
			successAppended = append(successAppended, rec)
			p.lookupRecordIDs(domain, true) // refresh memory cache of RecordIDs
			log.Printf("DNS successful append record to zone: %s: %s: %s\n", zone, rec.Type, rec.Name)

		} else {
			log.Printf("DNS append failure with provider message: %s\n", response.Message)
		}
	}

	return successAppended, nil
}

// ModifyRecord will modify a single pre-existing record in the specified zone.
// It returns the updated record.
func (p *Provider) ModifyRecord(ctx context.Context, zone string, rec libdns.Record) (libdns.Record, error) {
	var successRecord libdns.Record
	domain := getDomain(zone)
	domainID := p.getDomainID(domain)
	recordID := p.getRecordID(domain, rec.Type, rec.Name)

	if domainID == "" {
		return successRecord, fmt.Errorf("record modify cannot proceed: unknown zone: %s", zone)
	}

	// identify record type
	typeName, err := convertRecordTypeToProvider(rec.Type)
	if err != nil {
		return successRecord, err
	}

	// create JSON payload for HTTP post
	payload := buildPayload(rec)
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(payload)
	if err != nil {
		return successRecord, err
	}

	// configure http request
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, APIbase+"/"+domainID+"/"+typeName+"/"+recordID, &buf)
	if err != nil {
		return successRecord, err
	}

	var response providerResponse
	err = p.performAPIcall(req, &response)
	if err != nil {
		return libdns.Record{}, err
	}

	if response.Status == "Success" {
		successRecord = rec
		p.lookupRecordIDs(domain, true) // refresh memory cache of RecordIDs
		log.Printf("DNS successful modify record in zone: %s: %s: %s\n", zone, rec.Type, rec.Name)

	} else {
		log.Printf("DNS modify failure with provider message: %s\n", response.Message)
	}

	return successRecord, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	domain := getDomain(zone)
	domainID := p.getDomainID(domain)
	var successUpdated []libdns.Record

	if domainID == "" {
		return nil, fmt.Errorf("record update cannot proceed: unknown zone: %s", zone)
	}

	// manipulate each record individually
	for _, rec := range records {
		recordID := p.getRecordID(domain, rec.Type, rec.Name)

		if recordID == "" { // record doesn't exist yet
			success, err := p.AppendRecords(ctx, zone, []libdns.Record{rec})
			if err == nil {
				successUpdated = append(successUpdated, success...)
			}

			continue
		}

		// record exists
		success, err := p.ModifyRecord(ctx, zone, rec)
		if err == nil {
			successUpdated = append(successUpdated, success)
		}
	}

	return successUpdated, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	domain := getDomain(zone)
	domainID := p.getDomainID(domain)
	var successDeleted []libdns.Record

	if domainID == "" {
		return nil, fmt.Errorf("record delete cannot proceed: unknown zone: %s", zone)
	}

	// delete each record individually
	for _, rec := range records {
		// identify record type
		typeName, err := convertRecordTypeToProvider(rec.Type)
		if err != nil {
			continue // skip to next record
		}

		recordID := p.getRecordID(domain, rec.Type, rec.Name)
		if recordID == "" { // record doesn't exist
			continue // skip to next record
		}

		// configure http request
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, APIbase+"/"+domainID+"/"+typeName+"/"+recordID, nil)
		if err != nil {
			return nil, err
		}

		var response providerResponse
		err = p.performAPIcall(req, &response)
		if err != nil {
			return nil, err
		}

		if response.Status == "Success" {
			successDeleted = append(successDeleted, rec)
			p.lookupRecordIDs(domain, true) // refresh memory cache of RecordIDs
			log.Printf("DNS successful delete record in zone: %s: %s: %s\n", zone, rec.Type, rec.Name)

		} else {
			log.Printf("DNS delete record failure with provider message: %s\n", response.Message)
		}
	}

	return successDeleted, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
