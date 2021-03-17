package metaname

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (p *Provider) dns_zone(ctx context.Context, zone string) ([]metanameRR, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var records []metanameRR

	fqdn := strings.TrimRight(zone, ".")

	params := []interface{}{fqdn}

	var result metanameResponse

	if err := p.makeRPCRequest(ctx, "dns_zone", params, &result); err != nil {
		return nil, err
	}
	if result.Result == nil {
		return nil, fmt.Errorf("Metaname error: %s", result.Error.Message)
	}

	recs := result.Result.([]interface{})
	for _, r := range recs {
		rr := r.(map[string]interface{})
		aux := -1
		if rr["aux"] == "" {
			aux = 0
		} else if rr["aux"] != nil {
			aux = int(rr["aux"].(float64))
		}
		ttl := 0
		if rr["ttl"] != nil {
			ttl = int(rr["ttl"].(float64))
		}
		newRec := metanameRR{
			Reference: rr["reference"].(string),
			Name:      rr["name"].(string),
			Type:      rr["type"].(string),
			Aux:       aux,
			Ttl:       ttl,
			Data:      rr["data"].(string),
		}
		records = append(records, newRec)
	}

	return records, nil
}

func (p *Provider) create_dns_record(ctx context.Context, zone string, record metanameRR) (string, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	fqdn := strings.TrimRight(zone, ".")

	params := []interface{}{fqdn, record}
	var result metanameResponse
	if err := p.makeRPCRequest(ctx, "create_dns_record", params, &result); err != nil {
		return "", err
	}
	if result.Result == nil {
		errData, _ := json.Marshal(result.Error.Data)
		return "", fmt.Errorf("Metaname error from create_dns_record: %d %s (%s)", result.Error.Code, result.Error.Message, string(errData))
	}
	return result.Result.(string), nil
}

func (p *Provider) update_dns_record(ctx context.Context, zone string, reference string, record metanameRR) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	fqdn := strings.TrimRight(zone, ".")

	params := []interface{}{fqdn, reference, record}
	var result metanameResponse
	if err := p.makeRPCRequest(ctx, "update_dns_record", params, &result); err != nil {
		return err
	}
	if result.Error.Code < 0 {
		errData, _ := json.Marshal(result.Error.Data)
		return fmt.Errorf("Metaname error from update_dns_record: %s (%s)", result.Error.Message, string(errData))
	}
	return nil
}

func (p *Provider) delete_dns_record(ctx context.Context, zone string, reference string) (bool, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	fqdn := strings.TrimRight(zone, ".")

	params := []interface{}{fqdn, reference}

	var result metanameResponse
	if err := p.makeRPCRequest(ctx, "delete_dns_record", params, &result); err != nil {
		return false, err
	}
	if result.Result == nil {
		errData, _ := json.Marshal(result.Error.Data)
		return false, fmt.Errorf("Metaname error from delete_dns_record: %s (%s)", result.Error.Message, string(errData))
	}
	return true, nil

}

func (p *Provider) makeRPCRequest(ctx context.Context, method string, params []interface{}, response *metanameResponse) error {
	var req rpcRequest
	req.Jsonrpc = "2.0"
	req.Id = "abc"
	req.Method = method
	req.Params = append([]interface{}{p.AccountReference, p.APIKey}, params...)

	if p.Endpoint == "" {
		p.Endpoint = "https://metaname.net/api/1.1"
	}

	raw, err := json.Marshal(req)
	if err != nil {
		return err
	}

	hreq, err := http.NewRequestWithContext(ctx, "POST", p.Endpoint, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("error creating http request")
	}
	hreq.Header.Set("Content-type", "application/json")

	resp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		return fmt.Errorf("error performing http request")
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("error decoding JSON: %s", err)
	}

	return nil

}
