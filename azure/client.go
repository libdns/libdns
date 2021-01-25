package azure

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/libdns/libdns"
)

// NewClient invokes authentication and store client to the provider instance.
func (p *Provider) NewClient() error {
	recordSetsClient := dns.NewRecordSetsClient(p.SubscriptionId)

	clientCredentialsConfig := auth.NewClientCredentialsConfig(p.ClientId, p.ClientSecret, p.TenantId)
	authorizer, err := clientCredentialsConfig.Authorizer()
	if err != nil {
		return err
	}

	recordSetsClient.Authorizer = authorizer
	p.client = &recordSetsClient

	return nil
}

// getRecords gets all records in specified zone on Azure DNS.
func (p *Provider) getRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	var recordSets []*dns.RecordSet

	pages, err := p.client.ListAllByDNSZone(
		ctx,
		p.ResourceGroupName,
		strings.TrimSuffix(zone, "."),
		to.Int32Ptr(1000),
		"")
	if err != nil {
		return nil, err
	}

	for pages.NotDone() {
		recordSetsList := pages.Response()
		for _, v := range *recordSetsList.Value {
			recordSet := v
			recordSets = append(recordSets, &recordSet)
		}
		pages.NextWithContext(ctx)
	}

	records, _ := convertAzureRecordSetsToLibdnsRecords(recordSets)
	return records, nil
}

// createRecord creates a new record in the specified zone.
// It throws an error if the record already exists.
func (p *Provider) createRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	return p.createOrUpdateRecord(ctx, zone, record, "*")
}

// updateRecord creates or updates a record, either by updating existing record or creating new one.
func (p *Provider) updateRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	return p.createOrUpdateRecord(ctx, zone, record, "")
}

// deleteRecord deletes an existing records.
// Regardless of the value of the record, if the name and type match, the record will be deleted.
func (p *Provider) deleteRecord(ctx context.Context, zone string, record libdns.Record) (libdns.Record, error) {
	recordType, err := convertStringToRecordType(record.Type)
	if err != nil {
		return record, err
	}

	_, err = p.client.Delete(
		ctx,
		p.ResourceGroupName,
		strings.TrimSuffix(zone, "."),
		generateRecordSetName(record.Name, zone),
		recordType,
		"",
	)
	if err != nil {
		return record, err
	}

	return record, nil
}

// createOrUpdateRecord creates or updates a record.
// The behavior depends on the value of ifNoneMatch, set to "*" to allow to create a new record but prevent updating an existing record.
func (p *Provider) createOrUpdateRecord(ctx context.Context, zone string, record libdns.Record, ifNoneMatch string) (libdns.Record, error) {
	recordType, err := convertStringToRecordType(record.Type)
	if err != nil {
		return record, err
	}

	recordSet, err := convertLibdnsRecordToAzureRecordSet(record)
	if err != nil {
		return record, err
	}

	_, err = p.client.CreateOrUpdate(
		ctx,
		p.ResourceGroupName,
		strings.TrimSuffix(zone, "."),
		generateRecordSetName(record.Name, zone),
		recordType,
		recordSet,
		"",
		ifNoneMatch,
	)
	if err != nil {
		return record, err
	}

	return record, nil
}

// generateRecordSetName generates name for RecordSet object.
func generateRecordSetName(fqdn string, zone string) string {
	recordSetName := fqdn
	recordSetName = recordSetName[:len(recordSetName)-len(zone)]
	recordSetName = strings.TrimSuffix(recordSetName, ".")
	if recordSetName == "" {
		return "@"
	}
	return recordSetName
}

// convertStringToRecordType casts standard type name string to an Azure-styled dedicated type.
func convertStringToRecordType(typeName string) (dns.RecordType, error) {
	switch typeName {
	case "A":
		return dns.A, nil
	case "AAAA":
		return dns.AAAA, nil
	case "CAA":
		return dns.CAA, nil
	case "CNAME":
		return dns.CNAME, nil
	case "MX":
		return dns.MX, nil
	case "NS":
		return dns.NS, nil
	case "PTR":
		return dns.PTR, nil
	case "SOA":
		return dns.SOA, nil
	case "SRV":
		return dns.SRV, nil
	case "TXT":
		return dns.TXT, nil
	default:
		return dns.A, fmt.Errorf("The type %v cannot be interpreted.", typeName)
	}
}

