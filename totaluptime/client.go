package totaluptime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

var (
	// APIbase is the base provider API URL.
	APIbase = "https://api.totaluptime.com/CloudDNS/Domain"

	// DomainIDs is a cache of domain-to-ID cross-references.
	DomainIDs map[string]string

	// RecordIDs is a cache of domain/record-to-ID cross-references.
	RecordIDs map[string]string

	// Client is the currently-active HTTP client.
	Client http.Client
)

// getDomain returns the domain name as zone without trailing dot.
func getDomain(zone string) string {
	return strings.TrimSuffix(zone, ".")
}

// performAPIcall sets headers and sends the HTTP request across the wire.
func (p *Provider) performAPIcall(req *http.Request, result interface{}) error {
	auth := p.Username + ":" + p.Password
	basicAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Add("Authorization", "Basic "+basicAuth)
	req.Header.Set("Accept", "application/json")

	// execute API call
	resp, err := Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}

	return nil
}

// Domains type stores details about all domains in the account.
type Domains struct {
	Rows []struct {
		DomainName string `json:"domainName"`
		ID         string `json:"id"`
	} `json:"rows"`
}

// lookupDomainIDs will check and populate the DomainIDs cache under these conditions:
// 1) forceAPI is true; or 2) the current cache is empty and requires an API call.
func (p *Provider) lookupDomainIDs(forceAPI ...bool) error {
	// if forceAPI not specified, use cache if it exists
	if forceAPI == nil {
		if len(DomainIDs) > 0 {
			return nil // use existing cache
		}
	}

	// if forceAPI explicitly specified as false, use cache
	if len(forceAPI) > 0 {
		if !forceAPI[0] { // not true
			if len(DomainIDs) > 0 {
				return nil // use existing cache
			}
		}
	}

	// live API call to retrieve domains
	DomainIDs = make(map[string]string)

	// configure http request
	req, err := http.NewRequest(http.MethodGet, APIbase+"/All", nil)
	if err != nil {
		return err
	}

	var domains Domains
	err = p.performAPIcall(req, &domains)
	if err != nil {
		return err
	}

	// refresh the cache
	for _, d := range domains.Rows {
		DomainIDs[d.DomainName] = d.ID
	}

	return nil
}

// getDomainID returns the cached ID of the specified domain.
func (p *Provider) getDomainID(domain string) string {
	err := p.lookupDomainIDs()
	if err != nil {
		log.Println(err)
		return "" // error accessing cache
	}

	if id, ok := DomainIDs[domain]; ok {
		return id // cross-reference
	}

	log.Printf("Domain not found: %s\n", domain)
	return "" // domain not found
}

