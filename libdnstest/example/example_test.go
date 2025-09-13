package example_test

import (
	"context"
	"testing"

	"github.com/libdns/libdns"
	"github.com/libdns/libdns/libdnstest"
	"github.com/libdns/libdns/libdnstest/example"
)

func TestExampleProvider(t *testing.T) {
	provider := example.New("example.com.")
	testSuite := libdnstest.NewTestSuite(provider, "example.com.")
	testSuite.RunTests(t)
}

func TestExampleProviderWithSkippedRecords(t *testing.T) {
	provider := example.New("example.com.")
	testSuite := libdnstest.NewTestSuite(provider, "example.com.")

	// Example: skip some types that some provider may not implement
	testSuite.SkipRRTypes = map[string]bool{
		"MX":    true,
		"SRV":   true,
		"HTTPS": true,
		"SVCB":  true,
		"CAA":   true,
		"NS":    true,
		"AAAA":  true,
	}

	testSuite.RunTests(t)
}

type noZoneListerProvider struct {
	provider *example.Provider
}

func TestExampleProviderNoZoneLister(t *testing.T) {
	// let's pretend we have provider that does not implement ZoneLister
	provider := &noZoneListerProvider{provider: example.New("example.com.")}

	// this is how we test it:
	suite := libdnstest.NewTestSuite(libdnstest.WrapNoZoneLister(provider), "example.com.")
	suite.RunTests(t)
}

func (w *noZoneListerProvider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	return w.provider.GetRecords(ctx, zone)
}

func (w *noZoneListerProvider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return w.provider.AppendRecords(ctx, zone, records)
}

func (w *noZoneListerProvider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return w.provider.SetRecords(ctx, zone, records)
}

func (w *noZoneListerProvider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return w.provider.DeleteRecords(ctx, zone, records)
}

func TestExampleProviderStrict(t *testing.T) {
	provider := example.New("example.com.")
	testSuite := libdnstest.NewTestSuite(provider, "example.com.")
	// expect zone to have no records besides SOA and NS
	// otherwise tests will fail (strict mode)
	testSuite.ExpectEmptyZone = true
	testSuite.RunTests(t)
}
