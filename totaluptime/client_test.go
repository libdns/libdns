package totaluptime

import (
	"bytes"
	"fmt"
	"log"
	"testing"
)

func TestLookupDomainIDs(t *testing.T) {
	var p Provider
	setupMockServer()

	t.Run("initial request", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		var logString bytes.Buffer
		log.SetOutput(&logString)

		gotErr := p.lookupDomainIDs()
		// fmt.Printf(">>>DEBUG gotErr=%v\n", gotErr)
		AssertNil(t, gotErr)

		gotInt := len(DomainIDs)
		wantInt := 1 // one domain retrieved
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(DomainIDs)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "map[testdomain.com:test-domain-id]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("use cache if available", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		DomainIDs["testblue.com"] = "testblue-id-from-the-cache"
		var logString bytes.Buffer
		log.SetOutput(&logString)

		gotErr := p.lookupDomainIDs()
		// fmt.Printf(">>>DEBUG gotErr=%v\n", gotErr)
		AssertNil(t, gotErr)

		gotInt := len(DomainIDs)
		wantInt := 1 // one domain retrieved from cache
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(DomainIDs)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "map[testblue.com:testblue-id-from-the-cache]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Lookup DomainIDs from cache"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("forceAPI explicit false", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		DomainIDs["testblue.com"] = "testblue-id-from-the-cache"
		var logString bytes.Buffer
		log.SetOutput(&logString)

		gotErr := p.lookupDomainIDs(false)
		// fmt.Printf(">>>DEBUG gotErr=%v\n", gotErr)
		AssertNil(t, gotErr)

		gotInt := len(DomainIDs)
		wantInt := 1 // one domain retrieved from cache
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(DomainIDs)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "map[testblue.com:testblue-id-from-the-cache]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Lookup DomainIDs from cache"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("invalid json", func(t *testing.T) {
		DomainIDs = make(map[string]string) // reset cache
		setupMockServerInvalidJSON()

		gotErr := p.lookupDomainIDs()
		// fmt.Printf(">>>DEBUG gotErr=%v\n", gotErr)
		if gotErr == nil {
			t.Errorf("expecting error but got <nil>")
		}

		gotInt := len(DomainIDs)
		wantInt := 0 // no domains retrieved
		AssertInts(t, gotInt, wantInt)
	})
}

func TestGetDomainID(t *testing.T) {
	var p Provider
	DomainIDs = make(map[string]string) // reset cache
	setupMockServer()

	t.Run("lookup found", func(t *testing.T) {
		domain := "testdomain.com"

		got := p.getDomainID(domain)
		// fmt.Printf(">>>DEBUG got=%v; DomainIDs=%v\n", got, DomainIDs)

		want := "test-domain-id"
		AssertStrings(t, got, want)
	})

	t.Run("lookup missing", func(t *testing.T) {
		var logString bytes.Buffer
		log.SetOutput(&logString)
		domain := "missingdomain.com"

		got := p.getDomainID(domain)
		// fmt.Printf(">>>DEBUG got=%v; DomainIDs=%v\n", got, DomainIDs)

		want := ""
		AssertStrings(t, got, want)

		gotString := logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString := "Lookup DomainIDs from cache"
		AssertStringContains(t, gotString, wantString)

		wantString = "Domain not found: missingdomain.com"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("invalid cache", func(t *testing.T) {
		var logString bytes.Buffer
		log.SetOutput(&logString)
		setupMockServerInvalidJSON()
		DomainIDs = nil
		domain := "testdomain.com"

		got := p.getDomainID(domain)
		// fmt.Printf(">>>DEBUG got=%v; DomainIDs=%v\n", got, DomainIDs)

		want := "" // nothing returned because cache invalid
		AssertStrings(t, got, want)
	})
}

func TestLookupRecordIDs(t *testing.T) {
	var p Provider
	setupMockServer()
	domain := "testdomain.com"

	t.Run("initial request", func(t *testing.T) {
		RecordIDs = make(map[string]string) // reset cache
		var logString bytes.Buffer
		log.SetOutput(&logString)

		gotErr := p.lookupRecordIDs(domain)
		// fmt.Printf(">>>DEBUG gotErr=%v\n", gotErr)
		AssertNil(t, gotErr)

		gotInt := len(RecordIDs)
		wantInt := 6 // records retrieved from mock + flag item
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(RecordIDs)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "map[testdomain.com:populated testdomain.com/A/blue-a:test-a-id testdomain.com/CNAME/green-cname:test-cname-id testdomain.com/MX/@:test-mx-id testdomain.com/NS/@:test-ns-id testdomain.com/TXT/red-txt:test-txt-id]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Populate DomainIDs cache with live API call"
		AssertStringContains(t, gotString, wantString)

		wantString = "Populate Records cache with live API call for domain: testdomain.com"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("use cache if available", func(t *testing.T) {
		RecordIDs = make(map[string]string) // reset cache
		var logString bytes.Buffer
		log.SetOutput(&logString)
		RecordIDs[domain] = "populated" // flag item
		RecordIDs[domain+"/A/blue-a"] = "test-a-id-from-the-cache"

		gotErr := p.lookupRecordIDs(domain)
		// fmt.Printf(">>>DEBUG gotErr=%v\n", gotErr)
		AssertNil(t, gotErr)

		gotInt := len(RecordIDs)
		wantInt := 2 // records cached + flag item
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(RecordIDs)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "map[testdomain.com:populated testdomain.com/A/blue-a:test-a-id-from-the-cache]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Lookup RecordIDs from cache for domain: testdomain.com"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("forceAPI explicit false", func(t *testing.T) {
		RecordIDs = make(map[string]string) // reset cache
		RecordIDs[domain] = "populated"     // flag item
		RecordIDs[domain+"/A/blue-a"] = "test-a-id-from-the-cache"
		var logString bytes.Buffer
		log.SetOutput(&logString)

		gotErr := p.lookupRecordIDs(domain, false)
		// fmt.Printf(">>>DEBUG gotErr=%v\n", gotErr)
		AssertNil(t, gotErr)

		gotInt := len(RecordIDs)
		wantInt := 2 // record + flag item
		AssertInts(t, gotInt, wantInt)

		gotString := fmt.Sprint(RecordIDs)
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)
		wantString := "map[testdomain.com:populated testdomain.com/A/blue-a:test-a-id-from-the-cache]"
		AssertStrings(t, gotString, wantString)

		gotString = logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString = "Lookup RecordIDs from cache for domain: testdomain.com"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("invalid json", func(t *testing.T) {
		RecordIDs = make(map[string]string) // reset cache
		setupMockServerInvalidJSON()

		gotErr := p.lookupRecordIDs(domain)
		// fmt.Printf(">>>DEBUG gotErr=%v\n", gotErr)
		if gotErr == nil {
			t.Errorf("expecting error but got <nil>")
		}

		gotInt := len(RecordIDs)
		wantInt := 0 // no records retrieved
		AssertInts(t, gotInt, wantInt)
	})
}

func TestGetRecordID(t *testing.T) {
	var p Provider
	RecordIDs = make(map[string]string) // reset cache
	setupMockServer()

	t.Run("lookup found", func(t *testing.T) {
		domain := "testdomain.com"
		recType := "TXT"
		recName := "red-txt"

		got := p.getRecordID(domain, recType, recName)
		// fmt.Printf(">>>DEBUG got=%v; RecordIDs=%v\n", got, RecordIDs)

		want := "test-txt-id"
		AssertStrings(t, got, want)
	})

	t.Run("domain missing", func(t *testing.T) {
		var logString bytes.Buffer
		log.SetOutput(&logString)
		domain := "missingdomain.com"
		recType := "TXT"
		recName := "red-txt"

		got := p.getRecordID(domain, recType, recName)
		// fmt.Printf(">>>DEBUG got=%v; RecordIDs=%v\n", got, RecordIDs)

		want := ""
		AssertStrings(t, got, want)

		gotString := logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString := "Lookup DomainIDs from cache"
		AssertStringContains(t, gotString, wantString)

		wantString = "Domain not found: missingdomain.com"
		AssertStringContains(t, gotString, wantString)

		wantString = "record lookup cannot proceed: unknown domain: missingdomain.com"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("record missing", func(t *testing.T) {
		var logString bytes.Buffer
		log.SetOutput(&logString)
		domain := "testdomain.com"
		recType := "TXT"
		recName := "cant-touch-this"

		got := p.getRecordID(domain, recType, recName)
		// fmt.Printf(">>>DEBUG got=%v; RecordIDs=%v\n", got, RecordIDs)

		want := ""
		AssertStrings(t, got, want)

		gotString := logString.String()
		// fmt.Printf(">>>DEBUG gotString=%v\n", gotString)

		wantString := "Lookup RecordIDs from cache for domain: testdomain.com"
		AssertStringContains(t, gotString, wantString)

		wantString = "Record not found: testdomain.com/TXT/cant-touch-this"
		AssertStringContains(t, gotString, wantString)
	})

	t.Run("invalid cache", func(t *testing.T) {
		var logString bytes.Buffer
		log.SetOutput(&logString)
		setupMockServerInvalidJSON()
		RecordIDs = nil
		domain := "testdomain.com"
		recType := "TXT"
		recName := "red-txt"

		got := p.getRecordID(domain, recType, recName)
		// fmt.Printf(">>>DEBUG got=%v; RecordIDs=%v\n", got, RecordIDs)

		want := "" // nothing returned because cache invalid
		AssertStrings(t, got, want)
	})
}
