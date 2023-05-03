package directadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/libdns/libdns"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (p *Provider) getZoneRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	reqURL, err := url.Parse(p.ServerURL)
	if err != nil {
		return nil, err
	}

	reqURL.Path = "/CMD_API_DNS_CONTROL"

	queryString := make(url.Values)
	queryString.Set("json", "yes")
	queryString.Set("full_mx_records", "yes")
	queryString.Set("allow_dns_underscore", "yes")
	queryString.Set("ttl", "yes")
	queryString.Set("domain", zone)

	reqURL.RawQuery = queryString.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(p.User, p.LoginKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	var respData daZone
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return nil, err
	}

	recs := make([]libdns.Record, 0, len(respData.Records))
	for i := range respData.Records {
		libDnsRecord, err := respData.Records[i].libdnsRecord(zone)
		if err != nil {
			switch err {
			case ErrUnsupported:
				fmt.Printf("unsupported record conversion of type %v: %v", libDnsRecord.Type, libDnsRecord.Name)
				continue
			default:
				return nil, err
			}
		}
		recs = append(recs, libDnsRecord)
	}

	return recs, nil
}

func (p *Provider) appendZoneRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	reqURL, err := url.Parse(p.ServerURL)
	if err != nil {
		return libdns.Record{}, err
	}

	reqURL.Path = "/CMD_API_DNS_CONTROL"

	queryString := make(url.Values)
	queryString.Set("action", "add")
	queryString.Set("json", "yes")
	queryString.Set("full_mx_records", "yes")
	queryString.Set("allow_dns_underscore", "yes")
	queryString.Set("domain", zone)
	queryString.Set("type", record.Type)
	queryString.Set("name", record.Name)
	queryString.Set("value", record.Value)

	if record.Type != "NS" {
		queryString.Set("ttl", strconv.Itoa(int(record.TTL.Seconds())))
	}

	reqURL.RawQuery = queryString.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return libdns.Record{}, err
	}

	req.SetBasicAuth(p.User, p.LoginKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return libdns.Record{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close body: %v", err)
		}
	}(resp.Body)

	var respData daResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return libdns.Record{}, err
	}

	if len(respData.Error) > 0 {
		return libdns.Record{}, fmt.Errorf("%v: %v", respData.Error, respData.Result)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		log.Println(bodyString)

		return libdns.Record{}, err
	}

	record.ID = fmt.Sprintf("name=%v&value=%v", record.Name, record.Value)

	return record, nil
}

func (p *Provider) setZoneRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	reqURL, err := url.Parse(p.ServerURL)
	if err != nil {
		return libdns.Record{}, err
	}

	reqURL.Path = "/CMD_API_DNS_CONTROL"

	queryString := make(url.Values)
	queryString.Set("action", "edit")
	queryString.Set("json", "yes")
	queryString.Set("domain", zone)
	queryString.Set("type", record.Type)
	queryString.Set("name", record.Name)
	queryString.Set("value", record.Value)

	if record.Type != "NS" {
		queryString.Set("ttl", strconv.Itoa(int(record.TTL.Seconds())))
	}

	existingRecords, _ := p.getZoneRecords(ctx, zone)
	var existingRecordIndex = -1
	for i := range existingRecords {
		if existingRecords[i].Name == record.Name && existingRecords[i].Type == record.Type {
			existingRecordIndex = i
			break
		}
	}

	// If we're not -1, we found a matching existing record. This changes the API call
	// from create only to edit.
	if existingRecordIndex != -1 {
		editKey := fmt.Sprintf("%vrecs0", strings.ToLower(record.Type))
		editValue := existingRecords[existingRecordIndex].ID
		queryString.Set(editKey, editValue)
	}

	reqURL.RawQuery = queryString.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return libdns.Record{}, err
	}

	req.SetBasicAuth(p.User, p.LoginKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return libdns.Record{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close body: %v", err)
		}
	}(resp.Body)

	var respData daResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return libdns.Record{}, err
	}

	if len(respData.Error) > 0 {
		return libdns.Record{}, fmt.Errorf("%v: %v", respData.Error, respData.Result)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		log.Println(bodyString)

		return libdns.Record{}, err
	}

	record.ID = fmt.Sprintf("name=%v&value=%v", record.Name, record.Value)

	return record, nil
}

func (p *Provider) deleteZoneRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	reqURL, err := url.Parse(p.ServerURL)
	if err != nil {
		return libdns.Record{}, err
	}

	reqURL.Path = "/CMD_API_DNS_CONTROL"

	queryString := make(url.Values)
	queryString.Set("action", "select")
	queryString.Set("json", "yes")
	queryString.Set("domain", zone)

	editKey := fmt.Sprintf("%vrecs0", strings.ToLower(record.Type))
	editValue := fmt.Sprintf("name=%v&value=%v", record.Name, record.Value)
	queryString.Set(editKey, editValue)

	reqURL.RawQuery = queryString.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return libdns.Record{}, err
	}

	req.SetBasicAuth(p.User, p.LoginKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return libdns.Record{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close body: %v", err)
		}
	}(resp.Body)

	var respData daResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return libdns.Record{}, err
	}

	if len(respData.Error) > 0 {
		return libdns.Record{}, fmt.Errorf("%v: %v", respData.Error, respData.Result)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)
		log.Println(bodyString)

		return libdns.Record{}, err
	}

	return record, nil
}
