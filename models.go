package dnsexit

type Action int64

const (
	Set Action = iota
	Append
	Delete
)

type dnsExitPayload struct {
	Apikey        string           `json:"apikey"`
	Zone          string           `json:"domain"`
	AddRecords    *[]dnsExitRecord `json:"add,omitempty"`
	DeleteRecords *[]dnsExitRecord `json:"delete,omitempty"`
}

// TODO - look at co-ercing properties of LibDns mx records into MailZone/MailServer properties
// MailZone   string `json:"mail-zone,omitempty"`   // "mail-zone":"",
// MailServer string `json:"mail-server,omitempty"` // "mail-server":"mail2.dnsexit.com",

type dnsExitRecord struct {
	Type      string  `json:"type"`
	Name      string  `json:"name,omitempty"`
	Content   *string `json:"content,omitempty"`
	Priority  *int    `json:"priority,omitempty"`
	TTL       *int    `json:"ttl,omitempty"`
	Overwrite *bool   `json:"overwrite,omitempty"`
}

type dnsExitResponse struct {
	Code    int      `json:"code"`
	Details []string `json:"details"`
	Message string   `json:"message"`
}
