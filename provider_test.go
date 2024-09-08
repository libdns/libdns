package selectelv2_test

// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"testing"
// 	"time"

// 	"github.com/libdns/selectelv2"
// 	"github.com/joho/godotenv"
// 	"github.com/libdns/libdns"
// 	"github.com/stretchr/testify/assert"
// )

// var provider selectelv2.Provider
// var zone string
// var ctx context.Context

// var addedRecords []libdns.Record
// var sourceRecords []libdns.Record

// // load init data from .env
// func setup() {
// 	err := godotenv.Load(".env")
// 	if err != nil {
// 		panic("Error loading .env file")
// 	}

// 	provider = selectelv2.Provider{
// 		User:        os.Getenv("SELECTEL_USER"),
// 		Password:    os.Getenv("SELECTEL_PASSWORD"),
// 		AccountId:   os.Getenv("SELECTEL_ACCOUNT_ID"),
// 		ProjectName: os.Getenv("SELECTEL_PROJECT_NAME"),
// 		ZonesCache:  make(map[string]string),
// 	}
// 	zone = os.Getenv("SELECTEL_ZONE")
// 	ctx = context.Background()
// 	sourceRecords = []libdns.Record{
// 		{ // 0
// 			Type:  "A",
// 			Name:  fmt.Sprintf("test1.%s.", os.Getenv("SELECTEL_ZONE")),
// 			Value: "1.2.3.1",
// 			TTL:   61 * time.Second,
// 		},
// 		{ // 1
// 			Type:  "A",
// 			Name:  fmt.Sprintf("test2.%s.", os.Getenv("SELECTEL_ZONE")),
// 			Value: "1.2.3.2",
// 			TTL:   61 * time.Second,
// 		},
// 		{ // 2
// 			Type:  "A",
// 			Name:  "test3",
// 			Value: "1.2.3.3",
// 			TTL:   61 * time.Second,
// 		},
// 		{ // 3
// 			Type: "TXT",
// 			Name: "test1",
// 			Value: "test1 txt",
// 			TTL: 61 * time.Second,
// 		},
// 		{ // 4
// 			Type: "TXT",
// 			Name: fmt.Sprintf("test2.%s.", os.Getenv("SELECTEL_ZONE")),
// 			Value: "test2 txt",
// 			TTL: 61 * time.Second,
// 		},
// 		{ // 5
// 			Type: "TXT",
// 			Name: "test3",
// 			Value: "test3 txt",
// 			TTL: 61 * time.Second,
// 		},
// 	}
// }

// // testing GetRecord
// func TestProvider_GetRecords(t *testing.T) {
// 	setup()

// 	// delete sourceRec if exists
// 	provider.DeleteRecords(ctx, zone, sourceRecords)

// 	records, err := provider.GetRecords(ctx, zone)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, records)
// 	assert.True(t, len(records) > 0, "No records found")
// 	t.Logf("GetRecords test passed. Records found: %d", len(records))
// }

// // testing append record
// func TestProvider_AppendRecords(t *testing.T) {
// 	setup()
// 	// entries to add
// 	newRecords := []libdns.Record{
// 		sourceRecords[0],
// 		sourceRecords[1],
// 		sourceRecords[3],
// 		sourceRecords[4],
// 	}

// 	records, err := provider.AppendRecords(ctx, zone, newRecords)
// 	addedRecords = records
// 	assert.NoError(t, err)
// 	assert.NotNil(t, records)
// 	assert.Equal(t, 4, len(records))
// 	assert.Equal(t, "A", records[0].Type)
// 	assert.Equal(t, "TXT", records[2].Type)
// 	t.Logf("AppendRecords test passed. Append count: %d", len(records))
// }

// // testing set
// func TestProvider_SetRecords(t *testing.T) {
// 	setup()

// 	second := addedRecords[1]
// 	second.TTL = 62 * time.Second

// 	fourth := addedRecords[3]
// 	fourth.Value = "test 1 txt with additional line\nsecondline"

// 	fifth := sourceRecords[4]
// 	fifth.Value = "test 2 txt changed"

// 	// entries to set
// 	setRecords := []libdns.Record{
// 		{ // record from Append without id
// 			Type:  "A",
// 			Name:  "test1.", // <---- without zone, but with .
// 			Value: "1.2.3.1",
// 			TTL:   62 * time.Second, // <---- changed
// 		},
// 		second, // record from Append, but new ttl = 62
// 		sourceRecords[2], // new record
// 		fourth, // changed value. 2 lines
// 		fifth,
// 		sourceRecords[5],
// 	}

// 	records, err := provider.SetRecords(ctx, zone, setRecords)
// 	addedRecords = records
// 	assert.NoError(t, err)
// 	assert.NotNil(t, records)
// 	assert.Equal(t, 6, len(records))
// 	assert.Equal(t, "A", records[2].Type)
// 	assert.Equal(t, "1.2.3.2", records[1].Value)
// 	assert.Equal(t, 62, int(records[0].TTL.Seconds()))
// 	t.Logf("SetRecords test passed. Set count: %d", len(records))
// }

// // testing delete
// func TestProvider_DeleteRecords(t *testing.T) {
// 	setup()

// 	// entries to delete
// 	delRecords := []libdns.Record{
// 		addedRecords[0],
// 		sourceRecords[1],
// 		addedRecords[2],
// 		sourceRecords[3],
// 		addedRecords[4],
// 		sourceRecords[5],
// 	}

// 	records, err := provider.DeleteRecords(ctx, zone, delRecords)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, records)
// 	assert.Equal(t, 6, len(records))
// 	assert.Equal(t, "A", records[0].Type)
// 	assert.Equal(t, "1.2.3.2", records[1].Value)
// 	assert.Equal(t, 61, int(records[2].TTL.Seconds()))
// 	t.Logf("DeleteRecords test passed. Delete count: %d", len(records))
// }
