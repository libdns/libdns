package infomaniak

import (
	"context"
	"strconv"
	"testing"

	"github.com/libdns/libdns"
)

// TestClient instance of IkClient used to mock API calls
type TestClient struct {
	getter  func(ctx context.Context, zone string) ([]IkRecord, error)
	setter  func(ctx context.Context, zone string, record IkRecord) (*IkRecord, error)
	deleter func(ctx context.Context, zone string, id string) error
}

// GetDnsRecordsForZone implementation to fulfill IkClient interface
func (c *TestClient) GetDnsRecordsForZone(ctx context.Context, zone string) ([]IkRecord, error) {
	return c.getter(ctx, zone)
}

// CreateOrUpdateRecord implementation to fulfill IkClient interface
func (c *TestClient) CreateOrUpdateRecord(ctx context.Context, zone string, record IkRecord) (*IkRecord, error) {
	return c.setter(ctx, zone, record)
}

// DeleteRecord implementation to fulfill IkClient interface
func (c *TestClient) DeleteRecord(ctx context.Context, zone string, id string) error {
	return c.deleter(ctx, zone, id)
}

// assertEquals helper function that throws an error if the actual string value is not the expected value
func assertEquals(t *testing.T, name string, expected string, actual string) {
	if expected != actual {
		t.Fatalf("Expected %s=%s, got %s=%s", name, expected, name, actual)
	}
}

func Test_GetRecords_ReturnsRecords(t *testing.T) {
	subDomain := "subdomain"
	expectedRec := IkRecord{ID: "1893", Type: "AAAA", SourceIdn: subDomain + ".example.com", Target: "ns11.infomaniak.ch", TtlInSec: 301, Priority: 15}
	client := TestClient{getter: func(ctx context.Context, zone string) ([]IkRecord, error) { return []IkRecord{expectedRec}, nil }}
	provider := Provider{client: &client}
	result, _ := provider.GetRecords(nil, "example.com")

	if len(result) != 1 {
		t.Fatalf("Expected %d records, got %d", 1, len(result))
	}

	actualRec := result[0]
	assertEquals(t, "ID", expectedRec.ID, actualRec.ID)
	assertEquals(t, "Name", subDomain, actualRec.Name)
	assertEquals(t, "TTL", strconv.FormatInt(int64(expectedRec.TtlInSec), 10), strconv.FormatInt(int64(actualRec.TTL), 10))
	assertEquals(t, "Type", expectedRec.Type, actualRec.Type)
	assertEquals(t, "Value", expectedRec.Target, actualRec.Value)
	assertEquals(t, "Priority", strconv.FormatInt(int64(expectedRec.Priority), 10), strconv.FormatInt(int64(actualRec.Priority), 10))
}

