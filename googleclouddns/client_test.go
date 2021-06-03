package googleclouddns

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/httpreplay"
	"github.com/libdns/libdns"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var (
	testProject = `test-project`
)

func Test_GetRecords(t *testing.T) {
	p, rs, err := getTestDNSClient(`./replay/dns_listrecords.json`)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	p.Project = testProject
	records, err := p.getCloudDNSRecords(context.Background(), `test.development.io.`)
	if err != nil {
		t.Fatal("error listing records from the test zone:", err)
	}
	if len(records) != 12 {
		t.Fatal("expected twelve records back, received", len(records))
	}
	for _, record := range records {
		if record.Type != "TXT" {
			continue
		}
		if record.Value != "\"Hi there! This is a TXT record!\"" {
			t.Fatal("TXT record was not correct, received:", record.Value)
		}
	}
}

func Test_CreateFail(t *testing.T) {
	p, rs, err := getTestDNSClient(`./replay/dns_createrecordfail.json`)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	dnsRecord := libdns.Record{
		Type:  "A",
		Name:  "mail",
		Value: "192.168.2.1",
		TTL:   time.Second * 600,
	}
	p.Project = testProject
	_, err = p.setCloudDNSRecord(context.Background(), `test.development.io.`, []libdns.Record{dnsRecord}, false)
	var (
		gerr *googleapi.Error
		ok   bool
	)
	if gerr, ok = err.(*googleapi.Error); !ok {
		t.Fatalf("expected an error of type googleapi.Error but received %T", err)
	}
	if gerr.Code != 409 {
		t.Fatalf("expected error code 409 but received %d", gerr.Code)
	}
}

func Test_Create(t *testing.T) {
	p, rs, err := getTestDNSClient(`./replay/dns_createrecord.json`)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	dnsRecord := libdns.Record{
		Type:  "CNAME",
		Name:  "libdns",
		Value: "libdns.example.org.",
		TTL:   time.Second * 1200,
	}
	p.Project = testProject
	createRecords, err := p.setCloudDNSRecord(context.Background(), `test.development.io.`, []libdns.Record{dnsRecord}, false)
	if err != nil {
		t.Fatalf("attempt to create a record failed with the following error: %v", err)
	}
	if len(createRecords) != 1 {
		t.Fatalf("expected to receive 1 created libdns Record back but received %d", len(createRecords))
	}
	if createRecords[0].Name != "libdns" {
		t.Fatalf("expected a record name of 'libdns' but received %s", createRecords[0].Name)
	}
}

func Test_Patch(t *testing.T) {
	p, rs, err := getTestDNSClient(`./replay/dns_patchrecord.json`)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	dnsRecord := libdns.Record{
		Type:  "CNAME",
		Name:  "libdns",
		Value: "libdns.example.com.",
		TTL:   time.Second * 1200,
	}
	p.Project = testProject
	patchRecords, err := p.setCloudDNSRecord(context.Background(), `test.development.io.`, []libdns.Record{dnsRecord}, true)
	if err != nil {
		t.Fatalf("attempt to patch an existing record failed with the following error: %v", err)
	}
	if len(patchRecords) != 1 {
		t.Fatalf("expected to receive 1 patched libdns Record back but received %d", len(patchRecords))
	}
	if patchRecords[0].Name != "libdns" {
		t.Fatalf("expected a record name of 'libdns' but received %s", patchRecords[0].Name)
	}
}

func Test_Delete(t *testing.T) {
	p, rs, err := getTestDNSClient(`./replay/dns_deleterecord.json`)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	dnsRecord := libdns.Record{
		Type:  "CNAME",
		Name:  "libdns",
		Value: "libdns.example.com.",
		TTL:   time.Second * 1200,
	}
	p.Project = testProject
	err = p.deleteCloudDNSRecord(context.Background(), `test.development.io.`, dnsRecord.Name, dnsRecord.Type)
	if err != nil {
		t.Fatalf("attempt to create a record failed with the following error: %v", err)
	}
}

// makes it easier to pass around the recorder or
// the replayer to tests
type replayClose interface {
	Close() error
}

// getTestDNSClient returns a Client prepped for testing. If the replay
// file exists in the replay folder, it will use that for testing,
// otherwise it will do a live request
func getTestDNSClient(filename string) (*Provider, replayClose, error) {
	var client *http.Client
	var rs replayClose
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		// Setup recorder and write out to specified filename
		client, rs, err = setupRecorder(filename)
	} else {
		// Playback file exists, use that to run tests
		client, rs, err = setupPlayback(filename)
	}
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	scopeOption := option.WithScopes(dns.NdevClouddnsReadwriteScope)
	httpClientOption := option.WithHTTPClient(client)
	dnsService, err := dns.NewService(ctx, scopeOption, httpClientOption)
	if err != nil {
		return nil, nil, err
	}
	provider := Provider{
		service: dnsService,
	}
	return &provider, rs, err
}

func setupRecorder(filename string) (*http.Client, replayClose, error) {
	ctx := context.Background()
	now := time.Now().UTC()
	nowBytes, err := json.Marshal(now)
	if err != nil {
		return nil, nil, err
	}
	tokenSource, err := google.DefaultTokenSource(ctx, dns.NdevClouddnsReadwriteScope)
	if err != nil {
		return nil, nil, err
	}
	rec, err := httpreplay.NewRecorder(filename, nowBytes)
	if err != nil {
		return nil, nil, err
	}
	opt := option.WithTokenSource(tokenSource)
	resClient, err := rec.Client(ctx, opt)
	if err != nil {
		return nil, nil, err
	}
	return resClient, rec, nil
}

func setupPlayback(filename string) (*http.Client, replayClose, error) {
	ctx := context.Background()
	replayer, err := httpreplay.NewReplayer(filename)
	if err != nil {
		return nil, nil, err
	}
	var tm time.Time
	if err := json.Unmarshal(replayer.Initial(), &tm); err != nil {
		return nil, nil, err
	}
	resClient, err := replayer.Client(ctx)
	if err != nil {
		return nil, nil, err
	}
	return resClient, replayer, nil
}
