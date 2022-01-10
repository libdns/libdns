// Direct implementation of all required netucp DNS API endpoints

package netcup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// fixed netcup API URL, may be made variable later
const apiUrl = "https://ccp.netcup.net/run/webservice/servers/endpoint.php?JSON"

const loggingPrefixNetcup = "[netcup]"

// Executes a request to the netcup API with a given request value.
// Returns the response with raw response data, which needs to be unmarshalled  depending on the request.
func (p *Provider) doRequest(ctx context.Context, req request) (*response, error) {
	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiUrl, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	defer httpResp.Body.Close()

	responseBody, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	var response response
	if err = json.Unmarshal(responseBody, &response); err != nil {
		return nil, err
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("%v %v: %v", loggingPrefixNetcup, response.ShortMessage, response.LongMessage)
	}

	fmt.Printf("%v %v: %v\n", loggingPrefixNetcup, response.ShortMessage, response.LongMessage)

	return &response, nil
}

// login starts an API session that lasts for some minutes (see nectup API documentation).
// The session ID is returned, which is needed for all other requests.
func (p *Provider) login(ctx context.Context) (string, error) {
	loginRequest := request{
		Action: "login",
		Param: requestParam{
			CustomerNumber: p.CustomerNumber,
			APIKey:         p.APIKey,
			APIPassword:    p.APIPassword,
		},
	}

	res, err := p.doRequest(ctx, loginRequest)
	if err != nil {
		return "", err
	}

	var asd apiSessionData
	if err = json.Unmarshal(res.ResponseData, &asd); err != nil {
		return "", err
	}

	return asd.APISessionId, nil
}

// Stops the session with the given session ID.
func (p *Provider) logout(ctx context.Context, apiSessionID string) {
	logoutRequest := request{
		Action: "logout",
		Param: requestParam{
			CustomerNumber: p.CustomerNumber,
			APIKey:         p.APIKey,
			APISessionID:   apiSessionID,
		},
	}

	p.doRequest(ctx, logoutRequest)
}

// Provides information about the given zone, especially the TTL
func (p *Provider) infoDNSZone(ctx context.Context, zone string, apiSessionID string) (*dnsZone, error) {
	infoDNSZoneRequest := request{
		Action: "infoDnsZone",
		Param: requestParam{
			DomainName:     zone,
			CustomerNumber: p.CustomerNumber,
			APIKey:         p.APIKey,
			APISessionID:   apiSessionID,
		},
	}

	res, err := p.doRequest(ctx, infoDNSZoneRequest)
	if err != nil {
		return nil, err
	}

	var dz dnsZone
	if err = json.Unmarshal(res.ResponseData, &dz); err != nil {
		return nil, err
	}

	return &dz, nil
}

// Returns a slice of all records found in the given zone.
func (p *Provider) infoDNSRecords(ctx context.Context, zone string, apiSessionID string) (*dnsRecordSet, error) {
	infoDNSrecordsRequest := request{
		Action: "infoDnsRecords",
		Param: requestParam{
			DomainName:     zone,
			CustomerNumber: p.CustomerNumber,
			APIKey:         p.APIKey,
			APISessionID:   apiSessionID,
		},
	}

	res, err := p.doRequest(ctx, infoDNSrecordsRequest)
	if err != nil {
		return nil, err
	}

	var recordSet dnsRecordSet
	if err = json.Unmarshal(res.ResponseData, &recordSet); err != nil {
		return nil, err
	}

	return &recordSet, err
}

// Updates records in the given zone with the values in the dnsRecordSet. Records are appended when no ID is set and updated when
// an ID is set and it exists. Returns all records found in the zone (with the appends and updates applied).
func (p *Provider) updateDNSRecords(ctx context.Context, zone string, updateRecordSet dnsRecordSet, apiSessionID string) (*dnsRecordSet, error) {
	updateDNSrecordsRequest := request{
		Action: "updateDnsRecords",
		Param: requestParam{
			DomainName:     zone,
			CustomerNumber: p.CustomerNumber,
			APIKey:         p.APIKey,
			APISessionID:   apiSessionID,
			DNSRecordSet:   updateRecordSet,
		},
	}

	res, err := p.doRequest(ctx, updateDNSrecordsRequest)
	if err != nil {
		return nil, err
	}

	var recordSet dnsRecordSet
	if err = json.Unmarshal(res.ResponseData, &recordSet); err != nil {
		return nil, err
	}

	return &recordSet, err
}
