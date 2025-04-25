package simplydotcom

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/libdns/libdns"
)

// Provider implements the libdns interfaces for Simply.com.
type Provider struct {
	// AccountName is the account name of the Simply.com account. It is required.
	AccountName string `json:"account_name,omitempty"`
	// APIKey is the API key for the Simply.com account. It is required.
	APIKey string `json:"api_key,omitempty"`

	// BaseURL is the base URL of the Simply.com API. Default is https://api.simply.com/2/
	BaseURL string `json:"base_url,omitempty"`

	// MaxRetries is the maximum number of retries if rate limiting occurs. Default is 3.
	MaxRetries *int `json:"max_retries,omitempty"`

	client simplyClient
	mu     sync.Mutex // Mutex to protect concurrent access to client
}

func (p *Provider) init() {
	if p.client != nil {
		return // Already initialized
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client != nil {
		return // Already initialized
	}

	if p.BaseURL == "" {
		p.BaseURL = "https://api.simply.com/2/"
	}

	maxRetries := 3
	if p.MaxRetries != nil {
		maxRetries = *p.MaxRetries
	}

	p.client = &simplyApiClient{
		accountName: p.AccountName,
		apiKey:      p.APIKey,
		baseURL:     p.BaseURL,
		maxRetries:  maxRetries,
		httpClient:  http.DefaultClient,
	}
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.init()

	records, err := p.client.getDnsRecords(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS records for zone %s: %w", zone, err)
	}

	var libdnsRecords []libdns.Record
	for _, record := range records {
		libdnsRecord, err := record.toLibdns(zone)
		if err != nil {
			return nil, fmt.Errorf("failed to convert DNS record (id: %d) to libdns format: %w", record.Id, err)
		}
		libdnsRecords = append(libdnsRecords, libdnsRecord)
	}
	return libdnsRecords, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.init()

	// Create a map to track added record IDs for O(1) lookup
	recordIdSet := make(map[int]struct{}, len(records))

	for _, record := range records {
		recordInfo, err := p.client.addDnsRecord(ctx, zone, toSimply(record))
		if err != nil {
			return nil, fmt.Errorf("failed to add DNS record %s %s: %w", record.RR().Type, record.RR().Name, err)
		}

		// Add the ID to our set
		recordIdSet[recordInfo.Record.Id] = struct{}{}
	}

	respRecords, err := p.client.getDnsRecords(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("DNS records were added but failed to get DNS records for zone %s: %w", zone, err)
	}

	// Pre-allocate the slice with the known capacity (number of added records)
	addedRecords := make([]libdns.Record, 0, len(recordIdSet))

	// Filter the records to only include those that were added
	for _, record := range respRecords {
		if _, exists := recordIdSet[record.Id]; exists {
			libdnsRecord, err := record.toLibdns(zone)
			if err != nil {
				return nil, fmt.Errorf("failed to convert added DNS record (id: %d) to libdns format: %w", record.Id, err)
			}
			addedRecords = append(addedRecords, libdnsRecord)
		}
	}

	return addedRecords, nil
}

// SetRecords updates the zone so that the records described in the input are
// reflected in the output. It may create or update records or—depending on
// the record type—delete records to maintain parity with the input. No other
// records are affected. It returns the records which were set.
//
// For any (name, type) pair in the input, SetRecords ensures that the only
// records in the output zone with that (name, type) pair are those that were
// provided in the input.
//
// In RFC 9499 terms, SetRecords appends, modifies, or deletes records in the
// zone so that for each RRset in the input, the records provided in the input
// are the only members of their RRset in the output zone.
//
// The Simply.com implementation does not implement support for DNSSEC-related records.
//
// WARNING: Calls to SetRecords for Simply.com are not atomic. If an error occurs, one or
// more of the requested changes may have been applied.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.init()

	// Get existing records
	existingRecords, err := p.client.getDnsRecords(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing DNS records for zone %s: %w", zone, err)
	}

	// Plan the changes needed
	plannedChanges := p.planSetRecordsChanges(existingRecords, records)

	// Execute all planned changes
	affectedRecordIDs, err := p.executeSetRecordsChanges(ctx, zone, plannedChanges)
	if err != nil {
		return nil, fmt.Errorf("failed to execute DNS changes for zone %s: %w", zone, err)
	}

	// Get the updated records to return
	updatedRecords, err := p.getRecordsById(ctx, zone, affectedRecordIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated records for zone %s: %w", zone, err)
	}

	return updatedRecords, nil
}

