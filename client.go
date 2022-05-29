package netlify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/libdns/libdns"
)

// createRecord creates a DNS record in the specified zone. It returns the DNS
// record created
func (p *Provider) createRecord(ctx context.Context, zoneInfo netlifyZone, record libdns.Record) (netlifyDNSRecord, error) {
	jsonBytes, err := json.Marshal(netlifyRecord(record))
	if err != nil {
		return netlifyDNSRecord{}, err
	}
	reqURL := fmt.Sprintf("%s/dns_zones/%s/dns_records", baseURL, zoneInfo.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return netlifyDNSRecord{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	var result netlifyDNSRecord
	err = p.doAPIRequest(req, false, false, false, true, &result)
	if err != nil {
		return netlifyDNSRecord{}, err
	}

	return result, nil
}

// updateRecord updates a DNS record. oldRec must have both an ID and zone ID.
// Only the non-empty fields in newRec will be changed.
func (p *Provider) updateRecord(ctx context.Context, oldRec netlifyDNSRecord, newRec netlifyDNSRecord) (netlifyDNSRecord, error) {
	reqURL := fmt.Sprintf("%s/dns_zones/%s/dns_records/%s", baseURL, oldRec.DNSZoneID, oldRec.ID)
	jsonBytes, err := json.Marshal(newRec)
	if err != nil {
		return netlifyDNSRecord{}, err
	}

	// PATCH changes only the populated fields; PUT resets Type, Name, Content, and TTL even if empty
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, reqURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return netlifyDNSRecord{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	var result netlifyDNSRecord
	err = p.doAPIRequest(req, false, false, false, true, &result)
	return result, err
}

// getDNSRecords gets all record in a zone. It returns an array of the records
// in the zone
func (p *Provider) getDNSRecords(ctx context.Context, zoneInfo netlifyZone, rec libdns.Record, matchContent bool) ([]netlifyDNSRecord, error) {
	qs := make(url.Values)
	qs.Set("type", rec.Type)
	qs.Set("name", libdns.AbsoluteName(rec.Name, zoneInfo.Name))
	if matchContent {
		qs.Set("content", rec.Value)
	}

	reqURL := fmt.Sprintf("%s/dns_zones/%s/dns_records", baseURL, zoneInfo.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	var results []netlifyDNSRecord
	err = p.doAPIRequest(req, false, false, true, false, &results)
	var rest_to_return []netlifyDNSRecord
	for _, res := range results {
		if res.Hostname == libdns.AbsoluteName(rec.Name, zoneInfo.Name) && res.Type == rec.Type {
			rest_to_return = append(rest_to_return, res)
		}
	}
	if len(rest_to_return) == 0 {
		return nil, fmt.Errorf("Can't find DNS record %s", libdns.AbsoluteName(rec.Name, zoneInfo.Name))
	}
	if err != nil {
		return nil, err
	}
	return rest_to_return, nil
}

// getZoneInfo get the information from a DNS zone. It returns the dns zone
func (p *Provider) getZoneInfo(ctx context.Context, zoneName string) (netlifyZone, error) {
	p.zonesMu.Lock()
	defer p.zonesMu.Unlock()

	// if we already got the zone info, reuse it
	if p.zones == nil {
		p.zones = make(map[string]netlifyZone)
	}
	if zone, ok := p.zones[zoneName]; ok {
		return zone, nil
	}

	qs := make(url.Values)
	qs.Set("name", zoneName)
	reqURL := fmt.Sprintf("%s/dns_zones?%s", baseURL, qs.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return netlifyZone{}, err
	}

	var zones []netlifyZone
	err = p.doAPIRequest(req, true, false, true, false, &zones)
	if err != nil {
		return netlifyZone{}, err
	}
	if len(zones) != 1 {
		return netlifyZone{}, fmt.Errorf("expected 1 zone, got %d for %s", len(zones), zoneName)
	}

	// cache this zone for possible reuse
	p.zones[zoneName] = zones[0]

	return zones[0], nil
}

// doAPIRequest authenticates the request req and does the round trip. It returns
// nil if there was no error, the error otherwise. The decoded content is passed
// to the calling function by the result variable
func (p *Provider) doAPIRequest(req *http.Request, isZone bool, isDel bool, isGet bool, isSolo bool, result interface{}) error {
	req.Header.Set("Authorization", "Bearer "+p.PersonalAccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("got error status: HTTP %d: %+v", resp.StatusCode, string(body))
	}

	// delete DNS record
	if isDel && !isZone {
		if len(body) > 0 {
			var err netlifyDNSDeleteError
			json.Unmarshal(body, &result)
			return fmt.Errorf(err.Message)
		}
		return err
	}

	// get zone info
	if isZone && isGet {
		err = json.Unmarshal(body, &result)
		if err != nil {
			return err
		}
		return err
	}

	// get DNS records
	if !isZone && isGet && !isSolo {
		err = json.Unmarshal(body, &result)
		if err != nil {
			return err
		}
		return err
	}

	// get DNS record
	if !isZone && isGet && isSolo {
		err = json.Unmarshal(body, &result)
		if err != nil {
			return err
		}
		return err
	}

	// update DNS record
	if !isZone && isSolo && !isGet {
		err = json.Unmarshal(body, &result)
		if err != nil {
			return nil
		}
		return err
	}

	return err
}

const baseURL = "https://api.netlify.com/api/v1"
