package chat

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/dsh/events"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func headerFixture() session.ViewState {
	return session.ViewState{SelectedID: "s", Messages: []session.Message{
		{ID: "call", Seq: 5, Kind: session.MessageKindTool, Raw: []byte(`{"type":"tool/call","data":{"callId":"c","name":"shell"}}`)},
		{ID: "result", Seq: 12, Kind: session.MessageKindTool, Raw: []byte(`{"type":"tool/result","data":{"message":{"toolCallId":"c"}}}`)},
	}, TrajectoryRequestHeaders: []session.TrajectoryRequestHeader{
		{Seq: 4, Header: session.RequestInspectionHeader{Tools: []events.RequestHeaderDataHeaderToolsItem{{Name: "shell", Description: "historical", Parameters: map[string]any{"type": "object"}}}}},
		{Seq: 8, Header: session.RequestInspectionHeader{Tools: []events.RequestHeaderDataHeaderToolsItem{{Name: "shell", Description: "newer", Parameters: map[string]any{"type": "string"}}}}},
	}}
}
func TestTrajectorySchemaUsesHistoricalCallHeader(t *testing.T) {
	snapshot := headerFixture()
	schema, header := trajectoryRequestPayloads(snapshot, "result")
	if !strings.Contains(schema, "historical") || strings.Contains(schema, "newer") || !strings.Contains(header, `"seq": 4`) {
		t.Fatalf("wrong historical schema %s %s", schema, header)
	}
	snapshot.Messages = snapshot.Messages[1:]
	if schema, _ := trajectoryRequestPayloads(snapshot, "result"); schema != "" {
		t.Fatal("result-only window guessed schema")
	}
	snapshot = headerFixture()
	snapshot.TrajectoryRequestHeaders = nil
	if schema, header := trajectoryRequestPayloads(snapshot, "call"); schema != "" || header != "" {
		t.Fatal("schema invented without header")
	}
}
func TestTrajectorySchemaTabStructuredAndCompleteCopy(t *testing.T) {
	snapshot := headerFixture()
	selected, tab, query := "result", "overview", ""
	copied := ""
	tt := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{Snapshot: snapshot, SelectedID: &selected, DetailTab: &tab, SearchQuery: &query, OnCopyDetail: func(s string) { copied = s }})
	}, 1000, 700)
	if err := tt.Click("Schema"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("Structured JSON"); !ok {
		t.Fatalf("schema not using existing structured inspector; tab=%q texts=%v", tab, tt.Texts())
	}
	// The shared inspector copy receives the complete retained schema payload.
	want, _ := trajectoryRequestPayloads(snapshot, "result")
	if err := tt.Click(i18n.T(i18n.KeyTrajCopy)); err != nil {
		t.Fatal(err)
	}
	if copied != want {
		t.Fatal("schema copy truncated")
	}
	if err := tt.Click("Request header"); err != nil {
		t.Fatal(err)
	}
	if tab != "request-header" {
		t.Fatal("header tab did not activate")
	}
}
