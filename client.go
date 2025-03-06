package neoserv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/libdns/libdns"
	"github.com/pkg/errors"
)

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

var (
	urlBase         = "https://moj.neoserv.si"
	urlBaseP        = mustParseURL(urlBase)
	urlLogin        = urlBase + "/prijava/preveri"
	urlZones        = urlBase + "/storitve"
	urlZone         = urlBase + "/storitve/domena/dns"
	urlAddRecord    = urlBase + "/storitve/domena/shranidnszapis"
	urlEditRecord   = urlBase + "/storitve/domena/popravizapis"
	urlDeleteRecord = urlBase + "/storitve/domena/odstranizapis"
)

// init initializes the Provider with an HTTP client and caching.
func (p *Provider) init() error {
	if p.client != nil {
		return nil
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	p.client = &http.Client{
		Jar: jar,
	}
	p.zoneIdCache = make(map[string]string)
	return nil
}

// authenticate starts a new session and authenticates with the Neoserv.
func (p *Provider) authenticate(ctx context.Context) error {
	// Initialize the provider if it hasn't been already.
	if err := p.init(); err != nil {
		return errors.Wrap(err, "failed to initialize provider")
	}

	// Check if "avt12" cookie is already set and not expired.
	// If it is, we don't need to authenticate again.
	cookies := p.client.Jar.Cookies(urlBaseP)
	for _, cookie := range cookies {
		if cookie.Name == "avt12" {
			// If the cookie is expired, remove it and authenticate again.
			if cookie.Expires.After(time.Now()) {
				jar, err := cookiejar.New(nil)
				if err != nil {
					return errors.Wrap(err, "failed to create new cookie jar")
				}
				p.client.Jar = jar
				return p.authenticate(ctx)
			}
		}
	}

	// Prepare form data for authentication.
	form := url.Values{}
	form.Set("email", p.Username)
	form.Set("password", p.Password)

	// Create a new POST request.
	req, err := http.NewRequestWithContext(ctx, "POST", urlLogin, strings.NewReader(form.Encode()))
	if err != nil {
		return errors.Wrap(err, "failed to create authentication request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Perform the request.
	resp, err := p.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to perform authentication request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	// If Refresh header is set, authentication failed. This is a Neoserv-specific behavior.
	// We can't rely on the status code alone to determine if authentication succeeded.
	if resp.Header.Get("Refresh") != "" {
		// We could parse the Refresh header to get the error message, but for now we'll just return a generic error.
		return fmt.Errorf("authentication failed")
	}

	return nil
}

// getZoneID returns the zone ID for the given zone name.
func (p *Provider) getZoneID(ctx context.Context, zone string) (string, error) {
	// Authenticate if necessary.
	if err := p.authenticate(ctx); err != nil {
		return "", errors.Wrap(err, "failed to get zone ID")
	}

	// Check if the zone ID is already cached.
	if id, ok := p.zoneIdCache[zone]; ok {
		return id, nil
	}

	// Perform a GET request to get the list of zones.
	resp, err := p.client.Get(urlZones)
	if err != nil {
		return "", errors.Wrap(err, "failed to get zone ID: failed to get zones")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get zone ID: zones status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to get zone ID: failed to parse zones")
	}

	found := false
	doc.Find("a[href]").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if !strings.Contains(s.Text(), fmt.Sprintf("domena - %s", zone)) {
			return true
		}
		href, ok := s.Attr("href")
		if !ok || !strings.HasPrefix(href, "/storitve/domena/") {
			return true
		}
		parts := strings.Split(href, "/")
		if len(parts) != 4 {
			return true
		}
		p.zoneIdCache[zone] = parts[3]
		found = true
		return false
	})
	if !found {
		return "", fmt.Errorf("failed to get zone ID: zone %s not found", zone)
	}
	return p.zoneIdCache[zone], nil
}

// getRecords returns the records in the zone.
func (p *Provider) getRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	zoneId, err := p.getZoneID(ctx, zone)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get records")
	}

	resp, err := p.client.Get(fmt.Sprintf("%s/%s", urlZone, zoneId))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get records")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get records: status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get records: failed to parse response")
	}

	zoneName := doc.Find("#p-dns .row h2")
	if zoneName.Length() != 1 {
		return nil, fmt.Errorf("failed get records: failed to find zone name")
	}
	if zoneName.Text() != zone {
		return nil, fmt.Errorf("failed get records: zone name mismatch: %s != %s", zoneName.Text(), zone)
	}

	var table *goquery.Selection
	doc.Find("table[summary]").Each(func(i int, s *goquery.Selection) {
		if s.AttrOr("summary", "") == fmt.Sprintf("DNS nastavitve za %s", zone) {
			table = s
		}
	})
	if table == nil {
		return nil, fmt.Errorf("failed get records: failed to find records table")
	}

	var records []libdns.Record
	var errBreak error
	table.Find("tbody tr").EachWithBreak(func(i int, s *goquery.Selection) bool {
		// #, name, TTL, type, value, priority(never present), action
		// <td>#</td><th>name</th><td>TTL</td><td>type</td><td>value</td><td>priority</td><td>action</td>
		tds := s.Children()
		if tds.Length() != 7 {
			errBreak = fmt.Errorf("failed get records: unexpected number of columns: %s", tds.Text())
			return false
		}
		name := tds.Eq(1).Text()
		ttl := tds.Eq(2).Text()
		typ := tds.Eq(3).Text()
		value := tds.Eq(4).Text()
		action := tds.Eq(6)

		name = libdns.RelativeName(strings.TrimSpace(name), zone)
		ttld, err := time.ParseDuration(ttl + "s")
		if err != nil {
			errBreak = errors.Wrap(err, "failed get records: failed to parse TTL")
			return false
		}
		typ = strings.ToUpper(strings.TrimSpace(typ))
		value = strings.TrimSpace(value)
		editLink := action.Children().First().AttrOr("href", "")
		editLinkParts := strings.Split(editLink, "/")
		if len(editLinkParts) != 6 {
			errBreak = fmt.Errorf("failed get records: failed to extract record id: %s", editLink)
			return false
		}
		id := editLinkParts[4]

		records = append(records, libdns.Record{
			ID:    id,
			Type:  typ,
			Name:  name,
			Value: value,
			TTL:   ttld,
			// TODO: Priority, Weight, Target
		})
		return true
	})

	if errBreak != nil {
		return nil, errBreak
	}
	return records, nil
}

