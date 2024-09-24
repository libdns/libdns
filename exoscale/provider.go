package exoscale

import (
	"context"
	"errors"
	"fmt"
	"time"
	"strings"
	"sync"

	"github.com/libdns/libdns"
	egoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/egoscale/v3/credentials"
)

// Provider facilitates DNS record manipulation with Exoscale.
type Provider struct {
	// Exoscale API Key (required)
	APIKey string `json:"api_key,omitempty"`
	// Exoscale API Secret (required)
	APISecret string `json:"api_secret,omitempty"`

	client *egoscale.Client
	mutex  sync.Mutex
}

// Create a map to store the string to CreateDNSDomainRecordRequestType mappings
var dnsRecordTypeMap = map[string]egoscale.CreateDNSDomainRecordRequestType{
	"NS":    egoscale.CreateDNSDomainRecordRequestTypeNS,
	"CAA":   egoscale.CreateDNSDomainRecordRequestTypeCAA,
	"NAPTR": egoscale.CreateDNSDomainRecordRequestTypeNAPTR,
	"POOL":  egoscale.CreateDNSDomainRecordRequestTypePOOL,
	"A":     egoscale.CreateDNSDomainRecordRequestTypeA,
	"HINFO": egoscale.CreateDNSDomainRecordRequestTypeHINFO,
	"CNAME": egoscale.CreateDNSDomainRecordRequestTypeCNAME,
	"SSHFP": egoscale.CreateDNSDomainRecordRequestTypeSSHFP,
	"SRV":   egoscale.CreateDNSDomainRecordRequestTypeSRV,
	"AAAA":  egoscale.CreateDNSDomainRecordRequestTypeAAAA,
	"MX":    egoscale.CreateDNSDomainRecordRequestTypeMX,
	"TXT":   egoscale.CreateDNSDomainRecordRequestTypeTXT,
	"ALIAS": egoscale.CreateDNSDomainRecordRequestTypeALIAS,
	"URL":   egoscale.CreateDNSDomainRecordRequestTypeURL,
	"SPF":   egoscale.CreateDNSDomainRecordRequestTypeSPF,
}

// Function to get the DNSDomainRecordRequestType from a string
func (p *Provider) StringToDNSDomainRecordRequestType(recordType string) (egoscale.CreateDNSDomainRecordRequestType, error) {
	// Lookup the record type in the map
	if recordType, exists := dnsRecordTypeMap[recordType]; exists {
		return recordType, nil
	}
	return "", errors.New("invalid DNS record type")
}

// initClient will initialize the Exoscale API client with the provided api key and secret, and
// store the client in the Provider struct.
func (p *Provider) initClient() error {
	if p.client == nil {
		// Create new Exoscale client using the provided api_key and api_secret
		// TODO: If exoscale wants the UserAgent to be set for Caddy, then we will need to set it as parameter of the Provider
        client, err := egoscale.NewClient(
            credentials.NewStaticCredentials(p.APIKey, p.APISecret),
            egoscale.ClientOptWithUserAgent("libdns/exoscale"),
        )

        if err != nil {
            return fmt.Errorf("exoscale: initializing client: %w", err)
        }

		p.client = client

	}

    return nil
}

// Internal helper function that actually creates the records
func (p *Provider) createRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var createdRecords []libdns.Record

    domain, err := p.findExistingZone(unFQDN(zone))
    if err != nil {
		return nil, fmt.Errorf("exoscale: %w", err)
	}
	if domain == nil {
		return nil, fmt.Errorf("exoscale: zone %q not found", unFQDN(zone))
	}

    for _, r := range records {

        recordType, err := p.StringToDNSDomainRecordRequestType(r.Type)
        if err != nil {
            // What to do here ?? Just continue the loop or return error ??
            return nil, fmt.Errorf("exoscale: error while creating DNS record %w", err)
            // return createdRecords, fmt.Errorf("exoscale: error while creating DNS record %w", err)
            // continue
        }

        recordRequest := egoscale.CreateDNSDomainRecordRequest{
            Content: r.Value,
            Name:    r.Name,
            Ttl:     int64(r.TTL.Seconds()),
            Priority: int64(r.Priority),
            Type:    recordType,
        }

        op, err := p.client.CreateDNSDomainRecord(ctx, domain.ID, recordRequest)
        if err != nil {
            // What to do here ?? Just continue the loop or return error ??
            return nil, fmt.Errorf("exoscale: error while creating DNS record: %w", err)
            // return createdRecords, fmt.Errorf("exoscale: error while creating DNS record: %w", err)
            // continue
        }

        _, err = p.client.Wait(ctx, op, egoscale.OperationStateSuccess)
        if err != nil {
            // What to do here ?? Just continue the loop or return error ??
            return nil, fmt.Errorf("exoscale: error while waiting for DNS record creation: %w", err)
            // return createdRecords, fmt.Errorf("exoscale: error while creating DNS record: %w", err)
            // continue
        }

        // We need to set the ID of the libdns.Record, so we need to query for the created Records
        recordID, err := p.findExistingRecordID(domain.ID, r)
        if err != nil {
            // What to do here ?? Just continue the loop or return error ??
            return nil, fmt.Errorf("exoscale: error while finding DNS record %w", err)
            // return createdRecords, fmt.Errorf("exoscale: error while creating DNS record %w", err)
            // continue
        }

        if recordID == "" {
            // What to do here ?? Just continue the loop or return error ??
            return nil, fmt.Errorf("exoscale: record not Found %q", r.Name)
            // return createdRecords, fmt.Errorf("exoscale: record not Found %w", r.Name)
            // continue
        }

        r.ID = string(recordID)

        createdRecords = append(createdRecords, r)
	}

    return createdRecords, nil
}

