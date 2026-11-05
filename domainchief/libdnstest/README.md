# libdns test suite

Runs the shared [libdns test suite](https://github.com/libdns/libdns/tree/master/libdnstest) against the real Domain Chief API. It is a separate module so the provider itself only depends on `libdns`.

The suite creates and deletes `test-*` records in the test zone. Use a zone without important records and a token with the `domainchief:dns:read`, `domainchief:dns:write` and `domainchief:domains:read` scopes:

```bash
DOMAINCHIEF_API_TOKEN=ctt_... DOMAINCHIEF_TEST_ZONE=example.com. go test -v ./...
```

Domain Chief does not support SVCB and HTTPS records, so those are skipped. Domain Chief only stores a fixed set of TTLs and rounds other values to the closest one, so the suite's `SetRecords` step that updates records to a TTL of 600 seconds reads them back with a TTL of 300 seconds.
