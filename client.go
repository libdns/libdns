package loopia

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kolo/xmlrpc"
	"github.com/libdns/libdns"
)

const (
	apiurl = "https://api.loopia.se/RPCSERV"
)

type client struct {
	rpc   *xmlrpc.Client
	mutex sync.Mutex
}

type myRecord struct {
	*libdns.Record
}
type loopiaRecord struct {
	ID       int64  `xmlrpc:"record_id"`
	TTL      int    `xmlrpc:"ttl"`
	Type     string `xmlrpc:"type"`
	Value    string `xmlrpc:"rdata"`
	Priority int    `xmlrpc:"priority"`
}

// func (r myRecord) MarshalZerologObject(e *zerolog.Event) {
// 	e.Str("name", r.Name).Str("id", r.ID).Str("value", r.Value).Str("type", r.Type).Dur("ttl", r.TTL)
// }

// func (r loopiaRecord) MarshalZerologObject(e *zerolog.Event) {
// 	e.Int64("id", r.ID).Str("value", r.Value).Str("type", r.Type).Int("ttl", r.TTL).Int("priority", r.Priority)
// }

type myRecords []myRecord

// func (rr myRecords) MarshalZerologArray(a *zerolog.Array) {
// 	for _, r := range rr {
// 		a.Object(r)
// 	}
// }

func toMyRecords(from []libdns.Record) myRecords {
	to := make(myRecords, len(from))
	for i, fr := range from {
		to[i] = myRecord{&fr}
	}
	return to
}

func cleanZone(zone string) string {
	if strings.HasSuffix(zone, ".") {
		zone = zone[:len(zone)-1]
	}
	return zone
}

func validZone(zone string) bool {
	if zone == "" || len(zone) < 4 {
		return false
	}
	return true
}

func validRecord(r libdns.Record) bool {
	if r.Name == "" {
		return false
	}
	if r.Type == "" {
		return false
	}
	if r.Value == "" {
		return false
	}
	if r.TTL < 0 || r.TTL > (time.Hour*8*24) {
		return false
	}
	if r.ID != "" {
		_, err := strconv.ParseInt(r.ID, 10, 64)
		if err != nil {
			return false
		}
	}
	return true
}

func toLoopiaRecord(r libdns.Record) loopiaRecord {
	out := loopiaRecord{Type: r.Type, TTL: int(r.TTL / time.Second), Value: r.Value, ID: idToInt(r.ID)}
	return out
}

func idToInt(id string) int64 {
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0
	}
	return idInt
}

func (p *Provider) getRpc() *xmlrpc.Client {
	if p.rpc == nil {
		rpc, err := xmlrpc.NewClient(apiurl, nil)
		if err != nil {
			Log().Errorw("error", err)
			os.Exit(1)

		}
		p.rpc = rpc
	}
	return p.rpc
}

func (p *Provider) call(serviceMethod string, args []interface{}, reply interface{}) error {
	params := []interface{}{
		p.Username,
		p.Password,
		p.Customer,
	}
	params = append(params, args...)
	// logger.Debug().Str("serviceMethod", serviceMethod).Msgf("calling xmlrpc with %v", params)
	// Log().Debugw("calling xmlrpc",
	// 	"serviceMethod", serviceMethod,
	// 	"params", params,
	// )
	return p.getRpc().Call(
		serviceMethod,
		params,
		reply,
	)
}

func (p *Provider) getRecords(ctx context.Context, zone, name string) ([]libdns.Record, error) {
	// logger.Debug().Str("zone", zone).Str("name", name).Msg("getRecords")
	if !validZone(zone) {
		return nil, fmt.Errorf("invalid zone '%s'", zone)
	}
	if name == "" {
		return nil, fmt.Errorf("invalide name '%s'", name)
	}
	records := []loopiaRecord{}
	Log().Debugw("getRecords", "zone", zone, "name", name)
	err := p.call("getZoneRecords", params(zone, name), &records)
	if err != nil {
		// logger.Warn().Err(err).Str("zone", zone).Str("name", name).Msg("unexpected error getting zone records")
		return nil, err
	}
	result := []libdns.Record{}
	for _, r := range records {
		result = append(result, libdns.Record{
			ID:    strconv.FormatInt(r.ID, 10),
			Type:  r.Type,
			Name:  name,
			Value: r.Value,
			TTL:   time.Duration(r.TTL * int(time.Second)),
		})
	}
	Log().Debugw("end-getRecords", "zone", zone, "name", name, "count", len(result), "err", err)
	return result, nil
}

