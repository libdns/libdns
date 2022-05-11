package nicrudns

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

func (client *Client) DownloadZone(name string) (string, error) {
	url := fmt.Sprintf(DownloadZoneUrlPattern, client.config.DnsServiceName, name)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ``, errors.Wrap(err, RequestError.Error())
	}
	response, err := client.Do(req)
	if err != nil {
		return ``, errors.Wrap(err, ResponseError.Error())
	}
	if response.StatusCode != http.StatusOK {
		return ``, errors.Wrap(InvalidStatusCode, strconv.Itoa(response.StatusCode))
	}
	buf := bytes.NewBuffer(nil)
	_, err = buf.ReadFrom(response.Body)
	if err != nil {
		return ``, errors.Wrap(err, BufferReadError.Error())
	}
	apiResponse := &Response{}
	if err := xml.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(&apiResponse); err == nil {
		// if response structure is valid, this is unexpected result
		if apiResponse.Errors.Error.Text != `` {
			return ``, errors.Wrap(ApiNonSuccessError, describeError(apiResponse.Errors.Error))
		} else {
			return ``, errors.Wrap(ResponseError, `not a dns zone format`)
		}
	} else {
		// else OK
		return buf.String(), nil
	}
}
