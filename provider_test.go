package simplydotcom

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

// mockSimplyClient is a mock implementation of the simplyClient interface
type mockSimplyClient struct {
	records         []dnsRecordResponse
	recordsToReturn []dnsRecordResponse
	updatedRecords  map[int]dnsRecord
	deletedRecords  map[int]struct{}
	addedRecords    []dnsRecord
	nextRecordID    int

	getDnsRecordsFunc   func(context.Context, string) ([]dnsRecordResponse, error)
	updateDnsRecordFunc func(context.Context, string, int, dnsRecord) error
	deleteDnsRecordFunc func(context.Context, string, int) error
	addDnsRecordFunc    func(context.Context, string, dnsRecord) (createRecordResponse, error)
}

func newMockSimplyClient() *mockSimplyClient {
	return &mockSimplyClient{
		records:        []dnsRecordResponse{},
		updatedRecords: make(map[int]dnsRecord),
		deletedRecords: make(map[int]struct{}),
		addedRecords:   []dnsRecord{},
		nextRecordID:   1000,
	}
}

func (m *mockSimplyClient) getDnsRecords(ctx context.Context, zone string) ([]dnsRecordResponse, error) {
	if m.getDnsRecordsFunc != nil {
		return m.getDnsRecordsFunc(ctx, zone)
	}

	// If recordsToReturn is set, use that instead of records
	if m.recordsToReturn != nil {
		return m.recordsToReturn, nil
	}

	return m.records, nil
}

func (m *mockSimplyClient) updateDnsRecord(ctx context.Context, zone string, recordID int, record dnsRecord) error {
	if m.updateDnsRecordFunc != nil {
		return m.updateDnsRecordFunc(ctx, zone, recordID, record)
	}

	m.updatedRecords[recordID] = record

	// Update the record in the records list too
	for i, r := range m.records {
		if r.Id == recordID {
			m.records[i].Name = record.Name
			m.records[i].Type = record.Type
			m.records[i].Data = record.Data
			m.records[i].Ttl = record.Ttl
			m.records[i].Priority = record.Priority
			break
		}
	}

	return nil
}

func (m *mockSimplyClient) deleteDnsRecord(ctx context.Context, zone string, recordID int) error {
	if m.deleteDnsRecordFunc != nil {
		return m.deleteDnsRecordFunc(ctx, zone, recordID)
	}

	m.deletedRecords[recordID] = struct{}{}

	// Remove the record from the records list
	for i, r := range m.records {
		if r.Id == recordID {
			m.records = append(m.records[:i], m.records[i+1:]...)
			break
		}
	}

	return nil
}

func (m *mockSimplyClient) addDnsRecord(ctx context.Context, zone string, record dnsRecord) (createRecordResponse, error) {
	if m.addDnsRecordFunc != nil {
		return m.addDnsRecordFunc(ctx, zone, record)
	}

	m.addedRecords = append(m.addedRecords, record)

	// Create a new record with an ID
	newRecord := dnsRecordResponse{
		Id:        m.nextRecordID,
		dnsRecord: record,
	}
	m.nextRecordID++

	// Add it to the records list
	m.records = append(m.records, newRecord)

	return createRecordResponse{
		Record: struct {
			Id int "json:\"id\""
		}{Id: newRecord.Id},
	}, nil
}

