package nicrudns

type IClient interface {
	AddA(names []string, target string, ttl string) (*Response, error)
	AddAAAA(names []string, target string, ttl string) (*Response, error)
	AddCnames(names []string, target string, ttl string) (*Response, error)
	AddMx(names []string, target string, preference string, ttl string) (*Response, error)
	AddTxt(names []string, target string, ttl string) (*Response, error)
	CommitZone() (*Response, error)
	DeleteRecord(id int) (*Response, error)
	DownloadZone() (string, error)
	GetARecords(nameFilter string, targetFilter string) ([]*RR, error)
	GetAAAARecords(nameFilter string, targetFilter string) ([]*RR, error)
	GetCnameRecords(nameFilter string, targetFilter string) ([]*RR, error)
	GetMxRecords(nameFilter string, targetFilter string) ([]*RR, error)
	GetTxtRecords(nameFilter string, targetFilter string) ([]*RR, error)
	RollbackZone() (*Response, error)
	GetServices() ([]*Service, error)
}
