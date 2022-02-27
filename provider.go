// Package libdnstemplate implements a DNS record management client compatible
// with the libdns interfaces for Porkbun.
package libdns_porkbun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with Porkbun.
type Provider struct {
	APIKey       string `json:"api_key,omitempty"`
	APISecretKey string `json:"api_secret_key,omitempty"`
}

func (p *Provider) getApiHost() string {
	return "https://porkbun.com/api/json/v3/"
}

func (p *Provider) getRecordCoordinates(record libdns.Record) string {
	return fmt.Sprintf("%s-%s", record.Name, record.Type)
}

func (p *Provider) getCredentials() ApiCredentials {
	return ApiCredentials{p.APIKey, p.APISecretKey}
}

// Strips the trailing dot from a Zone
func trimZone(zone string) string {
	return strings.TrimSuffix(zone, ".")
}

func (p *Provider) CheckCredentials(ctx context.Context) (string, error) {
	client := http.Client{}

	credentialJson, err := json.Marshal(p.getCredentials())
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	req, err := http.NewRequest("POST", p.getApiHost()+"/ping", bytes.NewReader(credentialJson))

	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return "", errors.New(string(bodyBytes))
	}

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	resultObj := PingResponse{}

	err = json.Unmarshal(result, &resultObj)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	if resultObj.ResponseStatus.Status != "SUCCESS" {
		return "", resultObj.ResponseStatus
	}

	return resultObj.YourIP, nil
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	client := http.Client{}
	trimmedZone := trimZone(zone)

	credentialJson, err := json.Marshal(p.getCredentials())
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	req, err := http.NewRequest("POST", p.getApiHost()+"/dns/retrieve/"+trimmedZone, bytes.NewReader(credentialJson))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("could not get records: Zone: %s; Status: %v; Body: %s",
			trimmedZone, resp.StatusCode, string(bodyBytes))
	}

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	resultObj := ApiRecordsResponse{}

	err = json.Unmarshal(result, &resultObj)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	var records []libdns.Record

	for _, record := range resultObj.Records {
		ttl, err := time.ParseDuration(record.TTL + "s")
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		priority, _ := strconv.Atoi(record.Prio)
		records = append(records, libdns.Record{
			ID:       record.ID,
			Name:     record.Name,
			Priority: priority,
			TTL:      ttl,
			Type:     record.Type,
			Value:    record.Content,
		})
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	client := http.Client{}

	credentials := p.getCredentials()
	trimmedZone := trimZone(zone)

	var createdRecords []libdns.Record

	for _, record := range records {
		if record.TTL/time.Second < 600 {
			record.TTL = 600 * time.Second
		}
		ttlInSeconds := int(record.TTL / time.Second)
		trimmedName := libdns.RelativeName(record.Name, zone)
		reqBody := RecordCreateRequest{&credentials, record.Value, trimmedName, strconv.Itoa(ttlInSeconds), record.Type}
		reqJson, err := json.Marshal(reqBody)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/dns/create/%s", p.getApiHost(), trimmedZone), bytes.NewReader(reqJson))
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			return nil, fmt.Errorf("could not create record:(%s) in Zone: %s; Status: %v; Body: %s",
				fmt.Sprint(reqBody), trimmedZone, resp.StatusCode, string(bodyBytes))
		}

		result, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		resultObj := ResponseStatus{}

		err = json.Unmarshal(result, &resultObj)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		if resultObj.Status != "SUCCESS" {
			log.Fatal(resultObj)
			return nil, resultObj
		}
		createdRecords = append(createdRecords, record)
	}

	return createdRecords, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) UpdateRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	client := http.Client{}

	credentials := p.getCredentials()
	trimmedZone := trimZone(zone)

	var createdRecords []libdns.Record

	for _, record := range records {
		if record.TTL/time.Second < 600 {
			record.TTL = 600 * time.Second
		}
		ttlInSeconds := int(record.TTL / time.Second)
		trimmedName := libdns.RelativeName(record.Name, zone)
		reqBody := RecordUpdateRequest{&credentials, record.Value, strconv.Itoa(ttlInSeconds)}
		reqJson, err := json.Marshal(reqBody)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/dns/editByNameType/%s/%s/%s", p.getApiHost(), trimmedZone, record.Type, trimmedName), bytes.NewReader(reqJson))
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			return nil, fmt.Errorf("could not update record:(%s) in Zone: %s; Status: %v; Body: %s",
				fmt.Sprint(reqBody), trimmedZone, resp.StatusCode, string(bodyBytes))
		}

		result, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		resultObj := ResponseStatus{}

		err = json.Unmarshal(result, &resultObj)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		if resultObj.Status != "SUCCESS" {
			log.Fatal(resultObj)
			return nil, resultObj
		}
		createdRecords = append(createdRecords, record)
	}

	return createdRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	existingRecords, err := p.GetRecords(ctx, zone)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	existingCoordinates := NewSet()
	for _, r := range existingRecords {
		existingCoordinates.Add(p.getRecordCoordinates(r))
	}

	var updates []libdns.Record
	var creates []libdns.Record
	for _, r := range records {
		if existingCoordinates.Contains(p.getRecordCoordinates(r)) {
			updates = append(updates, r)
		} else {
			creates = append(creates, r)
		}
	}

	p.AppendRecords(ctx, zone, creates)
	p.UpdateRecords(ctx, zone, updates)

	return records, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	client := http.Client{}

	credentials := p.getCredentials()
	trimmedZone := trimZone(zone)

	var deletedRecords []libdns.Record

	for _, record := range records {
		reqJson, err := json.Marshal(credentials)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		trimmedName := libdns.RelativeName(record.Name, zone)

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/dns/deleteByNameType/%s/%s/%s", p.getApiHost(), trimmedZone, record.Type, trimmedName), bytes.NewReader(reqJson))
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			return nil, fmt.Errorf("could not delete record:(Type: %s Name: %s) in Zone: %s; Status: %v; Body: %s",
				record.Type, record.Name, trimmedZone, resp.StatusCode, string(bodyBytes))
		}

		result, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}

		resultObj := ResponseStatus{}

		err = json.Unmarshal(result, &resultObj)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		deletedRecords = append(deletedRecords, record)
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
