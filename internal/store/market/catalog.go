package market

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/state"
)

type CatalogService interface {
	List(context.Context, bool, string) (desktopmarket.Catalog, error)
	Details(context.Context, desktopmarket.CatalogItem, bool) (desktopmarket.Details, error)
}
type discoveryState struct {
	service                                CatalogService
	catalog                                desktopmarket.Catalog
	loaded, loading, detailsLoading        bool
	nativeCompatibleOnly                   bool
	err, detailsErr                        *state.Fault
	query, category, tab, selected, locale string
	sort, period                           string
	details                                desktopmarket.Details
	revision                               uint64
	page                                   int
	expanded                               bool
}

// Discovery is the published catalog projection. Its slices and the
// Selected item are shared with the store and must not be edited.
type Discovery struct {
	Catalog                                       desktopmarket.Catalog
	Rows                                          []desktopmarket.CatalogItem
	Total, Page                                   int
	Query, Category, Tab, SelectedID              string
	Err, DetailsErr                               *state.Fault
	Sort, Period                                  string
	Loading, DetailsLoading, NativeCompatibleOnly bool
	Selected                                      *desktopmarket.CatalogItem
	Details                                       desktopmarket.Details
	Revision                                      uint64
	RuntimeReady                                  bool
}

// InstallTarget is an immutable receipt for one displayed catalog and runtime.
type InstallTarget struct {
	ID, Spec                      string
	Revision, Runtime             uint64
	UpdateVersion, CurrentVersion string
	Inventory, UpdateCheck        uint64
}

func newDiscoveryState() discoveryState {
	dir, _ := os.UserCacheDir()
	path := ""
	if dir != "" {
		path = filepath.Join(dir, "whaleisle", "marketplace-cache.json")
	}
	return discoveryState{service: desktopmarket.NewService(path), category: "all", tab: "discover", sort: "catalog", period: "all", nativeCompatibleOnly: true}
}
func (s *Store) SetCatalogService(service CatalogService) {
	s.mu.Lock()
	s.invalidateUpdatesLocked()
	// The replaced discovery state's reads must not land in the new one.
	s.lists.Retire()
	s.details.Retire()
	s.discovery = newDiscoveryState()
	s.discovery.service = service
	s.publishLocked()
	s.mu.Unlock()
}
func (s *Store) SetNotify(notify func()) { s.mu.Lock(); s.notify = notify; s.mu.Unlock() }
func (s *Store) EnsureCatalog(locale string) {
	s.mu.Lock()
	needed := !s.discovery.loaded && !s.discovery.loading || s.discovery.locale != locale
	s.mu.Unlock()
	if needed {
		s.RefreshCatalog(false, locale)
	}
}
func (s *Store) RefreshCatalog(force bool, locale string) {
	s.mu.Lock()
	d := &s.discovery
	s.invalidateUpdatesLocked()
	// Renewing the list lease cancels the previous read.
	s.listL = s.lists.Renew(context.Background())
	lease := s.listL
	service := d.service
	d.loading = true
	d.locale = locale
	d.err = nil
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
	go func() {
		cat, e := service.List(lease.Context(), force, locale)
		s.mu.Lock()
		d := &s.discovery
		if !lease.Valid() {
			s.mu.Unlock()
			return
		}
		d.loading = false
		d.loaded = true
		if e != nil {
			d.err = state.FaultOf(e)
		} else {
			d.catalog = cat
			d.revision++
			d.page = 0
			d.expanded = false
			d.details = desktopmarket.Details{}
			d.detailsErr = nil
			d.detailsLoading = false
			s.details.Retire()
			d.selected = ""
		}
		s.publishLocked()
		s.mu.Unlock()
		s.fire()
		if e == nil {
			s.checkUpdatesWhenReady()
		}
	}()
}

// Discovery returns the published discovery projection. It is rebuilt
// once per change instead of filtered and copied on every read; Catalog,
// Rows and Selected share the immutable catalog items.
func (s *Store) Discovery() Discovery {
	if v := s.disc.Load(); v != nil {
		return v.Value
	}
	s.mu.Lock()
	if s.disc.Load() == nil {
		s.publishLocked()
	}
	s.mu.Unlock()
	return s.disc.Load().Value
}

// DiscoveryVersion is Discovery with its publication revision.
func (s *Store) DiscoveryVersion() *state.Version[Discovery] {
	s.Discovery()
	return s.disc.Load()
}

// buildDiscoveryLocked filters, sorts and pages the catalog. The listing
// period is measured against the time of the change that published it.
func (s *Store) buildDiscoveryLocked() Discovery {
	d := &s.discovery
	out := Discovery{Catalog: d.catalog, Query: d.query, Category: d.category, Tab: d.tab, Sort: d.sort, Period: d.period, SelectedID: d.selected, Err: d.err, DetailsErr: d.detailsErr, Loading: d.loading, DetailsLoading: d.detailsLoading, NativeCompatibleOnly: d.nativeCompatibleOnly, Details: d.details, Revision: d.revision, RuntimeReady: s.client != nil, Page: d.page}
	q := strings.ToLower(strings.TrimSpace(d.query))
	for i := range d.catalog.Items {
		item := d.catalog.Items[i]
		if item.ID == d.selected {
			out.Selected = &d.catalog.Items[i]
		}
		if d.tab == "favorites" && !s.personal.data.Favorites[item.ID] {
			continue
		}
		if d.category != "all" && d.category != "" && item.Category != d.category {
			continue
		}
		if d.nativeCompatibleOnly && !item.AllowsNativeMode() {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(item.ID+" "+item.Description+" "+item.PackageName), q) {
			continue
		}
		out.Rows = append(out.Rows, item)
	}
	now := time.Now
	if s.now != nil {
		now = s.now
	}
	out.Rows = filterCatalogListings(out.Rows, d.sort, d.period, now())
	out.Total = len(out.Rows)
	start := d.page * 60
	if d.expanded {
		start = 0
	}
	if start >= len(out.Rows) {
		start = 0
		out.Page = 0
	}
	end := start + 60
	if d.expanded {
		end = (d.page + 1) * 60
	}
	if end > len(out.Rows) {
		end = len(out.Rows)
	}
	out.Rows = out.Rows[start:end:end]
	return out
}

