// Package easydns implements a DNS record management client compatible
// with the libdns interfaces for EasyDNS.
// See https://cp.easydns.com/manage/security/ to manage Token and Key information
// for your account.
package easydns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with EasyDNS.
type Provider struct {
	// EasyDNS API Token (required)
	APIToken string `json:"api_token,omitempty"`
	// EasyDNS API Key (required)
	APIKey string `json:"api_key,omitempty"`
	// EasyDNS API URL (defaults to https://rest.easydns.net)
	APIUrl string `json:"api_url,omitempty"`
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	log.Println("Get Records for zone:", zone)
	client := http.Client{}
	var records []libdns.Record
	resultObj := ZoneRecordResponse{}

	url := fmt.Sprintf("%s/zones/records/all/%s?format=json", p.getApiUrl(), zone)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(p.APIToken, p.APIKey)
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("could not get records for domain: %s, HTTP Status: %s", zone, resp.Status)
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(result, &resultObj)
	if err != nil {
		return nil, err
	}

	if len(resultObj.Data) == 0 {
		return nil, fmt.Errorf("no records found for domain: %s", zone)
	}

	for _, r := range resultObj.Data {
		ttl, err := strconv.Atoi(r.TTL)
		if err != nil {
			return nil, err
		}
		priority, err := strconv.Atoi(r.Priority)
		if err != nil {
			priority = 0
		}
		records = append(records, libdns.Record{
			ID:       r.Id,
			Type:     r.Type,
			Name:     r.Host,
			Value:    r.Rdata,
			TTL:      time.Duration(ttl) * time.Second,
			Priority: priority,
		})
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	log.Println("Append Record(s) to zone:", zone)
	var appendedRecords []libdns.Record

	for _, record := range records {
		client := http.Client{}

		if record.TTL < time.Duration(300)*time.Second {
			record.TTL = time.Duration(300) * time.Second
		}

		reqData, err := json.Marshal(AddEntry{
			Domain:   zone,
			Host:     record.Name,
			TTL:      int(record.TTL.Seconds()),
			Priority: record.Priority,
			Type:     record.Type,
			Rdata:    record.Value,
		})
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/zones/records/add/%s/%s",
			p.getApiUrl(), zone, record.Type), bytes.NewBuffer(reqData))
		if err != nil {
			return nil, err
		}

		req.SetBasicAuth(p.APIToken, p.APIKey)
		req.Header.Add("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return nil, fmt.Errorf("could not add record for domain: %s, HTTP Status: %s", zone, resp.Status)
		}

		_, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		appendedRecords = append(appendedRecords, record)

	}

	return appendedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	log.Println("Update Record(s) in zone:", zone)

	var updatedRecords []libdns.Record

	currentRecords, err := p.GetRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		updated := false
		for _, currentRecord := range currentRecords {
			if currentRecord.Name == record.Name && currentRecord.Type == record.Type {
				client := http.Client{}

				if record.TTL < time.Duration(300)*time.Second {
					record.TTL = time.Duration(300) * time.Second
				}

				reqData, err := json.Marshal(UpdateEntry{
					Host:  record.Name,
					TTL:   int(record.TTL.Seconds()),
					Type:  record.Type,
					Rdata: record.Value,
				})
				if err != nil {
					return nil, err
				}

				req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/zones/records/%s",
					p.getApiUrl(), currentRecord.ID), bytes.NewBuffer(reqData))
				if err != nil {
					return nil, err
				}

				req.SetBasicAuth(p.APIToken, p.APIKey)
				req.Header.Add("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return nil, fmt.Errorf("could not update record for domain: %s, HTTP Status: %s", zone, resp.Status)
				}

				_, err = io.ReadAll(resp.Body)
				if err != nil {
					return nil, err
				}
				updatedRecords = append(updatedRecords, record)
				updated = true
				break
			}
		}
		if !updated {
			added, err := p.AppendRecords(ctx, zone, []libdns.Record{record})
			if err != nil {
				return nil, err
			}
			updatedRecords = append(updatedRecords, added...)
		}
	}

	return updatedRecords, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	log.Println("Delete Record(s) from zone:", zone)
	var deletedRecords []libdns.Record

	currentRecords, err := p.GetRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	for _, record := range records {

		for _, currentRecord := range currentRecords {
			if currentRecord.Name == record.Name && currentRecord.Type == record.Type {
				client := http.Client{}

				req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/zones/records/%s/%s",
					p.getApiUrl(), zone, currentRecord.ID), nil)
				if err != nil {
					return nil, err
				}

				req.SetBasicAuth(p.APIToken, p.APIKey)
				req.Header.Add("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return nil, fmt.Errorf("could not delete record for domain: %s, HTTP Status: %s", zone, resp.Status)
				}

				_, err = io.ReadAll(resp.Body)
				if err != nil {
					return nil, err
				}
				deletedRecords = append(deletedRecords, record)
			}
		}
	}

	return deletedRecords, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)

func (p *Provider) getApiUrl() string {
	if p.APIUrl != "" {
		return p.APIUrl
	}
	return "https://rest.easydns.net"
}
