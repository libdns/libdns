package testing

import (
	"context"
	"testing"

	"github.com/libdns/infomaniak"
	"github.com/libdns/libdns"
)

// Put your API token here - do not forget to remove it before committing!
const apiToken = "<YOUR_TOKEN>"

// use a subdomain that you normally don't use e.g. test.<your_domain> to prevent that your actual dns records are changed
const zone = "<YOUR_(SUB)_DOMAIN>"

// Provider used for integration test
var provider = infomaniak.Provider{APIToken: apiToken}

// contains the the created test records to clean up after the test
var testRecords = make([]libdns.Record, 0)

// cleanup ensures that all created records are removed after each test
func cleanup() {
	provider.DeleteRecords(context.TODO(), zone, testRecords)
	testRecords = make([]libdns.Record, 0)
}

// appendRecord calls provider, handles error and ensures that the appended records will be deleted at the end of the test
func appendRecord(t *testing.T, rec libdns.Record) []libdns.Record {
	appendedRecords, err := provider.AppendRecords(context.TODO(), zone, []libdns.Record{rec})
	if err != nil {
		t.Fatal(err)
	}
	testRecords = append(testRecords, appendedRecords...)
	return appendedRecords
}

// setRecord calls provider, handles error and ensures that the set records will be deleted at the end of the test
func setRecord(t *testing.T, rec libdns.Record) []libdns.Record {
	setRecords, err := provider.SetRecords(context.TODO(), zone, []libdns.Record{rec})
	if err != nil {
		t.Fatal(err)
	}
	testRecords = append(testRecords, setRecords...)
	return setRecords
}

// deleteRecord calls provider, handles error and ensures that the deleted records will not be deleted again at the end of the test
func deleteRecord(t *testing.T, rec libdns.Record) []libdns.Record {
	deletedRecs, err := provider.DeleteRecords(context.TODO(), zone, []libdns.Record{rec})
	if err != nil {
		t.Fatal(err)
	}

	indexOfDeletedRec := -1
	for i, testRec := range testRecords {
		if testRec == rec {
			indexOfDeletedRec = i
			break
		}
	}

	if len(testRecords) <= 1 {
		testRecords = make([]libdns.Record, 0)
	} else if indexOfDeletedRec > -1 {
		testRecords[indexOfDeletedRec] = testRecords[len(testRecords)-1]
		testRecords = testRecords[:len(testRecords)-1]
	}
	return deletedRecs
}

