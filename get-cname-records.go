package nicrudns

import (
	"github.com/pkg/errors"
	"regexp"
)

func (client *Client) GetCnameRecords(nameFilter string, targetFilter string) ([]*RR, error) {
	allRecords, err := client.GetRecords()
	if err != nil {
		return nil, err
	}

	nameFilterRegexp, err := regexp.Compile(nameFilter)
	if err != nil {
		return nil, errors.Wrap(err, NameFilterError.Error())
	}
	targetFilterRegexp, err := regexp.Compile(targetFilter)
	if err != nil {
		return nil, errors.Wrap(err, TargetFilterError.Error())
	}

	var records []*RR
	for _, record := range allRecords {
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
	return records, nil
}
