package totaluptime

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

func TestGetRecords(t *testing.T) {
	var p Provider
	setupMockServer()
	zone := "testdomain.com."

	t.Run("success", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		var logString bytes.Buffer
		log.SetOutput(&logString)

		got, err := p.GetRecords(context.TODO(), zone)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)
		AssertNil(t, err)

		gotString := fmt.Sprint(got)
		wantString := "[{test-a-id A blue-a 111.111.111.111 1h0m0s 0} {test-cname-id CNAME green-cname test-alias-domain.com. 1h0m0s 0} {test-mx-id MX @ test-mx-mailserver.com. 1h0m0s 0} {test-ns-id NS @ test-ns-name.com. 8h0m0s 0} {test-txt-id TXT red-txt test-txt-text-value 1m0s 0}]"
		AssertStrings(t, gotString, wantString)

		gotInt := len(got)
		wantInt := 5 // records retrieved from mock + flag item
		AssertInts(t, gotInt, wantInt)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("missing zone", func(t *testing.T) {
		zone := "missingzone.com."
		DomainIDs = make(map[string]string) // reset cache
		var logString bytes.Buffer
		log.SetOutput(&logString)

		got, err := p.GetRecords(context.TODO(), zone)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)
		AssertNil(t, got)

		gotErr := err.Error()
		wantErr := "record lookup cannot proceed: unknown domain: missingzone.com"
		AssertStrings(t, gotErr, wantErr)

		gotString := logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString := "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)

		wantString = "Domain not found: missingzone.com"
		AssertStringContains(t, gotString, wantString)
	})
}

func TestAppendRecords(t *testing.T) {
	var p Provider
	ctx := context.Background()
	zone := "testdomain.com."

	// records for appending
	records := []libdns.Record{
		libdns.Record{
			Type:  "A",
			Name:  "test-a-hostname",
			Value: "111.111.111.111",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:  "CNAME",
			Name:  "test-cname-hostname",
			Value: "test-cname-value",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:  "INVALID_TYPE",
			Name:  "test-INVALID-domain",
			Value: "test-INVALID-value",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:     "MX",
			Name:     "test-MX-domain",
			Value:    "test-MX-value",
			TTL:      3600 * time.Second,
			Priority: 10,
		},
		libdns.Record{
			Type:  "NS",
			Name:  "test-NS-domain",
			Value: "test-NS-value",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:  "TXT",
			Name:  "test-TXT-domain",
			Value: "test-TXT-value",
			TTL:   3600 * time.Second,
		},
	}

	t.Run("success API transaction", func(t *testing.T) {
		// setupMockServerSuccessTransaction()
		setupMockServer()
		DomainIDs = make(map[string]string) // reset cache
		var logString bytes.Buffer
		log.SetOutput(&logString)

		got, err := p.AppendRecords(ctx, zone, records)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)
		AssertNil(t, err)

		gotInt := len(got)
		// fmt.Printf(">>>DEBUG gotInt=%v\n", gotInt)
		wantInt := 5 // successful appended records
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(got)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "[{ A test-a-hostname 111.111.111.111 1h0m0s 0} { CNAME test-cname-hostname test-cname-value 1h0m0s 0} { MX test-MX-domain test-MX-value 1h0m0s 10} { NS test-NS-domain test-NS-value 1h0m0s 0} { TXT test-TXT-domain test-TXT-value 1h0m0s 0}]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful append record to zone: testdomain.com.: A: test-a-hostname"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful append record to zone: testdomain.com.: CNAME: test-cname-hostname"
		AssertStringContains(t, gotString, wantString)

		wantString = "Unable to identify record type; skipping: INVALID_TYPE: test-INVALID-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful append record to zone: testdomain.com.: MX: test-MX-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful append record to zone: testdomain.com.: NS: test-NS-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful append record to zone: testdomain.com.: TXT: test-TXT-domain"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("failed API transaction", func(t *testing.T) {
		setupMockServerFailedTransaction()
		DomainIDs = make(map[string]string) // reset cache
		var logString bytes.Buffer
		log.SetOutput(&logString)

		got, err := p.AppendRecords(ctx, zone, records)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)
		AssertNil(t, err)

		gotInt := len(got)
		// fmt.Printf(">>>DEBUG gotInt=%v\n", gotInt)
		wantInt := 0 // no records appended due to failed transaction mock
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(got)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "[]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS append failure with provider message: Invalid Port."
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("missing zone", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		zone := "missing.com."
		var logString bytes.Buffer
		log.SetOutput(&logString)

		got, err := p.AppendRecords(ctx, zone, records)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)

		AssertNil(t, got)

		gotErr := err.Error()
		wantErr := "record append cannot proceed: unknown zone: missing.com."
		AssertStrings(t, gotErr, wantErr)
	})
}

