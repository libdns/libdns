package nicrudns

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
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

	if response.StatusCode != http.StatusOK {
		return nil, errors.Wrap(err, strconv.Itoa(response.StatusCode))
	} else {
		buf = bytes.NewBuffer(nil)
		if _, err := buf.ReadFrom(response.Body); err != nil {
			return nil, errors.Wrap(err, BufferReadError.Error())
		}
		s := buf.String()

		buf = bytes.NewBuffer(nil)
		buf.WriteString(s)

		response := &Response{}
		if err := xml.NewDecoder(buf).Decode(&response); err != nil {
			return nil, errors.Wrap(err, XmlDecodeError.Error())
		}

		if response.Status != SuccessStatus {
			return nil, errors.Wrap(ApiNonSuccessError, describeError(response.Errors.Error))
		} else {
			return response, nil
		}
	}
}
