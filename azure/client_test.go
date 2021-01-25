package azure

import (
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/google/go-cmp/cmp"
	"github.com/libdns/libdns"
)

func Test_generateRecordSetName(t *testing.T) {
	t.Run("fqdn=test.example.com.", func(t *testing.T) {
		got := generateRecordSetName("test.example.com.", "example.com.")
		want := "test"
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("diff: %s", diff)
		}
	})
	t.Run("fqdn=example.com.", func(t *testing.T) {
		got := generateRecordSetName("example.com.", "example.com.")
		want := "@"
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("diff: %s", diff)
		}
	})
}

func Test_convertStringToRecordType(t *testing.T) {
	typeNames := []string{"A", "AAAA", "CAA", "CNAME", "MX", "NS", "PTR", "SOA", "SRV", "TXT"}
	for _, typeName := range typeNames {
		t.Run("type="+typeName, func(t *testing.T) {
			recordType, _ := convertStringToRecordType(typeName)
			got := fmt.Sprintf("%T:%v", recordType, recordType)
			want := "dns.RecordType:" + typeName
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("diff: %s", diff)
			}
		})
	}
	t.Run("type=ERR", func(t *testing.T) {
		_, err := convertStringToRecordType("ERR")
		got := err.Error()
		want := "The type ERR cannot be interpreted."
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("diff: %s", diff)
		}
	})
}

