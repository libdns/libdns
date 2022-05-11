package nicrudns

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
)

func (client *Client) DownloadZone() (string, error) {
	url := fmt.Sprintf(DownloadZoneUrlPattern, client.config.DnsServiceName, client.config.ZoneName)
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
	return buf.String(), nil
}
