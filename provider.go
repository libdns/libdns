// Package bluecat implements a DNS record management client compatible
// with the libdns interfaces for Bluecat Address Manager.
package bluecat

import (
	"context"
	"fmt"
	"sync"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with Bluecat Address Manager.
type Provider struct {
	// ServerURL is the base URL of the Bluecat Address Manager server
	// (e.g., "https://bluecat.example.com")
	ServerURL string `json:"server_url,omitempty"`

	// Username for authenticating with the Bluecat API
	Username string `json:"username,omitempty"`

	// Password for authenticating with the Bluecat API
	Password string `json:"password,omitempty"`

	// Configuration name in Bluecat (optional, defaults to first available)
	ConfigurationName string `json:"configuration_name,omitempty"`

	// View name in Bluecat (optional, defaults to first available)
	ViewName string `json:"view_name,omitempty"`

	client *Client
	mu     sync.Mutex
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	if err := p.ensureClient(ctx); err != nil {
		return nil, err
	}

	// Get the zone ID
	zoneID, err := p.client.GetZoneID(ctx, zone, p.ConfigurationName, p.ViewName)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID: %w", err)
	}

	// Get all resource records for the zone
	records, err := p.client.GetResourceRecords(ctx, zoneID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource records: %w", err)
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if err := p.ensureClient(ctx); err != nil {
		return nil, err
	}

	// Get the zone ID
	zoneID, err := p.client.GetZoneID(ctx, zone, p.ConfigurationName, p.ViewName)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID: %w", err)
	}

	var created []libdns.Record
	for _, record := range records {
		rec, err := p.client.CreateResourceRecord(ctx, zoneID, zone, record)
		if err != nil {
			return created, fmt.Errorf("failed to create record: %w", err)
		}
		created = append(created, rec)
	}

	return created, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if err := p.ensureClient(ctx); err != nil {
		return nil, err
	}

	// Get the zone ID
	zoneID, err := p.client.GetZoneID(ctx, zone, p.ConfigurationName, p.ViewName)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID: %w", err)
	}

	// Get existing records to determine what needs to be updated/created/deleted
	existingRecords, err := p.client.GetResourceRecords(ctx, zoneID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing records: %w", err)
	}

	// Build maps for easier comparison
	recordsByNameType := make(map[string][]libdns.Record)
	for _, rec := range records {
		rr := rec.RR()
		key := rr.Name + ":" + rr.Type
		recordsByNameType[key] = append(recordsByNameType[key], rec)
	}

	existingByNameType := make(map[string][]libdns.Record)
	for _, rec := range existingRecords {
		rr := rec.RR()
		key := rr.Name + ":" + rr.Type
		existingByNameType[key] = append(existingByNameType[key], rec)
	}

	var updated []libdns.Record

	// Delete records that exist in Bluecat but not in our new set
	for key, existingRecs := range existingByNameType {
		if _, exists := recordsByNameType[key]; !exists {
			// Delete all records with this name/type combo
			for _, rec := range existingRecs {
				if err := p.client.DeleteResourceRecord(ctx, rec); err != nil {
					return updated, fmt.Errorf("failed to delete record: %w", err)
				}
			}
		}
	}

	// Create or update records
	for _, rec := range records {
		rr := rec.RR()
		key := rr.Name + ":" + rr.Type

		if existing, exists := existingByNameType[key]; exists {
			// Update existing record - for simplicity, delete old and create new
			for _, oldRec := range existing {
				if err := p.client.DeleteResourceRecord(ctx, oldRec); err != nil {
					return updated, fmt.Errorf("failed to delete old record: %w", err)
				}
			}
		}

		// Create the new record
		created, err := p.client.CreateResourceRecord(ctx, zoneID, zone, rec)
		if err != nil {
			return updated, fmt.Errorf("failed to create/update record: %w", err)
		}
		updated = append(updated, created)
	}

	return updated, nil
}

// DeleteRecords deletes the specified records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if err := p.ensureClient(ctx); err != nil {
		return nil, err
	}

	// Get the zone ID
	zoneID, err := p.client.GetZoneID(ctx, zone, p.ConfigurationName, p.ViewName)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone ID: %w", err)
	}

	// Get all existing records to find the IDs we need to delete
	existingRecords, err := p.client.GetResourceRecords(ctx, zoneID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing records: %w", err)
	}

	var deleted []libdns.Record
	for _, record := range records {
		rr := record.RR()
		// Find matching record in existing records
		for _, existing := range existingRecords {
			existingRR := existing.RR()
			if matchesRecord(rr, existingRR) {
				if err := p.client.DeleteResourceRecord(ctx, existing); err != nil {
					return deleted, fmt.Errorf("failed to delete record: %w", err)
				}
				deleted = append(deleted, existing)
			}
		}
	}

	return deleted, nil
}

// ensureClient ensures the client is initialized and authenticated
func (p *Provider) ensureClient(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client != nil {
		return nil
	}

	if p.ServerURL == "" {
		return fmt.Errorf("server URL is required")
	}
	if p.Username == "" {
		return fmt.Errorf("username is required")
	}
	if p.Password == "" {
		return fmt.Errorf("password is required")
	}

	client, err := NewClient(p.ServerURL, p.Username, p.Password)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if err := client.Authenticate(ctx); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	p.client = client
	return nil
}

// matchesRecord checks if two records match based on the provided criteria
// If the input record has empty Type, TTL, or Data, those fields are ignored in the comparison
func matchesRecord(input, existing libdns.RR) bool {
	if input.Name != existing.Name {
		return false
	}
	if input.Type != "" && input.Type != existing.Type {
		return false
	}
	if input.TTL != 0 && input.TTL != existing.TTL {
		return false
	}
	if input.Data != "" && input.Data != existing.Data {
		return false
	}
	return true
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
