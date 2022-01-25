package godaddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

// Provider godaddy dns provider
type Provider struct {
	APIToken string
}

func getDomain(zone string) string {
	return strings.TrimSuffix(zone, ".")
}

func getRecordName(zone, name string) string {
	return strings.TrimSuffix(strings.TrimSuffix(name, zone), ".")
}

func (p *Provider) getApiHost() string {
	return "https://api.godaddy.com"
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	log.Println("GetRecords", zone)
	client := http.Client{}

	domain := getDomain(zone)

	req, err := http.NewRequest("GET", p.getApiHost()+"/v1/domains/"+domain+"/records", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "sso-key "+p.APIToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("could not get records: Domain: %s; Status: %v; Body: %s",
			domain, resp.StatusCode, string(bodyBytes))
	}

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	resultObj := []struct {
		Type  string `json:"type"`
		Name  string `json:"name"`
		Value string `json:"data"`
		TTL   int    `json:"ttl"`
	}{}

	err = json.Unmarshal(result, &resultObj)
	if err != nil {
		return nil, err
	}

	var records []libdns.Record

	for _, record := range resultObj {
		records = append(records, libdns.Record{
			Type:  record.Type,
			Name:  record.Name,
			Value: record.Value,
			TTL:   time.Duration(record.TTL) * time.Second,
		})
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	log.Println("AppendRecords", zone, records)
	var appendedRecords []libdns.Record

	for _, record := range records {
		client := http.Client{}

		type PostRecord struct {
			Data string `json:"data"`
		}

		if record.TTL < time.Duration(600)*time.Second {
			record.TTL = time.Duration(600) * time.Second
		}

		data, err := json.Marshal([]PostRecord{
			{
				Data: record.Value,
			},
		})
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("PUT", p.getApiHost()+"/v1/domains/"+getDomain(zone)+"/records/"+record.Type+"/"+getRecordName(zone, record.Name), bytes.NewBuffer(data))
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", "sso-key "+p.APIToken)
		req.Header.Add("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			return nil, fmt.Errorf("could not append records: Domain: %s; Record: %s, Status: %v; Body: %s; PUT: %s",
				getDomain(zone), getRecordName(zone, record.Name), resp.StatusCode, string(bodyBytes), data)
		}

		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		appendedRecords = append(appendedRecords, record)
	}

	return appendedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records
// or creating new ones. It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	log.Println("SetRecords", zone, records)
	return p.AppendRecords(ctx, zone, records)
}

// DeleteRecords deletes the records from the zone.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	log.Println("DeleteRecords", zone, records)

	currentRecords, err := p.GetRecords(ctx, zone)
	if err != nil {
		return nil, err
	}

	var deletedRecords []libdns.Record

	for _, record := range records {
		for i, currentRecord := range currentRecords {
			if currentRecord.Type == record.Type && currentRecord.Name == getRecordName(zone, record.Name) {
				currentRecords = append(currentRecords[:i], currentRecords[i+1:]...)
				deletedRecords = append(deletedRecords, currentRecord)
				break
			}
		}
	}

	type PostRecord struct {
		Data string `json:"data"`
		Name string `json:"name"`
		TTL  int    `json:"ttl"`
		Type string `json:"type"`
	}

	var data []PostRecord

	for _, record := range currentRecords {
		data = append(data, PostRecord{
			Data: record.Value,
			Name: record.Name,
			TTL:  int(record.TTL / time.Second),
			Type: record.Type,
		})
	}

	dataByte, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", p.getApiHost()+"/v1/domains/"+getDomain(zone)+"/records", bytes.NewBuffer(dataByte))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "sso-key "+p.APIToken)
	req.Header.Add("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("could not delete records: Domain: %s; Records: %v, Status: %v; Body: %s",
			zone, currentRecords, resp.StatusCode, string(bodyBytes))
	}

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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
