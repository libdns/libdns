package mythicbeasts

type mythicAuthResponse struct {
	Token     string `json:"access_token,omitempty"` // The bearer token for use in API requests
	Lifetime  int    `json:"expires_in,omitempty"`   // The maximum lifetime of the token in seconds
	TokenType string `json:"token_type,omitempty"`   // The token type must be 'bearer'
}

type mythicAuthResponseError struct {
	ErrorMessage     string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

type mythicRecord struct {
	Type  string `json:"type,omitempty"`
	Name  string `json:"host,omitempty"`
	Value string `json:"data,omitempty"`
	TTL   int    `json:"ttl,omitempty"`
}

type mythicRecords struct {
	Records []mythicRecord `json:"records,omitempty"`
}

type mythicRecordUpdate struct {
	Message        string `json:"message,omitempty"`
	RecordsAdded   int    `json:"records_added,omitempty"`
	RecordsRemoved int    `json:"records_removed,omitempty"`
}

type mythicError struct {
	Error string `json:"error,omitempty"`
}

type mythicErrors struct {
	Errors []string `json:",omitempty"`
}
