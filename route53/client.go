package route53

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	r53 "github.com/aws/aws-sdk-go/service/route53"
	"github.com/libdns/libdns"
)

// NewSession initializes the AWS client
func (p *Provider) NewSession() error {
	if p.MaxRetries == 0 {
		p.MaxRetries = 5
	}
	sess, err := session.NewSession(&aws.Config{
		MaxRetries: aws.Int(p.MaxRetries),
	})
	if err != nil {
		return err
	}

	p.client = r53.New(sess)

	return nil
}

func (p *Provider) getRecords(ctx context.Context, zoneID string) ([]libdns.Record, error) {
	getRecordsInput := &r53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		MaxItems:     aws.String("1000"),
	}

	var records []libdns.Record
	var recordSets []*r53.ResourceRecordSet

	for {
		getRecordResult, err := p.client.ListResourceRecordSetsWithContext(ctx, getRecordsInput)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case r53.ErrCodeNoSuchHostedZone:
					return records, fmt.Errorf("%s: %s", r53.ErrCodeNoSuchHostedZone, aerr.Error())
				case r53.ErrCodeInvalidInput:
					return records, fmt.Errorf("%s: %s", r53.ErrCodeInvalidInput, aerr.Error())
				default:
					return records, fmt.Errorf(aerr.Error())
				}
			} else {
				return records, fmt.Errorf(err.Error())
			}
		}

		recordSets = append(recordSets, getRecordResult.ResourceRecordSets...)
		if *getRecordResult.IsTruncated {
			getRecordsInput.StartRecordName = getRecordResult.NextRecordName
			getRecordsInput.StartRecordType = getRecordResult.NextRecordType
			getRecordsInput.StartRecordIdentifier = getRecordResult.NextRecordIdentifier
		} else {
			break
		}
	}

	for _, rrset := range recordSets {
		for _, rrsetRecord := range rrset.ResourceRecords {
			record := libdns.Record{
				Name:  *rrset.Name,
				Value: *rrsetRecord.Value,
				Type:  *rrset.Type,
				TTL:   time.Duration(*rrset.TTL) * time.Second,
			}

			records = append(records, record)
		}
	}

	return records, nil
}

func (p *Provider) getZoneID(ctx context.Context, zoneName string) (string, error) {
	getZoneInput := &r53.ListHostedZonesByNameInput{
		DNSName:  aws.String(zoneName),
		MaxItems: aws.String("1"),
	}

	getZoneResult, err := p.client.ListHostedZonesByNameWithContext(ctx, getZoneInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case r53.ErrCodeInvalidDomainName:
				return "", fmt.Errorf("%s: %s", r53.ErrCodeInvalidDomainName, aerr.Error())
			case r53.ErrCodeInvalidInput:
				return "", fmt.Errorf("%s: %s", r53.ErrCodeInvalidInput, aerr.Error())
			default:
				return "", fmt.Errorf(aerr.Error())
			}
		} else {
			return "", fmt.Errorf(err.Error())
		}
	}

	if len(getZoneResult.HostedZones) > 0 {
		if *getZoneResult.HostedZones[0].Name == zoneName {
			return *getZoneResult.HostedZones[0].Id, nil
		}
	}

	return "", fmt.Errorf("%s: No zones found for the domain %s", r53.ErrCodeHostedZoneNotFound, zoneName)
}

func (p *Provider) createRecord(ctx context.Context, zoneID string, record libdns.Record) (libdns.Record, error) {
	// AWS Route53 TXT record value must be enclosed in quotation marks on create
	if record.Type == "TXT" {
		record.Value = strconv.Quote(record.Value)
	}

	createInput := &r53.ChangeResourceRecordSetsInput{
		ChangeBatch: &r53.ChangeBatch{
			Changes: []*r53.Change{
				{
					Action: aws.String("CREATE"),
					ResourceRecordSet: &r53.ResourceRecordSet{
						Name: aws.String(record.Name),
						ResourceRecords: []*r53.ResourceRecord{
							{
								Value: aws.String(record.Value),
							},
						},
						TTL:  aws.Int64(int64(record.TTL)),
						Type: aws.String(record.Type),
					},
				},
			},
		},
		HostedZoneId: aws.String(zoneID),
	}

	err := p.applyChange(ctx, createInput)
	if err != nil {
		return record, err
	}

	return record, nil
}

func (p *Provider) updateRecord(ctx context.Context, zoneID string, record libdns.Record) (libdns.Record, error) {
	// AWS Route53 TXT record value must be enclosed in quotation marks on update
	if record.Type == "TXT" {
		record.Value = strconv.Quote(record.Value)
	}

	updateInput := &r53.ChangeResourceRecordSetsInput{
		ChangeBatch: &r53.ChangeBatch{
			Changes: []*r53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &r53.ResourceRecordSet{
						Name: aws.String(record.Name),
						ResourceRecords: []*r53.ResourceRecord{
							{
								Value: aws.String(record.Value),
							},
						},
						TTL:  aws.Int64(int64(record.TTL)),
						Type: aws.String(record.Type),
					},
				},
			},
		},
		HostedZoneId: aws.String(zoneID),
	}

	err := p.applyChange(ctx, updateInput)
	if err != nil {
		return record, err
	}

	return record, nil
}

func (p *Provider) deleteRecord(ctx context.Context, zoneID string, record libdns.Record) (libdns.Record, error) {
	deleteInput := &r53.ChangeResourceRecordSetsInput{
		ChangeBatch: &r53.ChangeBatch{
			Changes: []*r53.Change{
				{
					Action: aws.String("DELETE"),
					ResourceRecordSet: &r53.ResourceRecordSet{
						Name: aws.String(record.Name),
						ResourceRecords: []*r53.ResourceRecord{
							{
								Value: aws.String(record.Value),
							},
						},
						TTL:  aws.Int64(int64(record.TTL)),
						Type: aws.String(record.Type),
					},
				},
			},
		},
		HostedZoneId: aws.String(zoneID),
	}

	err := p.applyChange(ctx, deleteInput)
	if err != nil {
		return record, err
	}

	return record, nil
}

func (p *Provider) applyChange(ctx context.Context, input *r53.ChangeResourceRecordSetsInput) error {
	changeResult, err := p.client.ChangeResourceRecordSetsWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case r53.ErrCodeNoSuchHostedZone:
				return fmt.Errorf("%s: %s", r53.ErrCodeNoSuchHostedZone, aerr.Error())
			case r53.ErrCodeInvalidChangeBatch:
				return fmt.Errorf("%s: %s", r53.ErrCodeInvalidChangeBatch, aerr.Error())
			case r53.ErrCodeInvalidInput:
				return fmt.Errorf("%s: %s", r53.ErrCodeInvalidInput, aerr.Error())
			case r53.ErrCodePriorRequestNotComplete:
				return fmt.Errorf("%s: %s", r53.ErrCodePriorRequestNotComplete, aerr.Error())
			default:
				return fmt.Errorf(aerr.Error())
			}
		} else {
			return fmt.Errorf(err.Error())
		}
	}

	changeInput := &r53.GetChangeInput{
		Id: changeResult.ChangeInfo.Id,
	}

	// Wait for the RecordSetChange status to be "INSYNC"
	err = p.client.WaitUntilResourceRecordSetsChangedWithContext(ctx, changeInput)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	return nil
}
