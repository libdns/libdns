package mijnhost

import (
	"time"

	"github.com/libdns/libdns"
)

type RecordRequest struct {
	Record Record `json:"record"`
}
type RecordResponse struct {
	Type  string        `json:"type"`
	Name  string        `json:"name"`
	Value string        `json:"value"`
	TTL   time.Duration `json:"ttl"`
}

type RecordsResponse struct {
	Status            uint   `json:"status"`
	StatusDescription string `json:"status_description"`
	Data              struct {
		Records []Record `json:"records"`
	} `json:"data"`
}

type Record struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

type RecordList struct {
	Records []Record `json:"records"`
}

type SavedRecordResponse struct {
	Status            uint   `json:"status"`
	StatusDescription string `json:"status_description"`
}

func (r *Record) libDNSRecord(zone string) libdns.Record {
	return libdns.Record{
		//ID:    fmt.Sprintf("%d", r.DNSRecord.ID),
		Name:  libdns.RelativeName(r.Name, zone),
		Type:  r.Type,
		Value: r.Value,
		TTL:   time.Duration(r.TTL),
	}
}

func (r *RecordResponse) libDNSRecord(zone string) libdns.Record {
	return libdns.Record{
		// ID:       fmt.Sprintf("%d", r.ID),
		Name:  libdns.RelativeName(r.Name, zone),
		Type:  r.Type,
		Value: r.Value,
		TTL:   r.TTL,
		// Priority: r.Priority,
	}
}
func libdnsToRecordRequest(r libdns.Record) RecordRequest {
	return RecordRequest{
		Record: libdnsToRecord(r),
	}
}

func libdnsToRecord(r libdns.Record) Record {
	return Record{
		Type:  r.Type,
		Value: r.Value,
		Name:  r.Name,
		TTL:   int(r.TTL),
	}
}

func libdnsToRecords(r []libdns.Record) []Record {
	result := make([]Record, len(r)) // Create a new slice with the same length
	for _, v := range r {
		result = append(result, libdnsToRecord(v)) // Apply function
	}
	return result
}

func libdnsToRecordList(r []libdns.Record) RecordList {
	return RecordList{
		Records: libdnsToRecords(r),
	}
}
