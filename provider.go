package nicrudns

import (
	"context"
	"fmt"
	"github.com/libdns/libdns"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

// Provider facilitates DNS record manipulation with NIC.ru.
type Provider struct {
	OAuth2ClientID   string `json:"oauth2_client_id"`
	OAuth2SecretID   string `json:"oauth2_secret_id"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	NicRuServiceName string `json:"nic_ru_service_name"`
	CachePath        string `json:"cache_path"`
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	client := NewClient(&Config{
		Credentials: &Credentials{
			OAuth2ClientID: p.OAuth2ClientID,
			OAuth2SecretID: p.OAuth2SecretID,
			Username:       p.Username,
			Password:       p.Password,
		},
		ZoneName:       zone,
		DnsServiceName: p.NicRuServiceName,
		CachePath:      p.CachePath,
	})
	var records []libdns.Record
	rrs, err := client.GetRecords()
	if err != nil {
		return nil, err
	}
	for _, rr := range rrs {
		var ttl time.Duration
		if v, err := strconv.ParseInt(rr.Ttl, 10, 64); err != nil {
			ttl = time.Second * 0
		} else {
			ttl, _ = time.ParseDuration(fmt.Sprintf(`%ds`, v))
		}
		if rr.A != nil {
			records = append(records, libdns.Record{
				ID:    rr.ID,
				Type:  rr.Type,
				Name:  rr.Name,
				Value: rr.A.String(),
				TTL:   ttl,
			})
		}
		if rr.Cname != nil {
			records = append(records, libdns.Record{
				ID:    rr.ID,
				Type:  rr.Type,
				Name:  rr.Name,
				Value: rr.Cname.Name,
				TTL:   ttl,
			})
		}
		if rr.AAAA != nil {
			records = append(records, libdns.Record{
				ID:    rr.ID,
				Type:  rr.Type,
				Name:  rr.Name,
				Value: rr.AAAA.String(),
				TTL:   ttl,
			})
		}
		if rr.Txt != nil {
			records = append(records, libdns.Record{
				ID:    rr.ID,
				Type:  rr.Type,
				Name:  rr.Name,
				Value: rr.Txt.String,
				TTL:   ttl,
			})
		}
		if rr.Mx != nil {
			priority, err := strconv.ParseInt(rr.Mx.Preference, 10, 64)
			if err != nil {
				return nil, err
			}
			records = append(records, libdns.Record{
				ID:       rr.ID,
				Type:     rr.Type,
				Name:     rr.Name,
				Value:    rr.Mx.Exchange.Name,
				TTL:      ttl,
				Priority: int(priority),
			})
		}
	}
	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	client := NewClient(&Config{
		Credentials: &Credentials{
			OAuth2ClientID: p.OAuth2ClientID,
			OAuth2SecretID: p.OAuth2SecretID,
			Username:       p.Username,
			Password:       p.Password,
		},
		ZoneName:       zone,
		DnsServiceName: p.NicRuServiceName,
		CachePath:      p.CachePath,
	})
	var result []libdns.Record
	for _, record := range records {
		switch record.Type {
		case `A`:
			response, err := client.AddA([]string{record.Name}, record.Value, strconv.Itoa(int(record.TTL.Seconds())))
			if err != nil {
				return nil, err
			}
			result = append(result, libdns.Record{
				ID:    response.Data.Zone[0].Rr[0].ID,
				Type:  record.Type,
				Name:  response.Data.Zone[0].Rr[0].Name,
				Value: response.Data.Zone[0].Rr[0].A.String(),
				TTL:   record.TTL,
			})
		case `AAAA`:
			response, err := client.AddA([]string{record.Name}, record.Value, strconv.Itoa(int(record.TTL.Seconds())))
			if err != nil {
				return nil, err
			}
			result = append(result, libdns.Record{
				ID:    response.Data.Zone[0].Rr[0].ID,
				Type:  record.Type,
				Name:  response.Data.Zone[0].Rr[0].Name,
				Value: response.Data.Zone[0].Rr[0].AAAA.String(),
				TTL:   record.TTL,
			})
		case `CNAME`:
			response, err := client.AddCnames([]string{record.Name}, record.Value, strconv.Itoa(int(record.TTL.Seconds())))
			if err != nil {
				return nil, err
			}
			result = append(result, libdns.Record{
				ID:    response.Data.Zone[0].Rr[0].ID,
				Type:  record.Type,
				Name:  response.Data.Zone[0].Rr[0].Name,
				Value: response.Data.Zone[0].Rr[0].Cname.Name,
				TTL:   record.TTL,
			})
		case `MX`:
			response, err := client.AddCnames([]string{record.Name}, record.Value, strconv.Itoa(int(record.TTL.Seconds())))
			if err != nil {
				return nil, err
			}
			priority, err := strconv.ParseInt(response.Data.Zone[0].Rr[0].Mx.Preference, 10, 64)
			if err != nil {
				return nil, err
			}
			result = append(result, libdns.Record{
				ID:       response.Data.Zone[0].Rr[0].ID,
				Type:     record.Type,
				Name:     response.Data.Zone[0].Rr[0].Name,
				Value:    response.Data.Zone[0].Rr[0].Mx.Exchange.Name,
				TTL:      record.TTL,
				Priority: int(priority),
			})
		case `TXT`:
			response, err := client.AddCnames([]string{record.Name}, record.Value, strconv.Itoa(int(record.TTL.Seconds())))
			if err != nil {
				return nil, err
			}
			result = append(result, libdns.Record{
				ID:    response.Data.Zone[0].Rr[0].ID,
				Type:  record.Type,
				Name:  response.Data.Zone[0].Rr[0].Name,
				Value: response.Data.Zone[0].Rr[0].Txt.String,
				TTL:   record.TTL,
			})
		default:
			return nil, errors.Wrap(NotImplementedRecordType, record.Type)
		}
	}
	if _, err := client.CommitZone(); err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	client := NewClient(&Config{
		Credentials: &Credentials{
			OAuth2ClientID: p.OAuth2ClientID,
			OAuth2SecretID: p.OAuth2SecretID,
			Username:       p.Username,
			Password:       p.Password,
		},
		ZoneName:       zone,
		DnsServiceName: p.NicRuServiceName,
		CachePath:      p.CachePath,
	})
	allRecords, err := client.GetRecords()
	if err != nil {
		return nil, err
	}
	var result []libdns.Record
	for _, record := range records {
		//first delete exist records
		if rec := getRecordByID(record.ID, allRecords); rec != nil {
			id, err := strconv.ParseInt(rec.ID, 10, 64)
			if err != nil {
				return nil, err
			}
			if _, err := client.DeleteRecord(int(id)); err != nil {
				return nil, err
			}
		}
		// now add new records
		switch record.Type {
		case `A`:
			response, err := client.AddA([]string{record.Name}, record.Value, strconv.Itoa(int(record.TTL.Seconds())))
			if err != nil {
				return nil, err
			}
			result = append(result, libdns.Record{
				ID:    response.Data.Zone[0].Rr[0].ID,
				Type:  record.Type,
				Name:  response.Data.Zone[0].Rr[0].Name,
				Value: response.Data.Zone[0].Rr[0].A.String(),
				TTL:   record.TTL,
			})
		case `AAAA`:
			response, err := client.AddA([]string{record.Name}, record.Value, strconv.Itoa(int(record.TTL.Seconds())))
			if err != nil {
				return nil, err
			}
			result = append(result, libdns.Record{
				ID:    response.Data.Zone[0].Rr[0].ID,
				Type:  record.Type,
				Name:  response.Data.Zone[0].Rr[0].Name,
				Value: response.Data.Zone[0].Rr[0].AAAA.String(),
				TTL:   record.TTL,
			})
		case `CNAME`:
			response, err := client.AddCnames([]string{record.Name}, record.Value, strconv.Itoa(int(record.TTL.Seconds())))
			if err != nil {
				return nil, err
			}
			result = append(result, libdns.Record{
				ID:    response.Data.Zone[0].Rr[0].ID,
				Type:  record.Type,
				Name:  response.Data.Zone[0].Rr[0].Name,
				Value: response.Data.Zone[0].Rr[0].Cname.Name,
				TTL:   record.TTL,
			})
		case `TXT`:
			response, err := client.AddCnames([]string{record.Name}, record.Value, strconv.Itoa(int(record.TTL.Seconds())))
			if err != nil {
				return nil, err
			}
			result = append(result, libdns.Record{
				ID:    response.Data.Zone[0].Rr[0].ID,
				Type:  record.Type,
				Name:  response.Data.Zone[0].Rr[0].Name,
				Value: response.Data.Zone[0].Rr[0].Txt.String,
				TTL:   record.TTL,
			})
		case `MX`:
			response, err := client.AddCnames([]string{record.Name}, record.Value, strconv.Itoa(int(record.TTL.Seconds())))
			if err != nil {
				return nil, err
			}
			priority, err := strconv.ParseInt(response.Data.Zone[0].Rr[0].Mx.Preference, 10, 64)
			result = append(result, libdns.Record{
				ID:       response.Data.Zone[0].Rr[0].ID,
				Type:     record.Type,
				Name:     response.Data.Zone[0].Rr[0].Name,
				Value:    response.Data.Zone[0].Rr[0].Mx.Exchange.Name,
				TTL:      record.TTL,
				Priority: int(priority),
			})
		default:
			return nil, errors.Wrap(NotImplementedRecordType, record.Type)
		}
	}
	if _, err := client.CommitZone(); err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	client := NewClient(&Config{
		Credentials: &Credentials{
			OAuth2ClientID: p.OAuth2ClientID,
			OAuth2SecretID: p.OAuth2SecretID,
			Username:       p.Username,
			Password:       p.Password,
		},
		ZoneName:       zone,
		DnsServiceName: p.NicRuServiceName,
		CachePath:      p.CachePath,
	})
	var result []libdns.Record
	for _, record := range records {
		id, err := strconv.ParseInt(record.ID, 10, 64)
		if err != nil {
			return nil, err
		}
		if _, err := client.DeleteRecord(int(id)); err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	if _, err := client.CommitZone(); err != nil {
		return nil, err
	} else {
		return result, nil
	}
}

func getRecordByID(id string, records []*RR) *RR {
	for _, record := range records {
		if record.ID == id {
			return record
		}
	}
	return nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
