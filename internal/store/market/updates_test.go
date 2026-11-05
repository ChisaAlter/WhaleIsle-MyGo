package market

import (
	"context"
	"testing"
	"time"
	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/dsh"
)

type updateFixture struct {
	version string
	release chan struct{}
}

type automaticUpdateFixture struct {
	updateFixture
	checked chan struct{}
}

func (f automaticUpdateFixture) List(context.Context, bool, string) (desktopmarket.Catalog, error) {
	return desktopmarket.Catalog{Items: []desktopmarket.CatalogItem{{ID: "owner/demo", PackageName: "demo", InstallSpec: "demo"}}}, nil
}
func (f automaticUpdateFixture) LatestVersion(ctx context.Context, item desktopmarket.CatalogItem) (string, error) {
	f.checked <- struct{}{}
	return f.updateFixture.LatestVersion(ctx, item)
}

func TestAutomaticUpdatesWaitForCatalogAndInventoryInEitherOrder(t *testing.T) {
	for _, catalogFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "catalog-first", false: "inventory-first"}[catalogFirst], func(t *testing.T) {
			checked := make(chan struct{}, 4)
			s := NewWithPersonalPath("")
			client := &dsh.Client{}
			s.Attach(client, nil)
			s.SetCatalogService(automaticUpdateFixture{updateFixture{version: "1.1.0"}, checked})
			inventory := func() {
				s.finishRefresh(client, s.generation, s.refreshID, nil, []Entry{{Label: "demo", IdentityKnown: true, Installed: true, Removable: true, Source: "demo@1.0.0", Version: "1.0.0"}}, "", "", "")
			}
			catalog := func() {
				s.RefreshCatalog(false, "en")
				deadline := time.Now().Add(time.Second)
				for s.Discovery().Loading && time.Now().Before(deadline) {
					time.Sleep(time.Millisecond)
				}
				if s.Discovery().Loading {
					t.Fatal("catalog did not load")
				}
			}
			if catalogFirst {
				catalog()
			} else {
				inventory()
			}
			select {
			case <-checked:
				t.Fatal("checked updates before both reads completed")
			default:
			}
			if catalogFirst {
				inventory()
			} else {
				catalog()
			}
			select {
			case <-checked:
			case <-time.After(time.Second):
				t.Fatal("ready reads did not automatically check updates")
			}
			if u := awaitUpdates(t, s); len(u.Rows) != 1 || !u.Rows[0].Available {
				t.Fatal(u)
			}
			s.checkUpdatesWhenReady()
			select {
			case <-checked:
				t.Fatal("same completed reads checked twice")
			default:
			}
			s.Detach()
		})
	}
}

func (f updateFixture) List(context.Context, bool, string) (desktopmarket.Catalog, error) {
	return desktopmarket.Catalog{}, nil
}
func (f updateFixture) Details(context.Context, desktopmarket.CatalogItem, bool) (desktopmarket.Details, error) {
	panic("updates must not fetch README")
}
func (f updateFixture) LatestVersion(ctx context.Context, _ desktopmarket.CatalogItem) (string, error) {
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return f.version, nil
}
func updateStore(f updateFixture) *Store {
	s := NewWithPersonalPath("")
	s.Attach(&dsh.Client{}, nil)
	s.discovery.service = f
	s.discovery.revision = 1
	s.discovery.catalog.Items = []desktopmarket.CatalogItem{{ID: "owner/demo", PackageName: "demo", InstallSpec: "demo"}}
	s.rows = []Entry{{Label: "demo", IdentityKnown: true, Installed: true, Removable: true, Source: "demo@1.0.0", Version: "1.0.0"}}
	return s
}
func awaitUpdates(t *testing.T, s *Store) Updates {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		u := s.Updates()
		if !u.Loading {
			return u
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("check timed out")
	return Updates{}
}
func TestUpdatesPinVersionAndRejectStaleReceipts(t *testing.T) {
	s := updateStore(updateFixture{version: "1.1.0"})
	s.CheckUpdates()
	u := awaitUpdates(t, s)
	if len(u.Rows) != 1 || !u.Rows[0].Available || u.Rows[0].Current != "1.0.0" {
		t.Fatal(u)
	}
	target, err := s.CaptureUpdateTarget("owner/demo")
	if err != nil || target.Spec != "demo@1.1.0" {
		t.Fatal(target, err)
	}
	if err = s.ValidateTarget(target); err != nil {
		t.Fatal(err)
	}
	s.refreshID++
	if s.ValidateTarget(target) == nil {
		t.Fatal("stale inventory accepted")
	}
}
func TestUpdatesUnknownCannotBeSelected(t *testing.T) {
	s := updateStore(updateFixture{version: "dev"})
	s.CheckUpdates()
	u := awaitUpdates(t, s)
	if len(u.Rows) != 1 || u.Rows[0].Known || u.Rows[0].Available {
		t.Fatal(u)
	}
	if _, err := s.CaptureUpdateTarget("owner/demo"); err == nil {
		t.Fatal("unknown selected")
	}
}
func TestUpdatesDiscardReplacedRuntime(t *testing.T) {
	release := make(chan struct{})
	s := updateStore(updateFixture{version: "2.0.0", release: release})
	s.CheckUpdates()
	s.Detach()
	close(release)
	u := s.Updates()
	if u.Loading || len(u.Rows) != 0 {
		t.Fatal(u)
	}
}
