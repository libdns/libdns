package designate

import (
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/zones"
	"github.com/libdns/libdns"
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

	allPages, err := recordsets.ListByZone(p.dnsClient, p.zoneID, listOpts).AllPages()
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

func (p *Provider) createRecord(record libdns.Record, zone string) error {
	createOpts := recordsets.CreateOpts{
		Name:    record.Name + zone,
		Type:    record.Type,
		TTL:     int(record.TTL / time.Second),
		Records: []string{record.Value},
	}

	exist, err := p.getRecordID(record.Name, zone)
	if err != nil {
		return fmt.Errorf("cannot get recordID: %v", err)
	}

	if exist != "" {
		return errors.New("DNS record already exist")
	}

	_, err = recordsets.Create(p.dnsClient, p.zoneID, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("cannot create DNS record: %v", err)
	}

	return nil
}

func (p *Provider) updateRecord(record libdns.Record, recordID string) error {
	updateOpts := recordsets.UpdateOpts{
		TTL:     intToPointer(int(record.TTL / time.Second)),
		Records: []string{record.Value},
	}

	// Update updates a recordset in a given zone
	_, err := recordsets.Update(p.dnsClient, p.zoneID, recordID, updateOpts).Extract()
	if err != nil {
		return err
	}
	return nil
}

func (p *Provider) deleteRecord(recordID string) error {
	err := recordsets.Delete(p.dnsClient, p.zoneID, recordID).ExtractErr()
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) exportEnvVariables() error {
	err := os.Setenv("OS_REGION_NAME", p.AuthOpenStack.RegionName)
	if err != nil {
		return fmt.Errorf("cannot set environment variable: %v", err)
	}
	err = os.Setenv("OS_TENANT_ID", p.AuthOpenStack.TenantID)
	if err != nil {
		return fmt.Errorf("cannot set environment variable: %v", err)
	}
	err = os.Setenv("OS_IDENTITY_API_VERSION", p.AuthOpenStack.IdentityApiVersion)
	if err != nil {
		return fmt.Errorf("cannot set environment variable: %v", err)
	}
	err = os.Setenv("OS_PASSWORD", p.AuthOpenStack.Password)
	if err != nil {
		return fmt.Errorf("cannot set environment variable: %v", err)
	}
	err = os.Setenv("OS_AUTH_URL", p.AuthOpenStack.AuthURL)
	if err != nil {
		return fmt.Errorf("cannot set environment variable: %v", err)
	}
	err = os.Setenv("OS_USERNAME", p.AuthOpenStack.Username)
	if err != nil {
		return fmt.Errorf("cannot set environment variable: %v", err)
	}
	err = os.Setenv("OS_TENANT_NAME", p.AuthOpenStack.TenantName)
	if err != nil {
		return fmt.Errorf("cannot set environment variable: %v", err)
	}
	err = os.Setenv("OS_ENDPOINT_TYPE", p.AuthOpenStack.EndpointType)
	if err != nil {
		return fmt.Errorf("cannot set environment variable: %v", err)
	}
	return nil
}

func (p *Provider) isAuth() (bool, error) {
	if p.dnsClient != nil {
		_, err := p.dnsClient.GetAuthResult().ExtractTokenID()
		if err != nil {
			return true, err
		}
	}

	return false, nil
}

func (p *Provider) auth() error {
	authStatus, err := p.isAuth()
	if err != nil {
		return err
	}

	if authStatus {
		return nil
	}

	err = p.exportEnvVariables()
	if err != nil {
		return err
	}

	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return err
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return err
	}

	dnsClient, err := openstack.NewDNSV2(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		return err
	}
	p.dnsClient = dnsClient

	return nil
}

func (p *Provider) setZone(zone string) error {
	zoneID, err := p.setZoneID(zone)
	if err != nil {
		return err
	}
	p.zoneID = zoneID

	if p.zoneID == "" {
		return errors.New("zoneID does not exist")
	}
	return nil
}

func (p *Provider) setZoneID(zoneName string) (string, error) {
	listOpts := zones.ListOpts{}

	allPages, err := zones.List(p.dnsClient, listOpts).AllPages()
	if err != nil {
		return "", fmt.Errorf("trying to get zones list: %v", err)
	}

	allZones, err := zones.ExtractZones(allPages)
	if err != nil {
		return "", fmt.Errorf("trying to extract zones: %v", err)
	}

	for _, zone := range allZones {
		if zoneName == zone.Name {
			return zone.ID, nil
		}
	}

	return "", nil
}

func intToPointer(x int) *int {
	return &x
}
