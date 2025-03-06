// Package neoserv implements a DNS record management client compatible
// with the libdns interfaces for Neoserv.
package neoserv

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/libdns/libdns"
	"github.com/pkg/errors"
)

// Provider facilitates DNS record manipulation with Neoserv.si.
type Provider struct {
	// Email used to authenticate with moj.neoserv.si
	Username string `json:"username,omitempty"`
	// Password used to authenticate with moj.neoserv.si
	Password string `json:"password,omitempty"`

	// UnsupportedTTLisError determines whether an unsupported TTL value should be treated as an error.
	// If set to true, the provider will return an error if an unsupported TTL value is requested.
	// If set to false, the provider will set the TTL to the nearest supported value that is at least the requested
	// value.
	UnsupportedTTLisError bool `json:"unsupported_ttl_is_error,omitempty"`

	// client is the HTTP client used to communicate with the Neoserv API.
	client *http.Client
	// zoneIdCache is a map of zone names to their corresponding zone IDs.
	// This is used to avoid making unnecessary API calls to get the zone ID.
	zoneIdCache map[string]string
}

// Neoserv API does not support all TTL values. The following are the supported TTL values.
// Check Provider.UnsupportedTTLisError to determine how unsupported TTL values are handled.
var (
	TTL1m     = 1 * time.Minute
	TTL5m     = 5 * time.Minute
	TTL15m    = 15 * time.Minute
	TTL30m    = 30 * time.Minute
	TTL1h     = 1 * time.Hour
	TTL6h     = 6 * time.Hour
	TTL12h    = 12 * time.Hour
	TTL24h    = 24 * time.Hour
	TTL2d     = 2 * 24 * time.Hour
	TTL7d     = 7 * 24 * time.Hour
	TTL14d    = 14 * 24 * time.Hour
	TTL30d    = 30 * 24 * time.Hour
	ValidTTLs = []time.Duration{
		TTL1m, TTL5m, TTL15m, TTL30m, TTL1h, TTL6h, TTL12h, TTL24h, TTL2d, TTL7d, TTL14d, TTL30d,
	}
)

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	return p.getRecords(ctx, zone)
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	// Validate the records
	for _, record := range records {
		if record.ID != "" {
			return nil, fmt.Errorf("failed to append records: record ID must be empty")
		}
		ttl, err := p.getRecordTTL(record.TTL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to append records")
		}
		record.TTL = ttl
	}

	// Because Neoserv does not return the ID of the newly added record(s), we need to
	// identify the new records by comparing the list of records before and after the append operation.
	// We will return the new records with their IDs in the same order as the input records.
	appendedRecords := make([]libdns.Record, 0, len(records))

	// Get the records before the append operation
	oldRecords, err := p.getRecords(ctx, zone)
	if err != nil {
		return appendedRecords, errors.Wrap(err, "failed to append records")
	}

	// Append each record separately
	for _, record := range records {
		if err := p.createRecord(ctx, zone, record); err != nil {
			return appendedRecords, errors.Wrap(err, "failed to append records")
		}
		appendedRecords = append(appendedRecords, record)
	}

	// Get list of records after the append operation(s) to identify the new records
	afterRecords, err := p.getRecords(ctx, zone)
	if err != nil {
		return appendedRecords, errors.Wrap(err, "failed to append records")
	}
	// Create a set of current records for quick lookup
	oldRecordIDs := make(map[string]struct{}, len(oldRecords))
	for _, record := range oldRecords {
		oldRecordIDs[record.ID] = struct{}{}
	}
	// Identify the new records
	newRecords := make([]libdns.Record, 0, len(records))
	for _, record := range afterRecords {
		if _, ok := oldRecordIDs[record.ID]; !ok {
			newRecords = append(newRecords, record)
		}
	}

	// Assign new records to the appended records
	for i, appendedRecord := range appendedRecords {
		// Find the corresponding record in the new records
		matchingIdx := -1
		for j, newRecord := range newRecords {
			if sameRecord(&appendedRecord, &newRecord) {
				matchingIdx = j
				break
			}
		}
		if matchingIdx == -1 {
			return appendedRecords, fmt.Errorf("failed to append records: record %s not found in new records", appendedRecord.Name)
		}
		appendedRecords[i] = newRecords[matchingIdx]

		// Remove the record from the addedRecords
		newRecords[matchingIdx] = newRecords[len(newRecords)-1]
		newRecords = newRecords[:len(newRecords)-1]
	}

	return appendedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	// Validate the records
	for _, record := range records {
		ttl, err := p.getRecordTTL(record.TTL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to set records")
		}
		record.TTL = ttl
	}

	// Get the current records
	currentRecords, err := p.getRecords(ctx, zone)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set records")
	}
	currentRecordsSet := make(map[string]libdns.Record, len(currentRecords))
	for _, record := range currentRecords {
		currentRecordsSet[record.ID] = record
	}

	// Split the records into two groups: records to add and records to edit
	toAdd := make([]libdns.Record, 0, len(records))
	toAddIdx := make([]int, 0, len(records))
	toEditIdx := make([]int, 0, len(records))

	setRecords := make([]libdns.Record, len(records))
	for i, record := range records {
		if record.ID == "" {
			toAdd = append(toAdd, record)
			toAddIdx = append(toAddIdx, i)
		} else {
			if currentRecord, ok := currentRecordsSet[record.ID]; !ok {
				return nil, fmt.Errorf("failed to set records: record %s not found", record.Name)
			} else {
				if sameRecord(&record, &currentRecord) {
					setRecords[i] = record // No need to edit
				} else {
					toEditIdx = append(toEditIdx, i)
				}
			}
		}
	}

	// Append the records to add
	added, err := p.AppendRecords(ctx, zone, toAdd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to set records")
	}
	if len(added) != len(toAdd) {
		return nil, fmt.Errorf("failed to set records: expected %d records to be added, got %d", len(toAdd), len(added))
	}
	for i, idx := range toAddIdx {
		setRecords[idx] = added[i]
	}

	// Edit the records to edit
	for _, idx := range toEditIdx {
		if err := p.updateRecord(ctx, zone, records[idx]); err != nil {
			return nil, errors.Wrap(err, "failed to set records")
		}
		setRecords[idx] = records[idx]
	}

	return setRecords, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	for _, record := range records {
		if record.ID == "" {
			return nil, fmt.Errorf("failed to delete records: record ID is required")
		}
	}

	removed := make([]libdns.Record, 0, len(records))
	for _, record := range records {
		if err := p.deleteRecord(ctx, zone, record); err != nil {
			return removed, errors.Wrap(err, "failed to delete records")
		}
		removed = append(removed, record)
	}

	return removed, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