// DeleteRecords deletes the given records from the zone if they exist in the
// zone and exactly match the input. If the input records do not exist in the
// zone, they are silently ignored. DeleteRecords returns only the the records
// that were deleted, and does not return any records that were provided in the
// input but did not exist in the zone.
//
// DeleteRecords only deletes records from the zone that *exactly* match the
// input records—that is, the name, type, TTL, and value all must be identical
// to a record in the zone for it to be deleted.
//
// As a special case, you may leave any of the fields [libdns.Record.Type],
// [libdns.Record.TTL], or [libdns.Record.Value] empty ("", 0, and ""
// respectively). In this case, DeleteRecords will delete any records that
// match the other fields, regardless of the value of the fields that were left
// empty. Note that this behavior does *not* apply to the [libdns.Record.Name]
// field, which must always be specified.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.init()

	zoneRecords, err := p.client.getDnsRecords(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS records for zone %s: %w", zone, err)
	}

	deletedRecords := make([]libdns.Record, 0, len(records)) // Pre-allocate with estimated capacity

	for _, zoneRecord := range zoneRecords {
		zoneLibdnsRecord, err := zoneRecord.toLibdns(zone)
		if err != nil {
			return nil, fmt.Errorf("failed to convert zone record (id: %d) to libdns format: %w", zoneRecord.Id, err)
		}

		for _, inputRecord := range records {
			inputRR := inputRecord.RR()

			if isRecordMatch(zoneRecord, inputRR, zone) {
				// Delete the record immediately
				err := p.client.deleteDnsRecord(ctx, zone, zoneRecord.Id)
				if err != nil {
					return deletedRecords, fmt.Errorf("failed to delete DNS record %d in zone %s: %w", zoneRecord.Id, zone, err)
				}

				// Add to deleted records
				deletedRecords = append(deletedRecords, zoneLibdnsRecord)
				break // Once we've matched and deleted this zone record, move to the next one
			}
		}
	}

	return deletedRecords, nil
}

// isRecordMatch determines if a DNS record matches the criteria according to deletion rules.
// It checks name, type, TTL, and data value according to specified matching rules.
func isRecordMatch(candidateRec dnsRecordResponse, criteriaRec libdns.RR, zone string) bool {
	// libdns.Record.Name is required, so we must always match on name
	if libdns.AbsoluteName(candidateRec.Name, zone) != libdns.AbsoluteName(criteriaRec.Name, zone) {
		return false
	}

	// Match on type if criteria type is not empty
	if criteriaRec.Type != "" && candidateRec.Type != criteriaRec.Type {
		return false
	}

	// Match on TTL if criteria TTL is not zero
	if criteriaRec.TTL != 0 && time.Duration(candidateRec.Ttl)*time.Second != criteriaRec.TTL {
		return false
	}

	// Match on value if criteria value is not empty
	// For some record types, we need to convert recordToCheck to libdns format to ensure
	// consistent data format comparison
	if criteriaRec.Data != "" {
		// Convert to libdns.Record format for data comparison
		libdnsRec, err := candidateRec.toLibdns(zone)
		if err != nil {
			return false
		}

		if libdnsRec.RR().Data != criteriaRec.Data {
			return false
		}
	}

	// If we get here, the record matches all criteria
	return true
}

// recordKey is a struct that uniquely identifies a record by name and type, used by SetRecords
type recordKey struct {
	Name string
	Type string
}

// operationType represents the type of operation to perform on a DNS record, used by SetRecords
type operationType int

const (
	opUpdate operationType = iota
	opDelete
	opCreate
)

// plannedChange represents a change to be applied to a DNS record, used by SetRecords
type plannedChange struct {
	op        operationType
	recordID  int
	dnsRecord dnsRecord
}

// groupRecordsByNameAndType groups records by their name and type for easier RRSet handling.
func groupRecordsByNameAndType[T any](records []T, nameFunc func(T) string, typeFunc func(T) string) map[recordKey][]T {
	result := make(map[recordKey][]T, len(records))

	for _, record := range records {
		key := recordKey{
			Name: nameFunc(record),
			Type: typeFunc(record),
		}
		result[key] = append(result[key], record)
	}

	return result
}

