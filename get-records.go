package nicrudns

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

func (client *Client) GetRecords() ([]*RR, error) {
	url := fmt.Sprintf(GetRecordsUrlPattern, client.config.DnsServiceName, client.config.ZoneName)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, RequestError.Error())
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, ResponseError.Error())
	}
	buf := bytes.NewBuffer(nil)
	if _, err := buf.ReadFrom(response.Body); err != nil {
		return nil, errors.Wrap(err, BufferReadError.Error())
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.Wrap(InvalidStatusCode, strconv.Itoa(response.StatusCode))
	}
	apiResponse := &Response{}
	if err := xml.NewDecoder(buf).Decode(&apiResponse); err != nil {
		return nil, errors.Wrap(err, XmlDecodeError.Error())
	}
	if apiResponse.Status != SuccessStatus {
		return nil, errors.Wrap(ApiNonSuccessError, describeError(apiResponse.Errors.Error))
	}
	var records []*RR
	for _, zone := range apiResponse.Data.Zone {
		if zone.Name != client.config.ZoneName {
			continue
		}
		records = append(records, zone.Rr...)
	}
	return records, nil
}
