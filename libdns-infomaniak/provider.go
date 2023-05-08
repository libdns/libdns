package infomaniak

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with infomaniak.
type Provider struct {
	//infomaniak API token
	APIToken string `json:"api_token,omitempty"`

	//infomaniak client used to call API
	client IkClient

	//mutex to prevent race conditions
	mu sync.Mutex
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	ikRecords, err := p.getClient().GetDnsRecordsForZone(ctx, zone)
	if err != nil {
		return nil, err
	}

	libdnsRecords := make([]libdns.Record, 0, len(ikRecords))
	for _, rec := range ikRecords {
		libdnsRecords = append(libdnsRecords, rec.ToLibDnsRecord(zone))
	}

	return libdnsRecords, nil
}

// getRecordsByCoordinates returns the existing records in this zone by their coordinates
func (p *Provider) getRecordsByCoordinates(ctx context.Context, zone string) (map[string][]libdns.Record, error) {
	records, err := p.GetRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	recordsByCoordinats := make(map[string][]libdns.Record)
	for _, rec := range records {
		coordinates := getCoordinates(rec)
		recordsWithSameCoordinates := recordsByCoordinats[coordinates]
		if recordsWithSameCoordinates == nil {
			recordsWithSameCoordinates = make([]libdns.Record, 0)
		}
		recordsWithSameCoordinates = append(recordsWithSameCoordinates, rec)
		recordsByCoordinats[coordinates] = recordsWithSameCoordinates
	}
	return recordsByCoordinats, nil
}

// getCoordinates returns the coordinates of a record
func getCoordinates(record libdns.Record) string {
	return fmt.Sprintf("%s-%s", record.Name, record.Type)
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	mergedRecs, err := p.getRecordsMergedWithAlreadyExistingOnes(ctx, zone, records)
	if err != nil {
		return nil, err
	}

	createdRecs := make([]libdns.Record, 0)
	for _, rec := range mergedRecs {
		if rec.ID == "" {
			createdRec, err := p.getClient().CreateOrUpdateRecord(ctx, zone, ToInfomaniakRecord(&rec, zone))
			if err != nil {
				return nil, err
			}
			createdRecs = append(createdRecs, createdRec.ToLibDnsRecord(zone))
		}
	}
	return createdRecs, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	recsToSet, err := p.getRecordsMergedWithAlreadyExistingOnes(ctx, zone, records)
	if err != nil {
		return nil, err
	}

	createdOrUpdatedRecs := make([]libdns.Record, 0)
	for _, rec := range recsToSet {
		updatedRec, err := p.getClient().CreateOrUpdateRecord(ctx, zone, ToInfomaniakRecord(&rec, zone))
		if err != nil {
			return nil, err
		}
		createdOrUpdatedRecs = append(createdOrUpdatedRecs, updatedRec.ToLibDnsRecord(zone))
	}

	return createdOrUpdatedRecs, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	recsToDelete, err := p.getRecordsMergedWithAlreadyExistingOnes(ctx, zone, records)
	if err != nil {
		return nil, err
	}

	deletedRecs := make([]libdns.Record, 0)
	for _, rec := range recsToDelete {
		if rec.ID != "" {
			err := p.getClient().DeleteRecord(ctx, zone, rec.ID)
			if err != nil {
				return nil, err
			}
			deletedRecs = append(deletedRecs, rec)
		}
	}
	return deletedRecs, nil
}

// getRecordsMergedWithAlreadyExistingOnes returns records with an ID immediately, checks for records without ID if a record with the same coordinates
// already exists, if yes, then it returns the updated already existing records otherwise the new record
func (p *Provider) getRecordsMergedWithAlreadyExistingOnes(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	result := make([]libdns.Record, 0)
	recsWithoutId := make([]libdns.Record, 0)

	for _, rec := range records {
		if rec.ID == "" {
			recsWithoutId = append(recsWithoutId, rec)
		} else {
			result = append(result, rec)
		}
	}

	if len(recsWithoutId) > 0 {
		mergedRecs, err := p.mergeRecordsWithExistingOnes(ctx, zone, recsWithoutId)
		if err != nil {
			return nil, err
		}
		result = append(result, mergedRecs...)
	}
	return result, nil
}

// mergeRecordsWithExistingOnes takes a list of records without ID. If one or multiple records with the same coordinates already exist, these records' data
// are updated and returned, otherwise the new record is returned without any ID set
func (p *Provider) mergeRecordsWithExistingOnes(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if len(records) <= 0 {
		return make([]libdns.Record, 0), nil
	}
	existingRecords, err := p.getRecordsByCoordinates(ctx, zone)
	if err != nil {
		return nil, err
	}
	result := make([]libdns.Record, 0)
	for _, rec := range records {
		if rec.ID != "" {
			return nil, errors.New("Got record that already exists as parameter")
		}
		recordsWithSameCoords := existingRecords[getCoordinates(rec)]
		if recordsWithSameCoords != nil && len(recordsWithSameCoords) > 0 {
			for _, existingRec := range recordsWithSameCoords {
				copy := rec
				copy.ID = existingRec.ID
				result = append(result, copy)
			}
		} else {
			result = append(result, rec)
		}
	}
	return result, nil
}

// getClient returns a new instance of the infomaniak API client
func (p *Provider) getClient() IkClient {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client == nil {
		p.client = &Client{Token: p.APIToken, HttpClient: http.DefaultClient}
	}
	return p.client
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