func TestModifyRecord(t *testing.T) {
	var p Provider
	ctx := context.Background()

	t.Run("missing zone", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		zone := "missing.com."
		record := libdns.Record{}
		var logString bytes.Buffer
		log.SetOutput(&logString)

		_, err := p.ModifyRecord(ctx, zone, record)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)

		gotErr := err.Error()
		wantErr := "record modify cannot proceed: unknown zone: missing.com."
		AssertStrings(t, gotErr, wantErr)

		gotString := logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString := "Domain not found: missing.com"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("bad record type", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		zone := "testdomain.com."
		var logString bytes.Buffer
		log.SetOutput(&logString)

		record := libdns.Record{
			Type:  "INVALID_TYPE",
			Name:  "test-INVALID-domain",
			Value: "test-INVALID-value",
			TTL:   3600 * time.Second,
		}

		_, err := p.ModifyRecord(ctx, zone, record)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)

		gotErr := err.Error()
		wantErr := "type INVALID_TYPE cannot be interpreted"
		AssertStrings(t, gotErr, wantErr)

		gotString := logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString := "Record not found: testdomain.com/INVALID_TYPE/test-INVALID-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "Unable to identify record type; skipping: INVALID_TYPE: test-INVALID-domain"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("failed transaction", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		RecordIDs = make(map[string]string) // reset cache
		setupMockServerFailedTransaction()
		zone := "testdomain.com."
		var logString bytes.Buffer
		log.SetOutput(&logString)

		record := libdns.Record{
			Type:  "TXT",
			Name:  "red-txt",
			Value: "test-TXT-value",
			TTL:   3600 * time.Second,
		}

		_, err := p.ModifyRecord(ctx, zone, record)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)
		AssertNil(t, err)

		gotString := logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString := "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)

		wantString = "Populate Records cache with live API call for domain: testdomain.com"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS modify failure with provider message: Invalid Port."
		AssertStringContains(t, gotString, wantString)
	})
}

