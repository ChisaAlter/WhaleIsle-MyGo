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
			s := &mainState{dshClient: client, market: store}
			owner := s.captureMarketOwner(store)
			switch reconnect {
			case "same-client":
				store.Detach()
				store.Attach(client, nil)
			case "other-client":
				s.dshClient = &dsh.Client{}
			case "other-store":
				s.market = marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "other.json"))
			}
			for _, install := range []bool{false, true} {
				op, _ := s.beginMarketAttemptForOwner("fixture-plugin", nil, install, &owner)
				if op != nil || s.marketWorker != nil || s.marketInstallBusy {
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
	s := &mainState{dshClient: client, market: store}
	owner := s.captureMarketOwner(store)
	op, _ := s.beginMarketAttemptForOwner("", nil, false, &owner)
	if op == nil || op.client != client || op.store != store {
		t.Fatal("current rendered action did not retain its Host")
	}
	op.cancel()
}