// TestProvider_SetRecords tests the SetRecords method
func TestProvider_SetRecords(t *testing.T) {
	// Create test IPv4 and IPv6 addresses
	ipv4 := netip.MustParseAddr("192.0.2.1")
	// IPv6 address for when needed in tests
	// ipv6 := netip.MustParseAddr("2001:db8::1")

	uint16Ptr := func(v uint16) *uint16 {
		return &v
	}

	// Test cases
	tests := []struct {
		name               string
		existingRecords    []dnsRecordResponse
		inputRecords       []libdns.Record
		expectedUpdates    int
		expectedDeletes    int
		expectedCreates    int
		getDnsRecordsError error
		updateRecordError  error
		deleteRecordError  error
		addRecordError     error
		wantErr            bool
		// New field for verifying the final state
		expectedFinalRecords []dnsRecordResponse // The records we expect to exist after the operation
	}{
		{
			name: "Update existing record",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "198.51.100.1", // Different IP
						Ttl:  3600,
					},
				},
			},
			inputRecords: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
			},
			expectedUpdates: 1,
			expectedDeletes: 0,
			expectedCreates: 0,
			// Verify the final state
			expectedFinalRecords: []dnsRecordResponse{
				{
					Id: 1001, // ID remains the same
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1", // Updated to new IP
						Ttl:  3600,
					},
				},
			},
		},
		{
			name:            "Create new record",
			existingRecords: []dnsRecordResponse{
				// No matching record
			},
			inputRecords: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
			},
			expectedUpdates: 0,
			expectedDeletes: 0,
			expectedCreates: 1,
			// Verify the final state
			expectedFinalRecords: []dnsRecordResponse{
				{
					Id: 1000, // The mock client starts with ID 1000
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
			},
		},
		{
			name: "Delete excess record",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
				{
					Id: 1002,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.2",
						Ttl:  3600,
					},
				},
			},
			inputRecords: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
			},
			expectedUpdates: 1,
			expectedDeletes: 1,
			expectedCreates: 0,
			// Verify the final state - only one record should remain
			// Note: We can't know for sure which ID will remain as it's implementation-dependent
			// So we'll check this separately in the test validation
		},
		{
			name: "Mixed operations",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "198.51.100.1", // Will be updated
						Ttl:  3600,
					},
				},
				{
					Id: 1002,
					dnsRecord: dnsRecord{
						Name: "old",
						Type: "A",
						Data: "192.0.2.2",
						Ttl:  3600,
					},
				},
			},
			inputRecords: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
				libdns.Address{
					Name: "new",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
			},
			expectedUpdates: 1,
			expectedDeletes: 0, // "old" record should NOT be removed as it doesn't match any input record
			expectedCreates: 1, // "new" record should be created
			// Verify the final state
			expectedFinalRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1", // Updated
						Ttl:  3600,
					},
				},
				{
					Id: 1002,
					dnsRecord: dnsRecord{
						Name: "old",
						Type: "A",
						Data: "192.0.2.2", // Unchanged
						Ttl:  3600,
					},
				},
				{
					Id: 1000, // New record with next ID
					dnsRecord: dnsRecord{
						Name: "new",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
			},
		},
		{
			name: "Comprehensive RRSet handling",
			existingRecords: []dnsRecordResponse{
				// RRSet #1: "mail" MX records (will have one updated, one deleted)
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name:     "mail",
						Type:     "MX",
						Data:     "mail1.example.com.",
						Ttl:      3600,
						Priority: uint16Ptr(10),
					},
				},
				{
					Id: 1002,
					dnsRecord: dnsRecord{
						Name:     "mail",
						Type:     "MX",
						Data:     "mail2.example.com.",
						Ttl:      3600,
						Priority: uint16Ptr(20),
					},
				},
				// RRSet #2: "www" A records (will be completely replaced)
				{
					Id: 1003,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "198.51.100.1",
						Ttl:  3600,
					},
				},
				// RRSet #3: "api" A record (will be left untouched)
				{
					Id: 1004,
					dnsRecord: dnsRecord{
						Name: "api",
						Type: "A",
						Data: "198.51.100.2",
						Ttl:  3600,
					},
				},
			},
			inputRecords: []libdns.Record{
				// Only one "mail" MX record (so one will be updated, one deleted)
				libdns.MX{
					Name:       "mail",
					TTL:        7200 * time.Second, // Changed TTL
					Preference: 30,                 // Changed preference
					Target:     "mail1.example.com.",
				},
				// Two "www" A records (the existing one will be updated, new one created)
				libdns.Address{
					Name: "www",
					TTL:  7200 * time.Second,
					IP:   ipv4,
				},
				libdns.Address{
					Name: "www",
					TTL:  7200 * time.Second,
					IP:   netip.MustParseAddr("192.0.2.2"),
				},
				// New "cdn" A record (will be created)
				libdns.Address{
					Name: "cdn",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
			},
			expectedUpdates: 2, // Update 1 "mail" MX record and 1 "www" A record
			expectedDeletes: 1, // Delete 1 "mail" MX record
			expectedCreates: 2, // Create 1 new "www" A record and 1 "cdn" A record
			// Verify the final state
			expectedFinalRecords: []dnsRecordResponse{
				// Updated "mail" MX record
				{
					Id: 1001, // One of the mail records will be updated
					dnsRecord: dnsRecord{
						Name:     "mail",
						Type:     "MX",
						Data:     "mail1.example.com.",
						Ttl:      7200,          // Updated
						Priority: uint16Ptr(30), // Updated
					},
				},
				// Updated "www" A record
				{
					Id: 1003,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1", // Updated
						Ttl:  7200,        // Updated
					},
				},
				// New "www" A record (second record)
				{
					Id: 1000, // ID for first new record
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.2",
						Ttl:  7200,
					},
				},
				// New "cdn" A record
				{
					Id: 1001, // ID for second new record - but this could be 1000 + the number of creates
					dnsRecord: dnsRecord{
						Name: "cdn",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
				// Unchanged "api" A record
				{
					Id: 1004,
					dnsRecord: dnsRecord{
						Name: "api",
						Type: "A",
						Data: "198.51.100.2",
						Ttl:  3600,
					},
				},
			},
		},
		{
			name:               "Error getting DNS records",
			getDnsRecordsError: errors.New("failed to get records"),
			wantErr:            true,
		},
		{
			name: "Error updating record",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "198.51.100.1",
						Ttl:  3600,
					},
				},
			},
			inputRecords: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
			},
			updateRecordError: errors.New("failed to update record"),
			wantErr:           true,
		},
		{
			name: "Error deleting record",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
				{
					Id: 1002,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.2",
						Ttl:  3600,
					},
				},
			},
			inputRecords: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
			},
			deleteRecordError: errors.New("failed to delete record"),
			wantErr:           true,
		},
		{
			name:            "Error creating record",
			existingRecords: []dnsRecordResponse{
				// No matching record
			},
			inputRecords: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
			},
			addRecordError: errors.New("failed to create record"),
			wantErr:        true,
		},
		{
			name: "Different record types with same name",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
				{
					Id: 1002,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "AAAA",
						Data: "2001:db8::1",
						Ttl:  3600,
					},
				},
			},
			inputRecords: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  7200 * time.Second, // Changed TTL
					IP:   ipv4,
				},
			},
			expectedUpdates: 1, // Should update the A record
			expectedDeletes: 0, // Should NOT delete the AAAA record (different type)
			expectedCreates: 0,
			// Verify the final state
			expectedFinalRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  7200, // Updated TTL
					},
				},
				{
					Id: 1002,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "AAAA",
						Data: "2001:db8::1",
						Ttl:  3600, // Unchanged
					},
				},
			},
		},
		{
			name: "Multiple records of same type",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
			},
			inputRecords: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  7200 * time.Second,
					IP:   ipv4,
				},
				libdns.Address{
					Name: "www",
					TTL:  7200 * time.Second,
					IP:   netip.MustParseAddr("192.0.2.2"),
				},
			},
			expectedUpdates: 1, // Should update the first record
			expectedDeletes: 0,
			expectedCreates: 1, // Should create the second record
			// Verify the final state
			expectedFinalRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  7200, // Updated TTL
					},
				},
				{
					Id: 1000, // New record
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.2",
						Ttl:  7200,
					},
				},
			},
		},
		{
			name: "MX records handling",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name:     "mail",
						Type:     "MX",
						Data:     "mail1.example.com.",
						Ttl:      3600,
						Priority: uint16Ptr(10),
					},
				},
			},
			inputRecords: []libdns.Record{
				libdns.MX{
					Name:       "mail",
					TTL:        3600 * time.Second,
					Preference: 20, // Changed preference
					Target:     "mail1.example.com.",
				},
			},
			expectedUpdates: 1,
			expectedDeletes: 0,
			expectedCreates: 0,
			// Verify the final state
			expectedFinalRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name:     "mail",
						Type:     "MX",
						Data:     "mail1.example.com.",
						Ttl:      3600,
						Priority: uint16Ptr(20), // Updated preference
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Special case for "Delete excess record" test - we don't know exactly which
			// record will remain, so we'll handle it separately
			isDeleteExcessTest := tc.name == "Delete excess record"

			// Set up mock client
			mockClient := newMockSimplyClient()
			mockClient.records = tc.existingRecords

			// Set up error conditions if needed
			if tc.getDnsRecordsError != nil {
				mockClient.getDnsRecordsFunc = func(ctx context.Context, zone string) ([]dnsRecordResponse, error) {
					return nil, tc.getDnsRecordsError
				}
			}

			if tc.updateRecordError != nil {
				mockClient.updateDnsRecordFunc = func(ctx context.Context, zone string, recordID int, record dnsRecord) error {
					return tc.updateRecordError
				}
			}

			if tc.deleteRecordError != nil {
				mockClient.deleteDnsRecordFunc = func(ctx context.Context, zone string, recordID int) error {
					return tc.deleteRecordError
				}
			}

			if tc.addRecordError != nil {
				mockClient.addDnsRecordFunc = func(ctx context.Context, zone string, record dnsRecord) (createRecordResponse, error) {
					return createRecordResponse{}, tc.addRecordError
				}
			}

			// Create provider with mock client
			provider := &Provider{}
			provider.client = mockClient

			// Execute SetRecords
			ctx := context.Background()
			zone := "example.com."
			_, err := provider.SetRecords(ctx, zone, tc.inputRecords)

			// Check if error was expected
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check operations count
			if len(mockClient.updatedRecords) != tc.expectedUpdates {
				t.Errorf("expected %d updates, got %d", tc.expectedUpdates, len(mockClient.updatedRecords))
			}

			if len(mockClient.deletedRecords) != tc.expectedDeletes {
				t.Errorf("expected %d deletes, got %d", tc.expectedDeletes, len(mockClient.deletedRecords))
			}

			if len(mockClient.addedRecords) != tc.expectedCreates {
				t.Errorf("expected %d creates, got %d", tc.expectedCreates, len(mockClient.addedRecords))
			}

			// Special case: For the "Delete excess record" test,
			// we just verify that exactly one record remains with name="www" and type="A"
			if isDeleteExcessTest {
				var wwwRecords int
				for _, record := range mockClient.records {
					if record.Name == "www" && record.Type == "A" {
						wwwRecords++
						// Also verify the data and TTL
						if record.Data != "192.0.2.1" {
							t.Errorf("expected remaining record to have data=192.0.2.1, got %s", record.Data)
						}
						if record.Ttl != 3600 {
							t.Errorf("expected remaining record to have TTL=3600, got %d", record.Ttl)
						}
					}
				}
				if wwwRecords != 1 {
					t.Errorf("expected exactly 1 www A record to remain, got %d", wwwRecords)
				}
				return
			}

			// For non-error and non-special test cases, verify the final state
			if len(tc.expectedFinalRecords) > 0 {
				// The IDs might not match exactly for created records, so we'll create
				// maps of records by name+type for comparison
				finalRecordsByNameType := make(map[string][]dnsRecordResponse)
				expectedFinalByNameType := make(map[string][]dnsRecordResponse)

				// Build map of actual final records
				for _, record := range mockClient.records {
					key := record.Name + ":" + record.Type
					finalRecordsByNameType[key] = append(finalRecordsByNameType[key], record)
				}

				// Build map of expected final records
				for _, record := range tc.expectedFinalRecords {
					key := record.Name + ":" + record.Type
					expectedFinalByNameType[key] = append(expectedFinalByNameType[key], record)
				}

				// Check that we have the same set of name+type keys
				for key, expectedRecords := range expectedFinalByNameType {
					actualRecords, exists := finalRecordsByNameType[key]
					if !exists {
						t.Errorf("expected records for %s but found none", key)
						continue
					}

					// Check count of records for this name+type combination
					if len(actualRecords) != len(expectedRecords) {
						t.Errorf("expected %d records for %s, got %d",
							len(expectedRecords), key, len(actualRecords))
						continue
					}

					// For each expected record, find a matching actual record
					for _, expected := range expectedRecords {
						found := false
						for _, actual := range actualRecords {
							// For existing records that should be updated, we care about the ID
							if expected.Id >= 1000 && expected.Id < mockClient.nextRecordID-len(mockClient.addedRecords) {
								if expected.Id == actual.Id &&
									expected.Name == actual.Name &&
									expected.Type == actual.Type &&
									expected.Data == actual.Data &&
									expected.Ttl == actual.Ttl {

									// For MX records, also check priority
									if expected.Type == "MX" {
										if (expected.Priority == nil && actual.Priority != nil) ||
											(expected.Priority != nil && actual.Priority == nil) ||
											(expected.Priority != nil && actual.Priority != nil && *expected.Priority != *actual.Priority) {
											continue // Priority doesn't match
										}
									}

									found = true
									break
								}
							} else {
								// For new records, we don't care about the ID
								if expected.Name == actual.Name &&
									expected.Type == actual.Type &&
									expected.Data == actual.Data &&
									expected.Ttl == actual.Ttl {

									// For MX records, also check priority
									if expected.Type == "MX" {
										if (expected.Priority == nil && actual.Priority != nil) ||
											(expected.Priority != nil && actual.Priority == nil) ||
											(expected.Priority != nil && actual.Priority != nil && *expected.Priority != *actual.Priority) {
											continue // Priority doesn't match
										}
									}

									found = true
									break
								}
							}
						}

						if !found {
							t.Errorf("expected record {Name:%s, Type:%s, Data:%s, TTL:%d} not found in final records",
								expected.Name, expected.Type, expected.Data, expected.Ttl)
						}
					}
				}

				// Check for unexpected records
				for key := range finalRecordsByNameType {
					if _, exists := expectedFinalByNameType[key]; !exists {
						t.Errorf("found unexpected records for %s", key)
					}
				}
			}
		})
	}
}

