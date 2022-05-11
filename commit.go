package nicrudns

import (
	"encoding/xml"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
)

func (client *Client) CommitZone() (*Response, error) {
	url := fmt.Sprintf(CommitUrlPattern, client.config.DnsServiceName, client.config.ZoneName)
	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, RequestError.Error())
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, ResponseError.Error())
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.Wrap(err, InvalidStatusCode.Error())
	}
	apiResponse := Response{}
	if err := xml.NewDecoder(response.Body).Decode(&apiResponse); err != nil {
		return nil, errors.Wrap(err, XmlDecodeError.Error())
	}
	if apiResponse.Status != SuccessStatus {
		return nil, errors.Wrap(ApiNonSuccessError, describeError(apiResponse.Errors.Error))
	} else {
		return &apiResponse, nil
	}
}
