package browserpreview

import (
	"context"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/localservers"
)

type localDiscovery struct {
	cancel    context.CancelFunc
	workspace string
	servers   []localservers.Server
	problem   string
}

func (b *Controller) stopLocalDiscovery() {
	if b.discovery != nil {
		b.discovery.cancel()
		b.discovery = nil
	}
}

// Called on the UI thread. A hidden/closed panel or changed workspace retires
// its request, so late results never update another pane or change its URL.
func (b *Controller) applyLocalServers(ctx context.Context, request *localDiscovery, found []localservers.Server) bool {
	if ctx.Err() != nil || b.discovery != request {
		return false
	}
	scope, err := b.browserWorkspace()
	if err != nil || scope.key != request.workspace {
		return false
	}
	request.servers = found
	return true
}

func (b *Controller) resumeLocalDiscovery(t *Tab) {
	scope, err := b.browserWorkspace()
	if err != nil || t.source != nil || t.Document != nil || t.HTML != nil || t.Media != nil {
		b.stopLocalDiscovery()
		return
	}
	if b.discovery != nil && b.discovery.workspace == scope.key {
		return
	}
	b.stopLocalDiscovery()
	if b.parent == nil || b.parent.IsDestroyed() {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	request := &localDiscovery{cancel: cancel, workspace: scope.key}
	b.discovery = request
	parent := b.parent
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			result := localservers.DiscoverStatus(ctx)
			if ctx.Err() != nil {
				return
			}
			applied := make(chan struct{})
			parent.Update(func() {
				defer close(applied)
				if !b.applyLocalServers(ctx, request, result.Servers) {
					cancel()
					if b.discovery == request {
						b.discovery = nil
					}
				} else {
					request.problem = result.Error
				}
			})
			select {
			case <-ctx.Done():
				return
			case <-applied:
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (b *Controller) localServerControls(c *ui.Context, t *Tab) {
	// The fixed-height row keeps native child bounds stable as servers appear.
	ui.Row(c).Height(30).Gap(6).AlignItems(ui.Center).Children(func() {
		request := b.discovery
		scope, err := b.browserWorkspace()
		if request == nil || err != nil || scope.key != request.workspace {
			return
		}
		if request.problem != "" {
			ui.Text(c, "本机服务探测异常").Tooltip(request.problem + "；每 3 秒自动重试，已发现服务仍可打开。")
		}
		if len(request.servers) == 0 {
			return
		}
		ui.Text(c, "发现本机服务").Tooltip("每 3 秒探测本机固定开发端口；仅 TCP 连通性，不保证 HTTP 可用。不会启动服务。")
		ui.ScrollHorizontal(c).Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Gap(4).Children(func() {
				for _, server := range request.servers {
					if ui.Button(c, server.URL).Clicked() && b.current() == t && b.discovery == request {
						// Navigate resolves current workspace ownership at the actual click.
						b.Navigate(server.URL)
					}
				}
			})
		})
	})
}
