# libdns End-to-End Testing

This package provides reusable end-to-end testing utilities for libdns provider implementations.

## Provider Types

Most libdns providers implement basic record operations but not `ZoneLister`. Choose the appropriate interface:

- **RecordProvider**: Basic DNS record operations (most common)
- **FullProvider**: Complete interface including `ZoneLister`

## Usage

```go
import "github.com/libdns/libdns/e2e"

// Most providers (without ZoneLister)
suite := e2e.NewRecordTestSuite(provider, "test-zone.com.")
suite.RunRecordTests(t)

// Full providers (with ZoneLister)
suite := e2e.NewFullTestSuite(provider, "test-zone.com.")
suite.RunFullTests(t) // runs ListZones + all record tests
```

## Custom Record Types

Providers may have custom record implementations with additional fields:

```go
type MyRecord struct {
    libdns.RR
    Extra string `json:"extra"` // Provider-specific field
}

func (r MyRecord) RR() libdns.RR { return r.RR }

// Configure custom record constructor
suite := e2e.NewRecordTestSuite(provider, "test-zone.com.")
suite.AppendRecordFunc = func(rr libdns.RR) libdns.Record {
    return MyRecord{
        RR:    rr,
        Extra: "pretty please", // Provider-specific data
    }
}
suite.RunRecordTests(t)
```

## Test Coverage

- **ListZones**: Lists available zones (FullProvider only)
- **GetRecords**: Retrieves records from a zone  
- **AppendRecords**: Creates new records (uses `test-append*` names)
- **SetRecords**: Creates/updates/deletes records by (name,type), preserves unrelated records (uses `test-set*` names)
- **DeleteRecords**: Creates then deletes records (uses `test-delete*` names) 

For **real DNS providers**, use dedicated test zones since tests create/modify/delete DNS records.

## Dummy Provider

```go
import "github.com/libdns/libdns/e2e/dummy"

provider := dummy.New("example.com.")  
records, err := provider.GetRecords(ctx, "example.com.")
```

The dummy provider implements all libdns interfaces using in-memory storage. It's used to test the e2e framework itself and as a reference implementation.

## Zone Cleanup

The test suite provides a cleanup method for removing test records:

```go
suite := e2e.NewRecordTestSuite(provider, "test-zone.com.")

// Clean up test records after running tests
err := suite.AttemptZoneCleanup()
if err != nil {
    t.Logf("Cleanup warning: %v", err)
}
```

**Note**: `AttemptZoneCleanup()` deletes all DNS records with names starting with "test-" from the zone. This includes any existing records that begin with "test-", not just records created by the e2e tests. Use dedicated test zones when working with real DNS providers.