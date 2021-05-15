package leaseweb

// Structs for easy json marshalling.
// Only declare fields that are used.
type LeasewebRecordSet struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Content []string `json:"content"`
	TTL     int      `json:"ttl"`
}

type LeasewebRecordSets struct {
	ResourceRecordSets []LeasewebRecordSet `json:"resourceRecordSets"`
}