// SetNativeCompatibleOnly toggles filtering out Web UI-only catalog entries.
func (s *Store) SetNativeCompatibleOnly(enabled bool) {
	s.mu.Lock()
	s.discovery.nativeCompatibleOnly = enabled
	s.discovery.page = 0
	s.discovery.expanded = false
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SetQuery(q string) {
	s.mu.Lock()
	s.discovery.query = q
	s.discovery.page = 0
	s.discovery.expanded = false
	s.publishLocked()
	s.mu.Unlock()
}
func (s *Store) SetCategory(category string) {
	s.mu.Lock()
	s.discovery.category = category
	s.discovery.page = 0
	s.discovery.expanded = false
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SetTab(tab string) {
	s.mu.Lock()
	s.discovery.tab = tab
	s.discovery.page = 0
	s.discovery.expanded = false
	s.discovery.selected = ""
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SetPage(page int) {
	s.mu.Lock()
	if page < 0 {
		page = 0
	}
	s.discovery.page = page
	s.discovery.expanded = false
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}

// ShowMore appends the next catalog batch without replacing visible cards.
// SetPage remains the explicit page-selection API for existing callers.
func (s *Store) ShowMore() {
	s.mu.Lock()
	s.discovery.expanded = true
	s.discovery.page++
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) CloseDetails() {
	s.mu.Lock()
	d := &s.discovery
	s.details.Retire()
	d.selected = ""
	d.detailsLoading = false
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}

var detailTimeout = 15 * time.Second

func (s *Store) SelectDetails(id string, force bool) {
	s.mu.Lock()
	d := &s.discovery
	var item *desktopmarket.CatalogItem
	for _, i := range d.catalog.Items {
		if i.ID == id {
			x := i
			item = &x
			break
		}
	}
	if item == nil {
		s.mu.Unlock()
		return
	}
	// Renewing the details lease cancels the previous details read.
	s.detailL = s.details.Renew(context.Background())
	lease, revision := s.detailL, d.revision
	d.selected = id
	d.detailsLoading = true
	d.detailsErr = nil
	d.details = desktopmarket.Details{}
	service := d.service
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(lease.Context(), detailTimeout)
		defer cancel()
		details, e := service.Details(ctx, *item, force)
		s.mu.Lock()
		d := &s.discovery
		if !lease.Valid() || d.revision != revision || d.selected != id {
			s.mu.Unlock()
			return
		}
		d.detailsLoading = false
		if e != nil {
			d.detailsErr = state.FaultOf(e)
		} else {
			d.details = details
			if details.CapabilityKind != "" {
				for idx := range d.catalog.Items {
					if d.catalog.Items[idx].ID == id {
						// Published discovery views share the item slice;
						// apply the profile to a fresh copy.
						items := slices.Clone(d.catalog.Items)
						desktopmarket.ApplyCapabilityProfile(&items[idx], details.Capabilities)
						d.catalog.Items = items
						break
					}
				}
			}
		}
		s.publishLocked()
		s.mu.Unlock()
		s.fire()
	}()
}
func (s *Store) CaptureTarget(id string) (InstallTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := &s.discovery
	if s.client == nil {
		return InstallTarget{}, errors.New("Runtime is unavailable; catalog installation is disabled.")
	}
	if d.loading {
		return InstallTarget{}, errors.New("Wait for catalog refresh before installing.")
	}
	for _, i := range d.catalog.Items {
		if i.ID == id && desktopmarket.ValidInstallItem(i) {
			return InstallTarget{ID: i.ID, Spec: i.InstallSpec, Revision: d.revision, Runtime: s.runtimeL.Generation()}, nil
		}
	}
	return InstallTarget{}, errors.New("Catalog target is not installable.")
}
func (s *Store) ValidateTarget(t InstallTarget) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || s.runtimeL.Generation() != t.Runtime || s.discovery.revision != t.Revision || s.discovery.loading {
		return errors.New("Catalog or runtime changed; select the plugin again.")
	}
	if t.UpdateVersion != "" {
		return s.validateUpdateLocked(t)
	}
	for _, i := range s.discovery.catalog.Items {
		if i.ID == t.ID && i.InstallSpec == t.Spec && desktopmarket.ValidInstallItem(i) {
			return nil
		}
	}
	return errors.New("Catalog target is no longer available.")
}
func InstalledName(item desktopmarket.CatalogItem, rows []Entry) string {
	for _, r := range rows {
		if r.IdentityKnown && r.Installed && item.PackageName != "" && r.Label == item.PackageName {
			return r.Label
		}
	}
	id := desktopmarket.GitHubIdentity(item.InstallSpec)
	if id != "" {
		for _, r := range rows {
			if r.IdentityKnown && r.Installed && desktopmarket.GitHubIdentity(r.Source) == id {
				return r.Label
			}
		}
	}
	return ""
}
