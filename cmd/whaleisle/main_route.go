package main

import (
	"github.com/egoist/mygo"
)

// mainRoute is the single selection for mutually exclusive main destinations.
// The legacy *Open fields are read-only projections for existing view models.
// File docks, terminal panels and transient confirmation dialogs are orthogonal.
type mainRoute uint8

const (
	routeConversation mainRoute = iota
	routeComponents
	routeImport
	routeStartupDiagnostics
	routeUpdatePreferences
	routeSettings
	routeMarket
	routePlugins
	routeSchedule
	routeAssistant
	routeUsage
	routeIM
	routeRemote
	routeBrowser
	routeWebView
)

type mainRouteToken struct {
	route mainRoute
	epoch uint64
}

func (s *mainState) captureMainRoute() mainRouteToken {
	s.mu.Lock()
	defer s.mu.Unlock()
	return mainRouteToken{s.mainRoute, s.mainRouteEpoch}
}

func (s *mainState) mainRouteCurrent(token mainRouteToken) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mainRouteCurrentLocked(token)
}

func (s *mainState) mainRouteCurrentLocked(token mainRouteToken) bool {
	return !s.quitting && s.mainRoute == token.route && s.mainRouteEpoch == token.epoch
}

// setMainRouteLocked commits navigation only. It performs no I/O or callbacks
// under the state lock; syncMainRouteResources releases the prior surface.
func (s *mainState) setMainRouteLocked(route mainRoute) {
	s.mainRoute = route
	s.mainRouteEpoch++
	s.maintenanceOpen = route.isMaintenance()
	s.settingsOpen = route == routeSettings
	s.marketOpen = route == routeMarket || route == routePlugins
	s.marketManager = route == routePlugins
	s.scheduleOpen = route == routeSchedule
	s.usageOpen = route == routeUsage
	s.imOpen = route == routeIM
	s.remoteOpen = route == routeRemote
	s.browserPreviewOpen = route == routeBrowser
	s.webviewOpen = route == routeWebView
	s.toolsMenuOpen = false
	if route != routeSettings {
		s.settingsEmbeddedSection = ""
	}
}

// syncMainRouteResources is also called after frame callbacks, so locked
// platform/startup transitions have the same disposal path as user navigation.
func (s *mainState) syncMainRouteResources() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		mygo.RunOnMain(s.applyMainRouteResources)
		return
	}
	s.applyMainRouteResources()
}

// Native surfaces and their controllers have one UI-thread owner. Boot and
// recovery callbacks may request navigation from a worker, but never dispose
// browser tabs concurrently with rendering them.
func (s *mainState) applyMainRouteResources() {
	s.mu.Lock()
	route, epoch := s.mainRoute, s.mainRouteEpoch
	changed := epoch != s.mainRouteApplied
	s.mainRouteApplied = epoch
	var retired retiredWebView
	if route != routeWebView {
		retired = s.detachWebViewLocked()
	}
	browser := s.browserController
	s.mu.Unlock()
	retired.close()
	if changed && s.mainRouteCurrent(mainRouteToken{route, epoch}) {
		if route != routeImport {
			s.cancelImportCatalogScan()
		}
		s.assistantService().SetOpen(route == routeAssistant)
		if browser != nil && route != routeBrowser {
			browser.Suspend()
		}
		retireStaleInlineVisualPages()
	}
}

func (s *mainState) navigateMainRoute(route mainRoute) {
	s.mu.Lock()
	s.setMainRouteLocked(route)
	win := s.win
	s.mu.Unlock()
	s.syncMainRouteResources()
	if win != nil {
		win.Update(func() {})
	}
}

func (route mainRoute) isMaintenance() bool {
	return route == routeComponents || route == routeImport || route == routeStartupDiagnostics || route == routeUpdatePreferences
}

func panelMainRoute(panel string) mainRoute {
	switch panel {
	case "market":
		return routeMarket
	case "plugins":
		return routePlugins
	case "schedule":
		return routeSchedule
	case "assistant":
		return routeAssistant
	default:
		return routeConversation
	}
}