func Test_AppendRecords_DoesNotAppendRecordWithId(t *testing.T) {
	client := TestClient{
		setter: func(ctx context.Context, zone string, record IkRecord) (*IkRecord, error) {
			t.Fatalf("Expected that append is not called for record with ID")
			return nil, nil
		},
	}
	provider := Provider{client: &client}
	appendedRec, err := provider.AppendRecords(nil, "", []libdns.Record{{ID: "1893"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(appendedRec) > 0 {
		t.Fatalf("Expected 0 appended records, got %d", len(appendedRec))
	}
}

func Test_AppendRecords_AppendsNotExistingRecord(t *testing.T) {
	id := "12345"
	methodCalled := false
	client := TestClient{
		getter: func(ctx context.Context, zone string) ([]IkRecord, error) { return []IkRecord{}, nil },
		setter: func(ctx context.Context, zone string, record IkRecord) (*IkRecord, error) {
			if methodCalled {
				t.Fatalf("Expected append method to be only called once")
			} else {
				methodCalled = true
			}
			return &IkRecord{ID: id}, nil
		},
	}
	provider := Provider{client: &client}
	appendedRec, err := provider.AppendRecords(nil, "", []libdns.Record{{}})
	if err != nil {
		t.Fatal(err)
	}
	if !methodCalled {
		t.Fatalf("Expected record to be appended but was not")
	}
	if len(appendedRec) != 1 {
		t.Fatalf("Expected 1 appended record, got %d", len(appendedRec))
	}
	if appendedRec[0].ID != id {
		t.Fatalf("Expected appended record to be updated with ID %s, ID was %s", id, appendedRec[0].ID)
	}
}

func Test_AppendRecords_DoesNotAppendAlreadyExistingRecord(t *testing.T) {
	recType := "AAAA"
	name := "Test1"
	client := TestClient{
		getter: func(ctx context.Context, zone string) ([]IkRecord, error) {
			return []IkRecord{{SourceIdn: name, Type: recType}}, nil
		},
		setter: func(ctx context.Context, zone string, record IkRecord) (*IkRecord, error) {
			t.Fatalf("Expected that append is not called for already existing record")
			return nil, nil
		},
	}
	provider := Provider{client: &client}
	appendedRec, err := provider.AppendRecords(nil, "", []libdns.Record{{ID: "222", Type: recType, Name: name}})
	if err != nil {
		t.Fatal(err)
	}
	if len(appendedRec) > 0 {
		t.Fatalf("Expected 0 appended records, got %d", len(appendedRec))
	}
}

func Test_SetRecords_CreatesNewRecord(t *testing.T) {
	methodCalled := false
	client := TestClient{
		getter: func(ctx context.Context, zone string) ([]IkRecord, error) { return []IkRecord{}, nil },
		setter: func(ctx context.Context, zone string, record IkRecord) (*IkRecord, error) {
			if methodCalled {
				t.Fatalf("Expected set method to be only called once")
			} else {
				methodCalled = true
			}
			return &IkRecord{}, nil
		},
	}
	provider := Provider{client: &client}
	setRec, err := provider.SetRecords(nil, "", []libdns.Record{{}})
	if err != nil {
		t.Fatal(err)
	}
	if !methodCalled {
		t.Fatalf("Expected record to be appended but was not")
	}
	if len(setRec) != 1 {
		t.Fatalf("Expected 1 set record, got %d", len(setRec))
	}
}

func Test_SetRecords_UpdatesExistingRecordById(t *testing.T) {
	id := "789"
	methodCalled := false
	client := TestClient{
		setter: func(ctx context.Context, zone string, record IkRecord) (*IkRecord, error) {
			if methodCalled {
				t.Fatalf("Expected set method to be only called once")
			} else if record.ID == id {
				methodCalled = true
			}
			return &IkRecord{ID: id}, nil
		},
	}
	provider := Provider{client: &client}
	setRec, err := provider.SetRecords(nil, "", []libdns.Record{{ID: id}})
	if err != nil {
		t.Fatal(err)
	}
	if !methodCalled {
		t.Fatalf("Expected record to be update but was not")
	}
	if len(setRec) != 1 {
		t.Fatalf("Expected 1 set record, got %d", len(setRec))
	}
}

func Test_SetRecords_UpdatesExistingRecordByNameAndTypeIfNoIdProvided(t *testing.T) {
	id := "2247"
	recType := "MX"
	methodCalled := false
	client := TestClient{
		getter: func(ctx context.Context, zone string) ([]IkRecord, error) {
			return []IkRecord{{ID: id, Type: recType}}, nil
		},
		setter: func(ctx context.Context, zone string, record IkRecord) (*IkRecord, error) {
			if methodCalled {
				t.Fatalf("Expected set method to be only called once")
			} else if record.ID == id {
				methodCalled = true
			}
			return &IkRecord{ID: id}, nil
		},
	}
	provider := Provider{client: &client}
	setRec, err := provider.SetRecords(nil, "", []libdns.Record{{Type: recType}})
	if err != nil {
		t.Fatal(err)
	}
	if !methodCalled {
		t.Fatalf("Expected record to be update but was not")
	}
	if len(setRec) != 1 {
		t.Fatalf("Expected 1 set record, got %d", len(setRec))
	}
	if setRec[0].ID != id {
		t.Fatalf("Expected returned record to be updated with ID %s, ID was %s", id, setRec[0].ID)
	}
}

func Test_DeleteRecords_DoesNotDeleteRecordWithoutIdWhoseNameAndTypeDoesNotMatchWithAnyExistingRecord(t *testing.T) {
	client := TestClient{
		getter: func(ctx context.Context, zone string) ([]IkRecord, error) { return []IkRecord{}, nil },
		deleter: func(ctx context.Context, zone string, id string) error {
			t.Fatalf("Expected that delete is not called for record without ID")
			return nil
		},
	}
	provider := Provider{client: &client}
	deletedRecs, err := provider.DeleteRecords(nil, "", []libdns.Record{{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(deletedRecs) > 0 {
		t.Fatalf("Expected 0 deleted records, got %d", len(deletedRecs))
	}
}

func Test_Delete_Records_DeletesRecordWithSameNameAndTypeIfGivenRecordHasNoId(t *testing.T) {
	subDomain := "sub"
	recId := "23"
	recType := "A"

	methodCalled := false
	client := TestClient{
		getter: func(ctx context.Context, zone string) ([]IkRecord, error) {
			return []IkRecord{{ID: recId, SourceIdn: subDomain + "example.com", Type: recType}}, nil
		},
		deleter: func(ctx context.Context, zone string, id string) error {
			if methodCalled {
				t.Fatalf("Expected delete method to be only called once")
			} else if recId == id {
				methodCalled = true
			}
			return nil
		},
	}
	provider := Provider{client: &client}
	deletedRecs, err := provider.DeleteRecords(nil, "example.com", []libdns.Record{{Type: recType, Name: subDomain}})
	if err != nil {
		t.Fatal(err)
	}
	if !methodCalled {
		t.Fatalf("Expected record to be deleted but was not")
	}
	if len(deletedRecs) != 1 {
		t.Fatalf("Expected 1 deleted record, got %d", len(deletedRecs))
	}
	if deletedRecs[0].ID != recId {
		t.Fatalf("Expected that record with id %s is deleted, got id %s", recId, deletedRecs[0].ID)
	}
}

func Test_DeleteRecords_DeletesRecordWithId(t *testing.T) {
	methodCalled := false
	rec := libdns.Record{ID: "5557"}
	client := TestClient{
		deleter: func(ctx context.Context, zone string, id string) error {
			if methodCalled {
				t.Fatalf("Expected delete method to be only called once")
			} else if rec.ID == id {
				methodCalled = true
			}
			return nil
		},
	}
	provider := Provider{client: &client}
	deletedRecs, err := provider.DeleteRecords(nil, "", []libdns.Record{rec})
	if err != nil {
		t.Fatal(err)
	}
	if !methodCalled {
		t.Fatalf("Expected record to be deleted but was not")
	}
	if len(deletedRecs) != 1 {
		t.Fatalf("Expected 1 deleted record, got %d", len(deletedRecs))
	}
}