func TestProvider_SetRecords_GetRecordsById(t *testing.T) {
	// Test that getRecordsById returns the correct records
	mockClient := newMockSimplyClient()
	mockClient.records = []dnsRecordResponse{
		{
			Id: 1001,
			dnsRecord: dnsRecord{
				Name: "www",
				Type: "A",
				Data: "192.0.2.1",
				Ttl:  3600,
			},
		},
		{
			Id: 1002,
			dnsRecord: dnsRecord{
				Name:     "mail",
				Type:     "MX",
				Data:     "mail.example.com.",
				Ttl:      3600,
				Priority: func() *uint16 { v := uint16(10); return &v }(),
			},
		},
	}

	// Set up provider with mock client
	provider := &Provider{}
	provider.client = mockClient

	// Create affectedRecordIDs map with IDs we want to retrieve
	affectedRecordIDs := map[int]struct{}{
		1001: {},
	}

	// Call getRecordsById
	ctx := context.Background()
	zone := "example.com."
	records, err := provider.getRecordsById(ctx, zone, affectedRecordIDs)

	// Check results
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
		return
	}

	record := records[0]
	rr := record.RR()
	if rr.Type != "A" || rr.Name != "www" {
		t.Errorf("expected A record for www, got %s record for %s", rr.Type, rr.Name)
	}

	// Test error case
	mockClient.getDnsRecordsFunc = func(ctx context.Context, zone string) ([]dnsRecordResponse, error) {
		return nil, errors.New("failed to get records")
	}

	_, err = provider.getRecordsById(ctx, zone, affectedRecordIDs)
	if err == nil {
		t.Errorf("expected error but got nil")
	}
}

