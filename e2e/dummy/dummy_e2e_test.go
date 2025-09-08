package dummy_test

import (
	"context"
	"testing"

	"github.com/libdns/libdns"
	"github.com/libdns/libdns/e2e"
	"github.com/libdns/libdns/e2e/dummy"
)

func TestDummyProvider(t *testing.T) {
	provider := dummy.New("example.com.")
	testSuite := e2e.NewTestSuite(provider, "example.com.")
	testSuite.RunTests(t)
}

type noZoneListerProvider struct {
	provider *dummy.Provider
}

func TestDummyProviderNoZoneLister(t *testing.T) {
	// let's pretent we have provider that does not implement ZoneListener
	provider := &noZoneListerProvider{provider: dummy.New("example.com.")}

	// this how how we test it:
	suite := e2e.NewTestSuite(e2e.WrapNoZoneLister(provider), "example.com.")
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
