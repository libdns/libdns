package alidns

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const defRegID string = "cn-hangzhou"
const addrOfAPI string = "%s://alidns.aliyuncs.com/"
const dbgTAG string = "DEBUG:>\t"

// CredInfo implements param of the crediential
type CredInfo struct {
	AccKeyID     string `json:"access_key_id"`
	AccKeySecret string `json:"access_key_secret"`
	RegionID     string `json:"region_id,omitempty"`
}

// AliClint abstructs the alidns.Client
type AliClint struct {
	mutex   sync.Mutex
	APIHost string
	reqMap  []VKey
	sigStr  string
	sigPwd  string
}

// VKey implments of K-V struct
type VKey struct {
	key string
	val string
}

func newCredInfo(pAccKeyID, pAccKeySecret, pRegionID string) *CredInfo {
	if pAccKeyID == "" || pAccKeySecret == "" {
		return nil
	}
	if len(pRegionID) == 0 {
		pRegionID = defRegID
	}
	return &CredInfo{
		AccKeyID:     pAccKeyID,
		AccKeySecret: pAccKeySecret,
		RegionID:     pRegionID,
	}
}

func (c *Client) getAliClient(cred *CredInfo) error {
	cl0, err := c.Clint.getAliClientSche(cred, "https")
	if err != nil {
		return err
	}
	c.Clint = cl0
	return nil
}

func (c *Client) applyReq(cxt context.Context, method string, body io.Reader) (*http.Request, error) {
	if method == "" {
		method = "GET"
	}
	c0 := c.Clint
	c0.signReq(method)
	si0 := fmt.Sprintf("%s=%s", "Signature", c0.sigStr)
	mURL := fmt.Sprintf("%s?%s&%s", c0.APIHost, c0.reqMapToStr(), si0)
	req, err := http.NewRequestWithContext(cxt, method, mURL, body)
	req.Header.Set("Accept", "application/json")
	if err != nil {
		return &http.Request{}, err
	}
	return req, nil
}

func (c *AliClint) getAliClientSche(cred *CredInfo, scheme string) (*AliClint, error) {
	if cred == nil {
		return &AliClint{}, errors.New("alicloud: credentials missing")
	}
	if scheme == "" {
		scheme = "http"
	}

	cl0 := &AliClint{
		APIHost: fmt.Sprintf(addrOfAPI, scheme),
		reqMap: []VKey{
			VKey{key: "AccessKeyId", val: cred.AccKeyID},
			VKey{key: "Format", val: "json"},
			VKey{key: "SignatureMethod", val: "HMAC-SHA1"},
			VKey{key: "SignatureNonce", val: fmt.Sprintf("%d", time.Now().UnixNano())},
			VKey{key: "SignatureVersion", val: "1.0"},
			VKey{key: "Timestamp", val: time.Now().UTC().Format("2006-01-02T15:04:05Z")},
			VKey{key: "Version", val: "2015-01-09"},
		},
		sigStr: "",
		sigPwd: cred.AccKeySecret,
	}

	return cl0, nil
}

func (c *AliClint) signReq(method string) error {
	if c.sigPwd == "" || len(c.reqMap) == 0 {
		return errors.New("alicloud: AccessKeySecret or Request(includes AccessKeyId) is Misssing")
	}
	sort.Sort(byKey(c.reqMap))
	str := c.reqMapToStr()
	fmt.Println(dbgTAG+"req to str:", str)
	str = c.reqStrToSign(str, method)
	fmt.Println(dbgTAG+"url to sign:", str)
	c.sigStr = signStr(str, c.sigPwd)
	return nil
}

func (c *AliClint) addReqBody(key string, value string) error {
	if key == "" && value == "" {
		return errors.New("Key or Value is Empty")
	}
	el := VKey{key: key, val: value}
	c.mutex.Lock()
	for _, el0 := range c.reqMap {
		if el.key == el0.key {
			c.mutex.Unlock()
			return errors.New("Duplicate Keys")
		}
	}
	c.reqMap = append(c.reqMap, el)
	c.mutex.Unlock()
	return nil
}

func (c *AliClint) setReqBody(key string, value string) error {
	if key == "" && value == "" {
		return errors.New("Key or Value is Empty")
	}
	el := VKey{key: key, val: value}
	c.mutex.Lock()
	for in, el0 := range c.reqMap {
		if el.key == el0.key {
			(c.reqMap)[in] = el
			c.mutex.Unlock()
			return nil
		}
	}
	c.mutex.Unlock()
	return fmt.Errorf("Entry of %s not found", key)
}

func (c *AliClint) reqStrToSign(ins string, method string) string {
	if method == "" {
		method = "GET"
	}
	ecReq := urlEncode(ins)
	return fmt.Sprintf("%s&%s&%s", method, "%2F", ecReq)
}

func (c *AliClint) reqMapToStr() string {
	//str := ""
	m0 := c.reqMap
	urlEn := url.Values{}
	c.mutex.Lock()
	if m0 != nil {
		for _, o := range m0 {
			urlEn.Add(o.key, o.val)
		}
	}
	c.mutex.Unlock()
	return urlEn.Encode()
}

func signStr(ins string, sec string) string {
	sec = sec + "&"
	hm := hmac.New(sha1.New, []byte(sec))
	hm.Write([]byte(ins))
	sum := hm.Sum(nil)
	return base64.StdEncoding.EncodeToString(sum)
}

func urlEncode(ins string) string {
	str0 := ins
	str0 = strings.Replace(str0, "+", "%20", -1)
	str0 = strings.Replace(str0, "*", "%2A", -1)
	str0 = strings.Replace(str0, "%7E", "~", -1)
	str0 = url.QueryEscape(str0)
	return str0
}

type byKey []VKey

func (v byKey) Len() int {
	return len(v)
}

func (v byKey) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v byKey) Less(i, j int) bool {
	return v[i].key < v[j].key
}