// TotalUptimeRecords type stores details about resource records in a domain.
type TotalUptimeRecords struct {
	StatusCode  interface{} `json:"StatusCode"`
	Type        interface{} `json:"Type"`
	A6Record    interface{} `json:"A6Record"`
	AAAARecord  interface{} `json:"AAAARecord"`
	AFSDBRecord interface{} `json:"AFSDBRecord"`
	ARecord     struct {
		Message interface{} `json:"message"`
		Page    int         `json:"page"`
		Rows    []struct {
			StatusCode   interface{} `json:"StatusCode"`
			Type         interface{} `json:"Type"`
			AALFPac      string      `json:"aALFPac"`
			AALFPacID    string      `json:"aALFPacId"`
			AFailover    string      `json:"aFailover"`
			AFailoverID  string      `json:"aFailoverId"`
			AGeoZone     string      `json:"aGeoZone"`
			AGeoZoneID   string      `json:"aGeoZoneId"`
			AHostName    string      `json:"aHostName"`
			AIPAddress   string      `json:"aIPAddress"`
			AStamp       string      `json:"aStamp"`
			ATTL         string      `json:"aTTL"`
			AltIPAddress interface{} `json:"altIPAddress"`
			DomainID     interface{} `json:"domainID"`
			DomainName   interface{} `json:"domainName"`
			Errors       string      `json:"errors"`
			ID           string      `json:"id"`
			IsSameTTL    bool        `json:"isSameTTL"`
			Status       string      `json:"status"`
		} `json:"rows"`
		Status       string `json:"status"`
		Totalpages   int    `json:"totalpages"`
		Totalrecords int    `json:"totalrecords"`
		Userdata     string `json:"userdata"`
	} `json:"ARecord"`
	ATMARecord  interface{} `json:"ATMARecord"`
	CAARecord   interface{} `json:"CAARecord"`
	CNAMERecord struct {
		Message interface{} `json:"message"`
		Page    int         `json:"page"`
		Rows    []struct {
			StatusCode    interface{} `json:"StatusCode"`
			Type          interface{} `json:"Type"`
			CnameAliasFor string      `json:"cnameAliasFor"`
			CnameName     string      `json:"cnameName"`
			CnameStamp    string      `json:"cnameStamp"`
			CnameTTL      string      `json:"cnameTTL"`
			DomainID      interface{} `json:"domainID"`
			Errors        string      `json:"errors"`
			ID            string      `json:"id"`
			IsSameTTL     bool        `json:"isSameTTL"`
			Status        string      `json:"status"`
		} `json:"rows"`
		Status       string `json:"status"`
		Totalpages   int    `json:"totalpages"`
		Totalrecords int    `json:"totalrecords"`
		Userdata     string `json:"userdata"`
	} `json:"CNAMERecord"`
	DNAMERecord interface{} `json:"DNAMERecord"`
	DSRecord    interface{} `json:"DSRecord"`
	HINFORecord interface{} `json:"HINFORecord"`
	ISDNRecord  interface{} `json:"ISDNRecord"`
	LOCRecord   interface{} `json:"LOCRecord"`
	MBRecord    interface{} `json:"MBRecord"`
	MGRecord    interface{} `json:"MGRecord"`
	MINFORecord interface{} `json:"MINFORecord"`
	MRRecord    interface{} `json:"MRRecord"`
	MXRecord    struct {
		Message interface{} `json:"message"`
		Page    int         `json:"page"`
		Rows    []struct {
			StatusCode   interface{} `json:"StatusCode"`
			Type         interface{} `json:"Type"`
			DomainID     interface{} `json:"domainID"`
			Errors       string      `json:"errors"`
			ID           string      `json:"id"`
			IsSameTTL    bool        `json:"isSameTTL"`
			MxDomainName string      `json:"mxDomainName"`
			MxMailServer string      `json:"mxMailServer"`
			MxPreference string      `json:"mxPreference"`
			MxStamp      string      `json:"mxStamp"`
			MxTTL        string      `json:"mxTTL"`
			Status       string      `json:"status"`
		} `json:"rows"`
		Status       string `json:"status"`
		Totalpages   int    `json:"totalpages"`
		Totalrecords int    `json:"totalrecords"`
		Userdata     string `json:"userdata"`
	} `json:"MXRecord"`
	NAPTRRecord interface{} `json:"NAPTRRecord"`
	NSAPRecord  interface{} `json:"NSAPRecord"`
	NSRecord    struct {
		Message interface{} `json:"message"`
		Page    int         `json:"page"`
		Rows    []struct {
			StatusCode interface{} `json:"StatusCode"`
			Type       interface{} `json:"Type"`
			DomainID   interface{} `json:"domainID"`
			Errors     string      `json:"errors"`
			ID         string      `json:"id"`
			IsSameTTL  bool        `json:"isSameTTL"`
			NsHostName string      `json:"nsHostName"`
			NsName     string      `json:"nsName"`
			NsStamp    string      `json:"nsStamp"`
			NsTTL      string      `json:"nsTTL"`
			Status     string      `json:"status"`
		} `json:"rows"`
		Status       string `json:"status"`
		Totalpages   int    `json:"totalpages"`
		Totalrecords int    `json:"totalrecords"`
		Userdata     string `json:"userdata"`
	} `json:"NSRecord"`
	PTRRecord interface{} `json:"PTRRecord"`
	RPRecord  interface{} `json:"RPRecord"`
	RTRecord  interface{} `json:"RTRecord"`
	SOARecord struct {
		Message interface{} `json:"message"`
		Page    int         `json:"page"`
		Rows    []struct {
			StatusCode          interface{} `json:"StatusCode"`
			Type                interface{} `json:"Type"`
			DomainID            interface{} `json:"domainID"`
			Errors              string      `json:"errors"`
			ID                  string      `json:"id"`
			SoaExpireTime       string      `json:"soaExpireTime"`
			SoaMinDefaultTTL    string      `json:"soaMinDefaultTTL"`
			SoaPrimaryDNS       string      `json:"soaPrimaryDNS"`
			SoaRefreshInterval  string      `json:"soaRefreshInterval"`
			SoaResponsibleEmail string      `json:"soaResponsibleEmail"`
			SoaRetryInterval    string      `json:"soaRetryInterval"`
			SoaSerialNumber     string      `json:"soaSerialNumber"`
			SoaStamp            string      `json:"soaStamp"`
			SoaTTL              string      `json:"soaTTL"`
			SoaVersionNumber    string      `json:"soaVersionNumber"`
			Status              string      `json:"status"`
		} `json:"rows"`
		Status       string `json:"status"`
		Totalpages   int    `json:"totalpages"`
		Totalrecords int    `json:"totalrecords"`
		Userdata     string `json:"userdata"`
	} `json:"SOARecord"`
	SPFRecord interface{} `json:"SPFRecord"`
	SRVRecord struct {
		Message interface{} `json:"message"`
		Page    int         `json:"page"`
		Rows    []struct {
			StatusCode    interface{} `json:"StatusCode"`
			Type          interface{} `json:"Type"`
			DomainID      interface{} `json:"domainID"`
			Errors        string      `json:"errors"`
			ID            string      `json:"id"`
			IsSameTTL     bool        `json:"isSameTTL"`
			Port          string      `json:"port"`
			Priority      string      `json:"priority"`
			SrvDomainName string      `json:"srvDomainName"`
			SrvStamp      string      `json:"srvStamp"`
			SrvTTL        string      `json:"srvTTL"`
			Status        string      `json:"status"`
			Target        string      `json:"target"`
			Weight        string      `json:"weight"`
		} `json:"rows"`
		Status       string `json:"status"`
		Totalpages   int    `json:"totalpages"`
		Totalrecords int    `json:"totalrecords"`
		Userdata     string `json:"userdata"`
	} `json:"SRVRecord"`
	TLSARecord interface{} `json:"TLSARecord"`
	TXTRecord  struct {
		Message interface{} `json:"message"`
		Page    int         `json:"page"`
		Rows    []struct {
			StatusCode  interface{} `json:"StatusCode"`
			Type        interface{} `json:"Type"`
			DomainID    interface{} `json:"domainID"`
			Errors      string      `json:"errors"`
			ID          string      `json:"id"`
			IsSameTTL   bool        `json:"isSameTTL"`
			Status      string      `json:"status"`
			TxtHostName string      `json:"txtHostName"`
			TxtStamp    string      `json:"txtStamp"`
			TxtTTL      string      `json:"txtTTL"`
			TxtText     string      `json:"txtText"`
		} `json:"rows"`
		Status       string `json:"status"`
		Totalpages   int    `json:"totalpages"`
		Totalrecords int    `json:"totalrecords"`
		Userdata     string `json:"userdata"`
	} `json:"TXTRecord"`
	Type257Record     interface{} `json:"Type257Record"`
	WebRedirectRecord struct {
		Message interface{} `json:"message"`
		Page    int         `json:"page"`
		Rows    []struct {
			StatusCode       interface{} `json:"StatusCode"`
			Type             interface{} `json:"Type"`
			CarryPath        string      `json:"CarryPath"`
			Dest             string      `json:"Dest"`
			HostName         string      `json:"HostName"`
			RedirectType     string      `json:"RedirectType"`
			WebRedirectStamp string      `json:"WebRedirectStamp"`
			DomainID         interface{} `json:"domainID"`
			Errors           string      `json:"errors"`
			ID               string      `json:"id"`
			Status           string      `json:"status"`
		} `json:"rows"`
		Status       string `json:"status"`
		Totalpages   int    `json:"totalpages"`
		Totalrecords int    `json:"totalrecords"`
		Userdata     string `json:"userdata"`
	} `json:"WebRedirectRecord"`
	X25Record interface{} `json:"X25Record"`
	Message   interface{} `json:"message"`
	Status    string      `json:"status"`
	Userdata  interface{} `json:"userdata"`
}

