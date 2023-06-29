package easydns

// ZoneRecordResponse is the response from the EasyDNS API for a zone record request.
type ZoneRecordResponse struct {
	Timestamp int `json:"tm,omitempty"` // Result Timestamp
	Data      []struct {
		Id           string `json:"id,omitempty"`
		Domain       string `json:"domain,omitempty"`
		Host         string `json:"host,omitempty"`
		TTL          string `json:"ttl,omitempty"`
		Priority     string `json:"prio,omitempty"`
		Type         string `json:"type,omitempty"`
		Rdata        string `json:"rdata,omitempty"`
		LastModified string `json:"last_mod,omitempty"`
	} `json:"data,omitempty"`
	Count  int `json:"count,omitempty"`
	Total  int `json:"total,omitempty"`
	Start  int `json:"start,omitempty"`
	Max    int `json:"max,omitempty"`
	Status int `json:"status,omitempty"`
}

// AddEntry is the request body for adding a record to a zone (PUT request)
type AddEntry struct {
	Domain   string `json:"domain,omitempty"`
	Host     string `json:"host,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
	Priority int    `json:"prio,omitempty"`
	Type     string `json:"type,omitempty"`
	Rdata    string `json:"rdata,omitempty"`
}

// UpdateEntry is the request body for updating a record in a zone (POST request)
type UpdateEntry struct {
	Host  string `json:"host,omitempty"`
	TTL   int    `json:"ttl,omitempty"`
	Type  string `json:"type,omitempty"`
	Rdata string `json:"rdata,omitempty"`
}