// convertAzureRecordSetsToLibdnsRecords converts Azure-styled records to libdns records.
func convertAzureRecordSetsToLibdnsRecords(recordSets []*dns.RecordSet) ([]libdns.Record, error) {
	var records []libdns.Record

	for _, recordSet := range recordSets {
		switch typeName := strings.TrimPrefix(*recordSet.Type, "Microsoft.Network/dnszones/"); typeName {
		case "A":
			for _, v := range *recordSet.ARecords {
				record := libdns.Record{
					ID:    *recordSet.Etag,
					Type:  typeName,
					Name:  *recordSet.Fqdn,
					Value: *v.Ipv4Address,
					TTL:   time.Duration(*recordSet.TTL) * time.Second,
				}
				records = append(records, record)
			}
		case "AAAA":
			for _, v := range *recordSet.AaaaRecords {
				record := libdns.Record{
					ID:    *recordSet.Etag,
					Type:  typeName,
					Name:  *recordSet.Fqdn,
					Value: *v.Ipv6Address,
					TTL:   time.Duration(*recordSet.TTL) * time.Second,
				}
				records = append(records, record)
			}
		case "CAA":
			for _, v := range *recordSet.CaaRecords {
				record := libdns.Record{
					ID:    *recordSet.Etag,
					Type:  typeName,
					Name:  *recordSet.Fqdn,
					Value: strings.Join([]string{fmt.Sprint(*v.Flags), *v.Tag, *v.Value}, " "),
					TTL:   time.Duration(*recordSet.TTL) * time.Second,
				}
				records = append(records, record)
			}
		case "CNAME":
			record := libdns.Record{
				ID:    *recordSet.Etag,
				Type:  typeName,
				Name:  *recordSet.Fqdn,
				Value: *recordSet.CnameRecord.Cname,
				TTL:   time.Duration(*recordSet.TTL) * time.Second,
			}
			records = append(records, record)
		case "MX":
			for _, v := range *recordSet.MxRecords {
				record := libdns.Record{
					ID:    *recordSet.Etag,
					Type:  typeName,
					Name:  *recordSet.Fqdn,
					Value: strings.Join([]string{fmt.Sprint(*v.Preference), *v.Exchange}, " "),
					TTL:   time.Duration(*recordSet.TTL) * time.Second,
				}
				records = append(records, record)
			}
		case "NS":
			for _, v := range *recordSet.NsRecords {
				record := libdns.Record{
					ID:    *recordSet.Etag,
					Type:  typeName,
					Name:  *recordSet.Fqdn,
					Value: *v.Nsdname,
					TTL:   time.Duration(*recordSet.TTL) * time.Second,
				}
				records = append(records, record)
			}
		case "PTR":
			for _, v := range *recordSet.PtrRecords {
				record := libdns.Record{
					ID:    *recordSet.Etag,
					Type:  typeName,
					Name:  *recordSet.Fqdn,
					Value: *v.Ptrdname,
					TTL:   time.Duration(*recordSet.TTL) * time.Second,
				}
				records = append(records, record)
			}
		case "SOA":
			record := libdns.Record{
				ID:    *recordSet.Etag,
				Type:  typeName,
				Name:  *recordSet.Fqdn,
				Value: strings.Join([]string{*recordSet.SoaRecord.Host, *recordSet.SoaRecord.Email, fmt.Sprint(*recordSet.SoaRecord.SerialNumber), fmt.Sprint(*recordSet.SoaRecord.RefreshTime), fmt.Sprint(*recordSet.SoaRecord.RetryTime), fmt.Sprint(*recordSet.SoaRecord.ExpireTime), fmt.Sprint(*recordSet.SoaRecord.MinimumTTL)}, " "),
				TTL:   time.Duration(*recordSet.TTL) * time.Second,
			}
			records = append(records, record)
		case "SRV":
			for _, v := range *recordSet.SrvRecords {
				record := libdns.Record{
					ID:    *recordSet.Etag,
					Type:  typeName,
					Name:  *recordSet.Fqdn,
					Value: strings.Join([]string{fmt.Sprint(*v.Priority), fmt.Sprint(*v.Weight), fmt.Sprint(*v.Port), *v.Target}, " "),
					TTL:   time.Duration(*recordSet.TTL) * time.Second,
				}
				records = append(records, record)
			}
		case "TXT":
			for _, v := range *recordSet.TxtRecords {
				for _, txt := range *v.Value {
					record := libdns.Record{
						ID:    *recordSet.Etag,
						Type:  typeName,
						Name:  *recordSet.Fqdn,
						Value: txt,
						TTL:   time.Duration(*recordSet.TTL) * time.Second,
					}
					records = append(records, record)
				}
			}
		default:
			return []libdns.Record{}, fmt.Errorf("The type %v cannot be interpreted.", typeName)
		}
	}

	return records, nil
}

