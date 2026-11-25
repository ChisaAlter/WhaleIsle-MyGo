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
	s.mainRouteDirty.Store(true)
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
	if route != routeSettings {
		s.settingsEmbeddedSection = ""
	}
}

type mainTopCaptionMode uint8

const (
	// topCaptionNone reserves 0 DIP because the route paints its own full
	// native caption band (Workspace shell or Windows routeWebView).
	topCaptionNone mainTopCaptionMode = iota
	// topCaptionWindowsStandaloneStrip reserves a borderless, gitBadge-free
	// transparent draggable strip of chrome.EffectiveTitleBarHeight() (12 DIP)
	// for Windows standalone routes (routeBrowser, routeIM, routeRemote,
	// routeUsage, and offline maintenance). Subpages align their headers with
	// c.TitleBar().Left/Right at y=12..44 without a hairline slicing "- □ ×".
	topCaptionWindowsStandaloneStrip
	// topCaptionNonWindowsLocked keeps the macOS/Linux titlebar strip with
	// actions for locked non-conversation routes.
	topCaptionNonWindowsLocked
	// topCaptionNonWindowsDragStrip keeps the macOS/Linux draggable caption
	// strip for conversation/settings routes.
	topCaptionNonWindowsDragStrip
)

func mainTopCaptionModeForOS(goos string, ready bool, route mainRoute) mainTopCaptionMode {
	standalone := route == routeUsage || route == routeIM || route == routeRemote || route == routeBrowser
	if goos == "windows" {
		if route == routeWebView {
			return topCaptionNone
		}
		if standalone {
			return topCaptionWindowsStandaloneStrip
		}
		if !ready && route.isMaintenance() {
			return topCaptionWindowsStandaloneStrip
		}
		return topCaptionNone
	}
	locked := route != routeConversation && route != routeSettings
	if locked {
		return topCaptionNonWindowsLocked
	}
	return topCaptionNonWindowsDragStrip
}

// syncMainRouteResources is also called after frame callbacks, so locked
// platform/startup transitions have the same disposal path as user navigation.
func (s *mainState) syncMainRouteResources() {
	if !s.mainRouteDirty.Load() {
		return
	}
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
	s.mainRouteDirty.Store(false)
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
		s.noteRouteUse(route)
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