func Test_convertAzureRecordSetsToLibdnsRecords(t *testing.T) {
	t.Run("type=supported", func(t *testing.T) {
		azureRecordSets := []*dns.RecordSet{
			&dns.RecordSet{
				Name: to.StringPtr("record-a"),
				Type: to.StringPtr("Microsoft.Network/dnszones/A"),
				Etag: to.StringPtr("ETAG_A"),
				RecordSetProperties: &dns.RecordSetProperties{
					TTL:  to.Int64Ptr(30),
					Fqdn: to.StringPtr("test.example.com."),
					ARecords: &[]dns.ARecord{
						dns.ARecord{
							Ipv4Address: to.StringPtr("127.0.0.1"),
						},
					},
				},
			},
			&dns.RecordSet{
				Name: to.StringPtr("record-aaaa"),
				Type: to.StringPtr("Microsoft.Network/dnszones/AAAA"),
				Etag: to.StringPtr("ETAG_AAAA"),
				RecordSetProperties: &dns.RecordSetProperties{
					TTL:  to.Int64Ptr(30),
					Fqdn: to.StringPtr("record-aaaa.example.com."),
					AaaaRecords: &[]dns.AaaaRecord{
						dns.AaaaRecord{
							Ipv6Address: to.StringPtr("::1"),
						},
					},
				},
			},
			&dns.RecordSet{
				Name: to.StringPtr("record-caa"),
				Type: to.StringPtr("Microsoft.Network/dnszones/CAA"),
				Etag: to.StringPtr("ETAG_CAA"),
				RecordSetProperties: &dns.RecordSetProperties{
					TTL:  to.Int64Ptr(30),
					Fqdn: to.StringPtr("record-caa.example.com."),
					CaaRecords: &[]dns.CaaRecord{
						dns.CaaRecord{
							Flags: to.Int32Ptr(0),
							Tag:   to.StringPtr("issue"),
							Value: to.StringPtr("ca.example.com"),
						},
					},
				},
			},
			&dns.RecordSet{
				Name: to.StringPtr("record-cname"),
				Type: to.StringPtr("Microsoft.Network/dnszones/CNAME"),
				Etag: to.StringPtr("ETAG_CNAME"),
				RecordSetProperties: &dns.RecordSetProperties{
					TTL:  to.Int64Ptr(30),
					Fqdn: to.StringPtr("record-cname.example.com."),
					CnameRecord: &dns.CnameRecord{
						Cname: to.StringPtr("www.example.com"),
					},
				},
			},
			&dns.RecordSet{
				Name: to.StringPtr("record-mx"),
				Type: to.StringPtr("Microsoft.Network/dnszones/MX"),
				Etag: to.StringPtr("ETAG_MX"),
				RecordSetProperties: &dns.RecordSetProperties{
					TTL:  to.Int64Ptr(30),
					Fqdn: to.StringPtr("record-mx.example.com."),
					MxRecords: &[]dns.MxRecord{
						dns.MxRecord{
							Preference: to.Int32Ptr(10),
							Exchange:   to.StringPtr("mail.example.com"),
						},
					},
				},
			},
			&dns.RecordSet{
				Name: to.StringPtr("@"),
				Type: to.StringPtr("Microsoft.Network/dnszones/NS"),
				Etag: to.StringPtr("ETAG_NS"),
				RecordSetProperties: &dns.RecordSetProperties{
					TTL:  to.Int64Ptr(30),
					Fqdn: to.StringPtr("example.com."),
					NsRecords: &[]dns.NsRecord{
						dns.NsRecord{
							Nsdname: to.StringPtr("ns1.example.com"),
						},
						dns.NsRecord{
							Nsdname: to.StringPtr("ns2.example.com"),
						},
					},
				},
			},
			&dns.RecordSet{
				Name: to.StringPtr("record-ptr"),
				Type: to.StringPtr("Microsoft.Network/dnszones/PTR"),
				Etag: to.StringPtr("ETAG_PTR"),
				RecordSetProperties: &dns.RecordSetProperties{
					TTL:  to.Int64Ptr(30),
					Fqdn: to.StringPtr("record-ptr.example.com."),
					PtrRecords: &[]dns.PtrRecord{
						dns.PtrRecord{
							Ptrdname: to.StringPtr("hoge.example.com"),
						},
					},
				},
			},
			&dns.RecordSet{
				Name: to.StringPtr("@"),
				Type: to.StringPtr("Microsoft.Network/dnszones/SOA"),
				Etag: to.StringPtr("ETAG_SOA"),
				RecordSetProperties: &dns.RecordSetProperties{
					TTL:  to.Int64Ptr(30),
					Fqdn: to.StringPtr("example.com."),
					SoaRecord: &dns.SoaRecord{
						Host:         to.StringPtr("ns1.example.com"),
						Email:        to.StringPtr("hostmaster.example.com"),
						SerialNumber: to.Int64Ptr(1),
						RefreshTime:  to.Int64Ptr(7200),
						RetryTime:    to.Int64Ptr(900),
						ExpireTime:   to.Int64Ptr(1209600),
						MinimumTTL:   to.Int64Ptr(86400),
					},
				},
			},
			&dns.RecordSet{
				Name: to.StringPtr("record-srv"),
				Type: to.StringPtr("Microsoft.Network/dnszones/SRV"),
				Etag: to.StringPtr("ETAG_SRV"),
				RecordSetProperties: &dns.RecordSetProperties{
					TTL:  to.Int64Ptr(30),
					Fqdn: to.StringPtr("record-srv.example.com."),
					SrvRecords: &[]dns.SrvRecord{
						dns.SrvRecord{
							Priority: to.Int32Ptr(1),
							Weight:   to.Int32Ptr(10),
							Port:     to.Int32Ptr(5269),
							Target:   to.StringPtr("app.example.com"),
						},
					},
				},
			},
			&dns.RecordSet{
				Name: to.StringPtr("record-txt"),
				Type: to.StringPtr("Microsoft.Network/dnszones/TXT"),
				Etag: to.StringPtr("ETAG_TXT"),
				RecordSetProperties: &dns.RecordSetProperties{
					TTL:  to.Int64Ptr(30),
					Fqdn: to.StringPtr("record-txt.example.com."),
					TxtRecords: &[]dns.TxtRecord{
						dns.TxtRecord{
							Value: &[]string{"TEST VALUE"},
						},
					},
				},
			},
		}
		got, _ := convertAzureRecordSetsToLibdnsRecords(azureRecordSets)
		want := []libdns.Record{
			libdns.Record{
				ID:    "ETAG_A",
				Type:  "A",
				Name:  "test.example.com.",
				Value: "127.0.0.1",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_AAAA",
				Type:  "AAAA",
				Name:  "record-aaaa.example.com.",
				Value: "::1",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_CAA",
				Type:  "CAA",
				Name:  "record-caa.example.com.",
				Value: "0 issue ca.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_CNAME",
				Type:  "CNAME",
				Name:  "record-cname.example.com.",
				Value: "www.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_MX",
				Type:  "MX",
				Name:  "record-mx.example.com.",
				Value: "10 mail.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_NS",
				Type:  "NS",
				Name:  "example.com.",
				Value: "ns1.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_NS",
				Type:  "NS",
				Name:  "example.com.",
				Value: "ns2.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_PTR",
				Type:  "PTR",
				Name:  "record-ptr.example.com.",
				Value: "hoge.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_SOA",
				Type:  "SOA",
				Name:  "example.com.",
				Value: "ns1.example.com hostmaster.example.com 1 7200 900 1209600 86400",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_SRV",
				Type:  "SRV",
				Name:  "record-srv.example.com.",
				Value: "1 10 5269 app.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_TXT",
				Type:  "TXT",
				Name:  "record-txt.example.com.",
				Value: "TEST VALUE",
				TTL:   time.Duration(30) * time.Second,
			},
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("diff: %s", diff)
		}
	})
	t.Run("type=unsupported", func(t *testing.T) {
		azureRecordSets := []*dns.RecordSet{
			&dns.RecordSet{
				Type: to.StringPtr("Microsoft.Network/dnszones/ERR"),
			},
		}
		_, err := convertAzureRecordSetsToLibdnsRecords(azureRecordSets)
		got := err.Error()
		want := "The type ERR cannot be interpreted."
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("diff: %s", diff)
		}
	})
}

