package main

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"whaleisle/internal/desktop/deeplink"
	"whaleisle/internal/desktop/platformview"
	"whaleisle/internal/harness"
)

func TestMainRoutePluginEntryReplacesExclusivePageAndPreservesDrafts(t *testing.T) {
	for _, prior := range []mainRoute{routeMarket, routeSchedule, routeSettings, routeAssistant} {
		s := &mainState{draft: "unsent", fileEditDraft: "unsaved file", filesPaneOpen: true, rightPaneTab: "files"}
		s.navigateMainRoute(prior)
		s.openMainWebViewPlugin("http://127.0.0.1:8080", "dsh-example")
		if s.mainRoute != routeWebView || !s.webviewOpen || s.marketOpen || s.scheduleOpen || s.settingsOpen || s.assistantService().Open() {
			t.Fatalf("route %d still masks the explicit plugin page: route=%d", prior, s.mainRoute)
		}
		if s.draft != "unsent" || s.fileEditDraft != "unsaved file" || !s.filesPaneOpen || s.rightPaneTab != "files" {
			t.Fatal("navigation discarded conversation or dock state")
		}
		if s.webviewHost != nil {
			t.Fatal("a route selection without a native parent allocated a WebView")
		}
	}
}

func TestMainRouteTokenRejectsLateAndReopenedDestination(t *testing.T) {
	s := &mainState{}
	s.navigateMainRoute(routeWebView)
	ticket := s.captureMainRoute()
	if !s.mainRouteCurrent(ticket) {
		t.Fatal("current route rejected")
	}
	s.navigateMainRoute(routeSettings)
	s.navigateMainRoute(routeWebView)
	if s.mainRouteCurrent(ticket) {
		t.Fatal("a late platform or WebView creation can hijack a reopened destination")
	}
}

func routeTestGateway(t *testing.T) (*platformview.Gateway, string) {
	t.Helper()
	gateway, err := platformview.New(platformview.Session{Origin: "https://platform.invalid", Token: "fixture"}, platformview.ClientMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(gateway.Close)
	entry, err := gateway.Entry("usage")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(entry)
	if err != nil {
		t.Fatal(err)
	}
	return gateway, u.Host
}

func TestMainRouteExitRevokesAccountLeaseOnce(t *testing.T) {
	gateway, address := routeTestGateway(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancels := 0
	lease := &accountPlatformLease{cancel: func() { cancels++; cancel() }, gateway: gateway}
	s := &mainState{}
	s.navigateMainRoute(routeWebView)
	s.accountReturn.platform = lease
	connection, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = connection.Close()
	s.navigateMainRoute(routeSettings)
	s.syncMainRouteResources()
	closeAccountPlatformLease(lease)
	if ctx.Err() == nil || cancels != 1 || s.accountReturn.platform != nil || s.webviewHost != nil {
		t.Fatalf("account lease survived navigation or closed twice: cancels=%d", cancels)
	}
	connection, err = net.DialTimeout("tcp", address, time.Second)
	if err == nil {
		_ = connection.Close()
		t.Fatal("account transport still accepts connections after route exit")
	}
}

func TestMainRouteCancelledAccountLeaseRejectsLateGateway(t *testing.T) {
	lease := &accountPlatformLease{cancel: func() {}}
	closeAccountPlatformLease(lease)
	gateway, address := routeTestGateway(t)
	if lease.attachGateway(gateway) {
		t.Fatal("cancelled platform operation accepted a late gateway")
	}
	connection, err := net.DialTimeout("tcp", address, time.Second)
	if err == nil {
		_ = connection.Close()
		t.Fatal("late gateway was retained after cancellation")
	}
}

func TestMainRouteTransientModalKeepsLeaseAndNavigationRetiresIt(t *testing.T) {
	s := &mainState{}
	s.navigateMainRoute(routeWebView)
	closed := false
	s.accountReturn.platform = &accountPlatformLease{cancel: func() { closed = true }}
	s.dirPickOpen = true
	s.syncSettingsNativeSurfaces()
	if closed || !s.exclusiveSurfaceMask().transientModalMasked() {
		t.Fatal("transient modal was treated as navigation")
	}
	s.navigateMainRoute(routeConversation)
	if !closed {
		t.Fatal("actual navigation retained the modal's old account lease")
	}
}

func TestMainRouteOpenDeeplinkKeepsCurrentDestination(t *testing.T) {
	s := &mainState{}
	s.navigateMainRoute(routeSchedule)
	ticket := s.captureMainRoute()
	s.applyDeeplink(deeplink.Target{Action: deeplink.ActionOpen})
	if !s.mainRouteCurrent(ticket) || !s.scheduleOpen {
		t.Fatal("raise-only link changed the selected destination")
	}
	s.applyDeeplink(deeplink.Target{Action: deeplink.ActionMarket})
	if s.mainRoute != routeMarket || s.scheduleOpen || !s.marketOpen {
		t.Fatal("market link left an older destination selected")
	}
}

func TestMainRouteBootNotificationsDoNotStealNativePage(t *testing.T) {
	for _, route := range []mainRoute{routeConversation, routeSettings, routeImport, routeStartupDiagnostics} {
		s := &mainState{}
		s.navigateMainRoute(route)
		ticket := s.captureMainRoute()
		s.refreshBoot(harness.Status{State: harness.Starting})
		s.refreshBoot(harness.Status{State: harness.Ready})
		s.refreshBoot(harness.Status{State: harness.Failed})
		if !s.mainRouteCurrent(ticket) {
			t.Fatalf("Host status displaced route %v", route)
		}
	}
}
