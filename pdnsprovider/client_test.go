package pdnsprovider

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/libdns/libdns"
	"github.com/mittwald/go-powerdns/apis/zones"
)

func TestPDNSClient(t *testing.T) {
	var dockerCompose string
	var ok bool
	doRun, _ := strconv.ParseBool(os.Getenv("PDNS_RUN_INTEGRATION_TEST"))
	if !doRun {
		t.Skip("skipping because PDNS_RUN_INTEGRATION_TEST was not set")
	}
	if dockerCompose, ok = which("docker-compose"); !ok {
		t.Skip("docker-compose is not present, skipping")
	}
	err := runCmd(dockerCompose, "rm", "-sfv")
	if err != nil {
		t.Fatalf("docker-compose failed: %s", err)
	}
	err = runCmd(dockerCompose, "down", "-v")
	if err != nil {
		t.Fatalf("docker-compose failed: %s", err)
	}
	if err != nil {
		t.Fatalf("docker-compose failed: %s", err)
	}

	err = runCmd(dockerCompose, "up", "-d")
	defer func() {
		if skipCleanup, _ := strconv.ParseBool(os.Getenv("PDNS_SKIP_CLEANUP")); !skipCleanup {
			runCmd(dockerCompose, "down", "-v")
		}
	}()
	c, err := newClient("localhost", "http://localhost:8081", "secret", nil)
	if err != nil {
		t.Fatalf("failed client create: %s", err)
	}

	time.Sleep(time.Second * 30) // give everything time to finish coming up
	z := zones.Zone{
		Name: "example.org.",
		Type: zones.ZoneTypeZone,
		Kind: zones.ZoneKindNative,
		ResourceRecordSets: []zones.ResourceRecordSet{
			{
				Name: "1.example.org.",
				Type: "A",
				TTL:  60,
				Records: []zones.Record{
					{
						Content: "127.0.0.1",
					},
					{
						Content: "127.0.0.2",
					},
					{
						Content: "127.0.0.3",
					},
				},
			},
			{
				Name: "1.example.org.",
				Type: "TXT",
				TTL:  60,
				Records: []zones.Record{
					{
						Content: "\"This is text\"",
					},
				},
			},
			{
				Name: "2.example.org.",
				Type: "A",
				TTL:  60,
				Records: []zones.Record{
					{
						Content: "127.0.0.4",
					},
					{
						Content: "127.0.0.5",
					},
					{
						Content: "127.0.0.6",
					},
				},
			},
		},
		Serial: 1,
		Nameservers: []string{
			"ns1.example.org.",
			"ns2.example.org.",
		},
	}
	_, err = c.Client.Zones().CreateZone(context.Background(), c.sID, z)
	if err != nil {
		t.Fatalf("failed to create test zone: %s", err)
	}

	p := &Provider{
		ServerURL: "http://localhost:8081",
		ServerID:  "localhost",
		APIToken:  "secret",
		Debug:     os.Getenv("PDNS_DEBUG"),
	}
	for _, table := range []struct {
		name      string
		operation string
		zone      string
		records   []libdns.Record
		want      []string
	}{
		{
			name:      "Test Get Zone",
			operation: "records",
			zone:      "example.org.",
			records:   nil,
			want:      []string{"1.example.org.", "2.example.org."},
		},
		{
			name:      "Test Append Zone A record",
			operation: "append",
			zone:      "example.org.",
			records: []libdns.Record{
				{
					Name:  "1",
					Type:  "A",
					Value: "127.0.0.7",
				},
			},
			want: []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "127.0.0.7"},
		},
		{
			name:      "Test Append Zone TXT record",
			operation: "append",
			zone:      "example.org.",
			records: []libdns.Record{
				{
					Name:  "1",
					Type:  "TXT",
					Value: "\"This is also some text\"",
				},
			},
			want: []string{"\"This is text\"", "\"This is also some text\""},
		},

		{
			name:      "Test Delete Zone",
			operation: "delete",
			zone:      "example.org.",
			records: []libdns.Record{
				{
					Name:  "1",
					Type:  "A",
					Value: "127.0.0.7",
				},
			},
			want: []string{"127.0.0.1", "127.0.0.2", "127.0.0.3", "127.0.0.4", "127.0.0.5", "127.0.0.6"},
		},
	} {
		t.Run(table.name, func(t *testing.T) {
			var have []string
			switch table.operation {
			case "records":
				z, err := c.fullZone(context.Background(), table.zone)
				if err != nil {
					t.Errorf("error fetching full zone %s", err)
					return
				}

				for _, rr := range z.ResourceRecordSets {
					if rr.Type != "A" {
						continue
					}
					have = append(have, rr.Name)
				}
			case "append":
				_, err := p.AppendRecords(context.Background(), table.zone, table.records)
				if err != nil {
					t.Errorf("failed to append records: %s", err)
					return
				}
				z, err := c.fullZone(context.Background(), table.zone)
				if err != nil {
					t.Errorf("error fetching full zone %s", err)
					return
				}
				hash := makeLDRecHash(convertNamesToAbsolute(table.zone, table.records))
				wantedRRs := convertLDHash(hash)

				for _, wantedRR := range wantedRRs {
					found := false
					for _, gotRR := range z.ResourceRecordSets {
						if wantedRR.Name == gotRR.Name && wantedRR.Type == gotRR.Type {
							found = true
							for _, val := range gotRR.Records {
								have = append(have, val.Content)
							}
							break
						}
					}
					if !found {
						t.Errorf("rr not found: %s", wantedRR.Name)
						return
					}
				}
			case "delete":
				_, err := p.DeleteRecords(context.Background(), table.zone, table.records)
				if err != nil {
					t.Errorf("error deleting records: %s", err)
					return
				}
				z, err := c.fullZone(context.Background(), table.zone)
				if err != nil {
					t.Errorf("error fetching full zone %s", err)
					return
				}
				for _, rr := range z.ResourceRecordSets {
					if rr.Type != "A" {
						continue
					}
					for _, rec := range rr.Records {
						have = append(have, rec.Content)
					}
				}

			}
			sort.Strings(have)
			sort.Strings(table.want)
			if !reflect.DeepEqual(have, table.want) {
				t.Errorf("assertion failed: have: %#v want %#v", have, table.want)
			}

		})
	}

}

func which(cmd string) (string, bool) {
	pth, err := exec.LookPath(cmd)
	if err != nil {
		return "", false
	}
	return pth, true
}

func runCmd(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
