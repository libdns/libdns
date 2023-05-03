package directadmin

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/libdns/libdns"
	"os"
	"testing"
	"time"
)

func initProvider() (*Provider, string) {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		os.Exit(1)
	}

	zone := envOrFail("LIBDNS_DA_TEST_ZONE")

	provider := &Provider{
		ServerURL:        envOrFail("LIBDNS_DA_TEST_SERVER_URL"),
		User:             envOrFail("LIBDNS_DA_TEST_USER"),
		LoginKey:         envOrFail("LIBDNS_DA_TEST_LOGIN_KEY"),
		InsecureRequests: true,
	}
	return provider, zone
}

func envOrFail(key string) string {
	val := os.Getenv(key)
	if len(val) == 0 {
		fmt.Printf("Please notice that this test runs against a production direct admin DNS API\n"+
			"you sould never run the test with an in use, production zone.\n\n"+
			"To run these tests, you need to copy .env.example to .env and modify the values for your environment.\n\n"+
			"%v is required", key)
		os.Exit(1)
	}

	return val
}

func TestProvider_GetRecords(t *testing.T) {
	ctx := context.TODO()

	// Configure the DNS provider
	provider, zone := initProvider()

	// list records
	records, err := provider.GetRecords(ctx, zone)

	if len(records) == 0 {
		t.Errorf("expected >0 records")
	}

	if err != nil {
		t.Error(err)
	}

	// Hack to work around "unsupported record conversion of type SRV: _xmpp._tcp"
	// output not generating a new line. This breaks GoLands test results output
	// https://stackoverflow.com/a/68607772/95790
	fmt.Println()
}

func TestProvider_AppendRecords(t *testing.T) {
	ctx := context.TODO()

	// Configure the DNS provider
	provider, zone := initProvider()

	var tests = []struct {
		records       []libdns.Record
		expectSuccess bool
	}{
		{
			records: []libdns.Record{
				{
					Type:  "A",
					Name:  "libdnsTest",
					Value: "1.1.1.1",
					TTL:   300 * time.Second,
				},
			},
			expectSuccess: true,
		},
		{
			records: []libdns.Record{
				{
					Type:  "A",
					Name:  "libdnsTest",
					Value: "libdnsTest",
					TTL:   300 * time.Second,
				},
			},
			expectSuccess: false,
		},
		{
			records: []libdns.Record{
				{
					Type:  "AAAA",
					Name:  "libdnsTest",
					Value: "2606:4700:4700::1111",
					TTL:   300 * time.Second,
				},
			},
			expectSuccess: true,
		},
		{
			records: []libdns.Record{
				{
					Type:  "AAAA",
					Name:  "libdnsTest2",
					Value: "test2",
					TTL:   300 * time.Second,
				},
			},
			expectSuccess: false,
		},
		{
			records: []libdns.Record{
				{
					Type:  "A",
					Name:  "libdnsTest2",
					Value: "1.1.1.1",
					TTL:   300 * time.Second,
				},
				{
					Type:  "AAAA",
					Name:  "libdnsTest2",
					Value: "2606:4700:4700::1111",
					TTL:   300 * time.Second,
				},
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("%v records", 0)
		t.Run(testName, func(t *testing.T) {
			// Append Records
			_, err := provider.AppendRecords(ctx, zone, tt.records)

			if tt.expectSuccess && err != nil {
				t.Error(err)
			}

			if !tt.expectSuccess && err == nil {
				t.Error("expected an error, didn't see one")
			}
		})
	}
}

func TestProvider_SetRecords(t *testing.T) {
	ctx := context.TODO()

	// Configure the DNS provider
	provider, zone := initProvider()

	var tests = []struct {
		records       []libdns.Record
		expectSuccess bool
	}{
		{
			records: []libdns.Record{
				{
					Type:  "A",
					Name:  "libdnsTest",
					Value: "8.8.8.8",
					TTL:   300 * time.Second,
				},
			},
			expectSuccess: true,
		},
		{
			records: []libdns.Record{
				{
					Type:  "AAAA",
					Name:  "libdnsTest",
					Value: "2001:4860:4860::8888",
					TTL:   300 * time.Second,
				},
			},
			expectSuccess: true,
		},
		{
			records: []libdns.Record{
				{
					Type:  "A",
					Name:  "libdnsTest2",
					Value: "8.8.8.8",
					TTL:   300 * time.Second,
				},
				{
					Type:  "AAAA",
					Name:  "libdnsTest2",
					Value: "2001:4860:4860::8888",
					TTL:   300 * time.Second,
				},
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("%v records", 0)
		t.Run(testName, func(t *testing.T) {
			// Append Records
			_, err := provider.SetRecords(ctx, zone, tt.records)

			if tt.expectSuccess && err != nil {
				t.Error(err)
			}

			if !tt.expectSuccess && err == nil {
				t.Error("expected an error, didn't see one")
			}
		})
	}

	// Hack to work around "unsupported record conversion of type SRV: _xmpp._tcp"
	// output not generating a new line. This breaks GoLands test results output
	// https://stackoverflow.com/a/68607772/95790
	fmt.Println()
}

func TestProvider_DeleteRecords(t *testing.T) {
	ctx := context.TODO()

	// Configure the DNS provider
	provider, zone := initProvider()

	var tests = []struct {
		records       []libdns.Record
		expectSuccess bool
	}{
		{
			records: []libdns.Record{
				{
					Type:  "A",
					Name:  "libdnsTest",
					Value: "8.8.8.8",
					TTL:   300 * time.Second,
				},
			},
			expectSuccess: true,
		},
		{
			records: []libdns.Record{
				{
					Type:  "AAAA",
					Name:  "libdnsTest",
					Value: "2001:4860:4860::8888",
					TTL:   300 * time.Second,
				},
			},
			expectSuccess: true,
		},
		{
			records: []libdns.Record{
				{
					Type:  "A",
					Name:  "libdnsTest2",
					Value: "8.8.8.8",
					TTL:   300 * time.Second,
				},
				{
					Type:  "AAAA",
					Name:  "libdnsTest2",
					Value: "2001:4860:4860::8888",
					TTL:   300 * time.Second,
				},
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		testName := fmt.Sprintf("%v records", 0)
		t.Run(testName, func(t *testing.T) {
			// Append Records
			_, err := provider.DeleteRecords(ctx, zone, tt.records)

			if tt.expectSuccess && err != nil {
				t.Error(err)
			}

			if !tt.expectSuccess && err == nil {
				t.Error("expected an error, didn't see one")
			}
		})
	}
}
