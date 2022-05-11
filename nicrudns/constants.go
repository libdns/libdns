package nicrudns

const (
	BaseURL                 = `https://api.nic.ru`
	TokenURL                = BaseURL + `/oauth/token`
	GetRecordsUrlPattern    = BaseURL + `/dns-master/services/%s/zones/%s/records`
	DeleteRecordsUrlPattern = BaseURL + `/dns-master/services/%s/zones/%s/records/%d`
	AddRecordsUrlPattern    = BaseURL + `/dns-master/services/%s/zones/%s/records`
	GetServicesUrl          = BaseURL + `/dns-master/services`
	CommitUrlPattern        = BaseURL + `/dns-master/services/%s/zones/%s/commit`
	RollbackUrlPattern      = BaseURL + `/dns-master/services/%s/zones/%s/rollback`
	DownloadZoneUrlPattern  = BaseURL + `/dns-master/services/%s/zones/%s`
	SuccessStatus           = `success`
	OAuth2Scope             = `.+:/dns-master/.+`
)
