package mcpconfig

import (
	"encoding/json"
	"errors"
	"strings"

	"whaleisle/internal/dsh/api"
)

// TestDiagnostic is the last explicit connection test for one unchanged spec.
// Registered tools from list and raw test tool names are deliberately separate.
type TestDiagnostic struct {
	Result api.McpServerTestResult
	Err    string
	spec   string
}

func (c *Controller) SetQuery(query string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Query = query
}
func (c *Controller) SetEnabledFilter(filter string) {
	if filter != "enabled" && filter != "disabled" {
		filter = "all"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.EnabledFilter = filter
}

// Inspect opens only the already-loaded snapshot; it never starts a test or RPC.
func (c *Controller) Inspect(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.ExpandedID == id {
		c.state.ExpandedID = ""
		return
	}
	for _, row := range c.state.Rows {
		if row.Id == id {
			c.state.ExpandedID = id
			return
		}
	}
}

func ServerName(row api.McpServerSnapshotServersItem) string {
	var spec struct {
		Name string `json:"serverName"`
	}
	_ = json.Unmarshal(row.Spec, &spec)
	if spec.Name != "" {
		return spec.Name
	}
	return row.Id
}

// CatalogRows mirrors upstream searchableValues: identity, name and command/URL.
// Do not search the entire spec: that would index private env/header values.
func (s Snapshot) CatalogRows() []api.McpServerSnapshotServersItem {
	query := strings.ToLower(strings.TrimSpace(s.Query))
	rows := make([]api.McpServerSnapshotServersItem, 0, len(s.Rows))
	for _, row := range s.Rows {
		if (s.EnabledFilter == "enabled" && !row.Enabled) || (s.EnabledFilter == "disabled" && row.Enabled) {
			continue
		}
		var spec struct {
			Name      string `json:"serverName"`
			Transport string `json:"transport"`
			Command   string `json:"command"`
			URL       string `json:"url"`
		}
		_ = json.Unmarshal(row.Spec, &spec)
		endpoint := spec.Command
		if spec.Transport == "streamable-http" {
			endpoint = spec.URL
		}
		matched := query == ""
		for _, value := range []string{row.Id, spec.Name, endpoint} {
			if strings.Contains(strings.ToLower(value), query) {
				matched = true
				break
			}
		}
		if matched {
			rows = append(rows, row)
		}
	}
	return rows
}

func cloneDiagnostics(values map[string]TestDiagnostic) map[string]TestDiagnostic {
	if values == nil {
		return nil
	}
	out := make(map[string]TestDiagnostic, len(values))
	for id, value := range values {
		value.Result.ToolNames = append([]string(nil), value.Result.ToolNames...)
		out[id] = value
	}
	return out
}

// Reconcile after a successful list: a test belongs to exactly the tested spec.
func (c *Controller) reconcileDiagnostics() {
	specs := make(map[string]string, len(c.state.Rows))
	for _, row := range c.state.Rows {
		specs[row.Id] = string(row.Spec)
	}
	for id, result := range c.state.Tests {
		if spec, ok := specs[id]; !ok || spec != result.spec {
			delete(c.state.Tests, id)
		}
	}
	if _, ok := specs[c.state.ExpandedID]; !ok {
		c.state.ExpandedID = ""
	}
}

func (c *Controller) recordTest(id string, raw json.RawMessage, callErr error) error {
	diagnostic := TestDiagnostic{}
	for _, row := range c.state.Rows {
		if row.Id == id {
			diagnostic.spec = string(row.Spec)
			break
		}
	}
	if callErr == nil {
		callErr = json.Unmarshal(raw, &diagnostic.Result)
	}
	if callErr == nil && !diagnostic.Result.Ok {
		callErr = errors.New("MCP test did not return the required success acknowledgement")
	}
	if callErr != nil {
		diagnostic.Err = callErr.Error()
		diagnostic.Result = api.McpServerTestResult{}
	}
	if c.state.Tests == nil {
		c.state.Tests = make(map[string]TestDiagnostic)
	}
	c.state.Tests[id] = diagnostic
	return callErr
}