func TestProvider_SetRecords_ErrorAfterPartialChanges(t *testing.T) {
	// Test that if an error occurs after some changes have been applied,
	// we still return the error

	// Create test records
	existingRecords := []dnsRecordResponse{
		{
			Id: 1001,
			dnsRecord: dnsRecord{
				Name: "www",
				Type: "A",
				Data: "192.0.2.1",
				Ttl:  3600,
			},
		},
		{
			Id: 1002,
			dnsRecord: dnsRecord{
				Name: "www",
				Type: "A",
				Data: "192.0.2.2",
				Ttl:  3600,
			},
		},
	}

	inputRecords := []libdns.Record{
		libdns.Address{
			Name: "www",
			TTL:  7200 * time.Second,
			IP:   netip.MustParseAddr("192.0.2.3"),
		},
	}

	// Set up mock client that succeeds for first update but fails for delete
	mockClient := newMockSimplyClient()
	mockClient.records = existingRecords

	updateCalled := false
	mockClient.updateDnsRecordFunc = func(ctx context.Context, zone string, recordID int, record dnsRecord) error {
		updateCalled = true
		return nil // Succeed for update
	}

	mockClient.deleteDnsRecordFunc = func(ctx context.Context, zone string, recordID int) error {
		return errors.New("failed to delete record") // Fail for delete
	}

	// Create provider with mock client
	provider := &Provider{}
	provider.client = mockClient

	// Execute SetRecords
	ctx := context.Background()
	zone := "example.com."
	_, err := provider.SetRecords(ctx, zone, inputRecords)

	// Check results
	if err == nil {
		t.Errorf("expected error but got nil")
		return
	}

	// Verify that update was called before the error
	if !updateCalled {
		t.Errorf("update should have been called before the error")
	}
}

