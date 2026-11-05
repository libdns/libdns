package main

import (
	"os"
	"strings"
	"testing"

	"github.com/libdns/domainchief"
	"github.com/libdns/libdns/libdnstest"
)

// TestDomainChiefProvider runs the shared libdns test suite against the real Domain Chief API.
//
// It changes records in the test zone and needs a token with the `domainchief:dns:read`,
// `domainchief:dns:write` and `domainchief:domains:read` scopes.
func TestDomainChiefProvider(t *testing.T) {
	apiToken := os.Getenv("DOMAINCHIEF_API_TOKEN")
	testZone := os.Getenv("DOMAINCHIEF_TEST_ZONE")

	if apiToken == "" || testZone == "" {
		t.Skip("Skipping Domain Chief provider tests: DOMAINCHIEF_API_TOKEN and/or DOMAINCHIEF_TEST_ZONE environment variables must be set")
	}

	if !strings.HasSuffix(testZone, ".") {
		t.Fatal("We expect the test zone to have a trailing dot")
	}

	provider := &domainchief.Provider{
		APIToken: apiToken,
		BaseURL:  os.Getenv("DOMAINCHIEF_API_BASE_URL"),
	}

	suite := libdnstest.NewTestSuite(provider, testZone)
	// Domain Chief does not support SVCB and HTTPS records.
	suite.SkipRRTypes = map[string]bool{
		"SVCB":  true,
		"HTTPS": true,
	}
	suite.RunTests(t)
}
