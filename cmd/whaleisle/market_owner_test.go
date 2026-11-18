package main

import (
	"path/filepath"
	"testing"

	"whaleisle/internal/dsh"
	marketstore "whaleisle/internal/store/market"
)

func TestMarketRenderedOwnerCannotActAfterReconnect(t *testing.T) {
	for _, reconnect := range []string{"same-client", "other-client", "other-store"} {
		t.Run(reconnect, func(t *testing.T) {
			client := &dsh.Client{}
			store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "personal.json"))
			store.Attach(client, nil)
			s := &mainState{}
			coord := s.marketCoordinator()
			coord.SetClient(client)
			coord.SetStore(store)
			owner := coord.CaptureOwner(store)
			switch reconnect {
			case "same-client":
				store.Detach()
				store.Attach(client, nil)
			case "other-client":
				coord.SetClient(&dsh.Client{})
			case "other-store":
				coord.SetStore(marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "other.json")))
			}
			for _, install := range []bool{false, true} {
				op, _ := coord.BeginAttemptForOwner("fixture-plugin", nil, install, &owner)
				if op != nil || coord.IsBusy() {
					t.Fatal("stale rendered action acquired a new Host worker")
				}
			}
		})
	}
}

func TestMarketRenderedCurrentOwnerCanBegin(t *testing.T) {
	client := &dsh.Client{}
	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "personal.json"))
	store.Attach(client, nil)
	s := &mainState{}
	coord := s.marketCoordinator()
	coord.SetClient(client)
	coord.SetStore(store)
	owner := coord.CaptureOwner(store)
	op, _ := coord.BeginAttemptForOwner("", nil, false, &owner)
	if op == nil || op.Client != client || op.Store != store {
		t.Fatal("current rendered action did not retain its Host")
	}
	op.Cancel()
}
