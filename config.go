package nicrudns

type Config struct {
	Credentials    *Credentials // structure with credentials of nic.ru
	ZoneName       string       // zone name
	DnsServiceName string       // dns service name from nic.ru
	CachePath      string       // path to save cached auth token
}

type Credentials struct {
	OAuth2ClientID string
	OAuth2SecretID string
	Username       string //username *****/NIC-D
	Password       string
}