// getRecordTTL returns the closest valid TTL value to the provided TTL.
// Check Provider.UnsupportedTTLisError to determine how unsupported TTL values are handled.
func (p *Provider) getRecordTTL(ttl time.Duration) (time.Duration, error) {
	for _, validTTL := range ValidTTLs {
		if ttl < validTTL {
			if p.UnsupportedTTLisError {
				return 0, fmt.Errorf("unsupported TTL value: %s", ttl)
			}
			return validTTL, nil
		}

		if ttl == validTTL {
			return validTTL, nil
		}
	}

	if p.UnsupportedTTLisError {
		return 0, fmt.Errorf("unsupported TTL value: %s", ttl)
	}
	return ValidTTLs[len(ValidTTLs)-1], nil
}

// sameRecord checks if two records are the same. This is used to determine which new records were added,
// and which records were updated.
func sameRecord(a, b *libdns.Record) bool {
	return a.Name == b.Name && a.Type == b.Type && a.TTL == b.TTL && a.Value == b.Value
}

// createRecord creates a new record in the zone.
func (p *Provider) createRecord(ctx context.Context, zone string, record libdns.Record) error {
	zoneId, err := p.getZoneID(ctx, zone)
	if err != nil {
		return errors.Wrap(err, "failed to append record")
	}

	form := url.Values{}
	form.Set("record[type]", record.Type)
	form.Set("record[host]", record.Name)
	form.Set("record[cart_id]", zoneId)
	form.Set("record[ttl]", fmt.Sprintf("%d", int(record.TTL.Seconds())))
	form.Set("record[priority]", "10")    // TODO: Priority
	form.Set("record[weight]", "0")       // TODO: Weight
	form.Set("record[port]", "0")         // TODO: Port
	form.Set("record[caa_flag]", "0")     // TODO: CAA Flag
	form.Set("record[caa_type]", "issue") // TODO: CAA Type
	form.Set("record[caa_value]", ";")    // TODO: CAA Value
	form.Set("record[record]", record.Value)

	// Create a request with the context and form data
	request, err := http.NewRequestWithContext(ctx, "POST", urlAddRecord, strings.NewReader(form.Encode()))
	if err != nil {
		return errors.Wrap(err, "failed to append record")
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(request)
	if err != nil {
		return errors.Wrap(err, "failed to append record")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to append record: status %d", resp.StatusCode)
	}

	// Check the response body for success status
	var result map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return errors.Wrap(err, "failed to parse append record response")
	}

	status, ok := result["status"]
	if !ok || status != "1" {
		message, ok := result["message"]
		if ok {
			return fmt.Errorf("failed to append record: API returned non-ok status: %v", message)
		}
		return fmt.Errorf("failed to append record: API returned non-ok status: %v", result)
	}

	return nil
}

