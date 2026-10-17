package dsh

import (
	"context"
	"errors"
)

// EventDelivery is a process-local, per-response authority. It is never sent to
// the Host. A retained UI occurrence may revoke it while HTTP is in flight.
// The ordinary event stream remains the sole waterfall sender.
type EventDelivery struct {
	Client  *Client
	EventID string
	Context context.Context
	Valid   func() bool
}

func (d *EventDelivery) bind(parent context.Context, client *Client, eventID string) (context.Context, func(), error) {
	if d == nil {
		return parent, func() {}, nil
	}
	if d.Client != client || d.EventID != eventID || d.Context == nil || d.Context.Err() != nil || d.Valid == nil || !d.Valid() {
		return nil, func() {}, errors.New("interaction authority expired")
	}
	ctx, cancel := context.WithCancel(parent)
	stop := context.AfterFunc(d.Context, cancel)
	if d.Context.Err() != nil {
		cancel()
	}
	return ctx, func() { stop(); cancel() }, nil
}
