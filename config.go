package nicrudns

type Config struct {
	Credentials    *Credentials `json:"credentials,omitempty"`      // structure with credentials of nic.ru
	ZoneName       string       `json:"zone_name,omitempty"`        // zone name
	DnsServiceName string       `json:"dns_service_name,omitempty"` // dns service name from nic.ru
	CachePath      string       `json:"cache_path,omitempty"`       // path to save cached auth token
}

type Credentials struct {
	OAuth2ClientID string `json:"oauth2_client_id,omitempty"`
	OAuth2SecretID string `json:"oauth2_secret_id,omitempty"`
	Username       string `json:"username,omitempty"` //username *****/NIC-D
	Password       string `json:"password,omitempty"`
}
