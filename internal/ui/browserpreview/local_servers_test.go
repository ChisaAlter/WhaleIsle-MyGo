package browserpreview

import (
	"context"
	"testing"
	"whaleisle/internal/desktop/localservers"
)

func TestDiscoveryRejectsReplacedWorkspaceAndKeepsDraft(t *testing.T) {
	owner := WorkspaceOwner{Authority: "desktop", Cwd: "/one"}
	b := NewController(nil)
	b.SetWorkspaceOwner(func() WorkspaceOwner { return owner })
	scope, _ := b.browserWorkspace()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := &localDiscovery{cancel: cancel, workspace: scope.key}
	b.discovery = request
	b.Tabs = []*Tab{{Draft: "https://existing.example/path", workspace: scope}}
	found := []localservers.Server{{URL: "http://127.0.0.1:3000", Port: 3000}}
	if !b.applyLocalServers(ctx, request, found) || len(request.servers) != 1 {
		t.Fatal("current discovery rejected")
	}
	if b.current().Draft != "https://existing.example/path" {
		t.Fatal("discovery changed current URL")
	}
	owner.Cwd = "/two"
	if b.applyLocalServers(ctx, request, found) {
		t.Fatal("accepted previous workspace discovery")
	}
	b.stopLocalDiscovery()
	if b.discovery != nil || ctx.Err() == nil {
		t.Fatal("hide/close must cancel and clear discovery")
	}
	if b.applyLocalServers(ctx, request, found) {
		t.Fatal("accepted stopped discovery")
	}
}
