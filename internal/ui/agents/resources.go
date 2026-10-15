package agents

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"whaleisle/internal/desktop/workspacefs"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
)

func childResourceOptions(m Model, v session.AgentNavigationView) chat.ReadOnlyResourceOptions {
	read := func(ref string, more bool) {
		switch strings.ToLower(filepath.Ext(ref)) {
		case ".pdf", ".html", ".htm":
			if m.OnOpenDocument != nil {
				m.OnOpenDocument(v.Token, v.Address, ref)
			}
			return
		case ".xlsx":
			go m.Store.ReadAgentWorkbook(context.Background(), v.Token, v.Address, ref, workspacefs.XLSXPageRequest{})
			return
		}
		go m.Store.ReadAgentResource(context.Background(), v.Token, v.Address, ref, more)
	}
	return chat.ReadOnlyResourceOptions{OnReadTable: func(path string, request workspacefs.XLSXPageRequest) {
		go m.Store.ReadAgentWorkbook(context.Background(), v.Token, v.Address, path, request)
	}, OnRead: read, Reads: v.Resources, OnOpenFile: func(path string) { read(path, false) }, OnOpenURL: m.OnOpenURL}
}
func childDeliveriesView(c *ui.Context, m Model, v session.AgentNavigationView, st chat.ToolCardStyle) {
	deliveries := session.TurnDeliveries(v.Messages)
	turns := make([]uint64, 0, len(deliveries))
	for turn := range deliveries {
		turns = append(turns, turn)
	}
	sort.Slice(turns, func(i, j int) bool { return turns[i] < turns[j] })
	snapshot := m.Store.AgentTurnFiles(v.Token)
	for _, turn := range turns {
		chat.AddressedTurnDeliveryView(c, deliveries[turn], snapshot, snapshot.Scope.Binding != "", func(target session.TurnFileTarget, action string) {
			go m.Store.RunAgentTurnFileAction(context.Background(), v.Token, v.Address, session.TurnFileRequest{Target: target, Action: action})
		}, func(target session.TurnFileTarget) {
			if path, ok := m.Store.AgentTurnPreviewPath(v.Token, v.Address, target); ok {
				childResourceOptions(m, v).OnRead(path, false)
			}
		}, st)
	}
	// Also render explicitly opened inline-code/delivery references in one child panel.
	chat.ReadOnlyResourceResultsView(c, childResourceOptions(m, v), st)
}
