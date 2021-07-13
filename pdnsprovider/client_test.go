package pdnsprovider

import (
	"context"
	"fmt"
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
		Type      string
		records   []libdns.Record
		want      []string
	}{
		{
			name:      "Test Get Zone",
			operation: "records",
			zone:      "example.org.",
			records:   nil,
			Type:      "A",
			want:      []string{"1:127.0.0.1", "1:127.0.0.2", "1:127.0.0.3", "2:127.0.0.4", "2:127.0.0.5", "2:127.0.0.6"},
		},
		{
			name:      "Test Append Zone A record",
			operation: "append",
			zone:      "example.org.",
			Type:      "A",
			records: []libdns.Record{
				{
					Name:  "2",
					Type:  "A",
					Value: "127.0.0.7",
				},
			},
			want: []string{"1:127.0.0.1", "1:127.0.0.2", "1:127.0.0.3",
				"2:127.0.0.4", "2:127.0.0.5", "2:127.0.0.6", "2:127.0.0.7"},
		},
		{
			name:      "Test Append Zone TXT record",
			operation: "append",
			zone:      "example.org.",
			Type:      "TXT",
			records: []libdns.Record{
				{
					Name:  "1",
					Type:  "TXT",
					Value: "\"This is also some text\"",
				},
			},
			want: []string{"1:\"This is text\"", "1:\"This is also some text\""},
		},
		{
			name:      "Test Delete Zone",
			operation: "delete",
			zone:      "example.org.",
			Type:      "A",
			records: []libdns.Record{
				{
					Name:  "2",
					Type:  "A",
					Value: "127.0.0.7",
				},
			},
			want: []string{"1:127.0.0.1", "1:127.0.0.2", "1:127.0.0.3", "2:127.0.0.4", "2:127.0.0.5", "2:127.0.0.6"},
		},
		{
			name:      "Test Append and Add Zone",
			operation: "append",
			zone:      "example.org.",
			Type:      "A",
			records: []libdns.Record{
				{
					Name:  "2",
					Type:  "A",
					Value: "127.0.0.8",
				},
				{
					Name:  "3",
					Type:  "A",
					Value: "127.0.0.9",
				},
			},
			want: []string{"1:127.0.0.1", "1:127.0.0.2", "1:127.0.0.3",
				"2:127.0.0.4", "2:127.0.0.5", "2:127.0.0.6", "2:127.0.0.8",
				"3:127.0.0.9"},
		},
		{
			name:      "Test Set",
			operation: "set",
			zone:      "example.org.",
			Type:    "A",
			records: []libdns.Record{
				{
					Name:  "2",
					Type:  "A",
					Value: "127.0.0.1",
				},
				{
					Name:  "1",
					Type:  "A",
					Value: "127.0.0.1",
				},
			},
			want: []string{"1:127.0.0.1", "2:127.0.0.1", "3:127.0.0.9"},
		},
	} {
		t.Run(table.name, func(t *testing.T) {
			f := func(Name, Val string) string {
				return fmt.Sprintf("%s:%s", libdns.RelativeName(Name, table.zone), Val)
			}
			var have []string
			switch table.operation {
			case "records":
				recs, err := p.GetRecords(context.Background(), table.zone)
				if err != nil {
					t.Errorf("error fetching full zone %s", err)
					return
				}

				for _, rr := range recs {
					if rr.Type != table.Type {
						continue
					}
					have = append(have, fmt.Sprintf("%s:%s", rr.Name, rr.Value))
				}
			case "append", "set":
				var err error
				switch table.operation {
				case "append":
					_, err = p.AppendRecords(context.Background(), table.zone, table.records)
				default:
					_, err = p.SetRecords(context.Background(), table.zone, table.records)
				}
				if err != nil {
					t.Errorf("failed to %s records: %s", table.operation, err)
					return
				}
				z, err := c.fullZone(context.Background(), table.zone)
				if err != nil {
					t.Errorf("error fetching full zone %s", err)
					return
				}
				for _, rr := range z.ResourceRecordSets {
					if rr.Type == table.Type {
						for _, rec := range rr.Records {
							have = append(have, f(rr.Name, rec.Content))
						}
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
					if rr.Type != table.Type {
						continue
					}
					for _, rec := range rr.Records {
						have = append(have, f(rr.Name, rec.Content))
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
