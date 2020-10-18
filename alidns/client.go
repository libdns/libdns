package alidns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
)

// Client is an abstration of AliClint
type Client struct {
	Clint *AliClint
	mutex sync.Mutex
}

func (p *Provider) getClient() error {
	cred := newCredInfo(p.AccKeyID, p.AccKeySecret, p.RegionID)
	return p.getAliClient(cred)
}

func (p *Provider) addDomainRecord(ctx context.Context, rc aliDomaRecord) (recID string, err error) {
	p.mutex.Lock()
	p.getClient()
	p.Clint.addReqBody("Action", "AddDomainRecord")
	p.Clint.addReqBody("DomainName", rc.DName)
	p.Clint.addReqBody("RR", rc.Rr)
	p.Clint.addReqBody("Type", rc.DTyp)
	p.Clint.addReqBody("Value", rc.DVal)
	p.Clint.addReqBody("TTL", fmt.Sprintf("%d", rc.TTL))
	rs := aliResult{}
	err = p.doAPIRequest(ctx, &rs)
	recID = rs.RecID
	p.mutex.Unlock()
	if err != nil {
		return "", err
	}
	return recID, err
}

func (p *Provider) delDomainRecord(ctx context.Context, rc aliDomaRecord) (recID string, err error) {
	p.mutex.Lock()
	p.getClient()
	p.Clint.addReqBody("Action", "DeleteDomainRecord")
	p.Clint.addReqBody("RecordId", rc.RecID)
	rs := aliResult{}
	err = p.doAPIRequest(ctx, &rs)
	recID = rs.RecID
	p.mutex.Unlock()
	if err != nil {
		return "", err
	}
	return recID, err
}

func (p *Provider) setDomainRecord(ctx context.Context, rc aliDomaRecord) (recID string, err error) {
	p.mutex.Lock()
	p.getClient()
	p.Clint.addReqBody("Action", "UpdateDomainRecord")
	p.Clint.addReqBody("RecordId", rc.RecID)
	p.Clint.addReqBody("RR", rc.Rr)
	p.Clint.addReqBody("Type", rc.DTyp)
	p.Clint.addReqBody("Value", rc.DVal)
	p.Clint.addReqBody("TTL", fmt.Sprintf("%d", rc.TTL))
	rs := aliResult{}
	err = p.doAPIRequest(ctx, &rs)
	recID = rs.RecID
	p.mutex.Unlock()
	if err != nil {
		return "", err
	}
	return recID, err
}

func (p *Provider) getDomainRecord(ctx context.Context, recID string) (aliDomaRecord, error) {
	p.mutex.Lock()
	p.getClient()
	p.Clint.addReqBody("Action", "DescribeDomainRecordInfo")
	p.Clint.addReqBody("RecordId", recID)
	rs := aliResult{}
	err := p.doAPIRequest(ctx, &rs)
	rec := rs.ToDomaRecord()
	p.mutex.Unlock()
	if err != nil {
		return aliDomaRecord{}, err
	}
	return rec, err
}

func (p *Provider) queryDomainRecords(ctx context.Context, name string) ([]aliDomaRecord, error) {
	p.mutex.Lock()
	p.getClient()
	p.Clint.addReqBody("Action", "DescribeDomainRecords")
	p.Clint.addReqBody("DomainName", name)
	rs := aliResult{}
	err := p.doAPIRequest(ctx, &rs)
	p.mutex.Unlock()
	if err != nil {
		return []aliDomaRecord{}, err
	}
	return rs.DRecords.Record, err
}

func (p *Provider) queryDomainRecord(ctx context.Context, rr string, name string) (aliDomaRecord, error) {
	p.mutex.Lock()
	p.getClient()
	p.Clint.addReqBody("Action", "DescribeDomainRecords")
	p.Clint.addReqBody("DomainName", name)
	p.Clint.addReqBody("KeyWord", rr)
	p.Clint.addReqBody("SearchMode", "EXACT")
	rs := aliResult{}
	err := p.doAPIRequest(ctx, &rs)
	p.mutex.Unlock()
	if err != nil {
		return aliDomaRecord{}, err
	}
	if len(rs.DRecords.Record) == 0 {
		return aliDomaRecord{}, errors.New("the rr of the domain not found")
	}
	return rs.DRecords.Record[0], err
}

//queryMainDomain rseserved for absolute names to name,zone
func (p *Provider) queryMainDomain(ctx context.Context, name string) (string, string, error) {
	p.mutex.Lock()
	p.getClient()
	p.Clint.addReqBody("Action", "GetMainDomainName")
	p.Clint.addReqBody("InputString", name)
	rs := aliResult{}
	err := p.doAPIRequest(ctx, &rs)
	p.mutex.Unlock()
	fmt.Println("err:", err, "rs:", rs)
	if err != nil {
		return "", "", err
	}
	return rs.Rr, rs.DName, err
}

func (p *Provider) doAPIRequest(ctx context.Context, result interface{}) error {
	req, err := p.applyReq(ctx, "GET", nil)
	if err != nil {
		return err
	}
	fmt.Println(dbgTAG+"url:", req.URL.String(), "err:", err)

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()

	var buf []byte
	buf, err = ioutil.ReadAll(rsp.Body)
	strBody := string(buf)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(strBody), result)
	if err != nil {
		return err
	}
	if rsp.StatusCode != 200 {
		return fmt.Errorf("get error status: HTTP %d: %+v", rsp.StatusCode, result.(*aliResult).Msg)
	}
	p.Clint = nil
	return err
}
