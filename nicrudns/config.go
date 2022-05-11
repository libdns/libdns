package nicrudns

type Config struct {
	Credentials    *Credentials // структура с данными авторизации в апишке
	ZoneName       string       // имя зоны
	DnsServiceName string       // имя DNS-услуги
	CachePath      string       // путь для хранения токена авторизации
}

type Credentials struct {
	OAuth2ClientID string
	OAuth2SecretID string
	Username       string
	Password       string
}
