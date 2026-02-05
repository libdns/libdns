package main

import (
	"os"
	"testing"

	"github.com/libdns/arvancloud"
	"github.com/libdns/libdns/libdnstest"
)

func TestArvancloudProvider(t *testing.T) {
	apiToken := os.Getenv("ARVANCLOUD_API_KEY")
	testZone := os.Getenv("ARVANCLOUD_TEST_ZONE")

	if apiToken == "" || testZone == "" {
		t.Skip("Skipping Cloudflare provider tests: ARVANCLOUD_API_KEY and/or ARVANCLOUD_TEST_ZONE environment variables must be set")
	}


	provider := &arvancloud.Provider{
		AuthAPIKey:  apiToken,
	}

	suite := libdnstest.NewTestSuite(provider, testZone)
	suite.SkipRRTypes = map[string]bool{"SVCB": true, "HTTPS": true}
	suite.RunTests(t)
}