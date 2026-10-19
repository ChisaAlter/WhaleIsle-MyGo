package mcpconfig

import (
	"context"
	"strings"

	"whaleisle/internal/dsh/api"
)

// RefreshConnections is the explicit Refresh action from the original MCP page:
// retry failed managed servers, then reload the catalog. Background health and
// initial reads continue to use Refresh and never reconnect a server.
func (c *Controller) RefreshConnections(ctx context.Context, client Caller) {
	c.mu.Lock()
	if !c.begin(client) {
		c.mu.Unlock()
		return
	}
	var failed []api.McpServerSnapshotServersItem
	for _, row := range c.state.Rows {
		if row.Origin == "managed" && row.Connection != nil && row.Connection.Health == "failed" {
			failed = append(failed, row)
		}
	}
	c.mu.Unlock()

	var failures []string
	for _, row := range failed {
		var result api.RetryResult
		if err := client.Call(ctx, "mcpServers/retry", map[string]any{"request": api.McpServerIdRequest{Id: row.Id}}, &result); err != nil {
			failures = append(failures, ServerName(row)+": "+err.Error())
		}
	}
	snapshot, err := readRows(ctx, client)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Busy = false
	if err != nil {
		failures = append(failures, "Refresh failed: "+err.Error())
	} else {
		c.state.Rows = snapshot.Servers
		c.reconcileDiagnostics()
		c.state.Ready = true
		c.state.HealthRefreshFailed = false
	}
	c.state.Err = strings.Join(failures, "; ")
}