// findExistingZone Query Exoscale to find an existing zone for this name.
// Returns nil result if no zone could be found.
func (p *Provider) findExistingZone(zoneName string) (*egoscale.DNSDomain, error) {
	ctx := context.Background()

	zones, err := p.client.ListDNSDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("error while retrieving DNS zones: %w", err)
	}

	for _, zone := range zones.DNSDomains {
		if zone.UnicodeName == zoneName {
			return &zone, nil
		}
	}

	return nil, nil
}

// findExistingRecordID Query Exoscale to find an existing record for this name.
// Returns empty result if no record could be found.
func (p *Provider) findExistingRecordID(zoneID egoscale.UUID, record libdns.Record) (egoscale.UUID, error) {
	ctx := context.Background()

	records, err := p.client.ListDNSDomainRecords(ctx, zoneID)
	if err != nil {
		return "", fmt.Errorf("error while retrieving DNS records: %w", err)
	}

	for _, r := range records.DNSDomainRecords {
		// we must unquote TXT records as we receive "\"123d==\"" when we expect "123d=="
	    content := strings.TrimRight(strings.TrimLeft(r.Content, "\""), "\"")

		if r.Name == record.Name && string(r.Type) == record.Type && content == record.Value {
			return r.ID, nil
		}
	}

	return "", nil
}


func (p *Provider) getRecordsFromProvider(ctx context.Context, zone string) ([]libdns.Record, error) {
	var records []libdns.Record

    domain, err := p.findExistingZone(unFQDN(zone))
    if err != nil {
		return nil, fmt.Errorf("exoscale: %w", err)
	}
	if domain == nil {
		return nil, fmt.Errorf("exoscale: zone %q not found", zone)
	}

    domainRecords, err := p.client.ListDNSDomainRecords(ctx, domain.ID)
    if err != nil {
		return nil, fmt.Errorf("exoscale: %w", err)
	}

    for _, r := range domainRecords.DNSDomainRecords {
		record := libdns.Record{
			ID:       string(r.ID),
			Type:     string(r.Type),
			Name:     r.Name,
			Value:    r.Content,
			TTL:      time.Duration(r.Ttl),
			Priority: uint(r.Priority),
		}
        records = append(records, record)
	}

    return records, nil
}

// Internal helper function to get the lists of records to create and update respectively
func (p *Provider) getRecordsToCreateAndUpdate(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, []libdns.Record, error) {
	existingRecords, err := p.getRecordsFromProvider(ctx, zone)
	if err != nil {
		return nil, nil, err
	}
	var recordsToUpdate []libdns.Record

	updateMap := make(map[libdns.Record]bool)
	var recordsToCreate []libdns.Record

	// Figure out which records exist and need to be updated
	for _, r := range records {
		updateMap[r] = true
		for _, er := range existingRecords {
			if r.Name != er.Name {
				continue
			}
			if r.ID == "0" || r.ID == "" {
				r.ID = er.ID
			}
			recordsToUpdate = append(recordsToUpdate, r)
			updateMap[r] = false
		}
	}
	// If the record is not updating an existing record, we want to create it
	for r, updating := range updateMap {
		if updating {
			recordsToCreate = append(recordsToCreate, r)
		}
	}

	return recordsToCreate, recordsToUpdate, nil
}


// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
    p.mutex.Lock()
    defer p.mutex.Unlock()

    if err := p.initClient(); err != nil {
		return nil, err
	}

    return p.getRecordsFromProvider(ctx, zone)

}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
    p.mutex.Lock()
    defer p.mutex.Unlock()

    if err := p.initClient(); err != nil {
		return nil, err
	}

    return p.createRecords(ctx, zone, records)


}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
    defer p.mutex.Unlock()

    var setRecords []libdns.Record

    if err := p.initClient(); err != nil {
		return nil, err
	}

    recordsToCreate, recordsToUpdate, err := p.getRecordsToCreateAndUpdate(ctx, zone, records)
	if err != nil {
		return nil, err
	}

    // Create new records and append them to 'setRecords'
	createdRecords, err := p.createRecords(ctx, zone, recordsToCreate)
	if err != nil {
		return nil, err
	}
	for _, r := range createdRecords {
		setRecords = append(setRecords, r)
	}

    // Get Zone from zone name
    domain, err := p.findExistingZone(unFQDN(zone))
    if err != nil {
		return nil, fmt.Errorf("exoscale: %w", err)
	}
	if domain == nil {
		return nil, fmt.Errorf("exoscale: zone %q not found", unFQDN(zone))
	}

    for _, r := range recordsToUpdate {
		recordRequest := egoscale.UpdateDNSDomainRecordRequest{
            Content: r.Value,
            Name:    r.Name,
            Ttl:     int64(r.TTL.Seconds()),
            Priority: int64(r.Priority),
        }

        op, err := p.client.UpdateDNSDomainRecord(ctx, domain.ID, egoscale.UUID(r.ID), recordRequest)
        if err != nil {
            // What to do here ?? Just continue the loop or return error ??
            return nil, fmt.Errorf("exoscale: error while updating DNS record: %w", err)
            // return createdRecords, fmt.Errorf("exoscale: error while creating DNS record: %w", err)
            // continue
        }

        _, err = p.client.Wait(ctx, op, egoscale.OperationStateSuccess)
        if err != nil {
            // What to do here ?? Just continue the loop or return error ??
            return nil, fmt.Errorf("exoscale: error while waiting for DNS record update: %w", err)
            // return createdRecords, fmt.Errorf("exoscale: error while creating DNS record: %w", err)
            // continue
        }
        setRecords = append(setRecords, r)
	}

    return setRecords, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
    defer p.mutex.Unlock()

    var deletedRecords []libdns.Record

    if err := p.initClient(); err != nil {
		return nil, err
	}

    domain, err := p.findExistingZone(unFQDN(zone))
    if err != nil {
		return nil, fmt.Errorf("exoscale: %w", err)
	}
	if domain == nil {
		return nil, fmt.Errorf("exoscale: zone %q not found", unFQDN(zone))
	}

    for _, r := range records {
        recordID, err := p.findExistingRecordID(domain.ID, r)
        if err != nil {
            // What to do here ?? Just continue the loop or return error ??
            return nil, fmt.Errorf("exoscale: error while finding DNS record %w", err)
            // return createdRecords, fmt.Errorf("exoscale: error while creating DNS record %w", err)
            // continue
        }

        if recordID == "" {
            // What to do here ?? Just continue the loop or return error ??
            return nil, fmt.Errorf("exoscale: record not Found %q", r.Name)
            // return createdRecords, fmt.Errorf("exoscale: record not Found %w", r.Name)
            // continue
        }

        op, err := p.client.DeleteDNSDomainRecord(ctx, domain.ID, recordID)
        if err != nil {
            // What to do here ?? Just continue the loop or return error ??
            return nil, fmt.Errorf("exoscale: error while deleting DNS record: %w", err)
            // return createdRecords, fmt.Errorf("exoscale: error while deleting DNS record: %w", err)
            // continue
        }

        _, err = p.client.Wait(ctx, op, egoscale.OperationStateSuccess)
        if err != nil {
            // What to do here ?? Just continue the loop or return error ??
            return nil, fmt.Errorf("exoscale: error while waiting DNS record deletion: %w", err)
            // return createdRecords, fmt.Errorf("exoscale: error while deleting DNS record: %w", err)
            // continue
        }

        deletedRecords = append(deletedRecords, r)
	}

    return deletedRecords, nil
}

// unFQDN trims any trailing "." from fqdn. Exoscale's API does not use FQDNs.
func unFQDN(fqdn string) string {
	return strings.TrimSuffix(fqdn, ".")
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
