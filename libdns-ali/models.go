package alidns

import (
	"time"

	"github.com/libdns/libdns"
)

// aliRecList:the struct of query result
type aliRecList struct {
	ReqID      string         `json:"RequestId,omitempty"`
	TotalCount int            `json:"TotalCount,omitempty"`
	PgSize     int            `json:"PageSize,omitempty"`
	DRecords   aliDomaRecords `json:"DomainRecords,omitempty"`
	PgNum      int            `json:"PageNumber,omitempty"`
}

type aliDomaRecord struct {
	Rr     string `json:"RR,omitempty"`
	Line   string `json:"Line,omitempty"`
	Status string `json:"Status,omitempty"`
	Locked bool   `json:"Locked,omitempty"`
	DTyp   string `json:"Type,omitempty"`
	DName  string `json:"DomainName,omitempty"`
	DVal   string `json:"Value,omitempty"`
	RecID  string `json:"RecordId,omitempty"`
	TTL    int    `json:"TTL,omitempty"`
	Weight int    `json:"Weight,omitempty"`
}

type aliDomaRecords struct {
	Record []aliDomaRecord `json:"Record,omitempty"`
}

type aliResult struct {
	ReqID      string         `json:"RequestId,omitempty"`
	DRecords   aliDomaRecords `json:"DomainRecords,omitempty"`
	DLvl       int            `json:"DomainLevel,omitempty"`
	DVal       string         `json:"Value,omitempty"`
	TTL        int            `json:"TTL,omitempty"`
	DName      string         `json:"DomainName,omitempty"`
	Rr         string         `json:"RR,omitempty"`
	Msg        string         `json:"Message,omitempty"`
	Rcmd       string         `json:"Recommend,omitempty"`
	HostID     string         `json:"HostId,omitempty"`
	Code       string         `json:"Code,omitempty"`
	TotalCount int            `json:"TotalCount,omitempty"`
	PgSize     int            `json:"PageSize,omitempty"`
	PgNum      int            `json:"PageNumber,omitempty"`
	DTyp       string         `json:"Type,omitempty"`
	RecID      string         `json:"RecordId,omitempty"`
	Line       string         `json:"Line,omitempty"`
	Status     string         `json:"Status,omitempty"`
	Locked     bool           `json:"Locked,omitempty"`
	Weight     int            `json:"Weight,omitempty"`
}

func (r *aliDomaRecord) LibdnsRecord() libdns.Record {
	return libdns.Record{
		ID:    r.RecID,
		Type:  r.DTyp,
		Name:  r.Rr,
		Value: r.DVal,
		TTL:   time.Duration(r.TTL) * time.Second,
	}
}

func (r *aliResult) ToDomaRecord() aliDomaRecord {
	return aliDomaRecord{
		RecID:  r.RecID,
		DTyp:   r.DTyp,
		Rr:     r.Rr,
		DName:  r.DName,
		DVal:   r.DVal,
		TTL:    r.TTL,
		Line:   r.Line,
		Status: r.Status,
		Locked: r.Locked,
		Weight: r.Weight,
	}
}

// AlidnsRecord convert libdns.Record to aliDomaRecord
func alidnsRecord(r libdns.Record) aliDomaRecord {
	return aliDomaRecord{
		Rr:    r.Name,
		DTyp:  r.Type,
		DVal:  r.Value,
		RecID: r.ID,
		TTL:   int(r.TTL.Seconds()),
	}
}
