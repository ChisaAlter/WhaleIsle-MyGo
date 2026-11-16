package main

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/bridgealerts"
)

type staticBridgeSource struct {
	key atomic.Value
}

func (s *staticBridgeSource) Current(context.Context) (string, error) {
	v, _ := s.key.Load().(string)
	return v, nil
}
func (s *staticBridgeSource) Claim(context.Context, string) (bridgealerts.Batch, error) {
	return bridgealerts.Batch{}, bridgealerts.ErrNotClaimed
}
func (s *staticBridgeSource) Open(context.Context, string) error { return nil }

func TestBridgeAlertPollChangedSkipsUnchangedSnapshots(t *testing.T) {
	src := &staticBridgeSource{}
	src.key.Store("scope-1")
	mon := bridgealerts.New(src)
	defer mon.Close()
	run := &bridgeAlertUI{
		monitor:   mon,
		notices:   map[bridgealerts.Ticket]*mygo.Notification{},
		attempted: map[bridgealerts.Ticket]bool{},
	}
	mon.Poll()
	sig1, changed := bridgeAlertPollChanged(run, nil)
	if !changed {
		t.Fatal("initial bridge alert poll signature should trigger UI sync")
	}
	mon.Poll()
	sig2, changed := bridgeAlertPollChanged(run, &sig1)
	if changed {
		t.Fatalf("unchanged bridge alert poll unexpectedly signaled update: %+v vs %+v", sig1, sig2)
	}
	src.key.Store("scope-2")
	mon.Poll()
	if _, changed := bridgeAlertPollChanged(run, &sig2); !changed {
		t.Fatal("changed bridge alert scope did not trigger UI update")
	}
}