func Test_convertLibdnsRecordToAzureRecordSet(t *testing.T) {
	t.Run("type=supported", func(t *testing.T) {
		libdnsRecords := []libdns.Record{
			libdns.Record{
				ID:    "ETAG_A",
				Type:  "A",
				Name:  "test.example.com.",
				Value: "127.0.0.1",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_AAAA",
				Type:  "AAAA",
				Name:  "record-aaaa.example.com.",
				Value: "::1",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_CAA",
				Type:  "CAA",
				Name:  "record-caa.example.com.",
				Value: "0 issue ca.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_CNAME",
				Type:  "CNAME",
				Name:  "record-cname.example.com.",
				Value: "www.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_MX",
				Type:  "MX",
				Name:  "record-mx.example.com.",
				Value: "10 mail.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_NS",
				Type:  "NS",
				Name:  "example.com.",
				Value: "ns1.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_PTR",
				Type:  "PTR",
				Name:  "record-ptr.example.com.",
				Value: "hoge.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_SOA",
				Type:  "SOA",
				Name:  "example.com.",
				Value: "ns1.example.com hostmaster.example.com 1 7200 900 1209600 86400",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_SRV",
				Type:  "SRV",
				Name:  "record-srv.example.com.",
				Value: "1 10 5269 app.example.com",
				TTL:   time.Duration(30) * time.Second,
			},
			libdns.Record{
				ID:    "ETAG_TXT",
				Type:  "TXT",
				Name:  "record-txt.example.com.",
				Value: "TEST VALUE",
				TTL:   time.Duration(30) * time.Second,
			},
		}
		var got []dns.RecordSet
		for _, libdnsRecord := range libdnsRecords {
			convertedRecord, _ := convertLibdnsRecordToAzureRecordSet(libdnsRecord)
			got = append(got, convertedRecord)
		}
		want := []dns.RecordSet{
			dns.RecordSet{
				RecordSetProperties: &dns.RecordSetProperties{
					TTL: to.Int64Ptr(30),
					ARecords: &[]dns.ARecord{
						dns.ARecord{
							Ipv4Address: to.StringPtr("127.0.0.1"),
						},
					},
				},
			},
			dns.RecordSet{
				RecordSetProperties: &dns.RecordSetProperties{
					TTL: to.Int64Ptr(30),
					AaaaRecords: &[]dns.AaaaRecord{
						dns.AaaaRecord{
							Ipv6Address: to.StringPtr("::1"),
						},
					},
				},
			},
			dns.RecordSet{
				RecordSetProperties: &dns.RecordSetProperties{
					TTL: to.Int64Ptr(30),
					CaaRecords: &[]dns.CaaRecord{
						dns.CaaRecord{
							Flags: to.Int32Ptr(0),
							Tag:   to.StringPtr("issue"),
							Value: to.StringPtr("ca.example.com"),
						},
					},
				},
			},
			dns.RecordSet{
				RecordSetProperties: &dns.RecordSetProperties{
					TTL: to.Int64Ptr(30),
					CnameRecord: &dns.CnameRecord{
						Cname: to.StringPtr("www.example.com"),
					},
				},
			},
			dns.RecordSet{
				RecordSetProperties: &dns.RecordSetProperties{
					TTL: to.Int64Ptr(30),
					MxRecords: &[]dns.MxRecord{
						dns.MxRecord{
							Preference: to.Int32Ptr(10),
							Exchange:   to.StringPtr("mail.example.com"),
						},
					},
				},
			},
			dns.RecordSet{
				RecordSetProperties: &dns.RecordSetProperties{
					TTL: to.Int64Ptr(30),
					NsRecords: &[]dns.NsRecord{
						dns.NsRecord{
							Nsdname: to.StringPtr("ns1.example.com"),
						},
					},
				},
			},
			dns.RecordSet{
				RecordSetProperties: &dns.RecordSetProperties{
					TTL: to.Int64Ptr(30),
					PtrRecords: &[]dns.PtrRecord{
						dns.PtrRecord{
							Ptrdname: to.StringPtr("hoge.example.com"),
						},
					},
				},
			},
			dns.RecordSet{
				RecordSetProperties: &dns.RecordSetProperties{
					TTL: to.Int64Ptr(30),
					SoaRecord: &dns.SoaRecord{
						Host:         to.StringPtr("ns1.example.com"),
						Email:        to.StringPtr("hostmaster.example.com"),
						SerialNumber: to.Int64Ptr(1),
						RefreshTime:  to.Int64Ptr(7200),
						RetryTime:    to.Int64Ptr(900),
						ExpireTime:   to.Int64Ptr(1209600),
						MinimumTTL:   to.Int64Ptr(86400),
					},
				},
			},
			dns.RecordSet{
				RecordSetProperties: &dns.RecordSetProperties{
					TTL: to.Int64Ptr(30),
					SrvRecords: &[]dns.SrvRecord{
						dns.SrvRecord{
							Priority: to.Int32Ptr(1),
							Weight:   to.Int32Ptr(10),
							Port:     to.Int32Ptr(5269),
							Target:   to.StringPtr("app.example.com"),
						},
					},
				},
			},
			dns.RecordSet{
				RecordSetProperties: &dns.RecordSetProperties{
					TTL: to.Int64Ptr(30),
					TxtRecords: &[]dns.TxtRecord{
						dns.TxtRecord{
							Value: &[]string{"TEST VALUE"},
						},
					},
				},
			},
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("diff: %s", diff)
		}
	})
	t.Run("type=unsupported", func(t *testing.T) {
		libdnsRecords := []libdns.Record{
			libdns.Record{
				Type: "ERR",
			},
		}
		_, err := convertLibdnsRecordToAzureRecordSet(libdnsRecords[0])
		got := err.Error()
		want := "The type ERR cannot be interpreted."
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("diff: %s", diff)
		}
	})
}
