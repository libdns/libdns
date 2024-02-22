package libdnsdnsmadeeasy

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"sync"

	dme "github.com/john-k/dnsmadeeasy"
	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with DNSMadeEasy
type Provider struct {
	APIKey      string      `json:"api_key,omitempty"`
	SecretKey   string      `json:"secret_key,omitempty"`
	APIEndpoint dme.BaseURL `json:"api_endpoint,omitempty"`
	client      dme.Client
	once        sync.Once
	mutex       sync.Mutex
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.init(ctx)

	var records []libdns.Record

	// first, get the ID for our zone name
	zoneId, err := p.client.IdForDomain(zone)
	if err != nil {
		return nil, err
	}

	// get an array of DNSMadeEasy Records for our zone
	dmeRecords, err := p.client.EnumerateRecords(zoneId)
	if err != nil {
		return nil, err
	}

	// translate each DNSMadeEasy Domain Record to a libdns Record
	for _, rec := range dmeRecords {
		records = append(records, recordFromDmeRecord(rec))
	}

	return records, nil
}

func createRecords(client dme.Client, zone string, records []libdns.Record) ([]libdns.Record, error) {
	var dmeRecords []dme.Record

	// first, get the ID for our zone name
	zoneId, err := client.IdForDomain(zone)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		dmeRecord, err := dmeRecordFromRecord(record)
		if err != nil {
			return []libdns.Record{}, err
		}
		dmeRecords = append(dmeRecords, dmeRecord)
	}

	newDmeRecords, err := client.CreateRecords(zoneId, dmeRecords)
	if err != nil {
		return nil, err
	}

	var newRecords []libdns.Record
	for _, dmeRec := range newDmeRecords {
		newRec := recordFromDmeRecord(dmeRec)
		newRecords = append(newRecords, newRec)
	}

	return newRecords, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.init(ctx)

	return createRecords(p.client, zone, records)
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.init(ctx)

	// first, get the ID for our zone name
	zoneId, err := p.client.IdForDomain(zone)
	if err != nil {
		return nil, err
	}

	// get an array of DNSMadeEasy Records for our zone
	dmeRecords, err := p.client.EnumerateRecords(zoneId)
	if err != nil {
		return nil, err
	}

	// split our input records into those that need updating and those that need creating
	var existingRecords []libdns.Record
	var newRecords []libdns.Record
	for _, record := range records {
		foundIdx := slices.IndexFunc(dmeRecords, func(dmeRecord dme.Record) bool {
			return record.ID != "0" && fmt.Sprint(dmeRecord.ID) == record.ID
		})
		if foundIdx == -1 {
			newRecords = append(newRecords, record)
		} else {
			existingRecords = append(existingRecords, record)
		}
	}

	var dmeRecordsToUpdate []dme.Record
	for _, record := range existingRecords {
		newRecord, err := dmeRecordFromRecord(record)
		if err != nil {
			fmt.Printf("Could not convert %s record for %s: %s", record.Type, record.Name, err)
			continue
		}
		dmeRecordsToUpdate = append(dmeRecordsToUpdate, newRecord)
	}

	// update existing records
	// Note: this is performed first so that we don't leave our request
	// in a half-applied state
	updatedDmeRecords, err := p.client.UpdateRecords(zoneId, dmeRecordsToUpdate)
	if err != nil {
		return nil, err
	}

	// convert the DME Records to libdns records
	var updatedRecords []libdns.Record
	for _, record := range updatedDmeRecords {
		updatedRecords = append(updatedRecords, recordFromDmeRecord(record))
	}

	// create new records
	createdRecords, err := createRecords(p.client, zone, newRecords)
	if err != nil {
		return nil, err
	}

	// TODO: hopefully record ordering in the array isn't important
	return append(updatedRecords, createdRecords...), nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.init(ctx)

	// first, get the ID for our zone name
	zoneId, err := p.client.IdForDomain(zone)
	if err != nil {
		return nil, err
	}

	// convert an array of records into an array of integer
	// record ID to pass to the DME library
	var recordsToDelete []int
	for _, record := range records {
		id, err := strconv.Atoi(record.ID)
		if err != nil {
			fmt.Printf("Could not convert id '%s' to integer", record.ID)
			continue
		}
		recordsToDelete = append(recordsToDelete, id)
	}

	deletedRecords, err := p.client.DeleteRecords(zoneId, recordsToDelete)
	if err != nil {
		return nil, err
	}

	var returnRecords []libdns.Record
	for _, id := range deletedRecords {
		// find our deleted records ID in the original array argument
		recordId := slices.IndexFunc(records, func(rec libdns.Record) bool {
			return rec.ID == string(id)
		})
		if recordId == -1 {
			fmt.Printf("Could not find record id %d in supplied list of libdns.Record\n", id)
			continue
		}

		// add the full record to the array to be returned
		returnRecords = append(returnRecords, records[recordId])
	}

	return returnRecords, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