// updateRecord updates an existing record in the zone.
func (p *Provider) updateRecord(ctx context.Context, zone string, record libdns.Record) error {
	zoneId, err := p.getZoneID(ctx, zone)
	if err != nil {
		return errors.Wrap(err, "failed to edit record")
	}

	form := url.Values{}
	form.Set("record[type]", record.Type)
	form.Set("record[id]", record.ID)
	form.Set("record[host]", record.Name)
	form.Set("record[cart_id]", zoneId)
	form.Set("record[ttl]", fmt.Sprintf("%d", int(record.TTL.Seconds())))
	form.Set("record[record]", record.Value)
	// TODO: Priority, Weight, Port, CAA Flag, CAA Type, CAA Value

	// Create a request with the context and form data
	request, err := http.NewRequestWithContext(ctx, "POST", urlEditRecord, strings.NewReader(form.Encode()))
	if err != nil {
		return errors.Wrap(err, "failed to edit record")
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(request)
	if err != nil {
		return errors.Wrap(err, "failed to edit record")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to edit record: status %d", resp.StatusCode)
	}

	// Check the response body for success status
	var result map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return errors.Wrap(err, "failed to parse edit record response")
	}

	status, ok := result["status"]
	if !ok || status != "1" {
		message, ok := result["message"]
		if ok {
			return fmt.Errorf("failed to edit record: API returned non-ok status: %v", message)
		}
		return fmt.Errorf("failed to edit record: API returned non-ok status: %v", result)
	}
	return nil
}

// deleteRecord deletes an existing record in the zone.
func (p *Provider) deleteRecord(ctx context.Context, zone string, record libdns.Record) error {
	zoneId, err := p.getZoneID(ctx, zone)
	if err != nil {
		return errors.Wrap(err, "failed to delete record")
	}

	resp, err := p.client.Get(fmt.Sprintf("%s/%s/%s", urlDeleteRecord, record.ID, zoneId))
	if err != nil {
		return errors.Wrap(err, "failed to delete record")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete record: status %d", resp.StatusCode)
	}

	// Check the response body for success status
	var result map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return errors.Wrap(err, "failed to parse delete record response")
	}

	status, ok := result["status"]
	if !ok || status != "1" {
		message, ok := result["message"]
		if ok {
			if message == "Error: Missing record-id" {
				return fmt.Errorf("failed to delete record: record not found")
			}
			return fmt.Errorf("failed to delete record: API returned non-ok status: %v", message)
		}

		return fmt.Errorf("failed to delete record: API returned non-ok status: %v", result)
	}
	return nil
}
