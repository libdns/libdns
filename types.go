// All types in this file are for mapping the JSON data in the netcup DNS API requests or responses

package netcup

import (
	"encoding/json"
)

// dnsRecord is the netcup DNS record structure.
// DeleteRecord determines, whether the record should be deleted on an update
type dnsRecord struct {
	ID           string `json:"id"`
	HostName     string `json:"hostname"`
	RecType      string `json:"type"`
	Priority     int    `json:"priority,string"`
	Destination  string `json:"destination"`
	DeleteRecord bool   `json:"deleterecord"`
}

// Checks, if all the values of two records are the same, disregarding the ID. Needed to determine,
// which records need to be appended or updated.
func (rec *dnsRecord) equals(otherRec dnsRecord) bool {
	return rec.HostName == otherRec.HostName && rec.RecType == otherRec.RecType && rec.Destination == otherRec.Destination && rec.Priority == otherRec.Priority
}

// dnsRecordSet is used by the netcup API to wrap DnsRecords
type dnsRecordSet struct {
	DnsRecords []dnsRecord `json:"dnsrecords"`
}

// apiSessionData is returned by the netcup API in response to the login request and contains the session ID,
// which is needed for all consecutive requests for this session
type apiSessionData struct {
	APISessionId string `json:"apisessionid"`
}

// dnsZone contains information about the zone. Name: the zone name, TTL: time to live in seconds
type dnsZone struct {
	Name string `json:"name"`
	TTL  int64  `json:"ttl,string"`
}

// requestParam contains request parameters for all requests used in this libdns implementation.
// Not all of them are used in every request.
type requestParam struct {
	DomainName     string       `json:"domainname,omitempty"`
	CustomerNumber string       `json:"customernumber"`
	APIKey         string       `json:"apikey"`
	APIPassword    string       `json:"apipassword,omitempty"`
	APISessionID   string       `json:"apisessionid,omitempty"`
	DNSRecordSet   dnsRecordSet `json:"dnsrecordset,omitempty"`
}

// request maps the structure of the JSON body of every request to the netcup DNS API (there are only POST requests)
type request struct {
	Action string       `json:"action"`
	Param  requestParam `json:"param"`
}

// request maps the structure of the JSON body of every netcup DNS API response
type response struct {
	Action       string          `json:"action"`
	Status       string          `json:"status"`
	ShortMessage string          `json:"shortmessage"`
	LongMessage  string          `json:"longmessage"`
	ResponseData json.RawMessage `json:"responsedata"`
}
