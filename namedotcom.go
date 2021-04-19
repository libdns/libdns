package namedotcom

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/libdns/libdns"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"strconv"
	"time"
)

type (
	nameDotCom struct {
		Server string `json:"server,omitempty"`
		User   string `json:"user,omitempty"`
		Token  string `json:"token,omitempty"`
		Client *http.Client
	}

	// listRecordsResponse contains the response for the ListRecords function.
	listRecordsResponse struct {
		Records  []*nameDotComRecord `json:"records,omitempty"`
		NextPage int32               `json:"nextPage,omitempty"`
		LastPage int32               `json:"lastPage,omitempty"`
	}

	// nameDotComRecord is an individual DNS resource record for name.com.
	nameDotComRecord struct {
		ID         int32  `json:"id,omitempty"`
		DomainName string `json:"domainName,omitempty"`
		Host       string `json:"host,omitempty"`
		Fqdn       string `json:"fqdn,omitempty"`
		Type       string `json:"type,omitempty"`
		Answer     string `json:"answer,omitempty"`
		TTL        uint32 `json:"ttl,omitempty"`
		Priority   uint32 `json:"priority,omitempty"`
	}
)

type (
	// errorResponse is what is returned if the HTTP status code is not 200.
	errorResponse struct {
		// Message is the error message.
		Message string `json:"message,omitempty"`
		// Details may have some additional details about the error.
		Details string `json:"details,omitempty"`
	}
)

// Error errorResponse should implement the error interface
func (er errorResponse) Error() string {
	return er.Message + ": " + er.Details
}

// errorResponse  used to handle response errors
func (n *nameDotCom) errorResponse(resp *http.Response) error {
	er := &errorResponse{}
	err := json.NewDecoder(resp.Body).Decode(er)
	if err != nil {
		return errors.Wrap(err, "api returned unexpected response")
	}

	return errors.WithStack(er)
}

func (n *nameDotCom) doRequest(ctx context.Context, method, endpoint string, post io.Reader) (io.Reader, error) {
	uri := n.Server + endpoint
	req, err := http.NewRequestWithContext(ctx, method, uri, post)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(n.User, n.Token)

	resp, err := n.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, n.errorResponse(resp)
	}

	return resp.Body, nil
}

// NewNameDotComClient returns a new name.com client struct
func NewNameDotComClient(token, user, apiUrl string) *nameDotCom {
	return &nameDotCom{
		Server: apiUrl,
		User:   user,
		Token:  token,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

// fromLibDNSRecord maps a name.com record from a libdns record
func (n *nameDotComRecord) fromLibDNSRecord(record libdns.Record) {
	var id int64
	if record.ID != "" {
		id, _ = strconv.ParseInt(record.ID, 10, 32)
	}
	n.ID = int32(id)
	n.Type = record.Type
	n.Host = record.Name
	n.Answer = record.Value
	n.TTL = uint32(record.TTL.Seconds())
}

// toLibDNSRecord maps a name.com record to a libdns record
func (n *nameDotComRecord) toLibDNSRecord() libdns.Record {
	id := fmt.Sprint(n.ID)
	return libdns.Record{
		ID:    id,
		Type:  n.Type,
		Name:  n.Host,
		Value: n.Answer,
		TTL:   time.Duration(n.TTL) * time.Second,
	}
}
