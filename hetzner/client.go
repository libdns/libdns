package hetzner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/libdns/libdns"
)

type getAllRecordsResponse struct {
	Records []record `json:"records"`
}

type getAllZonesResponse struct {
	Zones []zone `json:"zones"`
}

type createRecordResponse struct {
	Record record `json:"record"`
}

type updateRecordResponse struct {
	Record record `json:"record"`
}

type zone struct {
	ID string `json:"id"`
}

type record struct {
	ID     string `json:"id,omitempty"`
	ZoneID string `json:"zone_id,omitempty"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Value  string `json:"value"`
	TTL    int    `json:"ttl"`
}

func doRequest(token string, request *http.Request) ([]byte, error) {
	request.Header.Add("Auth-API-Token", token)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%s (%d)", http.StatusText(response.StatusCode), response.StatusCode)
	}

	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func getZoneID(ctx context.Context, token string, zone string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://dns.hetzner.com/api/v1/zones?name=%s", url.QueryEscape(zone)), nil)
	data, err := doRequest(token, req)
	if err != nil {
		return "", err
	}

	result := getAllZonesResponse{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	if len(result.Zones) > 1 {
		return "", errors.New("zone is ambiguous")
	}

	return result.Zones[0].ID, nil
}

func getAllRecords(ctx context.Context, token string, zone string) ([]libdns.Record, error) {
	zoneID, err := getZoneID(ctx, token, zone)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://dns.hetzner.com/api/v1/records?zone_id=%s", zoneID), nil)
	data, err := doRequest(token, req)
	if err != nil {
		return nil, err
	}

	result := getAllRecordsResponse{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	records := []libdns.Record{}
	for _, r := range result.Records {
		records = append(records, libdns.Record{
			ID:    r.ID,
			Type:  r.Type,
			Name:  r.Name,
			Value: r.Value,
			TTL:   time.Duration(r.TTL) * time.Second,
		})
	}

	return records, nil
}

func createRecord(ctx context.Context, token string, zone string, r libdns.Record) (libdns.Record, error) {
	zoneID, err := getZoneID(ctx, token, zone)
	if err != nil {
		return libdns.Record{}, err
	}

	reqData := record{
		ZoneID: zoneID,
		Type:   r.Type,
		Name:   r.Name,
		Value:  r.Value,
		TTL:    int(r.TTL.Seconds()),
	}

	reqBuffer, err := json.Marshal(reqData)
	if err != nil {
		return libdns.Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://dns.hetzner.com/api/v1/records", bytes.NewBuffer(reqBuffer))
	data, err := doRequest(token, req)
	if err != nil {
		return libdns.Record{}, err
	}

	result := createRecordResponse{}
	if err := json.Unmarshal(data, &result); err != nil {
		return libdns.Record{}, err
	}

	return libdns.Record{
		ID:    result.Record.ID,
		Type:  result.Record.Type,
		Name:  result.Record.Name,
		Value: result.Record.Value,
		TTL:   time.Duration(result.Record.TTL) * time.Second,
	}, nil
}

func deleteRecord(ctx context.Context, token string, record libdns.Record) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("https://dns.hetzner.com/api/v1/records/%s", record.ID), nil)
	_, err = doRequest(token, req)
	if err != nil {
		return err
	}

	return nil
}

func updateRecord(ctx context.Context, token string, zone string, r libdns.Record) (libdns.Record, error) {
	zoneID, err := getZoneID(ctx, token, zone)
	if err != nil {
		return libdns.Record{}, err
	}

	reqData := record{
		ZoneID: zoneID,
		Type:   r.Type,
		Name:   r.Name,
		Value:  r.Value,
		TTL:    int(r.TTL.Seconds()),
	}

	reqBuffer, err := json.Marshal(reqData)
	if err != nil {
		return libdns.Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("https://dns.hetzner.com/api/v1/records/%s", r.ID), bytes.NewBuffer(reqBuffer))
	data, err := doRequest(token, req)
	if err != nil {
		return libdns.Record{}, err
	}

	result := updateRecordResponse{}
	if err := json.Unmarshal(data, &result); err != nil {
		return libdns.Record{}, err
	}

	return libdns.Record{
		ID:    result.Record.ID,
		Type:  result.Record.Type,
		Name:  result.Record.Name,
		Value: result.Record.Value,
		TTL:   time.Duration(result.Record.TTL) * time.Second,
	}, nil
}

func createOrUpdateRecord(ctx context.Context, token string, zone string, r libdns.Record) (libdns.Record, error) {
	if len(r.ID) == 0 {
		return createRecord(ctx, token, zone, r)
	}

	return updateRecord(ctx, token, zone, r)
}
