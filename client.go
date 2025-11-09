package allinkl

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/clbanning/mxj"
	"github.com/libdns/libdns"
	"github.com/tiaguinho/gosoap"
)

const ApiBase = "https://kasapi.kasserver.com/soap/wsdl/KasApi.wsdl"

var ChachedRecords = make(map[string][]allinklRecord)

func (p *Provider) GetAllRecords(ctx context.Context, zone string) ([]libdns.Record, error) {

	httpClient := &http.Client{
		Timeout: 1500 * time.Millisecond,
	}
	soap, err := gosoap.SoapClient(ApiBase, httpClient)
	if err != nil {
		fmt.Println("Error creating SOAP client:", err)
	}

	// Prepare parameters like PHP script
	params := map[string]interface{}{
		"zone_host": zone,
	}

	requestData := map[string]interface{}{
		"kas_login":        p.KasLogin,
		"kas_auth_type":    "plain",
		"kas_auth_data":    p.KasAuthPassword,
		"kas_action":       "get_dns_settings",
		"KasRequestParams": params,
	}

	// JSON encode the entire request like PHP
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		log.Fatalf("Error encoding JSON: %v", err)
	}

	// Call the SOAP method with JSON-encoded params
	res, err := soap.Call("KasApi", gosoap.Params{
		"Params": string(jsonData),
	})
	if err != nil {
		log.Fatalf("Error calling SOAP method: %v", err)
	}
	if res == nil {
		log.Fatal("Response is nil")
	}
	mv, err := mxj.NewMapXml([]byte(res.Body))
	if err != nil {
		log.Fatalf("Error converting XML to map: %v", err)
	}

	records := []interface{}{}
	// Defensive navigation through the nested map
	root, ok := mv["KasApiResponse"].(map[string]interface{})
	if ok {
		ret, ok := root["return"].(map[string]interface{})
		if ok {
			items := ret["item"]
			var itemList []interface{}
			switch v := items.(type) {
			case []interface{}:
				itemList = v
			case map[string]interface{}:
				itemList = []interface{}{v}
			}
			for _, item := range itemList {
				mitem, _ := item.(map[string]interface{})
				keyMap, _ := mitem["key"].(map[string]interface{})
				key, _ := keyMap["#text"].(string)
				if key == "Response" {
					val, _ := mitem["value"].(map[string]interface{})
					respItems := val["item"]
					var respList []interface{}
					switch v := respItems.(type) {
					case []interface{}:
						respList = v
					case map[string]interface{}:
						respList = []interface{}{v}
					}
					for _, respItem := range respList {
						respMap, _ := respItem.(map[string]interface{})
						rkeyMap, _ := respMap["key"].(map[string]interface{})
						rkey, _ := rkeyMap["#text"].(string)
						if rkey == "ReturnInfo" {
							rval, _ := respMap["value"].(map[string]interface{})
							arr := rval["item"]
							switch v := arr.(type) {
							case []interface{}:
								records = v
							case map[string]interface{}:
								records = []interface{}{v}
							}
						}
					}
				}
			}
		}
	}

	// Initialize recordsList with enough elements
	recordsList := make([]libdns.Record, len(records))
	rawRecords := make([]allinklRecord, len(records))

	for i, rec := range records {
		recMap, _ := rec.(map[string]interface{})

		items := recMap["item"]
		var kvList []interface{}
		switch v := items.(type) {
		case []interface{}:
			kvList = v
		case map[string]interface{}:
			kvList = []interface{}{v}
		}

		// Create a temporary record for this entry
		currentRecord := allinklRecord{}

		for _, kv := range kvList {
			kvMap, _ := kv.(map[string]interface{})
			keyMap, _ := kvMap["key"].(map[string]interface{})
			key, _ := keyMap["#text"].(string)
			valueMap, valueIsMap := kvMap["value"].(map[string]interface{})
			var value interface{}
			if valueIsMap {
				value, _ = valueMap["#text"]
			} else {
				value = kvMap["value"]
			}

			// Set fields in the temporary Record struct
			switch key {
			case "record_id":
				if strVal, ok := value.(string); ok {
					currentRecord.ID = strVal
				}
			case "record_zone":
				if strVal, ok := value.(string); ok {
					currentRecord.ZoneID = strVal
				}
			case "record_type":
				if strVal, ok := value.(string); ok {
					currentRecord.Type = strVal
				}
			case "record_name":
				if strVal, ok := value.(string); ok {
					currentRecord.Name = strVal
				} else if value == nil {
					// For apex/root domain records, name might be nil
					currentRecord.Name = "@"
				}
			case "record_data":
				if strVal, ok := value.(string); ok {
					currentRecord.Value = strVal
				}
			case "record_ttl":
				if ttlVal, ok := value.(float64); ok {
					currentRecord.TTL = int(ttlVal)
				} else if ttlStr, ok := value.(string); ok {
					// Try to parse string to int if needed
					var ttl int
					fmt.Sscanf(ttlStr, "%d", &ttl)
					currentRecord.TTL = ttl
				}
			}
		}

		rawRecords[i] = currentRecord
		var libdnsRecord, err = currentRecord.toLibdnsRecord(zone)
		if err != nil {
			log.Printf("Error converting record to libdns format: %v", err)
			continue
		}
		recordsList[i] = libdnsRecord
	}

	ChachedRecords[zone] = rawRecords

	return recordsList, nil
}