// TestProvider_GetRecords tests the GetRecords method
func TestProvider_GetRecords(t *testing.T) {
	tests := []struct {
		name            string
		existingRecords []dnsRecordResponse
		errorFunc       func(context.Context, string) ([]dnsRecordResponse, error)
		conversionError bool // If true, simulate conversion error from dnsRecordResponse to libdns.Record
	}{
		{
			name: "Successful retrieval of records",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
				{
					Id: 1002,
					dnsRecord: dnsRecord{
						Name:     "mail",
						Type:     "MX",
						Data:     "mail.example.com.",
						Ttl:      3600,
						Priority: func() *uint16 { v := uint16(10); return &v }(),
					},
				},
			},
		},
		{
			name:            "Empty record set",
			existingRecords: []dnsRecordResponse{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := newMockSimplyClient()
			mockClient.records = tc.existingRecords

			if tc.errorFunc != nil {
				mockClient.getDnsRecordsFunc = tc.errorFunc
			}

			provider := &Provider{}
			provider.client = mockClient

			ctx := context.Background()
			zone := "example.com."
			records, err := provider.GetRecords(ctx, zone)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify record count
			if len(records) != len(tc.existingRecords) {
				t.Errorf("expected %d records, got %d", len(tc.existingRecords), len(records))
				return
			}

			// Verify each record was converted correctly
			for i, record := range records {
				rr := record.RR()
				expected := tc.existingRecords[i]

				if rr.Name != expected.Name {
					t.Errorf("record %d: expected name %s, got %s", i, expected.Name, rr.Name)
				}

				if rr.Type != expected.Type {
					t.Errorf("record %d: expected type %s, got %s", i, expected.Type, rr.Type)
				}

				// Verify TTL
				expectedTTL := time.Duration(expected.Ttl) * time.Second
				if rr.TTL != expectedTTL {
					t.Errorf("record %d: expected TTL %v, got %v", i, expectedTTL, rr.TTL)
				}

				// For MX records, verify preference
				if expected.Type == "MX" {
					mx, ok := record.(libdns.MX)
					if !ok {
						t.Errorf("record %d: expected MX record but got different type", i)
						continue
					}

					if expected.Priority != nil && int(mx.Preference) != int(*expected.Priority) {
						t.Errorf("record %d: expected MX preference %d, got %d",
							i, *expected.Priority, mx.Preference)
					}
				}
			}
		})
	}
}

