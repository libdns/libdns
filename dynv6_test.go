package dynv6

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"testing"

	"github.com/libdns/libdns"
)

var (
	p   Provider
	ctx context.Context = context.Background()
)

func init() {
	flag.StringVar(&p.Token, "token", "", "dynv6 REST API token")
}

func TestGetZoneByName(t *testing.T) {
	zones, err := p.getZones(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, zoneItem := range zones {
		z, err := p.getZoneByName(ctx, zoneItem.Name)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(z)
	}
}

func TestGetZoneByID(t *testing.T) {
	zones, err := p.getZones(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, zoneItem := range zones {
		z, err := p.getZoneByID(ctx, zoneItem.ID)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(z)
	}
}

func TestListZones(t *testing.T) {
	zones, err := p.getZones(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(zones)
}

func TestGetRecords(t *testing.T) {
	zones, err := p.getZones(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, zoneItem := range zones {
		records, err := p.getRecords(ctx, zoneItem.ID)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(records)
	}
}

func generateRandInt(t *testing.T) uint16 {
	var data uint16
	err := binary.Read(rand.Reader, binary.BigEndian, &data)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAddUpdateDeleteRecord(t *testing.T) {
	zones, err := p.getZones(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, zoneItem := range zones {
		data := generateRandInt(t)
		r, err := p.addRecord(ctx, zoneItem.ID, &record{
			Name: fmt.Sprintf("test%d", data),
			Type: "TXT",
			Data: fmt.Sprint(data),
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Added record: %+v", r)
		if r.Data != fmt.Sprint(data) {
			t.Fatal("Data returned is not equal to data sent")
		}
		data = generateRandInt(t)
		r, err = p.updateRecord(ctx, zoneItem.ID, &record{
			ID:   r.ID,
			Name: r.Name,
			Type: r.Type,
			Data: fmt.Sprint(data),
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Updated record: %+v", r)
		if r.Data != fmt.Sprint(data) {
			t.Fatal("Data returned is not equal to data sent")
		}
		err = p.deleteRecord(ctx, zoneItem.ID, r.ID)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Deleted record: %+v", r)
	}
}

func TestLibdnsGetRecords(t *testing.T) {
	zones, err := p.getZones(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, zoneItem := range zones {
		records, err := p.GetRecords(ctx, zoneItem.Name)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(records)
	}
}

func TestLibdnsAppendSetDeleteRecords(t *testing.T) {
	zones, err := p.getZones(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, zoneItem := range zones {
		recs := []libdns.Record{}
		for i := 0; i < 3; i++ {
			data := generateRandInt(t)
			recs = append(recs, libdns.Record{
				Name:  fmt.Sprintf("test%d", data),
				Type:  "TXT",
				Value: fmt.Sprint(data),
			})
		}
		results, err := p.AppendRecords(ctx, zoneItem.Name, recs)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("AppendRecords returned: %+v", results)
		if len(results) != len(recs) {
			t.Fatal("AppendRecords: number of records returned not equal to records sent")
		}
		for i := range recs {
			data := generateRandInt(t)
			recs[i].Value = fmt.Sprint(data)
		}
		for i := 0; i < 2; i++ {
			data := generateRandInt(t)
			recs = append(recs, libdns.Record{
				Name:  fmt.Sprintf("test%d", data),
				Type:  "TXT",
				Value: fmt.Sprint(data),
			})
		}
		results, err = p.SetRecords(ctx, zoneItem.Name, recs)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("SetRecords returned: %+v", results)
		if len(results) != len(recs) {
			t.Fatal("SetRecords: number of records returned not equal to records sent")
		}
		results, err = p.DeleteRecords(ctx, zoneItem.Name, recs)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("DeleteRecords returned: %+v", results)
		if len(results) != len(recs) {
			t.Fatal("DeleteRecords: number of records returned not equal to records sent")
		}
	}
}
