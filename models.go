package selectelv2

const (
	cApiBaseUrl = "https://api.selectel.ru/domains/v2"
	cGetKeystoneTokenTemplate = "{\"auth\":{\"identity\":{\"methods\":[\"password\"],\"password\":{\"user\":{\"name\":\"{{.User}}\",\"domain\":{\"name\":\"{{.AccountId}}\"},\"password\":\"{{.Password}}\"}}},\"scope\":{\"project\":{\"name\":\"{{.ProjectName}}\",\"domain\":{\"name\":\"{{.AccountId}}\"}}}}}"
	cTokensUrl = "https://cloud.api.selcloud.ru/identity/v3/auth/tokens"
	cKeystoneTokenHeader = "X-Subject-Token"
)

var (
	httpMethods = httpMethod{
		post:   "POST",
		get:    "GET",
		patch:  "PATCH",
		delete: "DELETE",
		put:    "PUT",
	}
	
	recordMethods = recordMethod{
		get:	"GET",
		append:	"APPEND",
		set: 	"SET",
		delete:	"DELETE",
	}
)

type httpMethod struct {
	post string
	get string
	patch string
	delete string
	put string
}

type recordMethod struct {
	get string
	append string
	set string
	delete string
}


type Zones struct {
	Zones []Zone `json:"result"`
}

type Zone struct {
	Name string `json:"name"`
	
	// zoneId by selectel
	ID_ string `json:"id"`
}

type Recordset struct {
	Records []Record `json:"result"`
}

type Record struct {
	ID		string `json:"id,omitempty"`
	Type	string `json:"type"`
	Name	string `json:"name"`
	Value	[]RValue `json:"records"`
	TTL		int `json:"ttl"`
}

type RValue struct {
	Value string `json:"content"`
}

