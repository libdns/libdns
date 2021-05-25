// Package libdns-loopia implements a DNS record management client compatible
// with the libdns interfaces for Loopia.
package loopia

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/libdns/libdns"
)

func getRecords() []libdns.Record {
	return []libdns.Record{
		{ID: "14096733", Type: "A", Name: "*", Value: "192.168.42.1", TTL: time.Duration(5 * int(time.Minute))},
		{ID: "15838493", Type: "A", Name: "*", Value: "192.168.42.2", TTL: time.Duration(5 * int(time.Minute))},
		{ID: "14096734", Type: "NS", Name: "@", Value: "ns1.test.local.", TTL: time.Duration(int(time.Hour))},
		{ID: "15838494", Type: "NS", Name: "@", Value: "ns2.test.local.", TTL: time.Duration(10 * int(time.Minute))},
		{ID: "14096733", Type: "A", Name: "www", Value: "1.1.1.1", TTL: time.Duration(5 * int(time.Minute))},
	}
}

func TestProvider_GetRecords(t *testing.T) {
	tc := setupTest(t)
	defer teardownTest(tc)

	type args struct {
		ctx  context.Context
		zone string
	}
	tests := []struct {
		name     string
		provider *Provider
		args     args
		want     []libdns.Record
		wantErr  bool
	}{
		{"first", tc.getProvider(), args{context.TODO(), "test.local"}, getRecords(), false},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.provider
			got, err := p.GetRecords(tt.args.ctx, tt.args.zone)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.GetRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Provider.GetRecords()\n got\t %v,\n want\t %v", got, tt.want)
			}
		})
	}
}

func TestProvider_AppendRecords(t *testing.T) {
	tc := setupTest(t)
	defer teardownTest(tc)
	defer Log().Sync()
	type args struct {
		ctx     context.Context
		zone    string
		records []libdns.Record
	}
	tests := []struct {
		name     string
		provider *Provider
		args     args
		want     []libdns.Record
		wantErr  bool
	}{
		{"cdn", tc.getProvider(), args{context.TODO(), "test.local", []libdns.Record{
			{Type: "TXT", Name: "_test", Value: "some text", TTL: time.Duration(5 * time.Minute)},
		}}, []libdns.Record{{ID: "12345", Type: "TXT", Name: "_test", Value: "some text", TTL: 5 * time.Minute}}, false},
		// {"acme", tc.getProvider(),
		// 	args{
		// 		context.TODO(),
		// 		"test.local",
		// 		[]libdns.Record{
		// 			{Type: "TXT", Name: "_acme-challenge.test", Value: "UkBpoxq7XVRBML88YEq31EThjirZw9TAxlUFKUrJvEQ"},
		// 		},
		// 	},
		// 	nil,
		// 	false,
		// },
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.provider
			got, err := p.AppendRecords(tt.args.ctx, tt.args.zone, tt.args.records)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.AppendRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Provider.AppendRecords() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_SetRecords(t *testing.T) {
	tc := setupTest(t)
	defer teardownTest(tc)

	type args struct {
		ctx     context.Context
		zone    string
		records []libdns.Record
	}
	tests := []struct {
		name     string
		provider *Provider
		args     args
		want     []libdns.Record
		wantErr  bool
	}{
		{"nil records", tc.getProvider(), args{context.TODO(), "test.local", nil}, nil, true},
		{"empty records", tc.getProvider(), args{context.TODO(), "test.local", []libdns.Record{}}, nil, true},
		{"invalid record", tc.getProvider(), args{context.TODO(), "test.local", []libdns.Record{{Name: "www"}}}, nil, true},
		{"invalid ID", tc.getProvider(), args{context.TODO(), "test.local", []libdns.Record{{Name: "www", Type: "A", Value: "127.0.0.1", TTL: 5 * time.Minute}}}, nil, true},
		{"valid record", tc.getProvider(), args{context.TODO(), "test.local", []libdns.Record{{ID: "12345", Name: "www", Type: "A", Value: "127.0.0.1", TTL: 5 * time.Minute}}},
			[]libdns.Record{{ID: "12345", Name: "www", Type: "A", Value: "127.0.0.1", TTL: 5 * time.Minute}}, false},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.provider
			got, err := p.SetRecords(tt.args.ctx, tt.args.zone, tt.args.records)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.SetRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Provider.SetRecords() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProvider_DeleteRecords(t *testing.T) {
	tc := setupTest(t)
	defer teardownTest(tc)

	type args struct {
		ctx     context.Context
		zone    string
		records []libdns.Record
	}
	tests := []struct {
		name     string
		provider *Provider
		args     args
		want     []libdns.Record
		wantErr  bool
	}{
		{"invalid zone", tc.getProvider(), args{context.TODO(), "", nil}, nil, true},
		{"nil records", tc.getProvider(), args{context.TODO(), "test.local", nil}, nil, true},
		{"empty records", tc.getProvider(), args{context.TODO(), "test.local", []libdns.Record{}}, nil, true},
		{"no id records", tc.getProvider(), args{context.TODO(), "test.local", []libdns.Record{{Name: "test", Type: "A"}}}, nil, true},
		{"valid records", tc.getProvider(), args{context.TODO(), "test.local", []libdns.Record{{Name: "test", ID: "12345"}}}, []libdns.Record{{Name: "test", ID: "12345"}}, false},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.provider
			got, err := p.DeleteRecords(tt.args.ctx, tt.args.zone, tt.args.records)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.DeleteRecords() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Provider.DeleteRecords() = %v, want %v", got, tt.want)
			}
		})
	}
}