func TestSetRecords(t *testing.T) {
	var p Provider
	ctx := context.Background()
	zone := "testdomain.com."

	// new records for appending
	records := []libdns.Record{
		libdns.Record{
			Type:  "A",
			Name:  "test-a-hostname",
			Value: "111.111.111.111",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:  "CNAME",
			Name:  "test-cname-hostname",
			Value: "test-cname-value",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:  "INVALID_TYPE",
			Name:  "test-INVALID-domain",
			Value: "test-INVALID-value",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:     "MX",
			Name:     "test-MX-domain",
			Value:    "test-MX-value",
			TTL:      3600 * time.Second,
			Priority: 10,
		},
		libdns.Record{
			Type:  "NS",
			Name:  "test-NS-domain",
			Value: "test-NS-value",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:  "TXT",
			Name:  "test-TXT-domain",
			Value: "test-TXT-value",
			TTL:   3600 * time.Second,
		},
	}

	t.Run("appending records", func(t *testing.T) {
		// setupMockServerSuccessTransaction()
		DomainIDs = make(map[string]string) // reset cache
		RecordIDs = make(map[string]string) // reset cache
		setupMockServer()
		var logString bytes.Buffer
		log.SetOutput(&logString)

		got, err := p.SetRecords(ctx, zone, records)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)
		AssertNil(t, err)

		gotInt := len(got)
		// fmt.Printf(">>>DEBUG gotInt=%v\n", gotInt)
		wantInt := 5 // successful updated records
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(got)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "[{ A test-a-hostname 111.111.111.111 1h0m0s 0} { CNAME test-cname-hostname test-cname-value 1h0m0s 0} { MX test-MX-domain test-MX-value 1h0m0s 10} { NS test-NS-domain test-NS-value 1h0m0s 0} { TXT test-TXT-domain test-TXT-value 1h0m0s 0}]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)

		wantString = "Lookup DomainIDs from cache"
		AssertStringContains(t, gotString, wantString)

		wantString = "Populate Records cache with live API call for domain: testdomain.com"
		AssertStringContains(t, gotString, wantString)

		wantString = "Record not found: testdomain.com/A/test-a-hostname"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful append record to zone: testdomain.com.: A: test-a-hostname"
		AssertStringContains(t, gotString, wantString)

		wantString = "Record not found: testdomain.com/CNAME/test-cname-hostname"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful append record to zone: testdomain.com.: CNAME: test-cname-hostname"
		AssertStringContains(t, gotString, wantString)

		wantString = "Record not found: testdomain.com/INVALID_TYPE/test-INVALID-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "Unable to identify record type; skipping: INVALID_TYPE: test-INVALID-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "Record not found: testdomain.com/MX/test-MX-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful append record to zone: testdomain.com.: MX: test-MX-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "Record not found: testdomain.com/NS/test-NS-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful append record to zone: testdomain.com.: NS: test-NS-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "Record not found: testdomain.com/TXT/test-TXT-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful append record to zone: testdomain.com.: TXT: test-TXT-domain"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("updating records", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		RecordIDs = make(map[string]string) // reset cache
		setupMockServer()
		var logString bytes.Buffer
		log.SetOutput(&logString)

		// records for updating
		records := []libdns.Record{
			libdns.Record{
				Type:  "A",
				Name:  "blue-a",
				Value: "111.111.111.111",
				TTL:   3600 * time.Second,
			},
			libdns.Record{
				Type:  "CNAME",
				Name:  "green-cname",
				Value: "test-cname-value",
				TTL:   3600 * time.Second,
			},
			libdns.Record{
				Type:  "INVALID_TYPE",
				Name:  "test-INVALID-domain",
				Value: "test-INVALID-value",
				TTL:   3600 * time.Second,
			},
			libdns.Record{
				Type:     "MX",
				Name:     "@",
				Value:    "test-MX-value",
				TTL:      3600 * time.Second,
				Priority: 10,
			},
			libdns.Record{
				Type:  "NS",
				Name:  "@",
				Value: "test-NS-value",
				TTL:   3600 * time.Second,
			},
			libdns.Record{
				Type:  "TXT",
				Name:  "red-txt",
				Value: "test-TXT-value",
				TTL:   3600 * time.Second,
			},
		}

		got, err := p.SetRecords(ctx, zone, records)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)
		AssertNil(t, err)

		gotInt := len(got)
		// fmt.Printf(">>>DEBUG gotInt=%v\n", gotInt)
		wantInt := 5 // successful updated records
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(got)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "[{ A blue-a 111.111.111.111 1h0m0s 0} { CNAME green-cname test-cname-value 1h0m0s 0} { MX @ test-MX-value 1h0m0s 10} { NS @ test-NS-value 1h0m0s 0} { TXT red-txt test-TXT-value 1h0m0s 0}]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)

		wantString = "Lookup DomainIDs from cache"
		AssertStringContains(t, gotString, wantString)

		wantString = "Populate Records cache with live API call for domain: testdomain.com"
		AssertStringContains(t, gotString, wantString)

		wantString = "Lookup RecordIDs from cache for domain: testdomain.com"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful modify record in zone: testdomain.com.: A: blue-a"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful modify record in zone: testdomain.com.: CNAME: green-cname"
		AssertStringContains(t, gotString, wantString)

		wantString = "Record not found: testdomain.com/INVALID_TYPE/test-INVALID-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "Unable to identify record type; skipping: INVALID_TYPE: test-INVALID-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful modify record in zone: testdomain.com.: MX: @"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful modify record in zone: testdomain.com.: NS: @"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful modify record in zone: testdomain.com.: TXT: red-txt"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("failed API transaction", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		RecordIDs = make(map[string]string) // reset cache
		setupMockServerFailedTransaction()
		var logString bytes.Buffer
		log.SetOutput(&logString)

		got, err := p.SetRecords(ctx, zone, records)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)
		AssertNil(t, err)

		gotInt := len(got)
		// fmt.Printf(">>>DEBUG gotInt=%v\n", gotInt)
		wantInt := 0 // no records updated due to failed transaction mock
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(got)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "[]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS append failure with provider message: Invalid Port."
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("missing zone", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		zone := "missing.com."
		var logString bytes.Buffer
		log.SetOutput(&logString)

		got, err := p.SetRecords(ctx, zone, records)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)

		AssertNil(t, got)

		gotErr := err.Error()
		wantErr := "record update cannot proceed: unknown zone: missing.com."
		AssertStrings(t, gotErr, wantErr)
	})
}