func (p *Provider) AppendRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {

	httpClient := &http.Client{
		Timeout: 1500 * time.Millisecond,
	}
	soap, err := gosoap.SoapClient(ApiBase, httpClient)
	if err != nil {
		return libdns.RR{}, fmt.Errorf("error creating SOAP client: %w", err)
	}

	// Convert the record to RR to access its fields
	rr := record.RR()

	if rr.TTL/time.Second < 600 {
		rr.TTL = 600 * time.Second
	}
	ttlInSeconds := int(rr.TTL / time.Second)

	// Prepare parameters like PHP script
	params := map[string]interface{}{
		"record_name": rr.Name,
		"record_type": rr.Type,
		"record_data": rr.Data,
		"record_aux":  ttlInSeconds,
		"zone_host":   zone + ".",
	}

	// Include TTL if specified
	if rr.TTL != 0 {
		params["record_ttl"] = int(rr.TTL.Seconds())
	}

	requestData := map[string]interface{}{
		"kas_login":        p.KasLogin,
		"kas_auth_type":    "plain",
		"kas_auth_data":    p.KasAuthPassword,
		"kas_action":       "add_dns_settings",
		"KasRequestParams": params,
	}
	// JSON encode the entire request like PHP
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return libdns.RR{}, fmt.Errorf("error encoding JSON: %w", err)
	}

	// Call the SOAP method with JSON-encoded params
	res, err := soap.Call("KasApi", gosoap.Params{
		"Params": string(jsonData),
	})
	if err != nil {
		return libdns.RR{}, fmt.Errorf("error calling SOAP method: %w", err)
	}
	if res == nil {
		return libdns.RR{}, fmt.Errorf("response is nil")
	}

	mv, err := mxj.NewMapXml([]byte(res.Body))
	if err != nil {
		return libdns.RR{}, fmt.Errorf("error converting XML to map: %w", err)
	}

	// Parse response to check for success and get record ID
	root, ok := mv["KasApiResponse"].(map[string]interface{})
	if !ok {
		return libdns.RR{}, fmt.Errorf("invalid response format")
	}

	ret, ok := root["return"].(map[string]interface{})
	if !ok {
		return libdns.RR{}, fmt.Errorf("invalid response format: missing return")
	}

	// Check for errors in response
	items := ret["item"]
	var itemList []interface{}
	switch v := items.(type) {
	case []interface{}:
		itemList = v
	case map[string]interface{}:
		itemList = []interface{}{v}
	}

	for _, item := range itemList {
		mitem, _ := item.(map[string]interface{})
		keyMap, _ := mitem["key"].(map[string]interface{})
		key, _ := keyMap["#text"].(string)
		if key == "Response" {
			val, _ := mitem["value"].(map[string]interface{})
			if errorMsg, exists := val["KasFloodDelay"]; exists {
				return libdns.RR{}, fmt.Errorf("API flood delay: %v", errorMsg)
			}
		}
	}

	// Return the record with any updates from the server
	// Since the API doesn't return the new record details, we return the original
	return record, nil
}

func (p *Provider) getRecordByName(ctx context.Context, zone string, record libdns.Record, recursive bool) (allinklRecord, error) {

	for _, crecord := range ChachedRecords[zone] {
		if crecord.Name == record.RR().Name {
			return crecord, nil
		}
	}

	if !recursive {
		p.GetAllRecords(ctx, zone)
		return p.getRecordByName(ctx, zone, record, true)
	}

	return allinklRecord{}, fmt.Errorf("record not found: %s", record.RR().Name)
}