// lookupRecordIDs will check and populate the RecordIDs cache under these conditions:
// 1) forceAPI is true; or 2) the current cache for this domain is empty and requires an API call.
// This cache is per-domain and will reset when switching domains; therefore most efficient
// when multiple record manipulations for the same domain are grouped together if possible.
func (p *Provider) lookupRecordIDs(domain string, forceAPI ...bool) error {
	// if forceAPI not specified, use cache if it exists
	if forceAPI == nil {
		if _, ok := RecordIDs[domain]; ok { // some records for this domain exist
			return nil // use existing cache
		}
	}

	// if forceAPI explicitly specified as false, use cache
	if len(forceAPI) > 0 {
		if !forceAPI[0] { // not true
			if _, ok := RecordIDs[domain]; ok { // some records for this domain exist
				return nil // use existing cache
			}
		}
	}

	// lookup domainID required for record transactions
	domainID := p.getDomainID(domain)
	if domainID == "" {
		return fmt.Errorf("record lookup cannot proceed: unknown domain: %s", domain)
	}

	// live API call to retrieve records for specified domain
	RecordIDs = make(map[string]string) // reset cache

	// configure http request
	req, err := http.NewRequest(http.MethodGet, APIbase+"/"+domainID+"/AllRecords", nil)
	if err != nil {
		return err
	}

	var records TotalUptimeRecords
	err = p.performAPIcall(req, &records)
	if err != nil {
		return err
	}

	// populate RecordsID cache
	for _, val := range records.ARecord.Rows {
		RecordIDs[domain+"/A/"+val.AHostName] = val.ID
	}

	// CNAME records
	for _, val := range records.CNAMERecord.Rows {
		RecordIDs[domain+"/CNAME/"+val.CnameName] = val.ID
	}

	// MX records
	// TODO: decide how to handle multiple records same domain name
	for _, val := range records.MXRecord.Rows {
		RecordIDs[domain+"/MX/"+val.MxDomainName] = val.ID
	}

	// NS records
	// TODO: decide how to handle multiple records same hostname
	for _, val := range records.NSRecord.Rows {
		RecordIDs[domain+"/NS/"+val.NsHostName] = val.ID
	}

	// TXT records
	// TODO: decide how to handle multiple records same hostname
	for _, val := range records.TXTRecord.Rows {
		RecordIDs[domain+"/TXT/"+val.TxtHostName] = val.ID
	}

	RecordIDs[domain] = "populated" // flag item indicates cache populated for this domain
	return nil
}

