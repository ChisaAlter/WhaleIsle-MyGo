package settings

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestMCPHeaderAddUsesNewFormActionRatherThanEmptyEditID(t *testing.T) {
	added, edited := 0, 0
	a := &MCPAccess{HasListRPC: true, OnAdd: func() { added++ }, OnEdit: func(string) { edited++ }}
	tt := pass4aTester(t, func(c *ui.Context) { mcpPane(c, Model{MCP: a}, c.Theme().Text, c.Theme().TextMuted) })
	if err := tt.Click("Add MCP server"); err != nil {
		t.Fatal(err)
	}
	if added != 1 || edited != 0 {
		t.Fatal("Add did not use its dedicated form callback", added, edited)
	}
}
