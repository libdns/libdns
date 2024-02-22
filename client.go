package libdnsdnsmadeeasy

import (
	"context"
	"fmt"
	"strconv"
	"time"

	dme "github.com/john-k/dnsmadeeasy"
	"github.com/libdns/libdns"
)

func (p *Provider) init(ctx context.Context) {
	p.once.Do(func() {
		p.client = *dme.GetClient(
			ctx.Value("APIKey").(string),
			ctx.Value("SecretKey").(string),
			ctx.Value("APIEndpoint").(dme.BaseURL),
		)
	})
}

func recordFromDmeRecord(dmeRecord dme.Record) libdns.Record {
	var rec libdns.Record
	rec.ID = fmt.Sprint(dmeRecord.ID)
	rec.Type = dmeRecord.Type
	rec.Name = dmeRecord.Name
	rec.Value = dmeRecord.Value
	rec.TTL = time.Duration(dmeRecord.Ttl)

	// TODO: enable support for SRV weight field and embedding
	// "<port> <target>" in value when libdns releases support
	if dmeRecord.Type == "MX" {
		rec.Priority = dmeRecord.MxLevel
	} else if dmeRecord.Type == "SRV" {
		rec.Priority = dmeRecord.Priority
		//rec.Weight = dmeRecord.Weight
		//rec.Value = fmt.Sprintf("%d %s", dmeRecord.Port, dmeRecord.Value)
	}

	return rec
}

func dmeRecordFromRecord(record libdns.Record) (dme.Record, error) {
	var dmeRecord dme.Record
	id, err := strconv.Atoi(record.ID)
	if err != nil {
		return dme.Record{}, err
	}
	dmeRecord.ID = id
	dmeRecord.Name = record.Name
	dmeRecord.Type = record.Type
	dmeRecord.Value = record.Value
	dmeRecord.Ttl = int(record.TTL)
	if record.Type == "MX" {
		dmeRecord.MxLevel = record.Priority
	} else if record.Type == "SRV" {
		dmeRecord.Priority = record.Priority
		/*
			// TODO: enable support for SRV weight field and extracting
			// "<port> <target>" from value when libdns releases support
			dmeRecord.Weight = record.Weight
			fields := strings.Fields(record.Value)
			if len(fields) != 2 {
				return dme.Record{}, fmt.Errorf("malformed SRV value '%s'; expected: '<port> <target>'", record.Value)
			}

			port, err := strconv.Atoi(fields[0])
			if err != nil {
				return dme.Record{}, fmt.Errorf("invalid port %s: %v", fields[0], err)
			}
			if port < 0 {
				return dme.Record{}, fmt.Errorf("port cannot be < 0: %d", port)
			}
			dmeRecord.Port = port
			dmeRecord.Value = fields[1]
		*/
	}
	return dmeRecord, nil

}
