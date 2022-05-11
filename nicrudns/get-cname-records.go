package nicrudns

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"regexp"
	"strconv"
)

func (client *Client) GetCnameRecords(nameFilter string, targetFilter string) ([]*RR, error) {
	nameFilterRegexp, err := regexp.Compile(nameFilter)
	if err != nil {
		return nil, errors.Wrap(err, NameFilterError.Error())
	}
	targetFilterRegexp, err := regexp.Compile(targetFilter)
	if err != nil {
		return nil, errors.Wrap(err, TargetFilterError.Error())
	}

	url := fmt.Sprintf(GetRecordsUrlPattern, client.config.DnsServiceName, client.config.ZoneName)
	request, err := http.NewRequest(http.MethodGet, url, nil)
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
		return nil, errors.Wrap(ApiNonSuccessError, apiResponse.Status)
	}
	var records []*RR
	for _, zone := range apiResponse.Data.Zone {
		if zone.Name != client.config.ZoneName {
			continue
		}
		for _, record := range zone.Rr {
			if nameFilter != `` && !nameFilterRegexp.MatchString(record.Name) {
				continue
			}
			if record.Cname == nil {
				continue
			}
			if targetFilter != `` && !targetFilterRegexp.MatchString(record.Cname.Name) {
				continue
			}
			records = append(records, record)
		}
	}

	return records, nil
}
