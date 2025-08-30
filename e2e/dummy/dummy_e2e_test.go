package dummy_test

import (
	"testing"

	"github.com/libdns/libdns/e2e"
	"github.com/libdns/libdns/e2e/dummy"
)

func TestDummyProvider(t *testing.T) {
	provider := dummy.New("example.com.")
	testSuite := e2e.NewFullTestSuite(provider, "example.com.")
	testSuite.RunFullTests(t)
}

func TestDummyProviderMultipleZones(t *testing.T) {
	zones := []string{"zone1.com.", "zone2.net.", "zone3.org."}
	provider := dummy.New(zones...)

	for _, zone := range zones {
		t.Run("Zone_"+zone, func(t *testing.T) {
			testSuite := e2e.NewFullTestSuite(provider, zone)
			testSuite.RunFullTests(t)
		})
	}
}