// getDomainID returns the cached ID of the specified domain/recType/recName combination.
func (p *Provider) getRecordID(domain, recType, recName string) string {
	err := p.lookupRecordIDs(domain)
	if err != nil {
		log.Println(err)
		return "" // error accessing cache
	}

	cacheKey := domain + "/" + recType + "/" + recName
	if _, ok := RecordIDs[cacheKey]; ok {
		return RecordIDs[cacheKey] // cross-reference
	}

	log.Printf("Record not found: %s\n", cacheKey)
	return "" // record not found
}

// convertRecordTypeToProvider converts a libdns record type to provider code.
func convertRecordTypeToProvider(typeName string) (string, error) {
	switch typeName {
	case "A":
		return "ARecord", nil
	case "CNAME":
		return "CNAMERecord", nil
	case "MX":
		return "MXRecord", nil
	case "NS":
		return "NSRecord", nil
	case "TXT":
		return "TXTRecord", nil
	default:
		return "ARecord", fmt.Errorf("type %s cannot be interpreted", typeName)
	}
}

// providerPayload type stores the minimum required fields to interact with provider records.
type providerPayload struct {
	AHostName     string `json:"aHostName"`
	AIPAddress    string `json:"aIPAddress"`
	ATTL          string `json:"aTTL"`
	CnameName     string `json:"cnameName"`
	CnameAliasFor string `json:"cnameAliasFor"`
	CnameTTL      string `json:"cnameTTL"`
	MxDomainName  string `json:"mxDomainName"`
	MxPreference  string `json:"mxPreference"`
	MxMailServer  string `json:"mxMailServer"`
	MxTTL         string `json:"mxTTL"`
	NsHostName    string `json:"nsHostName"`
	NsName        string `json:"nsName"`
	NsTTL         string `json:"nsTTL"`
	TxtHostName   string `json:"txtHostName"`
	TxtText       string `json:"txtText"`
	TxtTTL        string `json:"txtTTL"`
}

// providerResponse type stores the response data from provider API calls.
type providerResponse struct {
	StatusCode interface{} `json:"StatusCode"`
	Type       interface{} `json:"Type"`
	Message    string      `json:"message"` // success or error details
	Status     string      `json:"status"`  // returns "Success"
	Data       interface{} `json:"data"`
	ID         string      `json:"id"`
	Options    interface{} `json:"options"`
}

// buildPayload configures the API query payload with provider requirements per record type.
func buildPayload(rec libdns.Record) providerPayload {
	var payload providerPayload

	switch rec.Type {
	case "A":
		payload.AHostName = rec.Name
		payload.AIPAddress = rec.Value
		payload.ATTL = fmt.Sprint(int64(rec.TTL / time.Second))

	case "CNAME":
		payload.CnameName = rec.Name
		payload.CnameAliasFor = rec.Value
		payload.CnameTTL = fmt.Sprint(int64(rec.TTL / time.Second))

	case "MX":
		payload.MxDomainName = rec.Name
		payload.MxMailServer = rec.Value
		payload.MxTTL = fmt.Sprint(int64(rec.TTL / time.Second))
		payload.MxPreference = fmt.Sprint(rec.Priority) // TODO: verify provider is honoring this

	case "NS":
		payload.NsHostName = rec.Name
		payload.NsName = rec.Value
		payload.NsTTL = fmt.Sprint(int64(rec.TTL / time.Second))

	case "TXT":
		payload.TxtHostName = rec.Name
		payload.TxtText = rec.Value
		payload.TxtTTL = fmt.Sprint(int64(rec.TTL / time.Second))

	default:
		// do not populate payload if record type is invalid
	}

	return payload
}