// convertLibdnsRecordToAzureRecordSet converts a libdns record to an Azure-styled record.
func convertLibdnsRecordToAzureRecordSet(record libdns.Record) (dns.RecordSet, error) {
	switch record.Type {
	case "A":
		recordSet := dns.RecordSet{
			RecordSetProperties: &dns.RecordSetProperties{
				TTL: to.Int64Ptr(int64(record.TTL / time.Second)),
				ARecords: &[]dns.ARecord{
					dns.ARecord{
						Ipv4Address: to.StringPtr(record.Value),
					},
				},
			},
		}
		return recordSet, nil
	case "AAAA":
		recordSet := dns.RecordSet{
			RecordSetProperties: &dns.RecordSetProperties{
				TTL: to.Int64Ptr(int64(record.TTL / time.Second)),
				AaaaRecords: &[]dns.AaaaRecord{
					dns.AaaaRecord{
						Ipv6Address: to.StringPtr(record.Value),
					},
				},
			},
		}
		return recordSet, nil
	case "CAA":
		values := strings.Split(record.Value, " ")
		flags, _ := strconv.ParseInt(values[0], 10, 32)
		recordSet := dns.RecordSet{
			RecordSetProperties: &dns.RecordSetProperties{
				TTL: to.Int64Ptr(int64(record.TTL / time.Second)),
				CaaRecords: &[]dns.CaaRecord{
					dns.CaaRecord{
						Flags: to.Int32Ptr(int32(flags)),
						Tag:   to.StringPtr(values[1]),
						Value: to.StringPtr(values[2]),
					},
				},
			},
		}
		return recordSet, nil
	case "CNAME":
		recordSet := dns.RecordSet{
			RecordSetProperties: &dns.RecordSetProperties{
				TTL: to.Int64Ptr(int64(record.TTL / time.Second)),
				CnameRecord: &dns.CnameRecord{
					Cname: to.StringPtr(record.Value),
				},
			},
		}
		return recordSet, nil
	case "MX":
		values := strings.Split(record.Value, " ")
		preference, _ := strconv.ParseInt(values[0], 10, 32)
		recordSet := dns.RecordSet{
			RecordSetProperties: &dns.RecordSetProperties{
				TTL: to.Int64Ptr(int64(record.TTL / time.Second)),
				MxRecords: &[]dns.MxRecord{
					dns.MxRecord{
						Preference: to.Int32Ptr(int32(preference)),
						Exchange:   to.StringPtr(values[1]),
					},
				},
			},
		}
		return recordSet, nil
	case "NS":
		recordSet := dns.RecordSet{
			RecordSetProperties: &dns.RecordSetProperties{
				TTL: to.Int64Ptr(int64(record.TTL / time.Second)),
				NsRecords: &[]dns.NsRecord{
					dns.NsRecord{
						Nsdname: to.StringPtr(record.Value),
					},
				},
			},
		}
		return recordSet, nil
	case "PTR":
		recordSet := dns.RecordSet{
			RecordSetProperties: &dns.RecordSetProperties{
				TTL: to.Int64Ptr(int64(record.TTL / time.Second)),
				PtrRecords: &[]dns.PtrRecord{
					dns.PtrRecord{
						Ptrdname: to.StringPtr(record.Value),
					},
				},
			},
		}
		return recordSet, nil
	case "SOA":
		values := strings.Split(record.Value, " ")
		serialNumber, _ := strconv.ParseInt(values[2], 10, 64)
		refreshTime, _ := strconv.ParseInt(values[3], 10, 64)
		retryTime, _ := strconv.ParseInt(values[4], 10, 64)
		expireTime, _ := strconv.ParseInt(values[5], 10, 64)
		minimumTTL, _ := strconv.ParseInt(values[6], 10, 64)
		recordSet := dns.RecordSet{
			RecordSetProperties: &dns.RecordSetProperties{
				TTL: to.Int64Ptr(int64(record.TTL / time.Second)),
				SoaRecord: &dns.SoaRecord{
					Host:         to.StringPtr(values[0]),
					Email:        to.StringPtr(values[1]),
					SerialNumber: to.Int64Ptr(serialNumber),
					RefreshTime:  to.Int64Ptr(refreshTime),
					RetryTime:    to.Int64Ptr(retryTime),
					ExpireTime:   to.Int64Ptr(expireTime),
					MinimumTTL:   to.Int64Ptr(minimumTTL),
				},
			},
		}
		return recordSet, nil
	case "SRV":
		values := strings.Split(record.Value, " ")
		priority, _ := strconv.ParseInt(values[0], 10, 32)
		weight, _ := strconv.ParseInt(values[1], 10, 32)
		port, _ := strconv.ParseInt(values[2], 10, 32)
		recordSet := dns.RecordSet{
			RecordSetProperties: &dns.RecordSetProperties{
				TTL: to.Int64Ptr(int64(record.TTL / time.Second)),
				SrvRecords: &[]dns.SrvRecord{
					dns.SrvRecord{
						Priority: to.Int32Ptr(int32(priority)),
						Weight:   to.Int32Ptr(int32(weight)),
						Port:     to.Int32Ptr(int32(port)),
						Target:   to.StringPtr(values[3]),
					},
				},
			},
		}
		return recordSet, nil
	case "TXT":
		recordSet := dns.RecordSet{
			RecordSetProperties: &dns.RecordSetProperties{
				TTL: to.Int64Ptr(int64(record.TTL / time.Second)),
				TxtRecords: &[]dns.TxtRecord{
					dns.TxtRecord{
						Value: &[]string{record.Value},
					},
				},
			},
		}
		return recordSet, nil
	default:
		return dns.RecordSet{}, fmt.Errorf("The type %v cannot be interpreted.", record.Type)
	}
}
