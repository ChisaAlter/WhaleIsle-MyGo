package market

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	desktopmarket "whaleisle/internal/desktop/market"
)

type versionService interface {
	LatestVersion(context.Context, desktopmarket.CatalogItem) (string, error)
}
type UpdateRow struct {
	ID, Name, Current, Latest, Reason string
	Known, Available                  bool
}
type Updates struct {
	Rows      []UpdateRow
	Loading   bool
	CheckedAt time.Time
	Err       string
}
type updateState struct {
	Updates
	request, runtime, revision, inventory uint64
	cancel                                context.CancelFunc
}

var npmRecordedVersion = regexp.MustCompile(`^[0-9A-Za-z.*~^<>=| +_-]+$`)

func npmInstalledSource(row Entry) bool {
	if row.Source == row.Label {
		return true
	}
	suffix, ok := strings.CutPrefix(row.Source, row.Label+"@")
	return ok && suffix != "" && npmRecordedVersion.MatchString(suffix) && !strings.Contains(suffix, "npm:")
}
func (s *Store) invalidateUpdatesLocked() {
	if s.updates.cancel != nil {
		s.updates.cancel()
	}
	s.updates.request++
	s.updates.Updates = Updates{}
}
func (s *Store) updatesCurrentLocked() bool {
	u := &s.updates
	return s.client != nil && !s.loading && s.err == nil && !s.discovery.loading && u.runtime == s.generation && u.revision == s.discovery.revision && u.inventory == s.refreshID
}
func (s *Store) Updates() Updates {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.updatesCurrentLocked() {
		return Updates{}
	}
	out := s.updates.Updates
	out.Rows = append([]UpdateRow(nil), out.Rows...)
	return out
}

// CheckUpdates performs metadata-only reads; it never queues an installation.
func (s *Store) CheckUpdates() {
	s.mu.Lock()
	if s.client == nil || s.loading || s.err != nil || s.discovery.loading {
		s.mu.Unlock()
		return
	}
	if s.updates.cancel != nil {
		s.updates.cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	request := s.updates.request + 1
	s.updates = updateState{Updates: Updates{Loading: true}, request: request, runtime: s.generation, revision: s.discovery.revision, inventory: s.refreshID, cancel: cancel}
	rows := append([]Entry(nil), s.rows...)
	items := append([]desktopmarket.CatalogItem(nil), s.discovery.catalog.Items...)
	service, ok := s.discovery.service.(versionService)
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		out := Updates{CheckedAt: time.Now()}
		for _, row := range rows {
			if !row.Installed || !row.IdentityKnown {
				continue
			}
			u := UpdateRow{Name: row.Label, Current: row.Version, Reason: "No verified npm update source"}
			var matches []desktopmarket.CatalogItem
			for _, item := range items {
				if InstalledName(item, []Entry{row}) != "" {
					matches = append(matches, item)
				}
			}
			if len(matches) == 1 {
				item := matches[0]
				u.ID = item.ID
				if ok && row.Removable && npmInstalledSource(row) && item.PackageName == row.Label && item.InstallSpec == item.PackageName && desktopmarket.ValidInstallItem(item) {
					version, err := service.LatestVersion(ctx, item)
					if err != nil {
						u.Reason = "Version check failed; retry"
					} else {
						u.Latest = version
						comparison, known := desktopmarket.CompareStableVersions(row.Version, version)
						u.Known = known
						u.Available = known && comparison < 0
						if known {
							u.Reason = ""
						} else {
							u.Reason = "Version comparison unavailable"
						}
					}
				}
			}
			out.Rows = append(out.Rows, u)
		}
		if ctx.Err() != nil {
			out.Err = "Update check interrupted; check again"
		}
		s.mu.Lock()
		if s.updates.request != request || !s.updatesCurrentLocked() {
			s.mu.Unlock()
			return
		}
		s.updates.Updates = out
		s.mu.Unlock()
		s.fire()
	}()
}
func (s *Store) CaptureUpdateTarget(id string) (InstallTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.updatesCurrentLocked() || s.updates.Loading || s.updates.Err != "" || time.Since(s.updates.CheckedAt) > 30*time.Minute {
		return InstallTarget{}, errors.New("Check updates again before selecting a version")
	}
	for _, u := range s.updates.Rows {
		if u.ID == id && u.Available {
			return InstallTarget{ID: id, Spec: u.Name + "@" + u.Latest, Revision: s.discovery.revision, Runtime: s.generation, UpdateVersion: u.Latest, CurrentVersion: u.Current, Inventory: s.refreshID, UpdateCheck: s.updates.request}, nil
		}
	}
	return InstallTarget{}, errors.New("No verified newer version available")
}
func (s *Store) validateUpdateLocked(t InstallTarget) error {
	if !s.updatesCurrentLocked() || s.updates.Loading || s.updates.Err != "" || t.UpdateCheck != s.updates.request || t.Inventory != s.refreshID || time.Since(s.updates.CheckedAt) > 30*time.Minute {
		return errors.New("Update check expired; check again")
	}
	for _, u := range s.updates.Rows {
		if u.ID == t.ID && u.Available && u.Current == t.CurrentVersion && u.Latest == t.UpdateVersion && t.Spec == u.Name+"@"+u.Latest {
			for _, item := range s.discovery.catalog.Items {
				if item.ID == t.ID && item.PackageName == u.Name && item.InstallSpec == u.Name && desktopmarket.ValidInstallItem(item) {
					return nil
				}
			}
		}
	}
	return errors.New("Update target changed; check again")
}
