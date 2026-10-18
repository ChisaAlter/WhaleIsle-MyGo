package market

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	desktopmarket "whaleisle/internal/desktop/market"
)

type CatalogService interface {
	List(context.Context, bool, string) (desktopmarket.Catalog, error)
	Details(context.Context, desktopmarket.CatalogItem, bool) (desktopmarket.Details, error)
}
type discoveryState struct {
	service                                                 CatalogService
	catalog                                                 desktopmarket.Catalog
	loaded, loading, detailsLoading                         bool
	err, detailsErr, query, category, tab, selected, locale string
	sort, period                                            string
	details                                                 desktopmarket.Details
	revision, requestID, detailID                           uint64
	cancel, detailCancel                                    context.CancelFunc
	page                                                    int
}
type Discovery struct {
	Catalog                                           desktopmarket.Catalog
	Rows                                              []desktopmarket.CatalogItem
	Total, Page                                       int
	Query, Category, Tab, SelectedID, Err, DetailsErr string
	Sort, Period                                      string
	Loading, DetailsLoading                           bool
	Selected                                          *desktopmarket.CatalogItem
	Details                                           desktopmarket.Details
	Revision                                          uint64
	RuntimeReady                                      bool
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
	return discoveryState{service: desktopmarket.NewService(path), category: "all", tab: "discover", sort: "catalog", period: "all"}
}
func (s *Store) SetCatalogService(service CatalogService) {
	s.mu.Lock()
	s.invalidateUpdatesLocked()
	s.discovery = newDiscoveryState()
	s.discovery.service = service
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
	if d.cancel != nil {
		d.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.requestID++
	request := d.requestID
	service := d.service
	d.loading = true
	d.locale = locale
	d.err = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		cat, e := service.List(ctx, force, locale)
		s.mu.Lock()
		d := &s.discovery
		if d.requestID != request {
			s.mu.Unlock()
			return
		}
		d.loading = false
		d.loaded = true
		if e != nil {
			d.err = e.Error()
		} else {
			d.catalog = cat
			d.revision++
			d.page = 0
			d.details = desktopmarket.Details{}
			d.detailsErr = ""
			d.detailsLoading = false
			d.detailID++
			if d.detailCancel != nil {
				d.detailCancel()
			}
			d.selected = ""
		}
		s.mu.Unlock()
		s.fire()
	}()
}
func (s *Store) Discovery() Discovery {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := &s.discovery
	out := Discovery{Catalog: d.catalog, Query: d.query, Category: d.category, Tab: d.tab, Sort: d.sort, Period: d.period, SelectedID: d.selected, Err: d.err, DetailsErr: d.detailsErr, Loading: d.loading, DetailsLoading: d.detailsLoading, Details: d.details, Revision: d.revision, RuntimeReady: s.client != nil, Page: d.page}
	out.Catalog.Items = append([]desktopmarket.CatalogItem(nil), d.catalog.Items...)
	out.Catalog.Categories = append([]desktopmarket.Category(nil), d.catalog.Categories...)
	out.Details.Requirements = append([]string(nil), d.details.Requirements...)
	q := strings.ToLower(strings.TrimSpace(d.query))
	for _, item := range d.catalog.Items {
		if item.ID == d.selected {
			x := item
			out.Selected = &x
		}
		if d.tab == "favorites" && !s.personal.data.Favorites[item.ID] {
			continue
		}
		if d.category != "all" && d.category != "" && item.Category != d.category {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(item.ID+" "+item.Description+" "+item.PackageName), q) {
			continue
		}
		out.Rows = append(out.Rows, item)
	}
	out.Rows = filterCatalogListings(out.Rows, d.sort, d.period, time.Now())
	out.Total = len(out.Rows)
	start := d.page * 60
	if start >= len(out.Rows) {
		start = 0
		out.Page = 0
	}
	end := start + 60
	if end > len(out.Rows) {
		end = len(out.Rows)
	}
	out.Rows = out.Rows[start:end]
	return out
}
func (s *Store) SetQuery(q string) {
	s.mu.Lock()
	s.discovery.query = q
	s.discovery.page = 0
	s.mu.Unlock()
}
func (s *Store) SetCategory(category string) {
	s.mu.Lock()
	s.discovery.category = category
	s.discovery.page = 0
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SetTab(tab string) {
	s.mu.Lock()
	s.discovery.tab = tab
	s.discovery.page = 0
	s.discovery.selected = ""
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SetPage(page int) {
	s.mu.Lock()
	if page < 0 {
		page = 0
	}
	s.discovery.page = page
	s.mu.Unlock()
	s.fire()
}
func (s *Store) CloseDetails() {
	s.mu.Lock()
	d := &s.discovery
	if d.detailCancel != nil {
		d.detailCancel()
	}
	d.detailID++
	d.selected = ""
	d.detailsLoading = false
	s.mu.Unlock()
	s.fire()
}
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
	if d.detailCancel != nil {
		d.detailCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.detailCancel = cancel
	d.detailID++
	request, revision := d.detailID, d.revision
	d.selected = id
	d.detailsLoading = true
	d.detailsErr = ""
	d.details = desktopmarket.Details{}
	service := d.service
	s.mu.Unlock()
	s.fire()
	go func() {
		details, e := service.Details(ctx, *item, force)
		s.mu.Lock()
		d := &s.discovery
		if d.detailID != request || d.revision != revision || d.selected != id {
			s.mu.Unlock()
			return
		}
		d.detailsLoading = false
		if e != nil {
			d.detailsErr = e.Error()
		} else {
			d.details = details
		}
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
			return InstallTarget{ID: i.ID, Spec: i.InstallSpec, Revision: d.revision, Runtime: s.generation}, nil
		}
	}
	return InstallTarget{}, errors.New("Catalog target is not installable.")
}
func (s *Store) ValidateTarget(t InstallTarget) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || s.generation != t.Runtime || s.discovery.revision != t.Revision || s.discovery.loading {
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
