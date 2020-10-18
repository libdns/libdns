package alidns

import (
	"context"
	"testing"
)

func Test_ClientAPIReq(t *testing.T) {
	p0.getClient()
	p0.Clint.addReqBody("Action", "DescribeDomainRecords")
	p0.Clint.addReqBody("KeyWords", "vi")
	//p0.Clint.SetReqBody("Timestamp", "2020-10-16T22:38:54Z")
	var rs aliDomaRecords
	rspData := aliResult{}
	err := p0.doAPIRequest(context.TODO(), &rspData)
	t.Log("data", rspData, "err:", err, "rs:", rs)
}

func Test_QueryDomainRecord(t *testing.T) {
	rr, name, _ := p0.queryMainDomain(context.Background(), "me.viscrop.top")
	r0, err := p0.queryDomainRecord(context.TODO(), rr, name)
	t.Log("result:", r0, "err:", err)
}

func Test_QueryDomainRecords(t *testing.T) {
	_, name, _ := p0.queryMainDomain(context.Background(), "me.viscrop.top")
	r0, err := p0.queryDomainRecords(context.TODO(), name)
	t.Log("result:", r0, "err:", err)
	_, name, _ = p0.queryMainDomain(context.Background(), "me.viscraop.top")
	r0, err = p0.queryDomainRecords(context.TODO(), name)
	t.Log("result:", r0, "err:", err)
}

func Test_DomainRecordOp(t *testing.T) {
	dr0 := aliDomaRecord{
		DName: "viscrop.top",
		Rr:    "baidu",
		DTyp:  "CNAME",
		DVal:  "baidu.com",
		TTL:   600,
	}
	r0, err := p0.addDomainRecord(context.TODO(), dr0)
	t.Log("result:", r0, "err:", err)
	dr0, err = p0.getDomainRecord(context.TODO(), r0)
	t.Log("result:", dr0, "err:", err)
	dr0.Rr = "bai"
	r0, err = p0.setDomainRecord(context.TODO(), dr0)
	t.Log("result:", r0, "err:", err)
	r0, err = p0.delDomainRecord(context.TODO(), dr0)
	t.Log("result:", r0, "err:", err)
}
