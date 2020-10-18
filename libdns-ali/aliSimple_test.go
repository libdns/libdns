package alidns

import (
	"context"
	"fmt"
	"testing"
)

func Test_URLEncode(t *testing.T) {
	s0 := urlEncode("AccessKeyId=testid&Action=DescribeDomainRecords")
	if s0 != "AccessKeyId%3Dtestid%26Action%3DDescribeDomainRecords" {
		t.Fail()
	}
	t.Log(s0)
}

var cl0 = &AliClint{
	APIHost: fmt.Sprintf(addrOfAPI, "https"),
	reqMap: []VKey{
		VKey{key: "AccessKeyId", val: "testid"},
		VKey{key: "Acction", val: "DescribeDomainRecords"},
		VKey{key: "SignatureMethod", val: "HMAC-SHA1"},
		VKey{key: "DomainName", val: "example.com"},
		VKey{key: "SignatureVersion", val: "1.0"},
		VKey{key: "SignatureNonce", val: "f59ed6a9-83fc-473b-9cc6-99c95df3856e"},
		VKey{key: "Timestamp", val: "2016-03-24T16:41:54Z"},
		VKey{key: "Version", val: "2015-01-09"},
	},
	sigStr: "",
	sigPwd: "testsecret",
}

func Test_AliClintReq(t *testing.T) {
	str := cl0.reqMapToStr()
	t.Log("map to str:" + str + "\n")
	str = cl0.reqStrToSign(str, "GET")
	t.Log("sign str:" + str + "\n")
	t.Log("signed base64:" + signStr(str, cl0.sigPwd) + "\n")
}

func Test_AppendDupReq(t *testing.T) {
	err := cl0.addReqBody("Version", "100")
	if err == nil {
		t.Fail()
	}
}

var p0 = Provider{
	AccKeyID:     "LTAI4G3UTA4x2XRm8HrgGJ63",
	AccKeySecret: "wO0q5OUKPWg8Iuy63VNuxLsdHGSH6d",
}

func Test_RequestUrl(t *testing.T) {
	p0.getClient()
	p0.Clint.addReqBody("Action", "DescribeDomainRecords")
	p0.Clint.addReqBody("DomainName", "viscrop.top")
	p0.Clint.setReqBody("Timestamp", "2020-10-16T20:10:54Z")
	r, err := p0.applyReq(context.TODO(), "GET", nil)
	t.Log("url:", r.URL.String(), "err:", err)
}
