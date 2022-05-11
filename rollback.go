package nicrudns

import (
	"encoding/xml"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
)

func (client *Client) RollbackZone() (*Response, error) {
	url := fmt.Sprintf(RollbackUrlPattern, client.config.DnsServiceName, client.config.ZoneName)
	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, RequestError.Error())
	}
	response, err := client.Do(request)
	//if response.StatusCode != http.StatusOK {
	//	return nil, errors.Wrap(err, InvalidStatusCode.Error())
	//}
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