// TestProvider_AppendRecords tests the AppendRecords method
func TestProvider_AppendRecords(t *testing.T) {
	ipv4 := netip.MustParseAddr("192.0.2.1")

	tests := []struct {
		name              string
		existingRecords   []dnsRecordResponse
		recordsToAdd      []libdns.Record
		expectedAddIDs    []int
		expectedAddedSize int
	}{
		{
			name:            "Add single record",
			existingRecords: []dnsRecordResponse{},
			recordsToAdd: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
			},
			expectedAddIDs:    []int{1000}, // Mock starts with ID 1000
			expectedAddedSize: 1,
		},
		{
			name: "Add multiple records",
			existingRecords: []dnsRecordResponse{
				{
					Id: 999,
					dnsRecord: dnsRecord{
						Name: "existing",
						Type: "A",
						Data: "192.0.2.100",
						Ttl:  3600,
					},
				},
			},
			recordsToAdd: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
				libdns.MX{
					Name:       "mail",
					TTL:        3600 * time.Second,
					Preference: 10,
					Target:     "mail.example.com.",
				},
			},
			expectedAddIDs:    []int{1000, 1001}, // Next IDs after 1000
			expectedAddedSize: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := newMockSimplyClient()
			mockClient.records = tc.existingRecords

			provider := &Provider{}
			provider.client = mockClient

			ctx := context.Background()
			zone := "example.com."
			addedRecords, err := provider.AppendRecords(ctx, zone, tc.recordsToAdd)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify number of records added
			if len(addedRecords) != tc.expectedAddedSize {
				t.Errorf("expected %d added records returned, got %d", tc.expectedAddedSize, len(addedRecords))
			}

			// Verify the records were added to the client
			if len(mockClient.addedRecords) != len(tc.recordsToAdd) {
				t.Errorf("expected %d records to be added to client, got %d",
					len(tc.recordsToAdd), len(mockClient.addedRecords))
			}

			// Verify the returned records match what was requested
			for i, addedRecord := range addedRecords {
				if i >= len(tc.recordsToAdd) {
					t.Errorf("unexpected extra record returned at index %d", i)
					continue
				}

				input := tc.recordsToAdd[i]
				inputRR := input.RR()
				addedRR := addedRecord.RR()

				if addedRR.Name != inputRR.Name {
					t.Errorf("record %d: expected name %s, got %s", i, inputRR.Name, addedRR.Name)
				}

				if addedRR.Type != inputRR.Type {
					t.Errorf("record %d: expected type %s, got %s", i, inputRR.Type, addedRR.Type)
				}

				if addedRR.TTL != inputRR.TTL {
					t.Errorf("record %d: expected TTL %v, got %v", i, inputRR.TTL, addedRR.TTL)
				}
			}
		})
	}
}

