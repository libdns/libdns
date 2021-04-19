package namedotcom

import (
	"context"
	"github.com/libdns/libdns"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	p              *Provider
	ctx            = context.Background()
	zone           string
	testRecords    []libdns.Record
	testSetRecords []libdns.Record
	rollingRecords []libdns.Record
)

func init() {
	p = &Provider{
		APIToken: os.Getenv("api_key"),
		User:     os.Getenv("user_name"),
		APIUrl:   "https://api.name.com",
	}

	testRecords = []libdns.Record{{
		Type:  "A",
		Name:  "TestRecord",
		Value: "192.168.1.33",
		TTL:   time.Duration(300),
	},
	}
	testSetRecords = []libdns.Record{{
		Type:  "A",
		Name:  "TestRecord",
		Value: "192.168.2.33",
		TTL:   time.Duration(300),
	},
	}
	zone = os.Getenv("test_zone")
}

func TestProvider_GetRecords(t *testing.T) {
	tests := []struct {
		name    string
		want    []libdns.Record
		wantErr bool
	}{
		{
			name:    "get_records_1_pass",
			want:    testRecords,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.GetRecords(ctx, zone)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else {
				rollingRecords = got
				t.Log(got, err)
			}
		})
	}
}

func TestProvider_AppendRecords(t *testing.T) {
	tests := []struct {
		name    string
		want    []libdns.Record
		wantErr bool
	}{
		{
			name:    "append_record_1_pass",
			want:    testRecords,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.AppendRecords(ctx, zone, testRecords)
			if (err != nil) != tt.wantErr {
				t.Errorf("AppendRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else {
				rollingRecords = got
				log.Println(got, err)
			}
		})
	}
}

func TestProvider_SetRecords(t *testing.T) {
	type args struct {
		ctx     context.Context
		zone    string
		records []libdns.Record
	}
	tests := []struct {
		name    string
		want    []libdns.Record
		wantErr bool
	}{
		{
			name:    "set_record_1_pass",
			want:    testSetRecords,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testName := strings.ToLower(testSetRecords[0].Name)
			t.Log(rollingRecords)

			for _, rec := range rollingRecords {
				if testName == rec.Name {
					testSetRecords[0].ID = rec.ID
				}
			}

			if testSetRecords[0].ID != "" {
				got, err := p.SetRecords(ctx, zone, testSetRecords)
				if (err != nil) != tt.wantErr {
					t.Errorf("SetRecords() error = %v, wantErr %v", err, tt.wantErr)
					return
				} else {
					rollingRecords = got
					t.Log(got, err)
				}
			} else {
				t.Log("skipping, record id is not set.")
				t.Skip()
			}
		})
	}
}

func TestProvider_DeleteRecords(t *testing.T) {
	tests := []struct {
		name    string
		want    []libdns.Record
		wantErr bool
	}{
		{
			name:    "delete_record_1_pass",
			want:    testRecords,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testName := strings.ToLower(testSetRecords[0].Name)

			for _, rec := range rollingRecords {
				if testName == rec.Name {
					testSetRecords[0].ID = rec.ID
				}
			}

			if testSetRecords[0].ID != "" {
				got, err := p.DeleteRecords(ctx, zone, testSetRecords)
				if (err != nil) != tt.wantErr {
					t.Errorf("DeleteRecords() error = %v, wantErr %v", err, tt.wantErr)
					return
				} else {
					t.Log(got, err)
				}
			} else {
				t.Log("skipping, record id is not set.")
				t.Skip()
			}
		})
	}
}
