package inwx

import (
	"fmt"

	"github.com/kolo/xmlrpc"
	"github.com/mitchellh/mapstructure"
)

type Client struct {
	rpcClient *xmlrpc.Client
}

type Response struct {
	Code         int    `xmlrpc:"code"`
	Message      string `xmlrpc:"msg"`
	ReasonCode   string `xmlrpc:"reasonCode"`
	Reason       string `xmlrpc:"reason"`
	ResponseData any    `xmlrpc:"resData"`
}

type ErrorResponse struct {
	Code       int    `xmlrpc:"code"`
	Message    string `xmlrpc:"msg"`
	ReasonCode string `xmlrpc:"reasonCode"`
	Reason     string `xmlrpc:"reason"`
}

type NameserverInfoRequest struct {
	Domain   string `xmlrpc:"domain,omitempty"`
	Name     string `xmlrpc:"name,omitempty"`
	Type     string `xmlrpc:"type,omitempty"`
	Content  string `xmlrpc:"content,omitempty"`
	TTL      int    `xmlrpc:"ttl,omitempty"`
	Priority int    `xmlrpc:"prio,omitempty"`
}

type NameserverInfoResponse struct {
	RoID    int                `mapstructure:"roId"`
	Domain  string             `mapstructure:"domain"`
	Type    string             `mapstructure:"type"`
	Count   int                `mapstructure:"count"`
	Records []NameserverRecord `mapstructure:"record"`
}

type NameserverCreateRecordRequest struct {
	Domain   string `xmlrpc:"domain"`
	Name     string `xmlrpc:"name"`
	Type     string `xmlrpc:"type"`
	Content  string `xmlrpc:"content"`
	TTL      int    `xmlrpc:"ttl"`
	Priority int    `xmlrpc:"prio"`
}

type NameserverCreateRecordResponse struct {
	ID int `mapstructure:"id"`
}

type NameserverUpdateRecordRequest struct {
	ID       int    `xmlrpc:"id"`
	Name     string `xmlrpc:"name"`
	Type     string `xmlrpc:"type"`
	Content  string `xmlrpc:"content"`
	TTL      int    `xmlrpc:"ttl"`
	Priority int    `xmlrpc:"prio"`
}

type NameserverDeleteRecordRequest struct {
	ID int `mapstructure:"id"`
}

type NameserverRecord struct {
	ID       int    `mapstructure:"id"`
	Name     string `mapstructure:"name"`
	Type     string `mapstructure:"type"`
	Content  string `mapstructure:"content"`
	TTL      int    `mapstructure:"ttl"`
	Priority int    `mapstructure:"prio"`
}

type AccountLoginRequest struct {
	User string `xmlrpc:"user"`
	Pass string `xmlrpc:"pass"`
}

type AccountLoginResponse struct {
	TFA string `mapstructure:"tfa"`
}

type AccountUnlockRequest struct {
	TAN string `xmlrpc:"tan"`
}

const (
	endpointURL     = "https://api.domrobot.com/xmlrpc/"
	testEndpointURL = "https://api.ote.domrobot.com/xmlrpc/"
)

func newClient(endpointURL string) (*Client, error) {
	rpcClient, err := xmlrpc.NewClient(endpointURL, nil)

	if err != nil {
		return nil, err
	}

	return &Client{rpcClient}, nil
}

func (c *Client) GetRecords(domain string) ([]NameserverRecord, error) {
	response, err := c.call("nameserver.info", NameserverInfoRequest{
		Domain: domain,
	})

	if err != nil {
		return []NameserverRecord{}, err
	}

	data := NameserverInfoResponse{}
	err = mapstructure.Decode(response, &data)

	if err != nil {
		return []NameserverRecord{}, err
	}

	return data.Records, nil
}

func (c *Client) FindRecords(record NameserverRecord, domain string, matchContent bool) ([]NameserverRecord, error) {
	request := NameserverInfoRequest{
		Domain: domain,
		Type:   record.Type,
		Name:   record.Name,
	}

	if matchContent {
		request.Content = record.Content
	}

	response, err := c.call("nameserver.info", request)

	if err != nil {
		return []NameserverRecord{}, err
	}

	data := NameserverInfoResponse{}
	err = mapstructure.Decode(response, &data)

	if err != nil {
		return []NameserverRecord{}, err
	}

	return data.Records, nil
}

func (c *Client) CreateRecord(record NameserverRecord, domain string) (int, error) {
	response, err := c.call("nameserver.createRecord", NameserverCreateRecordRequest{
		Domain:   domain,
		Name:     record.Name,
		Type:     record.Type,
		Content:  record.Content,
		TTL:      record.TTL,
		Priority: record.Priority,
	})

	if err != nil {
		return 0, err
	}

	data := NameserverCreateRecordResponse{}
	mapstructure.Decode(response, &data)

	return data.ID, nil
}

func (c *Client) UpdateRecord(record NameserverRecord) error {
	if record.ID == 0 {
		return fmt.Errorf("Record cannot be updated because the ID is not set.")
	}

	_, err := c.call("nameserver.updateRecord", NameserverUpdateRecordRequest{
		ID:       record.ID,
		Name:     record.Name,
		Type:     record.Type,
		Content:  record.Content,
		TTL:      record.TTL,
		Priority: record.Priority,
	})

	return err
}

func (c *Client) DeleteRecord(record NameserverRecord) error {
	_, err := c.call("nameserver.deleteRecord", NameserverDeleteRecordRequest{
		ID: record.ID,
	})

	return err
}

func (c *Client) Login(username string, password string, sharedSecret string) (bool, error) {
	response, err := c.call("account.login", AccountLoginRequest{
		User: username,
		Pass: password,
	})

	if err != nil {
		return false, err
	}

	data := AccountLoginResponse{}
	mapstructure.Decode(response, &data)

	return data.TFA == "GOOGLE-AUTH", err
}

func (c *Client) Logout() error {
	_, err := c.call("account.logout", nil)

	return err
}

func (c *Client) Unlock(tan string) error {
	_, err := c.call("account.unlock", AccountUnlockRequest{
		TAN: tan,
	})

	return err
}

func (c *Client) call(method string, params any) (any, error) {
	var response Response

	err := c.rpcClient.Call(method, params, &response)

	if err != nil {
		return nil, err
	}

	return response.ResponseData, checkResponse(response)
}

func (r *ErrorResponse) Error() string {
	if r.Reason != "" {
		return fmt.Sprintf("(%d) %s. Reason: (%s) %s",
			r.Code, r.Message, r.ReasonCode, r.Reason)
	}

	return fmt.Sprintf("(%d) %s", r.Code, r.Message)
}

func checkResponse(r Response) error {
	if c := r.Code; c >= 1000 && c <= 1500 {
		return nil
	}

	return &ErrorResponse{
		Code:       r.Code,
		Message:    r.Message,
		Reason:     r.Reason,
		ReasonCode: r.ReasonCode,
	}
}
