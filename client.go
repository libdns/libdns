package selectelv2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"text/template"

	"github.com/libdns/libdns"
)

// init once. Get KeystoneToken
func (p *Provider) init(ctx context.Context) error  {
		// create ZonesCache
		p.ZonesCache = make(map[string]string)

		// Compile the template
		tmpl, err := template.New("getKeystoneToken").Parse(cGetKeystoneTokenTemplate)
		if err != nil {
			return fmt.Errorf("GetKeystoneTokenTemplate error: %s", err)
		}
		var tokensBody bytes.Buffer
		err = tmpl.Execute(&tokensBody, p)
		if err != nil {
			return fmt.Errorf("GetKeystoneTokenTemplate execute error: %s", err)
		}

		// Request a KeystoneToken
		request, err := http.NewRequestWithContext(ctx, httpMethods.post, cTokensUrl, &tokensBody)
		if err != nil {
			return fmt.Errorf("getKeystoneToken NewRequestWithContext error: %s", err)
		}
		request.Header.Add("Content-Type", "application/json")

		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			return fmt.Errorf("getKeystoneToken client.Do error: %s", err)
		}
		defer response.Body.Close() // Guaranteed to close the body after the function is completed

		if response.StatusCode < 200 || response.StatusCode >= 300 {
			err = fmt.Errorf("%s (%d)", http.StatusText(response.StatusCode), response.StatusCode)
			return fmt.Errorf("getKeystoneToken response.StatusCode error: %s", err)
		}

		// Getting header $KeystoneTokenHeader
		p.KeystoneToken = response.Header.Get(cKeystoneTokenHeader)
		if p.KeystoneToken == "" {
			return fmt.Errorf("$KeystoneTokenHeader is missing")
		}

		return nil
}

// Functional part of procedure GetRecords -> uniRecords
func (p *Provider) getRecords(ctx context.Context, zoneId string, zone string) ([]libdns.Record, error) {
	records, err := p.getSelectelRecords(ctx, zoneId)
	if err != nil {
		return nil, err
	}
	return mapRecordsToLibds(zone, records), nil
}

// Functional part of procedure AppendRecords -> uniRecords
func (p *Provider) appendRecords(ctx context.Context, zone string, zoneId string, records []libdns.Record) ([]libdns.Record, error) {
	var resultRecords []libdns.Record
	var resultErr error
	for _, libRecord := range records {
		record := libdnsToRecord(zone, libRecord)
		// name normalizing
		body, err := json.Marshal(record)
		if err != nil {
			resultErr = err
			continue
		}
		// add recordset record request to api
		recordB, err := p.makeApiRequest(ctx, httpMethods.post, "/zones/%s/rrset", bytes.NewReader(body), zoneId)
		if err != nil {
			resultErr = err
			continue
		}
		selRecord, err := deserialization[Record](recordB)
		if err != nil {
			resultErr = err
			continue
		}
		resultRecords = append(resultRecords, recordToLibdns(zone, selRecord))
	}
	return resultRecords, resultErr
}

// Functional part of procedure SetRecords -> uniRecords
func (p *Provider) setRecords(ctx context.Context, zone string, zoneId string, records []libdns.Record) ([]libdns.Record, error) {
	zoneRecords, err := p.getSelectelRecords(ctx, zoneId)
	if err != nil {
		return nil, err
	}

	var resultRecords []libdns.Record
	var resultErr error
	for _, libRecord := range records {
		// check for already exists
		id, exists := idFromRecordsByLibRecord(zoneRecords, libRecord, zone)
		if exists {
			// if zone recordset contain record
			libRecord.ID = id
			record, err := p.updateSelectelRecord(ctx, zone, zoneId, libdnsToRecord(zone, libRecord))
			if err != nil {
				resultErr = err
			} else {
				resultRecords = append(resultRecords, recordToLibdns(zone, record))				
			}
		} else {
			// if not contain
			libRecords_, err := p.appendRecords(ctx, zone, zoneId, []libdns.Record{libRecord})
			if err != nil {
				resultErr = err
			} else {
				resultRecords = append(resultRecords, libRecords_...)
			}
		}
	}
	return resultRecords, resultErr
}

// Functional part of procedure DeleteRecords -> uniRecords
func (p *Provider) deleteRecords(ctx context.Context, zone string, zoneId string, records []libdns.Record) ([]libdns.Record, error) {
	zoneRecords, err := p.getSelectelRecords(ctx, zoneId)
	if err != nil {
		return nil, err
	}

	var resultRecords []libdns.Record
	var resultErr error
	for _, libRecord := range records {
		// check for already exists
		id, exists := idFromRecordsByLibRecord(zoneRecords, libRecord, zone)
		if exists {
			libRecord.ID = id
			// delete recordset record request to api
			_, err := p.makeApiRequest(ctx, httpMethods.delete, "/zones/%s/rrset/%s", nil, zoneId, libRecord.ID)
			if err != nil {
				resultErr = err
			} else {
				resultRecords = append(resultRecords, libRecord)
			}
		} else {
			resultErr = fmt.Errorf("no %s record %s for delete", libRecord.Type, libRecord.Name)
		}
	}
	return resultRecords, resultErr
}


func (p *Provider) uniRecords(method string,ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	// init
	var err error
	p.once.Do(func() {
		err = p.init(ctx)
	})
	
	if err != nil {
		p.once = sync.Once{} // reset p.once for next init
		return nil, err
	}
	
	// get zoneId
	zoneId, err := p.getZoneID(ctx, zone)
	if err != nil {
		return nil, err
	}

	// calling the appropriate method
	var libRecords []libdns.Record
	switch method {
	case recordMethods.get:
		libRecords, err = p.getRecords(ctx, zoneId, zone)
	case recordMethods.append:
		libRecords, err = p.appendRecords(ctx, zone, zoneId, records)
	case recordMethods.set:
		libRecords, err = p.setRecords(ctx, zone, zoneId, records)
	case recordMethods.delete:
		libRecords, err = p.deleteRecords(ctx, zone, zoneId, records)
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}

	return libRecords, err
}
