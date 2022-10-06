package njalla

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/libdns/libdns"
)

func doRequest(token string, request *http.Request) ([]byte, error) {
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Njalla "+token)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func getAllRecords(ctx context.Context, token string, zone string) ([]libdns.Record, error) {
	body, err := json.Marshal(NjallaRequest{Method: "list-records", Params: struct {
		Domain string `json:"domain"`
	}{Domain: zone}})
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", "https://njal.la/api/1/", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	data, err := doRequest(token, request)
	if err != nil {
		return nil, err
	}

	result := struct {
		Result struct {
			Records []NjallaRecord `json:"records"`
		} `json:"result"`
	}{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	records := []libdns.Record{}
	for _, record := range result.Result.Records {
		records = append(records, libdns.Record{
			ID:    record.ID,
			Type:  record.Type,
			Name:  record.Name,
			Value: record.Content,
			TTL:   time.Duration(time.Duration(record.TTL).Seconds()),
		})
	}
	return records, nil
}

func createRecord(ctx context.Context, token string, zone string, record libdns.Record) (libdns.Record, error) {
	body, err := json.Marshal(NjallaRequest{Method: "add-record", Params: struct {
		Domain  string `json:"domain"`
		Name    string `json:"name"`
		Content string `json:"content"`
		TTL     int    `json:"ttl"`
		Type    string `json:"type"`
	}{
		Domain:  zone,
		Name:    record.Name,
		Content: record.Value,
		TTL:     int(record.TTL),
		Type:    record.Type,
	}})
	if err != nil {
		return libdns.Record{}, err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", "https://njal.la/api/1/", bytes.NewBuffer(body))
	if err != nil {
		return libdns.Record{}, err
	}

	data, err := doRequest(token, request)
	if err != nil {
		return libdns.Record{}, err
	}

	result := struct {
		Result NjallaRecord `json:"result"`
	}{}
	if err := json.Unmarshal(data, &result); err != nil {
		return libdns.Record{}, err
	}
	log.Println(result)
	return libdns.Record{
		ID:    result.Result.ID,
		Type:  result.Result.Type,
		Name:  result.Result.Name,
		Value: result.Result.Content,
		TTL:   time.Duration(time.Duration(result.Result.TTL).Seconds()),
	}, nil
}

func editRecord(ctx context.Context, token string, zone string, record libdns.Record) (libdns.Record, error) {
	body, err := json.Marshal(NjallaRequest{Method: "edit-record", Params: struct {
		Domain  string `json:"domain"`
		ID      string `json:"id"`
		Content string `json:"content"`
	}{
		Domain:  zone,
		ID:      record.ID,
		Content: record.Value,
	}})
	if err != nil {
		return libdns.Record{}, err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", "https://njal.la/api/1/", bytes.NewBuffer(body))
	if err != nil {
		return libdns.Record{}, err
	}

	data, err := doRequest(token, request)
	if err != nil {
		return libdns.Record{}, err
	}

	result := struct {
		Result NjallaRecord `json:"result"`
	}{}
	if err := json.Unmarshal(data, &result); err != nil {
		return libdns.Record{}, err
	}
	log.Println(result)
	return libdns.Record{
		ID:    result.Result.ID,
		Type:  result.Result.Type,
		Name:  result.Result.Name,
		Value: result.Result.Content,
		TTL:   time.Duration(time.Duration(result.Result.TTL).Seconds()),
	}, nil
}

func removeRecord(ctx context.Context, token string, zone string, record libdns.Record) error {
	body, err := json.Marshal(NjallaRequest{Method: "remove-record", Params: struct {
		Domain string `json:"domain"`
		ID     string `json:"id"`
	}{
		Domain: zone,
		ID:     record.ID,
	}})
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", "https://njal.la/api/1/", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	_, err = doRequest(token, request)
	return err
}

func createOrEditRecord(ctx context.Context, token string, zone string, record libdns.Record) (libdns.Record, error) {
	if len(record.ID) == 0 {
		return createRecord(ctx, token, zone, record)
	}
	return editRecord(ctx, token, zone, record)
}
