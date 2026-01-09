package bluecat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

// Client handles communication with the Bluecat API
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	apiToken   string
	authHeader string
}

// NewClient creates a new Bluecat API client
func NewClient(baseURL, username, password string) (*Client, error) {
	// Trim trailing slash from baseURL
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Authenticate authenticates with the Bluecat API and stores the token
func (c *Client) Authenticate(ctx context.Context) error {
	url := c.baseURL + "/api/v2/sessions"

	reqBody := map[string]string{
		"username": c.username,
		"password": c.password,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal auth request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var authResp struct {
		APIToken                        string `json:"apiToken"`
		BasicAuthenticationCredentials string `json:"basicAuthenticationCredentials"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	c.apiToken = authResp.APIToken
	c.authHeader = authResp.BasicAuthenticationCredentials

	return nil
}

// GetZoneID retrieves the zone ID for a given zone name
func (c *Client) GetZoneID(ctx context.Context, zone, configName, viewName string) (int64, error) {
	// Clean up zone name (remove trailing dot)
	zone = strings.TrimSuffix(zone, ".")

	// Try to find the most specific zone by searching with absoluteName filter
	// Walk from most specific to least specific
	domainParts := strings.Split(zone, ".")
	for i := 0; i < len(domainParts); i++ {
		searchZone := strings.Join(domainParts[i:], ".")
		if searchZone == "" {
			continue
		}
		
		fmt.Printf("DEBUG: Searching for zone with absoluteName: %s\n", searchZone)
		
		// Use filter to search for zone by absoluteName
		apiURL := fmt.Sprintf("%s/api/v2/zones?filter=absoluteName:eq('%s')", c.baseURL, searchZone)
		
		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			continue
		}

		req.Header.Set("Authorization", "Basic "+c.authHeader)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var zonesResp struct {
				Data []struct {
					ID           int64  `json:"id"`
					Name         string `json:"name"`
					AbsoluteName string `json:"absoluteName"`
				} `json:"data"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&zonesResp); err == nil && len(zonesResp.Data) > 0 {
				resp.Body.Close()
				fmt.Printf("DEBUG: Found zone: %s (ID: %d)\n", zonesResp.Data[0].AbsoluteName, zonesResp.Data[0].ID)
				return zonesResp.Data[0].ID, nil
			}
		}
		resp.Body.Close()
	}

	return 0, fmt.Errorf("no zone found for %s", zone)
}

