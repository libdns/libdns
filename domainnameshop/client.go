package domainnameshop

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

const defaultBaseURL string = "https://api.domeneshop.no/v0"

// We set a default ttl that's used if TTL is not specified by other users
// By default domainname.shop uses 1 hour long TTL which might be too long in a lot of usecases
// The api specifies that TTL must be in seconds but also in must multiples of 60
const defaultTtl = time.Duration(2 * time.Minute)

// Domain JSON data structure.
type Domain struct {
	Name           string   `json:"domain"`
	ID             int      `json:"id"`
	ExpiryDate     string   `json:"expiry_date"`
	Nameservers    []string `json:"nameservers"`
	RegisteredDate string   `json:"registered_date"`
	Registrant     string   `json:"registrant"`
	Renew          bool     `json:"renew"`
	Services       Service  `json:"services"`
	Status         string
}

type Service struct {
	DNS       bool   `json:"dns"`
	Email     bool   `json:"email"`
	Registrar bool   `json:"registrar"`
	Webhotel  string `json:"webhotel"`
}

// DNSRecord JSON data structure.
type DNSRecord struct {
	ID   int    `json:"id,omitempty"`
	Host string `json:"host"`
	Data string `json:"data"`
	Type string `json:"type"`
	TTL  int    `json:"ttl"`
}

func doRequest(token string, secret string, request *http.Request) ([]byte, error) {
	request.SetBasicAuth(token, secret)
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%s (%d)", http.StatusText(response.StatusCode), response.StatusCode)
	}

	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func getDomainID(ctx context.Context, token string, secret string, zone string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(defaultBaseURL+"/domains?domain=%s", url.QueryEscape(removeFQDNTrailingDot(zone))), nil)
	if err != nil {
		return "", err
	}
	data, err := doRequest(token, secret, req)
	if err != nil {
		return "", err
	}

	var result []Domain
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	if len(result) > 1 {
		return "", errors.New("zone is ambiguous")
	}

	return strconv.Itoa(result[0].ID), nil
}

func getDNSRecord(ctx context.Context, token string, secret string, domainID string, recordID int) (libdns.Record, error) {
	var result DNSRecord
	reqBufferGet, err := json.Marshal(result)
	if err != nil {
		return libdns.Record{}, err
	}

	reqGet, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(defaultBaseURL+"/domains/%s/dns/%d", domainID, recordID), bytes.NewBuffer(reqBufferGet))
	if err != nil {
		return libdns.Record{}, err
	}
	dataGet, err := doRequest(token, secret, reqGet)
	if err != nil {
		return libdns.Record{}, err
	}

	if err := json.Unmarshal(dataGet, &result); err != nil {
		return libdns.Record{}, err
	}

	return libdns.Record{
		ID:    strconv.Itoa(result.ID),
		Type:  result.Type,
		Name:  result.Host,
		Value: result.Data,
		TTL:   time.Duration(result.TTL) * time.Second,
	}, nil
}

func deleteDNSRecord(ctx context.Context, token string, secret string, record libdns.Record, zone string) error {
	domainID, err := getDomainID(ctx, token, secret, zone)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf(defaultBaseURL+"/domains/%s/dns/%s", domainID, record.ID), nil)
	if err != nil {
		return err
	}
	_, err = doRequest(token, secret, req)
	if err != nil {
		return err
	}

	return nil
}

func getAllDomainRecords(ctx context.Context, token string, secret string, zone string) ([]libdns.Record, error) {
	domainID, err := getDomainID(ctx, token, secret, zone)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(defaultBaseURL+"/domains/%s/dns", domainID), nil)
	if err != nil {
		return nil, err
	}
	data, err := doRequest(token, secret, req)
	if err != nil {
		return nil, err
	}

	var result []DNSRecord
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	records := []libdns.Record{}
	for _, r := range result {
		records = append(records, libdns.Record{
			ID:    strconv.Itoa(r.ID),
			Type:  r.Type,
			Name:  r.Host,
			Value: r.Data,
			TTL:   time.Duration(r.TTL) * time.Second,
		})
	}

	return records, nil
}

func createDNSRecord(ctx context.Context, token string, secret string, zone string, r libdns.Record) (libdns.Record, error) {
	domainID, err := getDomainID(ctx, token, secret, zone)
	if err != nil {
		return libdns.Record{}, err
	}

	reqData := DNSRecord{
		Type: r.Type,
		Host: normalizeRecordName(r.Name, zone),
		Data: r.Value,
		TTL:  int(r.TTL.Seconds()),
	}
	if reqData.TTL == 0 {
		reqData.TTL = int(defaultTtl.Seconds())
	}

	reqBuffer, err := json.Marshal(reqData)
	if err != nil {
		return libdns.Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf(defaultBaseURL+"/domains/%s/dns", domainID), bytes.NewBuffer(reqBuffer))
	if err != nil {
		return libdns.Record{}, err
	}
	data, err := doRequest(token, secret, req)
	if err != nil {
		return libdns.Record{}, err
	}
	var result DNSRecord
	if err := json.Unmarshal(data, &result); err != nil {
		return libdns.Record{}, err
	}

	return getDNSRecord(ctx, token, secret, domainID, result.ID)
}

func updateDNSRecord(ctx context.Context, token string, secret string, zone string, r libdns.Record) (libdns.Record, error) {
	domainID, err := getDomainID(ctx, token, secret, zone)
	if err != nil {
		return libdns.Record{}, err
	}

	id, err := strconv.Atoi(r.ID)
	if err != nil {
		return libdns.Record{}, err
	}

	reqData := DNSRecord{
		Type: r.Type,
		Host: normalizeRecordName(r.Name, zone),
		Data: r.Value,
		TTL:  int(r.TTL.Seconds()),
	}
	if reqData.TTL == 0 {
		reqData.TTL = int(defaultTtl.Seconds())
	}

	reqBuffer, err := json.Marshal(reqData)
	if err != nil {
		return libdns.Record{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf(defaultBaseURL+"/domains/%s/dns/%d", domainID, id), bytes.NewBuffer(reqBuffer))
	if err != nil {
		return libdns.Record{}, err
	}
	data, err := doRequest(token, secret, req)
	if err != nil {
		return libdns.Record{}, err
	}
	_ = data

	return getDNSRecord(ctx, token, secret, domainID, id)
}

func createOrUpdateDNSRecord(ctx context.Context, token string, secret string, zone string, r libdns.Record) (libdns.Record, error) {
	if len(r.ID) == 0 {
		return createDNSRecord(ctx, token, secret, zone, r)
	}

	return updateDNSRecord(ctx, token, secret, zone, r)
}
func removeFQDNTrailingDot(fqdn string) string {
	return strings.TrimSuffix(fqdn, ".")
}
func normalizeRecordName(recordName string, zone string) string {
	normalized := removeFQDNTrailingDot(recordName)
	normalized = strings.TrimSuffix(normalized, removeFQDNTrailingDot(zone))
	if normalized == "" {
		normalized = "@"
	}
	return removeFQDNTrailingDot(normalized)
}