func (p *Provider) addRecord(ctx context.Context, zone string, record libdns.Record, withSubdomain bool) (*libdns.Record, error) {
	// logger.Debug().Str("zone", zone).Object("record", myRecord{&record}).Bool("withSubdomain", withSubdomain).Msg("addRecord")
	Log().Debugw("addRecord",
		"zone", zone,
		"record", record,
		"withSubdomain", withSubdomain,
	)
	if withSubdomain {
		var response string
		err := p.call("addSubdomain", params(zone, record.Name), &response)
		if err != nil {
			// logger.Error().Str("response", response).Msg("unexpected error on addSubdomain")
			Log().Errorw("unexpected error adding subdomain", "err", err, "response", response)
			return nil, err
		}
	}
	new := &loopiaRecord{Type: record.Type, TTL: int(record.TTL / time.Second), Value: record.Value}
	var result string
	if err := p.call("addZoneRecord", params(zone, record.Name, new), &result); err != nil || result != "OK" {
		// logger.Error().Err(err).Str("response", result).Msg("unexpected error adding zone record using loopia api")
		return nil, err
	}
	Log().Debugw("getting records to fetch ID", "zone", zone, "name", record.Name)
	records, err := p.getRecords(ctx, zone, record.Name)
	if err != nil {
		// logger.Error().Err(err).Str("name", record.Name).Msg("unable to get new records for ID")
		return nil, err
	}
	for _, r := range records {
		id := r.ID
		r.ID = record.ID
		Log().Debugw("comparing", "a", r, "b", record)
		if r == record {
			// match
			r.ID = id
			return &r, nil
		}
	}

	return nil, fmt.Errorf("unable to get record for ID")
}

func params(args ...interface{}) []interface{} {
	return args
}

func (p *Provider) getZoneRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	// logger.Debug().Str("zone", zone).Msg("getZoneRecords")
	if !validZone(zone) {
		return nil, fmt.Errorf("invalide zone '%s'", zone)
	}
	zone = cleanZone(zone)
	names := []string{}
	err := p.call("getSubdomains", params(zone), &names)
	if err != nil {
		return nil, err
	}
	// logger.Debug().Int("names", len(names)).Msg("got names")
	result := []libdns.Record{}
myloop:
	for _, name := range names {
		select {
		case <-ctx.Done():
			break myloop
		default:
			records, err := p.getRecords(ctx, zone, name)
			if err != nil {
				return nil, fmt.Errorf("error getting zone records for %s: %w", name, err)
			}
			result = append(result, records...)
		}
	}
	return result, err
}

func (p *Provider) addDNSEntries(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	Log().Debugw("addDNSEntries",
		"zone", zone,
		"records", records,
	)
	if !validZone(zone) {
		return nil, fmt.Errorf("invalide zone '%s'", zone)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("records is nil or empty")
	}
	for i, r := range records {
		if !validRecord(r) {
			return nil, fmt.Errorf("record %d is invalid", i)
		}
	}
	zone = cleanZone(zone)
	result := []libdns.Record{}
	// api, _ := p.getApi()
	cache := make(map[string][]libdns.Record)
	subsCreated := make(map[string]bool)
OUTER:
	for _, new := range records {
		select {
		case <-ctx.Done():
			break OUTER
		default:
			if cache[new.Name] == nil {
				existingRecords, err := p.getRecords(ctx, zone, new.Name)
				if err != nil {
					return result, err
				}
				cache[new.Name] = existingRecords
			}
			withSubdomain := false
			if len(cache[new.Name]) == 0 && !subsCreated[new.Name] {
				withSubdomain = true
			}
			for _, existing := range cache[new.Name] {
				id := existing.ID
				existing.ID = ""
				if existing == new {
					Log().Debugw("identical record exists, skipping",
						"record", new,
						"id", id)
					existing.ID = id
					result = append(result, existing)
					continue OUTER
				}
				existing.ID = id
			}
			if withSubdomain {
				subsCreated[new.Name] = true
			}
			cn, err := p.addRecord(ctx, zone, new, withSubdomain)
			if err != nil {
				return result, err
			}
			// logger.Debug().Stack().Object("added", myRecord{cn}).Msg("added record returned")
			Log().Debugw("added record returned", "record", cn)
			result = append(result, *cn)
		}
	}
	Log().Debug("done with addDNSEntries")
	return result, nil
}

