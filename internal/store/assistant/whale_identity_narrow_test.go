package assistant

import (
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func TestWhaleIdentityNarrowWriteChecksAndReadbackPreserveOtherDrafts(t *testing.T) {
	for _, action := range []string{"name", "title"} {
		t.Run(action, func(t *testing.T) {
			var e WhaleSettingsEditor
			loadWhale(t, &e, "host-a")
			e.Edit("host-a", "name", "New Whale")
			e.Edit("host-a", "title", "Captain draft")
			e.Edit("host-a", "memory", "retained memory")
			e.Edit("host-a", "persona", "retained persona")
			q, cmd, ok := e.BeginWrite("host-a", action)
			if !ok {
				t.Fatal("narrow identity write denied")
			}
			current := whaleFixture()
			if action == "name" {
				current.UserTitle = "title changed elsewhere"
			} else {
				current.Name = "name changed elsewhere"
			}
			if err := cmd.Check(current); err != nil {
				t.Fatal("narrow identity CAS inspected an unrelated field", err)
			}
			conflict := current
			if action == "name" {
				conflict.Name = "changed same field"
			} else {
				conflict.UserTitle = "changed same field"
			}
			if cmd.Check(conflict) == nil {
				t.Fatal("same-field concurrent edit escaped preflight")
			}
			actual := current
			if action == "name" {
				actual.Name = "New Whale"
			} else {
				actual.UserTitle = "Captain draft"
			}
			if !e.FinishWrite(q, cmd, &actual, true, nil) {
				t.Fatal("readback not accepted")
			}
			got := e.Snapshot()
			if got.Name != "New Whale" || got.UserTitle != "Captain draft" || got.Memory != "retained memory" || got.PersonaText != "retained persona" || got.Error != "" {
				t.Fatal("narrow readback committed or erased other drafts", got)
			}
			replacement := pluginrpc.WhaleSettings{HomeDir: "/other", SessionID: "other"}
			if cmd.Check(replacement) == nil {
				t.Fatal("identity changed without invalidating command")
			}
		})
	}
}