// TestProvider_DeleteRecords tests the DeleteRecords method
func TestProvider_DeleteRecords(t *testing.T) {
	ipv4 := netip.MustParseAddr("192.0.2.1")

	tests := []struct {
		name            string
		existingRecords []dnsRecordResponse
		recordsToDelete []libdns.Record
		expectedDeleted int
		errorExpected   bool
	}{
		{
			name: "Delete single record",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
			},
			recordsToDelete: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
			},
			expectedDeleted: 1,
		},
		{
			name: "Delete multiple records",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
				{
					Id: 1002,
					dnsRecord: dnsRecord{
						Name:     "mail",
						Type:     "MX",
						Data:     "mail.example.com.",
						Ttl:      3600,
						Priority: func() *uint16 { v := uint16(10); return &v }(),
					},
				},
			},
			recordsToDelete: []libdns.Record{
				libdns.Address{
					Name: "www",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
				libdns.MX{
					Name:       "mail",
					TTL:        3600 * time.Second,
					Preference: 10,
					Target:     "mail.example.com.",
				},
			},
			expectedDeleted: 2,
		},
		{
			name: "Delete non-existing record",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
			},
			recordsToDelete: []libdns.Record{
				libdns.Address{
					Name: "non-existing",
					TTL:  3600 * time.Second,
					IP:   ipv4,
				},
			},
			expectedDeleted: 0, // No records should be deleted
		},
		{
			name: "Match by name only",
			existingRecords: []dnsRecordResponse{
				{
					Id: 1001,
					dnsRecord: dnsRecord{
						Name: "www",
						Type: "A",
						Data: "192.0.2.1",
						Ttl:  3600,
					},
				},
			},
			recordsToDelete: []libdns.Record{
				// Use a simple wrapper around a basic record with just the name
				libdns.RR{
					Name: "www",
					// Not specifying the type, TTL, etc.
				},
			},
			expectedDeleted: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := newMockSimplyClient()
			mockClient.records = tc.existingRecords

			provider := &Provider{}
			provider.client = mockClient

			ctx := context.Background()
			zone := "example.com."
			deletedRecords, err := provider.DeleteRecords(ctx, zone, tc.recordsToDelete)

			if tc.errorExpected {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify number of records deleted
			if len(deletedRecords) != tc.expectedDeleted {
				t.Errorf("expected %d deleted records returned, got %d", tc.expectedDeleted, len(deletedRecords))
			}

			// For "Match by name only" test, verify special case
			if tc.name == "Match by name only" {
				// Verify that the record was deleted despite type mismatch
				for _, remaining := range mockClient.records {
					if remaining.Name == "www" {
						t.Errorf("expected 'www' record to be deleted regardless of type, but it still exists")
					}
				}
				return // Skip the generic verification below which would fail due to type mismatch
			}

			// Verify the records were removed from the client's records
			for _, recordToDelete := range tc.recordsToDelete {
				inputRR := recordToDelete.RR()
				// Check if this record should have been deleted
				shouldBeDeleted := false

				for _, existing := range tc.existingRecords {
					if isMatching(existing, inputRR) {
						shouldBeDeleted = true
						break
					}
				}

				if shouldBeDeleted {
					// Make sure it was deleted from mockClient.records
					for _, remaining := range mockClient.records {
						if isMatching(remaining, inputRR) {
							t.Errorf("record %s of type %s should have been deleted but still exists",
								inputRR.Name, inputRR.Type)
						}
					}
				}
			}
		})
	}
}

// Helper function for DeleteRecords tests to check if a record matches deletion criteria
func isMatching(record dnsRecordResponse, criteria libdns.RR) bool {
	// Always match on name
	if record.Name != criteria.Name {
		return false
	}

	// Match on type if specified
	if criteria.Type != "" && record.Type != criteria.Type {
		return false
	}

	// For simplicity in tests, we don't check data and TTL in this helper
	return true
}
