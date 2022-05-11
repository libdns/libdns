package nicrudns

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
)

func (client *Client) Add(request *Request) (*Response, error) {

	buf := bytes.NewBuffer(nil)
	if err := xml.NewEncoder(buf).Encode(request); err != nil {
		return nil, errors.Wrap(err, XmlEncodeError.Error())
	}

	url := fmt.Sprintf(AddRecordsUrlPattern, client.config.DnsServiceName, client.config.ZoneName)

	req, err := http.NewRequest(http.MethodPut, url, buf)
	if err != nil {
		return nil, errors.Wrap(err, RequestError.Error())
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, ResponseError.Error())
	}

	//if response.StatusCode != http.StatusOK {
	//	return nil, errors.Wrap(err, InvalidStatusCode.Error())
	//}

	buf = bytes.NewBuffer(nil)
	if _, err := buf.ReadFrom(response.Body); err != nil {
		return nil, errors.Wrap(err, BufferReadError.Error())
	}

	apiResponse := &Response{}
	if err := xml.NewDecoder(buf).Decode(&apiResponse); err != nil {
		return nil, errors.Wrap(err, XmlDecodeError.Error())
	}

	if apiResponse.Status != SuccessStatus {
		return nil, errors.Wrap(ApiNonSuccessError, describeError(apiResponse.Errors.Error))
	} else {
		return apiResponse, nil
	}
}
