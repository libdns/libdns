package dnsimple

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/libdns/libdns"
)

// Provider facilitates DNS record manipulation with DNSimple.
type Provider struct {
	APIAccessToken string `json:"api_access_token,omitempty"`
	AccountID      string `json:"account_id,omitempty"`
	APIURL         string `json:"api_url,omitempty"`

	client dnsimple.Client
	once   sync.Once
	mutex  sync.Mutex
}

// initClient will initialize the DNSimple API client with the provided access token and
// store the client in the Provider struct, along with setting the API URL and Account ID.
func (p *Provider) initClient(ctx context.Context) {
	p.once.Do(func() {
		// Create new DNSimple client using the provided access token.
		tc := dnsimple.StaticTokenHTTPClient(ctx, p.APIAccessToken)
		c := *dnsimple.NewClient(tc)
		// Set the API URL if using a non-default API hostname (e.g. sandbox).
		if p.APIURL != "" {
			c.BaseURL = p.APIURL
		}
		// If no Account ID is provided, we can call the API to get the corresponding
		// account id for the provided access token.
		if p.AccountID == "" {
			resp, _ := c.Identity.Whoami(context.Background())
			accountID := strconv.FormatInt(resp.Data.Account.ID, 10)
			p.AccountID = accountID
		}

		p.client = c
	})
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.initClient(ctx)

	var records []libdns.Record

	resp, err := p.client.Zones.ListRecords(ctx, p.AccountID, zone, &dnsimple.ZoneRecordListOptions{})
	if err != nil {
		return nil, err
	}
	for _, r := range resp.Data {
		records = append(records, libdns.Record{
			ID:       strconv.FormatInt(r.ID, 10),
			Type:     r.Type,
			Name:     r.Name,
			Value:    r.Content,
			TTL:      time.Duration(r.TTL),
			Priority: uint(r.Priority),
		})
	}

	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.initClient(ctx)

	var appendedRecords []libdns.Record

	// Get the Zone ID from zone name
	resp, err := p.client.Zones.GetZone(ctx, p.AccountID, zone)
	if err != nil {
		return appendedRecords, err
	}
	zoneID := strconv.FormatInt(resp.Data.ID, 10)

	for _, r := range records {
		zra := dnsimple.ZoneRecordAttributes{
			ZoneID:   zoneID,
			Type:     r.Type,
			Name:     &r.Name,
			Content:  r.Value,
			TTL:      int(r.TTL),
			Priority: int(r.Priority),
		}
		resp, err := p.client.Zones.CreateRecord(ctx, p.AccountID, zone, zra)
		if err != nil {
			return appendedRecords, fmt.Errorf("Failed to create record: %s, error: %v", r.Name, err.Error())
		}
		// See https://developer.dnsimple.com/v2/zones/records/#createZoneRecord
		switch resp.HTTPResponse.StatusCode {
		case http.StatusCreated:
			r.ID = strconv.FormatInt(resp.Data.ID, 10)
			appendedRecords = append(appendedRecords, r)
		case http.StatusBadRequest:
			return appendedRecords, fmt.Errorf("Received HTTP 400, could not create record: %s", r.Name)
		case http.StatusUnauthorized:
			return appendedRecords, fmt.Errorf("Received HTTP 401 due to authentication issues, could not create record: %s", r.Name)
		default:
			return appendedRecords, fmt.Errorf("Unexpected error: %s, could not create record: %s", resp.HTTPResponse.Status, r.Name)
		}
	}
	return appendedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.initClient(ctx)

	var setRecords []libdns.Record

	existingRecords, err := p.GetRecords(ctx, zone)
	if err != nil {
		return nil, err
	}
	var recordsToCreate []libdns.Record
	var recordsToUpdate []libdns.Record

	// Figure out which records are new and need to be created, and which records exist and need to be updated
	for _, r := range records {
		for _, er := range existingRecords {
			if r.Name == er.Name {
				if r.ID == "0" || r.ID == "" {
					r.ID = er.ID
				}
				recordsToUpdate = append(recordsToUpdate, r)
				break
			}
		}
		recordsToCreate = append(recordsToCreate, r)
	}

	// Create new records and append them to 'setRecords'
	createdRecords, err := p.AppendRecords(ctx, zone, recordsToCreate)
	if err != nil {
		return setRecords, err
	}
	for _, r := range createdRecords {
		setRecords = append(setRecords, r)
	}

	// Get the Zone ID from zone name
	resp, err := p.client.Zones.GetZone(ctx, p.AccountID, zone)
	if err != nil {
		return setRecords, err
	}
	zoneID := strconv.FormatInt(resp.Data.ID, 10)

	// Update existing records and append them to 'SetRecords'
	for _, r := range recordsToUpdate {
		zra := dnsimple.ZoneRecordAttributes{
			ZoneID:   zoneID,
			Type:     r.Type,
			Name:     &r.Name,
			Content:  r.Value,
			TTL:      int(r.TTL),
			Priority: int(r.Priority),
		}
		id, err := strconv.ParseInt(r.ID, 10, 64)
		if err != nil {
			return setRecords, err
		}
		resp, err := p.client.Zones.UpdateRecord(ctx, p.AccountID, zone, id, zra)
		if err != nil {
			return setRecords, fmt.Errorf("Failed to update record: %s, error: %v", r.Name, err.Error())
		}
		// https://developer.dnsimple.com/v2/zones/records/#updateZoneRecord
		switch resp.HTTPResponse.StatusCode {
		case http.StatusOK:
			r.ID = strconv.FormatInt(resp.Data.ID, 10)
			setRecords = append(setRecords, r)
		case http.StatusBadRequest:
			return setRecords, fmt.Errorf("Received HTTP 400, could not update record: %s", r.Name)
		case http.StatusUnauthorized:
			return setRecords, fmt.Errorf("Received HTTP 401 due to authentication issues, could not update record: %s", r.Name)
		default:
			return setRecords, fmt.Errorf("Unexpected error: %s, could not update record: %s", resp.HTTPResponse.Status, r.Name)
		}
	}
	return setRecords, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.initClient(ctx)

	var deleted []libdns.Record
	var failed []libdns.Record
	var noID []libdns.Record

	for _, r := range records {
		// If the record does not have an ID, we'll try to find it by calling the API later
		// and extrapolating its ID based on the record name, but continue for now.
		if r.ID == "0" || r.ID == "" {
			noID = append(noID, r)
			continue
		}

		id, err := strconv.ParseInt(r.ID, 10, 64)
		if err != nil {
			failed = append(failed, r)
			continue
		}

		resp, err := p.client.Zones.DeleteRecord(ctx, p.AccountID, zone, id)
		if err != nil {
			failed = append(failed, r)
		}
		// See https://developer.dnsimple.com/v2/zones/records/#deleteZoneRecord for API response codes
		switch resp.HTTPResponse.StatusCode {
		case http.StatusNoContent:
			deleted = append(deleted, r)
		case http.StatusBadRequest:
			fmt.Printf("Received a HTTP 400, could not delete record: %s", r.Name)
			failed = append(failed, r)
		case http.StatusUnauthorized:
			fmt.Printf("Received a HTTP 401, suggesting authentication issues with the DNSimple client")
			failed = append(failed, r)
		default:
			failed = append(failed, r)
		}
	}
	// If we received records without an ID earlier, we're going to try and figure out the ID by calling
	// GetRecords and comparing the record name. If we're able to find it, we'll delete it, otherwise
	// we'll append it to our list of failed to delete records.
	if len(noID) > 0 {
		fetchedRecords, err := p.GetRecords(ctx, zone)
		if err != nil {
			fmt.Printf("Failed to populate IDs for records where one wasn't provided, err: %s", err.Error())
		} else {
			for _, r := range noID {
				for _, fr := range fetchedRecords {
					if fr.Name == r.Name {
						id, err := strconv.ParseInt(fr.ID, 10, 64)
						if err != nil {
							failed = append(failed, r)
							break // Break out of the inner loop, but we still want to try the other records
						}
						resp, err := p.client.Zones.DeleteRecord(ctx, p.AccountID, zone, id)
						if err != nil {
							failed = append(failed, r)
						}
						// See https://developer.dnsimple.com/v2/zones/records/#deleteZoneRecord for API response codes
						switch resp.HTTPResponse.StatusCode {
						case http.StatusNoContent:
							deleted = append(deleted, r)
						case http.StatusBadRequest:
							fmt.Printf("Received a HTTP 400, could not delete record: %s", r.Name)
							failed = append(failed, r)
						case http.StatusUnauthorized:
							fmt.Printf("Received a HTTP 401, suggesting authentication issues with the DNSimple client")
							failed = append(failed, r)
						default:
							failed = append(failed, r)
						}
						break
					}
				}
				fmt.Printf("Could not figure out ID for record: %s", r.Name)
				failed = append(failed, r)
			}
		}
	}
	// Print out all the records we failed to delete.
	for _, r := range failed {
		fmt.Printf("Failed to delete record: %s", r.Name)
	}

	return deleted, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
