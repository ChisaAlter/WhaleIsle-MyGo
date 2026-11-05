package chat

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

func trajectoryRequestPayloads(snapshot session.ViewState, rowID string) (schema, header string) {
	var source *session.Message
	messages := trajectoryHistoryMessages(snapshot)
	for i := range messages {
		if messageToTrajRow(messages[i]).ID == rowID {
			source = &messages[i]
			break
		}
	}
	if source == nil || source.Seq <= 0 {
		return "", ""
	}
	seq := source.Seq
	name := ""
	// Resolve the selected call against the complete loaded tool tree. All
	// subcalls inherit their root's historical request declaration, as original.
	selectedCalls := session.ProjectToolMessages([]session.Message{*source})
	callID := ""
	if len(selectedCalls) > 0 && selectedCalls[0].Tool != nil {
		callID = selectedCalls[0].Tool.CallID
	}
	if callID != "" {
		for _, root := range session.ProjectToolMessages(messages) {
			stack := []*session.ToolCallView{root.Tool}
			seen := map[*session.ToolCallView]bool{}
			for len(stack) > 0 {
				call := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if call == nil || seen[call] {
					continue
				}
				seen[call] = true
				if call.CallID == callID {
					seq = root.Seq
					if call.Call == nil && call.State != "preparing" {
						return "", ""
					}
					name = call.Name
					break
				}
				stack = append(stack, call.Children...)
			}
			if name != "" {
				break
			}
		}
	}
	var selected *session.TrajectoryRequestHeader
	for i := range snapshot.TrajectoryRequestHeaders {
		h := &snapshot.TrajectoryRequestHeaders[i]
		if h.Seq < seq && (selected == nil || h.Seq > selected.Seq) {
			selected = h
		}
	}
	if selected == nil {
		return "", ""
	}
	raw, _ := json.MarshalIndent(selected, "", "  ")
	header = string(raw)
	for _, tool := range selected.Header.Tools {
		if tool.Name == name {
			raw, _ := json.MarshalIndent(tool, "", "  ")
			schema = string(raw)
			break
		}
	}
	return schema, header
}

func paintTrajectoryRequestTabs(c *ui.Context, selected *string, schema, header string, accent, muted ui.Color) {
	if schema == "" && header == "" {
		return
	}
	ui.Row(c).Gap(6).Children(func() {
		for _, tab := range []struct{ id, label, body string }{{"schema", "Schema", schema}, {"request-header", "Request header", header}} {
			if tab.body == "" {
				continue
			}
			ink := muted
			if *selected == tab.id {
				ink = accent
			}
			if ui.Button(c, tab.label).FontSize(theme.InterfaceSize(11)).BorderWidth(0).Background(ui.Transparent).TextColor(ink).Clicked() {
				*selected = tab.id
			}
		}
	})
}
