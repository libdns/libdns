package nicrudns

import (
	"bytes"
	"encoding/xml"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

func (client *Client) GetServices() ([]*Service, error) {
	request, err := http.NewRequest(http.MethodGet, GetServicesUrl, nil)
	if err != nil {
		return nil, errors.Wrap(err, RequestError.Error())
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, ResponseError.Error())
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.Wrap(InvalidStatusCode, strconv.Itoa(response.StatusCode))
	}

	buf := bytes.NewBuffer(nil)
	if _, err := buf.ReadFrom(response.Body); err != nil {
		return nil, errors.Wrap(err, BufferReadError.Error())
	}

	apiResponse := &Response{}
	if err := xml.NewDecoder(buf).Decode(&apiResponse); err != nil {
		return nil, errors.Wrap(err, XmlDecodeError.Error())
	}
	if apiResponse.Status != SuccessStatus {
		return nil, errors.Wrap(ApiNonSuccessError, describeError(apiResponse.Errors.Error))
	}

	return apiResponse.Data.Service, nil
}