func TestDeleteRecords(t *testing.T) {
	var p Provider
	ctx := context.Background()
	zone := "testdomain.com."

	// records for deleting
	records := []libdns.Record{
		libdns.Record{
			Type:  "A",
			Name:  "blue-a",
			Value: "111.111.111.111",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:  "CNAME",
			Name:  "green-cname",
			Value: "test-cname-value",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:  "INVALID_TYPE",
			Name:  "test-INVALID-domain",
			Value: "test-INVALID-value",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:     "MX",
			Name:     "@",
			Value:    "test-MX-value",
			TTL:      3600 * time.Second,
			Priority: 10,
		},
		libdns.Record{
			Type:  "NS",
			Name:  "@",
			Value: "test-NS-value",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:  "TXT",
			Name:  "red-txt",
			Value: "test-TXT-value",
			TTL:   3600 * time.Second,
		},
		libdns.Record{
			Type:  "TXT",
			Name:  "extra-txt",
			Value: "test-extra-TXT-value",
			TTL:   3600 * time.Second,
		},
	}

	t.Run("success API transaction", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		RecordIDs = make(map[string]string) // reset cache
		setupMockServer()
		var logString bytes.Buffer
		log.SetOutput(&logString)

		got, err := p.DeleteRecords(ctx, zone, records)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)
		AssertNil(t, err)

		gotInt := len(got)
		// fmt.Printf(">>>DEBUG gotInt=%v\n", gotInt)
		wantInt := 5 // successful deleted records
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(got)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "[{ A blue-a 111.111.111.111 1h0m0s 0} { CNAME green-cname test-cname-value 1h0m0s 0} { MX @ test-MX-value 1h0m0s 10} { NS @ test-NS-value 1h0m0s 0} { TXT red-txt test-TXT-value 1h0m0s 0}]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)

		wantString = "Populate Records cache with live API call for domain: testdomain.com"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful delete record in zone: testdomain.com.: A: blue-a"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful delete record in zone: testdomain.com.: CNAME: green-cname"
		AssertStringContains(t, gotString, wantString)

		wantString = "Unable to identify record type; skipping: INVALID_TYPE: test-INVALID-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful delete record in zone: testdomain.com.: MX: @"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful delete record in zone: testdomain.com.: NS: @"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS successful delete record in zone: testdomain.com.: TXT: red-txt"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("failed API transaction", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		RecordIDs = make(map[string]string) // reset cache
		setupMockServerFailedTransaction()
		var logString bytes.Buffer
		log.SetOutput(&logString)

		got, err := p.DeleteRecords(ctx, zone, records)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)
		AssertNil(t, err)

		gotInt := len(got)
		// fmt.Printf(">>>DEBUG gotInt=%v\n", gotInt)
		wantInt := 0 // no records deleted due to failed transaction mock
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(got)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "[]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)

		wantString = "DNS delete record failure with provider message: Invalid Port."
		AssertStringContains(t, gotString, wantString)

		wantString = "Unable to identify record type; skipping: INVALID_TYPE: test-INVALID-domain"
		AssertStringContains(t, gotString, wantString)

		wantString = "Record not found: testdomain.com/TXT/extra-txt"
		AssertStringContains(t, gotString, wantString)

		wantString = "Record not found in zone; skipping delete: testdomain.com.: TXT: extra-txt"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("missing zone", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		RecordIDs = make(map[string]string) // reset cache
		zone := "missing.com."
		var logString bytes.Buffer
		log.SetOutput(&logString)

		got, err := p.DeleteRecords(ctx, zone, records)
		// fmt.Printf(">>>DEBUG got=%v; err=%v\n", got, err)

		AssertNil(t, got)

		gotErr := err.Error()
		wantErr := "record delete cannot proceed: unknown zone: missing.com."
		AssertStrings(t, gotErr, wantErr)
	})
}
