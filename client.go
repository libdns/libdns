package dnsexit

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/libdns/libdns"
	"github.com/pkg/errors"
)

const (
	// API URL to POST updates to
	updateURL = "https://api.dnsexit.com/dns/"
)

var (
	// Set environment variable to "TRUE" to enable debug logging
	debug  = (os.Getenv("LIBDNS_DNSEXIT_DEBUG") == "TRUE")
	client = resty.New()
)

// Query Google DNS for A/AAAA/TXT record for a given DNS name
func (p *Provider) getDomain(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var libRecords []libdns.Record

	// The API only supports adding/updating/deleting records and no way
	// to get current records. So instead, we just make
	// simple DNS queries to get the A, AAAA, and TXT records.
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 10 * time.Second}
			return d.DialContext(ctx, network, "8.8.8.8:53")
		},
	}

	ips, err := r.LookupHost(ctx, zone)
	if err != nil {
		var dnsErr *net.DNSError
		// Ignore missing dns record
		if !(errors.As(err, &dnsErr) && dnsErr.IsNotFound) {
			return libRecords, errors.Wrapf(err, "error looking up host")
		}
	}

	for _, ip := range ips {
		parsed, err := netip.ParseAddr(ip)
		if err != nil {
			return libRecords, errors.Wrapf(err, "error parsing ip")
		}

		if parsed.Is4() {
			libRecords = append(libRecords, libdns.Record{
				Type:  "A",
				Name:  "@",
				Value: ip,
			})
		} else {
			libRecords = append(libRecords, libdns.Record{
				Type:  "AAAA",
				Name:  "@",
				Value: ip,
			})
		}
	}

	txt, err := r.LookupTXT(ctx, zone)
	if err != nil {
		var dnsErr *net.DNSError
		// Ignore missing dns record
		if !(errors.As(err, &dnsErr) && dnsErr.IsNotFound) {
			return libRecords, errors.Wrapf(err, "error looking up txt")
		}
	}
	for _, t := range txt {
		if t == "" {
			continue
		}
		libRecords = append(libRecords, libdns.Record{
			Type:  "TXT",
			Name:  "@",
			Value: t,
		})
	}

	return libRecords, nil
}

// Set or clear the value of a DNS entry
func (p *Provider) amendRecords(zone string, records []libdns.Record, action Action) ([]libdns.Record, error) {

	var payloadRecords []dnsExitRecord
	p.mutex.Lock()
	defer p.mutex.Unlock()

	////////////////////////////////////////////////
	// BUILD PAYLOAD
	////////////////////////////////////////////////
	for _, record := range records {
		if record.TTL/time.Second < 600 {
			record.TTL = 600 * time.Second
		}
		ttlInSeconds := int(record.TTL / time.Second)

		relativeName := libdns.RelativeName(record.Name, zone)
		trimmedName := relativeName
		if relativeName == "@" {
			trimmedName = ""
		}

		currentRecord := dnsExitRecord{}
		currentRecord.Type = record.Type
		currentRecord.Name = trimmedName

		if action != Delete {
			recordValue := record.Value
			currentRecord.Content = &recordValue
			recordPriority := int(record.Priority)
			currentRecord.Priority = &recordPriority
			recordTTL := ttlInSeconds
			currentRecord.TTL = &recordTTL
		}
		if action == Set {
			truevalue := true
			currentRecord.Overwrite = &truevalue
		}
		payloadRecords = append(payloadRecords, currentRecord)
	}

	payload := dnsExitPayload{}
	payload.Apikey = p.APIKey
	payload.Zone = zone

	switch action {
	case Delete:
		payload.DeleteRecords = &payloadRecords
	case Set:
		fallthrough
	case Append:
		payload.AddRecords = &payloadRecords
	default:
		return nil, errors.New(fmt.Sprintf("Unknown action type: %d", action))
	}

	////////////////////////////////////////////////
	//SEND PAYLOAD
	////////////////////////////////////////////////

	// Explore response object
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if debug {
		fmt.Println("Request Info:")
		fmt.Println("Body:", string(reqBody))
	}
	// Make the API request to DNSExit
	// POST Struct, default is JSON content type. No need to set one
	resp, err := client.R().
		SetBody(payload).
		SetResult(&dnsExitResponse{}).
		SetError(&dnsExitResponse{}).
		Post(updateURL)

	if err != nil {
		return nil, err
	}

	//TODO - query the response code and text to determine which updates where successful, and return both records and response text in all cases, rather than just assuming all records for a 0 code and no records for other codes.

	// On any non-zero return code return the API response as the error text.
	if !isResposeStatusOK(resp.Body()) {
		respBody := string(resp.String())
		return nil, errors.New(fmt.Sprintf("API request failed, response=%s", respBody))
	}

	return records, nil
}

// Convert API response code to human friendly error
func isResposeStatusOK(body []byte) bool {
	var respJson dnsExitResponse
	json.Unmarshal(body, &respJson)
	return respJson.Code == 0
}
