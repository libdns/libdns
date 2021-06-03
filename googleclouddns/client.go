package googleclouddns

import (
	"context"
	"fmt"
	"time"

	"github.com/libdns/libdns"
	"google.golang.org/api/dns/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	// zoneMapTTL timeout for the Google Cloud DNS zone map
	zoneMapTTL = time.Minute * 5
)

// newService initializes the Google client for the provider using the specified JSON file for credentials if set.
func (p *Provider) newService(ctx context.Context) error {
	var err error
	if p.service == nil {
		scopeOption := option.WithScopes(dns.NdevClouddnsReadwriteScope)
		if p.ServiceAccountJSON != "" {
			p.service, err = dns.NewService(ctx, scopeOption, option.WithCredentialsFile(p.ServiceAccountJSON))
		} else {
			p.service, err = dns.NewService(ctx, scopeOption)
		}
	}
	return err
}

// getCloudDNSRecords returns all the records for the specified zone. It breaks up a single Google Record
// with multiple Values into separate libdns.Records.
func (p *Provider) getCloudDNSRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if err := p.newService(ctx); err != nil {
		return nil, err
	}

	gcdZone, err := p.getCloudDNSZone(zone)
	if err != nil {
		return nil, err
	}
	rrsReq := p.service.ResourceRecordSets.List(p.Project, gcdZone)
	records := make([]libdns.Record, 0)
	if err := rrsReq.Pages(ctx, func(page *dns.ResourceRecordSetsListResponse) error {
		for _, googleRecord := range page.Rrsets {
			records = append(records, convertToLibDNS(googleRecord, zone)...)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return records, nil
}

// setCloudDNSRecord will attempt to create a new Google Cloud DNS record set based on the libdns.Records or patch an existing one if
// it already exists and patch is true.
func (p *Provider) setCloudDNSRecord(ctx context.Context, zone string, values []libdns.Record, patch bool) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if err := p.newService(ctx); err != nil {
		return nil, err
	}

	gcdZone, err := p.getCloudDNSZone(zone)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("no records available to add to zone %s", zone)
	}
	name := values[0].Name
	fullName := libdns.AbsoluteName(name, zone)
	rrs := dns.ResourceRecordSet{
		Name:    fullName,
		Rrdatas: make([]string, 0),
		Ttl:     int64(values[0].TTL / time.Second),
		Type:    values[0].Type,
	}
	for _, record := range values {
		rrs.Rrdatas = append(rrs.Rrdatas, record.Value)
	}
	googleRecord, err := p.service.Projects.ManagedZones.Rrsets.Create(p.Project, gcdZone, &rrs).Context(ctx).Do()
	if err != nil {
		if gErr, ok := err.(*googleapi.Error); !ok || (gErr.Code == 409 && !patch) {
			return nil, err
		}
		// Record exists and we'd really like to get this libdns.Record into the zone so how about we try patching it instead...
		googleRecord, err = p.service.Projects.ManagedZones.Rrsets.Patch(p.Project, gcdZone, rrs.Name, rrs.Type, &rrs).Context(ctx).Do()
		if err != nil {
			return nil, err
		}
	}
	return convertToLibDNS(googleRecord, zone), nil
}

// deleteCloudDNSRecord will delete the specified record set.
func (p *Provider) deleteCloudDNSRecord(ctx context.Context, zone, name, dnsType string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if err := p.newService(ctx); err != nil {
		return err
	}

	gcdZone, err := p.getCloudDNSZone(zone)
	if err != nil {
		return err
	}
	fullName := libdns.AbsoluteName(name, zone)
	_, err = p.service.Projects.ManagedZones.Rrsets.Delete(p.Project, gcdZone, fullName, dnsType).Context(ctx).Do()
	return err
}

// getCloudDNSZone will return the Google Cloud DNS zone name for the specified zone. The data is cached
// for five minutes to avoid repeated calls to the GCP API servers.
func (p *Provider) getCloudDNSZone(zone string) (string, error) {
	if p.zoneMap == nil || time.Now().Sub(p.zoneMapLastUpdated) > zoneMapTTL {
		p.zoneMap = make(map[string]string)
		zonesLister := p.service.ManagedZones.List(p.Project)
		err := zonesLister.Pages(context.Background(), func(response *dns.ManagedZonesListResponse) error {
			for _, zone := range response.ManagedZones {
				p.zoneMap[zone.DnsName] = zone.Name
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		p.zoneMapLastUpdated = time.Now()
	}
	if zoneName, ok := p.zoneMap[zone]; ok {
		return zoneName, nil
	}
	return "", fmt.Errorf("unable to find Google managaged zone for domain %s", zone)
}

func convertToLibDNS(googleRecord *dns.ResourceRecordSet, zone string) []libdns.Record {
	records := make([]libdns.Record, 0)
	for _, value := range googleRecord.Rrdatas {
		// there can be multiple values per record  so
		// let's treat each one as a separate libdns Record
		record := libdns.Record{
			Type:  googleRecord.Type,
			Name:  libdns.RelativeName(googleRecord.Name, zone),
			Value: value,
			TTL:   time.Duration(googleRecord.Ttl) * time.Second,
		}
		records = append(records, record)
	}
	return records
}
