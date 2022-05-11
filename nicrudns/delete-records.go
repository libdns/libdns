package nicrudns

import (
	"encoding/xml"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

func (client *Client) DeleteRecord(id int) (*Response, error) {
	url := fmt.Sprintf(DeleteRecordsUrlPattern, client.config.DnsServiceName, client.config.ZoneName, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, RequestError.Error())
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, ResponseError.Error())
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.Wrap(InvalidStatusCode, strconv.Itoa(response.StatusCode))
	}
	apiResponse := Response{}
	if err := xml.NewDecoder(response.Body).Decode(&apiResponse); err != nil {
		return nil, errors.Wrap(err, XmlDecodeError.Error())
	}
	if apiResponse.Status != SuccessStatus {
		return nil, errors.Wrap(ApiNonSuccessError, apiResponse.Status)
	} else {
		return &apiResponse, nil
	}
}
