package designate

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/zones"
	"github.com/libdns/libdns"
	"github.com/pkg/errors"
	"os"
	"time"
)

func (p *Provider) getRecords(recordSets []recordsets.RecordSet) ([]libdns.Record, error) {
	var records []libdns.Record
	for _, j := range recordSets {
		tmp := libdns.Record{
			ID:   j.ID,
			Type: j.Type,
			Name: j.Name,
			TTL:  time.Duration(j.TTL) * time.Second,
		}
		records = append(records, tmp)
	}

	return records, nil
}

func (p *Provider) getRecordID(recordName string, zone string) (string, error) {
	recordName = recordName + zone
	listOpts := recordsets.ListOpts{
		Type: "TXT",
	}

	allPages, err := recordsets.ListByZone(p.DNSClient, p.Request.ZoneID, listOpts).AllPages()
	if err != nil {
		return "", err
	}

	allRecordSets, err := recordsets.ExtractRecordSets(allPages)
	if err != nil {
		return "", err
	}

	for _, rr := range allRecordSets {
		if recordName == rr.Name {
			return rr.ID, nil
		}
	}

	return "", nil
}

func (p *Provider) setRecordID(recordID string) {
	p.Request.RecordID = recordID
}

func (p *Provider) setDescription(desc string) error {
	if desc == "" {
		return errors.New("example description")
	}
	p.DNSOptions.DNSDescription = desc

	return nil
}

func (p *Provider) createRecord(record libdns.Record, zone string) error {
	createOpts := recordsets.CreateOpts{
		Name:    record.Name + zone,
		Type:    record.Type,
		TTL:     int(record.TTL / time.Second),
		Records: []string{record.Value},
	}

	exist, err := p.getRecordID(record.Name, zone)
	if err != nil {
		return errors.Wrap(err, "cannot get recordID")
	}

	if exist != "" {
		return errors.New("DNS record already exist")
	}

	_, err = recordsets.Create(p.DNSClient, p.Request.ZoneID, createOpts).Extract()
	if err != nil {
		return errors.Wrap(err, "cannot create DNS record")
	}

	return nil
}

func (p *Provider) updateRecord(record libdns.Record, zone string) error {
	updateOpts := recordsets.UpdateOpts{
		TTL:     IntToPointer(int(record.TTL / time.Second)),
		Records: []string{record.Value},
	}

	// Update updates a recordset in a given zone
	_, err := recordsets.Update(p.DNSClient, p.Request.ZoneID, p.Request.RecordID, updateOpts).Extract()
	if err != nil {
		return err
	}
	return nil
}

func (p *Provider) deleteRecord(record libdns.Record, zone string) error {
	err := recordsets.Delete(p.DNSClient, p.Request.ZoneID, p.Request.RecordID).ExtractErr()
	if err != nil {
		return err
	}

	return nil
}

func New(zoneName string) (*Provider, error) {
	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return nil, err
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, err
	}

	dnsClient, err := openstack.NewDNSV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		return nil, err
	}

	zoneID, err := setZoneID(zoneName, dnsClient)
	if err != nil {
		return nil, err
	}

	if zoneID == "" {
		return nil, errors.New("zoneID does not exist")
	}

	return &Provider{
		DNSClient: dnsClient,
		Request:   Request{ZoneID: zoneID},
	}, nil
}

func setZoneID(zoneName string, dnsClient *gophercloud.ServiceClient) (string, error) {
	listOpts := zones.ListOpts{}

	allPages, err := zones.List(dnsClient, listOpts).AllPages()
	if err != nil {
		return "", errors.Wrap(err, "trying to get zones list")
	}

	allZones, err := zones.ExtractZones(allPages)
	if err != nil {
		return "", errors.Wrap(err, "trying to extract zones")
	}

	for _, zone := range allZones {
		if zoneName == zone.Name {
			return zone.ID, nil
		}
	}

	return "", nil
}

func IntToPointer(x int) *int {
	return &x
}
