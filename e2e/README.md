# libdns End-to-End Testing

This package provides reusable end-to-end testing utilities for libdns provider implementations.

## Usage

```go
import "github.com/libdns/libdns/e2e"

// Test your provider implementation
provider := YourProvider{...}
testSuite := e2e.NewTestSuite(provider, "your-test-zone.com.")
testSuite.RunAllTests(t)
```

For **real DNS provider implementations**, use dedicated test zones since the tests will create/modify/delete DNS records with names like `test-append`, `test-set`, etc.

## Test Coverage

- **ListZones**: Lists available zones
- **GetRecords**: Retrieves records from a zone  
- **AppendRecords**: Creates new records (uses `test-append*` names)
- **SetRecords**: Updates records by (name,type), preserves unrelated records (uses `test-set*` names)
- **DeleteRecords**: Creates then deletes records (uses `test-delete*` names)
- **RecordLifecycle**: Complete create → update → delete workflow (uses `test-lifecycle` names) 

## Dummy Provider

```go
import "github.com/libdns/libdns/e2e/dummy"

provider := dummy.New("example.com.")  
records, err := provider.GetRecords(ctx, "example.com.")
```

The dummy provider implements all libdns interfaces using in-memory storage. It's used to test the e2e framework itself and as a reference implementation.

## Running Tests  

```bash
go test ./e2e/dummy
```