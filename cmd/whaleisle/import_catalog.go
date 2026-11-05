package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"os"
	"path/filepath"
	"strings"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

type importCatalogScan struct {
	request uint64
	busy    bool
	cancel  context.CancelFunc
	owner   *mygo.Window
}

func (s *mainState) rescanImportCatalog(win *mygo.Window, source string) {
	s.requestImportCatalog(win, source, "")
}

// Each worker belongs to the source selected for this request. A later scan or
// cancellation retires even a result already queued on the native UI thread.
func (s *mainState) requestImportCatalog(win *mygo.Window, source, extraRoot string) {
	s.mu.Lock()
	if s.launcherImportBusy || s.quitting {
		s.mu.Unlock()
		return
	}
	var previous *importer.Catalog
	if current := s.launcherImportCatalog; current != nil {
		copy := current.Snapshot()
		previous = &copy
	}
	if s.launcherImportScan.cancel != nil {
		s.launcherImportScan.cancel()
	}
	s.launcherImportScan.request++
	id := s.launcherImportScan.request
	ctx, cancel := context.WithCancel(context.Background())
	s.launcherImportScan.cancel, s.launcherImportScan.busy = cancel, true
	s.launcherImportScan.owner = win
	s.mu.Unlock()
	notify := s.launcherSurfaceInvalidator(win)
	notify()
	go func() {
		var cat importer.Catalog
		var err error
		if previous != nil {
			cat, err = importer.RescanCatalogContext(ctx, *previous, source)
		} else {
			src := strings.TrimSpace(source)
			if src == "" {
				src = importer.OfficialDshHome()
				if st, statErr := os.Stat(src); src == "" || statErr != nil || !st.IsDir() {
					src = filepath.Join(importer.ElectronUserDataDir(), "dsh-home")
				}
			}
			cat, err = importer.BuildCatalogContext(ctx, src, paths.DshHome())
		}
		if err == nil && extraRoot != "" && ctx.Err() == nil {
			importer.AppendSkillDirectory(&cat, extraRoot)
		}
		if ctx.Err() != nil {
			cancel()
			return
		}
		apply := func() {
			defer cancel()
			s.mu.Lock()
			if s.quitting || s.launcherImportBusy || s.launcherImportScan.request != id || ctx.Err() != nil {
				s.mu.Unlock()
				return
			}
			s.launcherImportScan.busy, s.launcherImportScan.cancel = false, nil
			s.launcherImportScan.owner = nil
			if err != nil {
				s.launcherImport = err.Error()
			} else {
				if current := s.launcherImportCatalog; previous != nil && current != nil {
					importer.MergeCatalogSelections(&cat, current.Snapshot())
				}
				s.launcherImportCatalog = &cat
				s.launcherImport = fmt.Sprintf(i18n.T(i18n.KeyLauncherImportSummary), cat.SourceHome, cat.DestHome, len(cat.Sessions), len(cat.Skills), len(cat.Plugins), len(cat.MCP), len(cat.Settings), len(cat.Presets))
				s.launcherImportCategories = map[string]bool{}
				for category, count := range cat.CategoryCounts() {
					s.launcherImportCategories[category] = count[0] > 0
				}
				if previous == nil {
					s.launcherImportAttachments = false
					for _, row := range cat.Sessions {
						if cat.HasAttachments && !row.Unsupported {
							s.launcherImportAttachments = true
							break
						}
					}
				}
			}
			s.mu.Unlock()
			notify()
		}
		if win != nil && !win.IsDestroyed() {
			win.Update(apply)
		} else if win == nil {
			apply()
		} else {
			s.cancelImportCatalogScanFor(win, id)
		}
	}()
}

func (s *mainState) cancelImportCatalogScan() {
	s.mu.Lock()
	s.launcherImportScan.request++
	cancel := s.launcherImportScan.cancel
	s.launcherImportScan.cancel, s.launcherImportScan.busy = nil, false
	s.launcherImportScan.owner = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Retire only the scan owned by this destroyed origin, never a newer request.
func (s *mainState) cancelImportCatalogScanFor(owner *mygo.Window, id uint64) {
	s.mu.Lock()
	if s.launcherImportScan.owner != owner || s.launcherImportScan.request != id {
		s.mu.Unlock()
		return
	}
	s.launcherImportScan.request++
	cancel := s.launcherImportScan.cancel
	s.launcherImportScan.cancel, s.launcherImportScan.busy = nil, false
	s.launcherImportScan.owner = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *mainState) cancelImportCatalogScanOwner(owner *mygo.Window) {
	if owner == nil {
		return
	}
	s.mu.Lock()
	id := s.launcherImportScan.request
	s.mu.Unlock()
	s.cancelImportCatalogScanFor(owner, id)
}
