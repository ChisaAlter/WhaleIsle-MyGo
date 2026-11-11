package chat

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"sort"
	"whaleisle/internal/store/session"
)

const trajectoryRequestKind session.MessageKind = "trajectory-request"

func trajectoryHistoryMessages(snapshot session.ViewState) []session.Message {
	if snapshot.TrajectoryMessages != nil {
		return snapshot.TrajectoryMessages
	}
	return snapshot.Messages
}

func trajectoryMessages(snapshot session.ViewState) []session.Message {
	out := append([]session.Message(nil), trajectoryHistoryMessages(snapshot)...)
	for _, request := range snapshot.TrajectoryRequests {
		body, _ := json.MarshalIndent(request, "", "  ")
		raw, _ := json.Marshal(request.Events)
		model := ""
		if request.ProviderMetadata != nil {
			model = request.ProviderMetadata.Provider + " / " + request.ProviderMetadata.Model
		}
		out = append(out, session.Message{ID: "request:" + request.ID, Seq: request.StartSeq, Kind: trajectoryRequestKind, Role: "request", Body: fmt.Sprintf("%s · %s", request.Purpose, request.Status), Args: string(body), Raw: raw, Model: model,
			ConversationTurn: request.Turn, ConversationStep: request.Step,
			Streaming: request.Status == "running", TrajectoryTime: &session.TrajectoryTiming{OperationID: request.ID, StartedAt: request.StartedAt, CompletedAt: request.CompletedAt, Closed: request.Status != "running"}})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Seq == 0 {
			return false
		}
		if out[j].Seq == 0 {
			return true
		}
		return out[i].Seq < out[j].Seq
	})
	return out
}
func trajectoryRequestBodies(snapshot session.ViewState, rowID string) (string, string) {
	for _, r := range snapshot.TrajectoryRequests {
		if "request:"+r.ID == rowID {
			body, _ := json.MarshalIndent(r, "", "  ")
			events, _ := json.MarshalIndent(r.Events, "", "  ")
			return string(body), string(events)
		}
	}
	return "", ""
}
func paintTrajectoryLifecycleTabs(c *ui.Context, selected *string, available bool, accent, muted ui.Color) {
	if !available {
		return
	}
	ui.Row(c).Gap(6).Children(func() {
		for _, tab := range []struct{ id, label string }{{"request", "Request"}, {"request-events", "Events"}} {
			ink := muted
			if *selected == tab.id {
				ink = accent
			}
			if ui.Button(c, tab.label).TextColor(ink).Cursor(ui.CursorPointer).Clicked() {
				*selected = tab.id
			}
		}
	})
}
