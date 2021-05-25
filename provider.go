// Package libdns-loopia implements a DNS record management client compatible
// with the libdns interfaces for Loopia.
package loopia

import (
	"context"
	"fmt"

	"github.com/libdns/libdns"
)

type contextKey string

func (c contextKey) String() string {
	return "libdns-loopia " + string(c)
}

var (
	contextKeyStack = contextKey("stack")
)

func addStack(ctx context.Context, name string) context.Context {
	v := ctx.Value(contextKeyStack)
	if v != nil {
		return context.WithValue(ctx, contextKeyStack, fmt.Sprintf("%v", v)+"."+name)
	}
	return context.WithValue(ctx, contextKeyStack, name)
}

// Provider facilitates DNS record manipulation with Loopia.
type Provider struct {
	client
	// TODO: put config fields here (with snake_case json
	// struct tags on exported fields), for example:
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Customer string `json:"customer,omitempty"`
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.getZoneRecords(ctx, zone)
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	Log().Debug("AppendRecords")
	result, err := p.addDNSEntries(ctx, zone, records)
	if err != nil {
		Log().Warnw("error appending",
			"err", err,
			"result", result,
		)
	}
	Log().Debugw("done", "result", result, "err", err)
	return result, err
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.setDNSEntries(ctx, zone, records)
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.removeDNSEntries(ctx, zone, records)
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