// GetResourceRecords retrieves all resource records for a zone
func (c *Client) GetResourceRecords(ctx context.Context, zoneID int64) ([]libdns.Record, error) {
	url := fmt.Sprintf("%s/api/v2/zones/%d/resourceRecords", c.baseURL, zoneID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+c.authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource records: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get resource records with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var recordsResp struct {
		Data []BluecatResourceRecord `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&recordsResp); err != nil {
		return nil, fmt.Errorf("failed to decode resource records: %w", err)
	}

	// Convert Bluecat records to libdns records
	var records []libdns.Record
	for _, bcRec := range recordsResp.Data {
		rec, err := convertBluecatToLibdns(bcRec)
		if err != nil {
			// Skip records we can't convert
			continue
		}
		records = append(records, rec)
	}

	return records, nil
}

// CreateResourceRecord creates a new resource record in the specified zone
func (c *Client) CreateResourceRecord(ctx context.Context, zoneID int64, zone string, record libdns.Record) (libdns.Record, error) {
	url := fmt.Sprintf("%s/api/v2/zones/%d/resourceRecords", c.baseURL, zoneID)

	bcRecord, err := convertLibdnsToBluecat(record, zone)
	if err != nil {
		return nil, fmt.Errorf("failed to convert record: %w", err)
	}

	body, err := json.Marshal(bcRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+c.authHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create resource record with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var createdRecord BluecatResourceRecord
	if err := json.NewDecoder(resp.Body).Decode(&createdRecord); err != nil {
		return nil, fmt.Errorf("failed to decode created record: %w", err)
	}

	return convertBluecatToLibdns(createdRecord)
}

// DeleteResourceRecord deletes a resource record
func (c *Client) DeleteResourceRecord(ctx context.Context, record libdns.Record) error {
	// Extract the record ID from ProviderData
	recordID := getRecordID(record)
	
	if recordID == 0 {
		return fmt.Errorf("record ID not found in provider data")
	}

	url := fmt.Sprintf("%s/api/v2/resourceRecords/%d", c.baseURL, recordID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+c.authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete resource record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete resource record with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// getConfigurationID retrieves the configuration ID by name, or the first one if name is empty
func (c *Client) getConfigurationID(ctx context.Context, configName string) (int64, error) {
	apiURL := c.baseURL + "/api/v2/configurations"

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+c.authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get configurations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to get configurations with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var configResp struct {
		Data []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		return 0, fmt.Errorf("failed to decode configurations: %w", err)
	}

	if len(configResp.Data) == 0 {
		return 0, fmt.Errorf("no configurations found")
	}

	// If a specific config name was requested, find it
	if configName != "" {
		for _, cfg := range configResp.Data {
			if cfg.Name == configName {
				return cfg.ID, nil
			}
		}
		return 0, fmt.Errorf("configuration %s not found", configName)
	}

	// Otherwise return the first one
	return configResp.Data[0].ID, nil
}

// getViewID retrieves the view ID by name, or the first one if name is empty
func (c *Client) getViewID(ctx context.Context, configID int64, viewName string) (int64, error) {
	apiURL := fmt.Sprintf("%s/api/v2/configurations/%d/views", c.baseURL, configID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+c.authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get views: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to get views with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var viewResp struct {
		Data []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&viewResp); err != nil {
		return 0, fmt.Errorf("failed to decode views: %w", err)
	}

	if len(viewResp.Data) == 0 {
		return 0, fmt.Errorf("no views found")
	}

	// If a specific view name was requested, find it
	if viewName != "" {
		for _, v := range viewResp.Data {
			if v.Name == viewName {
				return v.ID, nil
			}
		}
		return 0, fmt.Errorf("view %s not found", viewName)
	}

	// Otherwise return the first one
	return viewResp.Data[0].ID, nil
}

// BluecatResourceRecord represents a resource record in the Bluecat API
type BluecatResourceRecord struct {
	ID               int64  `json:"id,omitempty"`
	Type             string `json:"type"`
	Name             string `json:"name"`
	AbsoluteName     string `json:"absoluteName,omitempty"`
	TTL              int    `json:"ttl,omitempty"`
	RecordType       string `json:"recordType,omitempty"`
	RData            string `json:"rdata,omitempty"`
	Text             string `json:"text,omitempty"`
	LinkedRecordName string `json:"linkedRecordName,omitempty"`
	Priority         int    `json:"priority,omitempty"`
	Weight           int    `json:"weight,omitempty"`
	Port             int    `json:"port,omitempty"`
	Addresses        []struct {
		Address string `json:"address"`
	} `json:"addresses,omitempty"`
}

// convertBluecatToLibdns converts a Bluecat resource record to a libdns record
func convertBluecatToLibdns(bcRec BluecatResourceRecord) (libdns.Record, error) {
	// Calculate relative name
	name := bcRec.Name
	if name == "" {
		name = "@"
	}

	ttl := time.Duration(bcRec.TTL) * time.Second

	// Determine the record type and create appropriate struct
	switch bcRec.RecordType {
	case "A", "AAAA":
		// HostRecord uses addresses field
		var ipStr string
		if len(bcRec.Addresses) > 0 {
			ipStr = bcRec.Addresses[0].Address
		} else if bcRec.RData != "" {
			ipStr = bcRec.RData
		} else {
			return nil, fmt.Errorf("no IP address found in record")
		}
		addr, err := netip.ParseAddr(ipStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse IP address: %w", err)
		}
		return libdns.Address{
			Name:         name,
			TTL:          ttl,
			IP:           addr,
			ProviderData: bcRec.ID,
		}, nil

	case "CNAME":
		return libdns.CNAME{
			Name:         name,
			TTL:          ttl,
			Target:       bcRec.LinkedRecordName,
			ProviderData: bcRec.ID,
		}, nil

	case "TXT":
		// TXTRecord uses 'text' field
		textData := bcRec.Text
		if textData == "" {
			textData = bcRec.RData
		}
		return libdns.TXT{
			Name:         name,
			TTL:          ttl,
			Text:         textData,
			ProviderData: bcRec.ID,
		}, nil

	case "MX":
		return libdns.MX{
			Name:         name,
			TTL:          ttl,
			Preference:   uint16(bcRec.Priority),
			Target:       bcRec.LinkedRecordName,
			ProviderData: bcRec.ID,
		}, nil

	case "NS":
		return libdns.NS{
			Name:         name,
			TTL:          ttl,
			Target:       bcRec.LinkedRecordName,
			ProviderData: bcRec.ID,
		}, nil

	case "SRV":
		// Parse service and protocol from name (format: _service._protocol.name)
		parts := strings.SplitN(name, ".", 3)
		var service, protocol, recordName string
		if len(parts) >= 3 {
			service = strings.TrimPrefix(parts[0], "_")
			protocol = strings.TrimPrefix(parts[1], "_")
			recordName = parts[2]
		}

		return libdns.SRV{
			Service:      service,
			Transport:    protocol,
			Name:         recordName,
			TTL:          ttl,
			Priority:     uint16(bcRec.Priority),
			Weight:       uint16(bcRec.Weight),
			Port:         uint16(bcRec.Port),
			Target:       bcRec.LinkedRecordName,
			ProviderData: bcRec.ID,
		}, nil

	default:
		// Skip unsupported record types rather than returning generic RR
		// as per libdns documentation requirements
		return nil, fmt.Errorf("unsupported record type: %s", bcRec.RecordType)
	}
}

// convertLibdnsToBluecat converts a libdns record to a Bluecat resource record
func convertLibdnsToBluecat(record libdns.Record, zone string) (BluecatResourceRecord, error) {
	rr := record.RR()

	// Remove trailing dot from zone for proper absolute name construction
	zone = strings.TrimSuffix(zone, ".")

	// Construct absolute name
	var absoluteName string
	if rr.Name == "@" || rr.Name == "" {
		absoluteName = zone
	} else {
		absoluteName = rr.Name + "." + zone
	}

	bcRec := BluecatResourceRecord{
		Name:         rr.Name,
		AbsoluteName: absoluteName,
		TTL:          int(rr.TTL.Seconds()),
	}

	// Set type-specific fields with proper Bluecat type names
	switch rec := record.(type) {
	case libdns.Address:
		// Use HostRecord type for A/AAAA records
		bcRec.Type = "HostRecord"
		if rec.IP.Is4() {
			bcRec.RecordType = "A"
		} else {
			bcRec.RecordType = "AAAA"
		}
		// HostRecord requires addresses field instead of rdata
		bcRec.Addresses = []struct {
			Address string `json:"address"`
		}{
			{Address: rec.IP.String()},
		}

	case libdns.CNAME:
		bcRec.Type = "AliasRecord"
		bcRec.RecordType = "CNAME"
		bcRec.LinkedRecordName = rec.Target

	case libdns.TXT:
		bcRec.Type = "TXTRecord"
		bcRec.RecordType = "TXT"
		// TXTRecord uses 'text' field
		bcRec.Text = rec.Text

	case libdns.MX:
		bcRec.Type = "MXRecord"
		bcRec.RecordType = "MX"
		bcRec.Priority = int(rec.Preference)
		bcRec.LinkedRecordName = rec.Target

	case libdns.NS:
		bcRec.Type = "GenericRecord"
		bcRec.RecordType = "NS"
		bcRec.LinkedRecordName = rec.Target

	case libdns.SRV:
		bcRec.Type = "SRVRecord"
		bcRec.RecordType = "SRV"
		// Construct the full SRV name: _service._protocol.name
		bcRec.Name = fmt.Sprintf("_%s._%s.%s", rec.Service, rec.Transport, rec.Name)
		if rec.Name == "@" || rec.Name == "" {
			bcRec.AbsoluteName = fmt.Sprintf("_%s._%s.%s", rec.Service, rec.Transport, zone)
		} else {
			bcRec.AbsoluteName = fmt.Sprintf("_%s._%s.%s.%s", rec.Service, rec.Transport, rec.Name, zone)
		}
		bcRec.Priority = int(rec.Priority)
		bcRec.Weight = int(rec.Weight)
		bcRec.Port = int(rec.Port)
		bcRec.LinkedRecordName = rec.Target

	case libdns.RR:
		bcRec.Type = "GenericRecord"
		bcRec.RecordType = rec.Type
		bcRec.RData = rec.Data
	}

	// Parse TTL if provided in string form
	if bcRec.TTL == 0 && rr.TTL > 0 {
		bcRec.TTL = int(rr.TTL.Seconds())
	}

	return bcRec, nil
}

// Helper method to extract provider data as int64
func getRecordID(record libdns.Record) int64 {
	switch rec := record.(type) {
	case libdns.Address:
		if id, ok := rec.ProviderData.(int64); ok {
			return id
		}
	case libdns.CNAME:
		if id, ok := rec.ProviderData.(int64); ok {
			return id
		}
	case libdns.TXT:
		if id, ok := rec.ProviderData.(int64); ok {
			return id
		}
	case libdns.MX:
		if id, ok := rec.ProviderData.(int64); ok {
			return id
		}
	case libdns.NS:
		if id, ok := rec.ProviderData.(int64); ok {
			return id
		}
	case libdns.SRV:
		if id, ok := rec.ProviderData.(int64); ok {
			return id
		}
	case libdns.RR:
		// RR types don't have ProviderData, try to extract from the record itself
		// This shouldn't happen in normal operation
		fmt.Printf("DEBUG: Trying to get ID from RR type (this shouldn't happen)\n")
	}

	fmt.Printf("DEBUG: Failed to extract record ID from type %T with ProviderData: %v\n", record, record.RR())
	return 0
}