func (p *Provider) SetRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	searchedRecord, err := p.getRecordByName(ctx, zone, record, false)
	if err != nil {
		return libdns.RR{}, fmt.Errorf("record not found: %s", record.RR().Name)
	}

	httpClient := &http.Client{
		Timeout: 1500 * time.Millisecond,
	}

	soap, err := gosoap.SoapClient(ApiBase, httpClient)
	if err != nil {
		return libdns.RR{}, fmt.Errorf("error creating SOAP client: %w", err)
	}

	rr := record.RR()

	if rr.TTL/time.Second < 600 {
		rr.TTL = 600 * time.Second
	}
	ttlInSeconds := int(rr.TTL / time.Second)

	params := map[string]interface{}{
		"record_name": rr.Name,
		"record_type": rr.Type,
		"record_data": rr.Data,
		"record_aux":  ttlInSeconds,
		"record_id":   searchedRecord.ID,
	}
	requestData := map[string]interface{}{
		"kas_login":        p.KasLogin,
		"kas_auth_type":    "plain",
		"kas_auth_data":    p.KasAuthPassword,
		"kas_action":       "update_dns_settings",
		"KasRequestParams": params,
	}

	// JSON encode the entire request like PHP
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return libdns.RR{}, fmt.Errorf("error encoding JSON: %w", err)
	}

	// Call the SOAP method with JSON-encoded params
	res, err := soap.Call("KasApi", gosoap.Params{
		"Params": string(jsonData),
	})

	if err != nil {
		return libdns.RR{}, fmt.Errorf("error calling SOAP method: %w", err)
	}
	if res == nil {
		return libdns.RR{}, fmt.Errorf("response is nil")
	}
	mv, err := mxj.NewMapXml([]byte(res.Body))
	if err != nil {
		return libdns.RR{}, fmt.Errorf("error converting XML to map: %w", err)
	}
	// Parse response to check for success
	root, ok := mv["KasApiResponse"].(map[string]interface{})
	if !ok {
		return libdns.RR{}, fmt.Errorf("invalid response format")
	}
	ret, ok := root["return"].(map[string]interface{})
	if !ok {
		return libdns.RR{}, fmt.Errorf("invalid response format: missing return")
	}

	// Check for errors in response
	items := ret["item"]
	var itemList []interface{}
	switch v := items.(type) {
	case []interface{}:
		itemList = v
	case map[string]interface{}:
		itemList = []interface{}{v}
	}

	for _, item := range itemList {
		mitem, _ := item.(map[string]interface{})
		keyMap, _ := mitem["key"].(map[string]interface{})
		key, _ := keyMap["#text"].(string)
		if key == "Response" {
			val, _ := mitem["value"].(map[string]interface{})
			if errorMsg, exists := val["KasFloodDelay"]; exists {
				return libdns.RR{}, fmt.Errorf("API flood delay: %v", errorMsg)
			}
		}
	}

	// If we reach here, the record was successfully updated
	updatedRecord := libdns.RR{
		Type: record.RR().Type,
		Name: record.RR().Name,
		Data: record.RR().Data,
		TTL:  record.RR().TTL,
	}

	return updatedRecord, nil
}

func (p *Provider) DeleteRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	searchedRecord, err := p.getRecordByName(ctx, zone, record, false)
	if err != nil {
		return libdns.RR{}, fmt.Errorf("record not found: %s", record.RR().Name)
	}

	httpClient := &http.Client{
		Timeout: 1500 * time.Millisecond,
	}

	soap, err := gosoap.SoapClient(ApiBase, httpClient)
	if err != nil {
		return libdns.RR{}, fmt.Errorf("error creating SOAP client: %w", err)
	}
	params := map[string]interface{}{
		"record_id": searchedRecord.ID,
	}
	requestData := map[string]interface{}{
		"kas_login":        p.KasLogin,
		"kas_auth_type":    "plain",
		"kas_auth_data":    p.KasAuthPassword,
		"kas_action":       "delete_dns_settings",
		"KasRequestParams": params,
	}

	// JSON encode the entire request like PHP
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return libdns.RR{}, fmt.Errorf("error encoding JSON: %w", err)
	}

	// Call the SOAP method with JSON-encoded params
	res, err := soap.Call("KasApi", gosoap.Params{
		"Params": string(jsonData),
	})

	if err != nil {
		return libdns.RR{}, fmt.Errorf("error calling SOAP method: %w", err)
	}
	if res == nil {
		return libdns.RR{}, fmt.Errorf("response is nil")
	}
	mv, err := mxj.NewMapXml([]byte(res.Body))
	if err != nil {
		return libdns.RR{}, fmt.Errorf("error converting XML to map: %w", err)
	}
	// Parse response to check for success
	root, ok := mv["KasApiResponse"].(map[string]interface{})
	if !ok {
		return libdns.RR{}, fmt.Errorf("invalid response format")
	}
	ret, ok := root["return"].(map[string]interface{})
	if !ok {
		return libdns.RR{}, fmt.Errorf("invalid response format: missing return")
	}
	// Check for errors in response
	items := ret["item"]
	var itemList []interface{}
	switch v := items.(type) {
	case []interface{}:
		itemList = v
	case map[string]interface{}:
		itemList = []interface{}{v}
	}
	for _, item := range itemList {
		mitem, _ := item.(map[string]interface{})
		keyMap, _ := mitem["key"].(map[string]interface{})
		key, _ := keyMap["#text"].(string)
		if key == "Response" {
			val, _ := mitem["value"].(map[string]interface{})
			if errorMsg, exists := val["KasFloodDelay"]; exists {
				return libdns.RR{}, fmt.Errorf("API flood delay: %v", errorMsg)
			}
		}
	}
	// If we reach here, the record was successfully deleted
	deletedRecord := libdns.RR{
		Type: record.RR().Type,
		Name: record.RR().Name,
		Data: record.RR().Data,
		TTL:  record.RR().TTL,
	}
	// Remove the record from the cache
	for i, crecord := range ChachedRecords[zone] {
		if crecord.ID == searchedRecord.ID {
			ChachedRecords[zone] = append(ChachedRecords[zone][:i], ChachedRecords[zone][i+1:]...)
			break
		}
	}
	return deletedRecord, nil

}