// planSetRecordsChanges determines what changes are needed to make the zone match the desired state, used by SetRecords
func (p *Provider) planSetRecordsChanges(existingRecords []dnsRecordResponse, inputRecords []libdns.Record) []plannedChange {
	var plannedChanges []plannedChange

	// Group existing records by name and type
	existingByKey := groupRecordsByNameAndType(existingRecords,
		func(r dnsRecordResponse) string { return r.Name },
		func(r dnsRecordResponse) string { return r.Type })

	// Group input records by name and type
	inputByKey := groupRecordsByNameAndType(inputRecords,
		func(r libdns.Record) string { return r.RR().Name },
		func(r libdns.Record) string { return r.RR().Type })

	// Process each RRSet in the input
	for key, inputSet := range inputByKey {
		existingSet, exists := existingByKey[key]

		if exists {
			// Update at most min(len(existingSet), len(inputSet)) records
			minLen := len(existingSet)
			if len(inputSet) < minLen {
				minLen = len(inputSet)
			}

			// Update the records we're keeping
			for i := 0; i < minLen; i++ {
				simplyRec := toSimply(inputSet[i])
				plannedChanges = append(plannedChanges, plannedChange{
					op:        opUpdate,
					recordID:  existingSet[i].Id,
					dnsRecord: simplyRec,
				})
			}

			// Delete excess existing records
			for i := minLen; i < len(existingSet); i++ {
				plannedChanges = append(plannedChanges, plannedChange{
					op:       opDelete,
					recordID: existingSet[i].Id,
				})
			}

			// Add new records if we have more inputs than existing
			for i := minLen; i < len(inputSet); i++ {
				simplyRec := toSimply(inputSet[i])
				plannedChanges = append(plannedChanges, plannedChange{
					op:        opCreate,
					dnsRecord: simplyRec,
				})
			}
		} else {
			// No existing records with this name and type, create all
			for _, inputRecord := range inputSet {
				simplyRec := toSimply(inputRecord)
				plannedChanges = append(plannedChanges, plannedChange{
					op:        opCreate,
					dnsRecord: simplyRec,
				})
			}
		}
	}

	return plannedChanges
}

// executeSetRecordsChanges applies the planned changes to the DNS zone, used by SetRecords.
func (p *Provider) executeSetRecordsChanges(ctx context.Context, zone string, plannedChanges []plannedChange) (map[int]struct{}, error) {
	affectedRecordIDs := make(map[int]struct{}, len(plannedChanges))

	for _, change := range plannedChanges {
		switch change.op {
		case opUpdate:
			err := p.client.updateDnsRecord(ctx, zone, change.recordID, change.dnsRecord)
			if err != nil {
				return affectedRecordIDs, fmt.Errorf("failed to update DNS record %d in zone %s: %w", change.recordID, zone, err)
			}
			affectedRecordIDs[change.recordID] = struct{}{}

		case opDelete:
			err := p.client.deleteDnsRecord(ctx, zone, change.recordID)
			if err != nil {
				return affectedRecordIDs, fmt.Errorf("failed to delete DNS record %d in zone %s: %w", change.recordID, zone, err)
			}

		case opCreate:
			recordInfo, err := p.client.addDnsRecord(ctx, zone, change.dnsRecord)
			if err != nil {
				return affectedRecordIDs, fmt.Errorf("failed to add DNS record (%s %s) in zone %s: %w",
					change.dnsRecord.Type, change.dnsRecord.Name, zone, err)
			}
			affectedRecordIDs[recordInfo.Record.Id] = struct{}{}
		}
	}

	return affectedRecordIDs, nil
}

// getRecordsById retrieves the records that match the provided record IDs.
func (p *Provider) getRecordsById(ctx context.Context, zone string, affectedRecordIDs map[int]struct{}) ([]libdns.Record, error) {
	// Fetch all records after changes
	allRecords, err := p.client.getDnsRecords(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated records for zone %s: %w", zone, err)
	}

	// Collect updated or created records
	updatedRecords := make([]libdns.Record, 0, len(affectedRecordIDs))
	for _, record := range allRecords {
		if _, wasAffected := affectedRecordIDs[record.Id]; wasAffected {
			libdnsRecord, err := record.toLibdns(zone)
			if err != nil {
				return nil, fmt.Errorf("failed to convert affected DNS record (id: %d) to libdns format: %w", record.Id, err)
			}
			updatedRecords = append(updatedRecords, libdnsRecord)
		}
	}

	return updatedRecords, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
