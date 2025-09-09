# libdns Testing Framework

This package provides reusable testing utilities for libdns provider implementations.

## Usage

```go
import "github.com/libdns/libdns/libdnstest"

suite := libdnstest.NewTestSuite(provider, "example.com.")
suite.RunTests(t)
```

The TestSuite provides a `SkipRRTypes` map to exclude specific record types from testing:

```go
suite := libdnstest.NewTestSuite(provider, "example.com.")
suite.SkipRRTypes = map[string]bool{
    "MX":    true,  // Skip MX record tests
    "SRV":   true,  // Skip SRV record tests
    "CAA":   true,  // Skip CAA record tests
    "NS":    true,  // Skip NS record tests
    "SVCB":  true,  // Skip SVCB record tests
    "HTTPS": true,  // Skip HTTPS record tests
}
suite.RunTests(t)
```

Note: Essential record types (A, CNAME, TXT) cannot be skipped as they are used by the testing framework itself.

## Providers Without ZoneLister

If your provider doesn't implement `ZoneLister`, use the `WrapNoZoneLister` helper:

```go
provider := YourProvider{...} // implements RecordGetter, RecordAppender, RecordSetter, RecordDeleter
wrappedProvider := libdnstest.WrapNoZoneLister(provider)
suite := libdnstest.NewTestSuite(wrappedProvider, "example.com.")
suite.RunTests(t) // ListZones test will be skipped automatically
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
suite := libdnstest.NewTestSuite(provider, "example.com.")
suite.AppendRecordFunc = func(record libdns.Record) libdns.Record {
    return MyRecord{
        RR:    record.RR(),
        Extra: "pretty please", // Provider-specific data
    }
}
suite.RunTests(t)
```

## Test Coverage

<dl>
<dt>ListZones</dt>
<dd>Lists available zones (requires ZoneLister interface)</dd>
<dt>GetRecords</dt>
<dd>Retrieves records from a zone</dd>
<dt>AppendRecords</dt>
<dd>Creates new records (uses <code>test-append*</code> names)</dd>
<dt>SetRecords</dt>
<dd>Creates/updates/deletes records by (name,type), preserves unrelated records (uses <code>test-set*</code> names)</dd>
<dt>DeleteRecords</dt>
<dd>Creates then deletes records (uses <code>test-delete*</code> names)</dd>
</dl>

> [!WARNING]
> When testing **real DNS providers** run the tests on dedicated test zones. **Your DNS records may be deleted or overwritten.** Even though tests use "test-" prefixed record names, bugs in the provider or test framework could cause additional data loss.
> Copy this note to README file of specific providers tests.

### Zone Cleanup

The test suite automatically cleans up test records using `AttemptZoneCleanup()` before all tests and after each individual test. This method deletes all DNS records with names starting with "test-" from the zone.

**Use dedicated test zones when working with real DNS providers.**

## Example Provider

```go
import "github.com/libdns/libdns/libdnstest/example"

provider := example.New("example.com.")
records, err := provider.GetRecords(ctx, "example.com.")
```

The example provider implements all libdns interfaces using in-memory storage. It serves as a double-entry system to ensure there is some implementation that can pass these tests. The example provider does not guarantee DNS compliance, but works for the currently defined tests.