// getRecords calls provider, handles error and returns records that exist for zone
func getRecords(t *testing.T, zone string) []libdns.Record {
	result, err := provider.GetRecords(context.TODO(), zone)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// aTestRecord returns a record that can be used for testing purposes
func aTestRecord(name string, value string) libdns.Record {
	return libdns.Record{
		Type:     "MX",
		Name:     libdns.RelativeName(name, zone),
		Value:    value,
		TTL:      3600,
		Priority: 6,
	}
}

// assertExists ensures that a record exists based on it's name, type and value
func assertExists(t *testing.T, record libdns.Record) {
	if !isRecordExisting(t, record) {
		t.Fatalf("Expected for record %#v to exist, but it does not", record)
	}
}

// assertNotExists ensures that a record with the given name, type and value does not exist
func assertNotExists(t *testing.T, record libdns.Record) {
	if isRecordExisting(t, record) {
		t.Fatalf("Expected for record %#v to not exist, but it does", record)
	}
}

// isRecordExisting tests that a record with given name, type and value exists or not exists (based on shouldExist parameter)
func isRecordExisting(t *testing.T, record libdns.Record) bool {
	existingRecs := getRecords(t, zone)
	for _, existingRec := range existingRecs {
		if existingRec.Name == record.Name && existingRec.Type == record.Type && existingRec.Value == record.Value {
			return true
		}
	}
	return false
}

func Test_DeleteRecords_DeletesRecordById(t *testing.T) {
	defer cleanup()

	recToDelete := aTestRecord(zone, "127.0.0.1")
	recToDelete = setRecord(t, recToDelete)[0]
	deleteRecord(t, recToDelete)

	assertNotExists(t, recToDelete)
}

func Test_DeleteRecords_DeletesRecordByNameAndType(t *testing.T) {
	defer cleanup()

	recToDeleteWithoutId := aTestRecord(zone, "127.0.0.1")
	setRecord(t, recToDeleteWithoutId)

	recToDeleteWithoutId.ID = ""
	deleteRecord(t, recToDeleteWithoutId)

	assertNotExists(t, recToDeleteWithoutId)
}

func Test_AppendRecords_AppendsNewRecord(t *testing.T) {
	defer cleanup()

	recToAppend := aTestRecord(zone, "127.0.0.1")
	appendedRecords := appendRecord(t, recToAppend)
	if len(appendedRecords) != 1 {
		t.Fatalf("Expected 1 record appended, got %d", len(appendedRecords))
	}

	appendedRecord := appendedRecords[0]

	if appendedRecord.ID == "" {
		t.Fatalf("Expected an ID for newly appended record")
	}
	assertExists(t, recToAppend)
}

func Test_AppendRecords_DoesNotOverwriteExistingRecordWithSameNameAndType(t *testing.T) {
	defer cleanup()

	originalRecord := aTestRecord(zone, "127.0.0.1")
	appendRecord(t, originalRecord)

	recThatShouldNotOverwriteFirst := originalRecord
	recThatShouldNotOverwriteFirst.Value = "127.0.0.0"
	addedRecords := appendRecord(t, recThatShouldNotOverwriteFirst)

	if len(addedRecords) > 0 {
		t.Fatalf("Expected that already existing record is not overwritten but it was")
	}

	assertExists(t, originalRecord)
	assertNotExists(t, recThatShouldNotOverwriteFirst)
}

func Test_SetRecords_CreatesNewRecord(t *testing.T) {
	defer cleanup()

	recToCreate := aTestRecord(zone, "127.0.0.1")
	setRecs := setRecord(t, recToCreate)

	if len(setRecs) != 1 {
		t.Fatalf("Expected 1 record updated, got %d", len(setRecs))
	}

	createdRec := setRecs[0]
	if createdRec.ID == "" {
		t.Fatalf("No ID set for newly created record")
	}

	assertExists(t, recToCreate)
}

func Test_SetRecords_UpdatesRecordById(t *testing.T) {
	defer cleanup()

	recToUpdate := aTestRecord(zone, "127.0.0.1")
	recToUpdate = setRecord(t, recToUpdate)[0]

	updatedRec := recToUpdate
	updatedRec.Value = "127.0.0.0"
	result := setRecord(t, updatedRec)

	if len(result) != 1 {
		t.Fatalf("Expected 1 record updated, got %d", len(result))
	}

	assertNotExists(t, recToUpdate)
	assertExists(t, updatedRec)
}

func Test_SetRecords_OverwritesExistingRecordWithSameNameAndType(t *testing.T) {
	defer cleanup()

	recToUpdate := aTestRecord(zone, "127.0.0.1")
	recToUpdate = setRecord(t, recToUpdate)[0]

	updatedRec := recToUpdate
	updatedRec.ID = ""
	updatedRec.Value = "127.0.0.0"
	result := setRecord(t, updatedRec)

	if len(result) != 1 {
		t.Fatalf("Expected 1 record updated, got %d", len(result))
	}
	if recToUpdate.ID != result[0].ID {
		t.Fatalf("Expected record with ID %s back, but got %s", recToUpdate.ID, result[0].ID)
	}

	assertNotExists(t, recToUpdate)
	assertExists(t, updatedRec)
}

func Test_GetRecords_DoesNotReturnRecordsOfParentZone(t *testing.T) {
	defer cleanup()

	setRecord(t, aTestRecord(zone, "127.0.0.1"))
	result := getRecords(t, "subzone."+zone)
	if len(result) > 0 {
		t.Fatalf("Expected 0 records, got %d", len(result))
	}
}

func Test_GetRecords_ReturnsRecordOfChildZone(t *testing.T) {
	defer cleanup()

	setRecord(t, aTestRecord("subzone."+zone, "127.0.0.1"))
	result := getRecords(t, zone)
	if len(result) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(result))
	}
}
