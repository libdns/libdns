package selectelv2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

// Generic function to deserialize a JSON string into a variable of type T
func deserialization[T any](data []byte) (T, error) {
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}
	return result, nil
}


// Generate url from path
func urlGenerator(path string, args ...interface{}) string {
	return fmt.Sprintf(cApiBaseUrl+path, args...)
}

// API request function
// ctx - context
// method - http method (GET, POST, DELETE, PATCH, PUT)
// path - path to connect to apiBaseUrl
// body - data transferred in the body
// hideToken - flag for hiding the token in the header
// args - substitution arguments in path
func (p *Provider) makeApiRequest(ctx context.Context, method string, path string , body io.Reader, args ...interface{}) ([]byte, error) {	

	// Make request
	request, err := http.NewRequestWithContext(ctx, method, urlGenerator(path, args...) , body)
	if err != nil {
		return nil, err
	}

	// add headers
	request.Header.Add("X-Auth-Token", p.KeystoneToken)
	request.Header.Add("Content-Type", "application/json")

	// request api
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close() // Guaranteed to close the body after the function is completed

	// get the body and return it
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	// check status
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("%s (%d): %s", http.StatusText(response.StatusCode), response.StatusCode, string(data))
	}

	return data, nil
}

// Get zoneId by zone name
func (p *Provider) getZoneID(ctx context.Context, zone string) (string, error) {
	// try get zoneId from cache
	zoneId := p.ZonesCache[zone]
	if zoneId == "" {
		// if not in cache, get from api
		zonesB, err := p.makeApiRequest(ctx, httpMethods.get, fmt.Sprintf("/zones?filter=%s", url.QueryEscape(zone)), nil)
		if err != nil {
			return "", err
		}
		zones, err := deserialization[Zones](zonesB)
		if err != nil {
			return "", err
		}
		if len(zones.Zones) == 0 {
			return "", fmt.Errorf("no zoneId for zone %s", zone)
		}
		zoneId = zones.Zones[0].ID_
	}
	return zoneId, nil
}

// Convert Record to libdns.Record
func recordToLibdns(zone string, record Record) libdns.Record {
    // for TTL
    ttlDuration := time.Duration(record.TTL) * time.Second

    // for Value
    var valueString string
    for _, recVal := range record.Value {
        valueString += recVal.Value + "\n"
    }
    // remove last \n
    if len(valueString) > 0 {
        valueString = valueString[:len(valueString)-1]
    }
	// for TXT raplace all \"
	if record.Type == "TXT" {
		valueString = strings.ReplaceAll(valueString, "\"", "")
	}

    return libdns.Record{
        ID:       record.ID,
        Type:     record.Type,
        Name:     nameNormalizer(record.Name, zone),
        Value:    valueString,
        TTL:      ttlDuration,
    }
}

// map []Record to []libdns.Record
func mapRecordsToLibds(zone string, records []Record) []libdns.Record {
	libdnsRecords := make([]libdns.Record, len(records))
	for i, record := range records {
		libdnsRecords[i] = recordToLibdns(zone, record)
	}
	return libdnsRecords
}

// Convert Record to libdns.Record
func libdnsToRecord(zone string, libdnsRecord libdns.Record) Record {
    // for TTL
    ttl := libdnsRecord.TTL.Seconds()

    // for Value
	recVals := strings.Split(libdnsRecord.Value, "\n")
    valueRV := make([]RValue, len(recVals))
    for i, recVal := range recVals {
		if libdnsRecord.Type == "TXT" {
			// if TXT, add to any preffix&suffix \"
			recVal = strings.Trim(recVal, "\"")
			valueRV[i] = RValue{Value: "\"" + recVal + "\""}
		} else {
			valueRV[i] = RValue{Value: recVal}
		}
    }

    return Record{
        ID:       libdnsRecord.ID,
        Type:     libdnsRecord.Type,
        Name:     nameNormalizer(libdnsRecord.Name, zone),
        Value:    valueRV,
        TTL:      int(ttl),
    }
}

// Get Selectel records
func (p *Provider) getSelectelRecords(ctx context.Context, zoneId string) ([]Record, error) {
	recordB, err := p.makeApiRequest(ctx, httpMethods.get, "/zones/%s/rrset", nil, zoneId)
	if err != nil {
		return nil, err
	}
	recordset, err := deserialization[Recordset](recordB)
	if err != nil {
		return nil, err
	}
	return recordset.Records, nil
}

// Update Selectel record
func (p *Provider) updateSelectelRecord(ctx context.Context, zone string, zoneId string, record Record) (Record, error) {
	body, err := json.Marshal(record)
	if err != nil {
		return Record{}, err
	}
	_, err = p.makeApiRequest(ctx, httpMethods.patch, "/zones/%s/rrset/%s", bytes.NewReader(body), zoneId, record.ID)
	if err != nil {
		return Record{}, err
	}
	return record, nil
}

// Normalize name in zone namespace
//
// test => test.zone.
// test.zone => test.zone.
// test.zone. => test.zone.
// test.subzone => test.subzone.zone.
// ...
func nameNormalizer(name string, zone string) string {
	name = strings.TrimSuffix(name, ".")
	zone = strings.TrimSuffix(zone, ".")
	if strings.HasSuffix(name, "."+zone) {
		return name + "."
	}
	return name + "." + zone + "."
}


// Check if an element with Id == id || (Name == name  && Type == type) exists 
func idFromRecordsByLibRecord(records []Record, libRecord libdns.Record, zone string) (string, bool) {
	nameNorm := nameNormalizer(libRecord.Name, zone)
	for _, record := range records {
		recordNameNorm := nameNormalizer(record.Name, zone)
		if libRecord.ID == record.ID || (recordNameNorm == nameNorm && libRecord.Type == record.Type) {
			return record.ID ,true // Element found
		}
	}
	return "", false // Element not found
}
// // map []Record to []libdns.Record
// func maplibdnsToRecord(libdnsRecords []libdns.Record) []Record {
// 	records := make([]Record, len(libdnsRecords))
// 	for i, record := range libdnsRecords {
// 		records[i] = libdnsToRecord(record)
// 	}
// 	return records
// }