func (p *Provider) setDNSEntries(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	// logger.Debug().Str("zone", zone).Array("records", toMyRecords(records)).Msg("setDNSEntries")
	if !validZone(zone) {
		return nil, fmt.Errorf("invalide zone '%s'", zone)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("records is nil or empty")
	}
	for i, r := range records {
		if !validRecord(r) {
			return nil, fmt.Errorf("record %d is invalid", i)
		}
		if idToInt(r.ID) < 1 {
			return nil, fmt.Errorf("record %d has invalid ID", i)
		}
	}
	zone = cleanZone(zone)
	result := []libdns.Record{}
myloop:
	for _, r := range records {
		select {
		case <-ctx.Done():
			break myloop
		default:
			updated := toLoopiaRecord(r)
			// logger.Debug().Object("record", myRecord{&r}).Object("updated", updated).Msg("updating record")
			var response string
			err := p.call("updateZoneRecord", params(zone, r.Name, updated), &response)
			if err != nil {
				// logger.Error().Err(err).Str("response", response).Msg("unexpected error when updating record")
				return result, err
			}
			result = append(result, r)
		}
	}

	return result, nil
}

func (p *Provider) removeDNSEntries(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if !validZone(zone) {
		return nil, fmt.Errorf("invalide zone '%s'", zone)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("records is nil or empty")
	}
	for i, r := range records {
		if idToInt(r.ID) < 1 {
			return nil, fmt.Errorf("record %d has invalid ID", i)
		}
		if r.Name == "" {
			return nil, fmt.Errorf("record %d has invalid name", i)
		}
	}
	zone = cleanZone(zone)
	result := []libdns.Record{}
firstloop:
	for _, r := range records {
		select {
		case <-ctx.Done():
			break firstloop
		default:
			// logger.Debug().Object("record", myRecord{&r}).Msg("Removing")
			var response string
			err := p.call("removeZoneRecord", params(zone, r.Name, idToInt(r.ID)), &response)
			if err != nil {
				// logger.Error().Err(err).Str("response", response).Msg("unexpected error when deleting record")
				return result, err
			}
			result = append(result, r)
		}
	}
	names := make(map[string]bool)
	// logger.Debug().Msg("checking removed result if subdomain is to be deleted")
secondloop:
	for _, r := range result {
		select {
		case <-ctx.Done():
			break secondloop
		default:
			if !names[r.Name] {
				names[r.Name] = true
				res, err := p.getRecords(ctx, zone, r.Name)
				if err != nil {
					Log().Warnw("unexpected error getting zone records", "err", err)
					continue
				}
				// logger.Debug().Int("records", len(res)).Str("name", r.Name).Msg("records left")
				if len(res) == 0 {
					// logger.Debug().Str("name", r.Name).Str("zone", zone).Msg("removing subdomain")
					var response string
					err := p.call("removeSubdomain", params(zone, r.Name), &response)
					if err != nil {
						Log().Warnw("unexpected error deleting subdomain", "err", err, "response", response)
					}
				}
			}
		}
	}
	p.getZoneRecords(ctx, zone)
	return result, nil
}
