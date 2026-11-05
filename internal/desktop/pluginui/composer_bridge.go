package pluginui

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/coder/websocket/wsjson"
)

// ComposerRequest is a closed native draft protocol, not a generic Host RPC.
type ComposerRequest struct {
	Request string          `json:"request"`
	Op      string          `json:"op"`
	Subject Subject         `json:"subject"`
	Edit    ComposerEdit    `json:"edit"`
	Gesture ComposerGesture `json:"gesture"`
}

func (o *Owner) SetComposerBridge(bridge ComposerBridge) {
	o.mu.Lock()
	o.composer = bridge
	o.composerLast = ""
	o.mu.Unlock()
	o.RefreshComposer()
}

// RefreshComposer is called by the native render owner. Its result cannot be
// delivered to a replacement page or another selected Session.
func (o *Owner) RefreshComposer() {
	o.mu.Lock()
	bridge, page, subject := o.composer, o.page, o.selected
	valid := !o.closed && !o.incompatible && bridge != nil && page != nil && subject.Kind == "session"
	o.mu.Unlock()
	if !valid {
		return
	}
	draft, err := bridge.Read(subject)
	if err != nil || !draft.Owner.Subject.equal(subject) {
		return
	}
	encoded, _ := json.Marshal(draft)
	o.mu.Lock()
	current := !o.closed && o.page == page && o.selected.equal(subject) && o.composerLast != string(encoded)
	if current {
		o.composerLast = string(encoded)
	}
	o.mu.Unlock()
	if !current {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if wsjson.Write(ctx, page.socket, map[string]any{"type": "composerState", "page": page.id, "draft": draft}) != nil {
		o.disconnect(page)
	}
}

func (o *Owner) handleComposer(page *bridgePage, request *ComposerRequest) {
	if request == nil || request.Request == "" || len(request.Request) > 64 {
		return
	}
	o.mu.Lock()
	bridge := o.composer
	valid := !o.closed && !o.incompatible && o.page == page && bridge != nil && o.selected.equal(request.Subject) && request.Subject.Kind == "session" && request.Subject.SessionID != ""
	o.mu.Unlock()
	var err error
	if !valid {
		err = errors.New("Plugin composer owner expired")
	} else {
		switch request.Op {
		case "read":
		case "edit":
			e := request.Edit
			count := 0
			if e.Text != nil {
				count++
				if len(*e.Text) > 1024*1024 {
					err = errors.New("Plugin draft is too large")
				}
			}
			if e.Replace != nil {
				count++
				if e.Replace.Start < 0 || e.Replace.End < e.Replace.Start || len(e.Replace.Text) > 1024*1024 {
					err = errors.New("Invalid plugin insertion")
				}
			}
			if e.RemoveFileID != 0 {
				count++
			}
			if e.RetryFileID != 0 {
				count++
			}
			if e.PickFiles {
				count++
			}
			if e.PickImage {
				count++
			}
			if !e.Owner.Subject.equal(request.Subject) || count != 1 {
				err = errors.New("Invalid plugin draft action")
			}
			if err == nil {
				err = bridge.Apply(e)
			}
		case "submit":
			if !request.Gesture.Owner.Subject.equal(request.Subject) {
				err = errors.New("Plugin composer owner changed")
			} else {
				err = bridge.Submit(request.Gesture)
			}
		default:
			err = errors.New("Unsupported plugin draft action")
		}
	}
	reply := map[string]any{"type": "composerReply", "page": page.id, "request": request.Request, "ok": err == nil}
	if err != nil {
		reply["error"] = err.Error()
	} else {
		draft, readErr := bridge.Read(request.Subject)
		if readErr != nil {
			reply["syncError"] = readErr.Error()
		} else if !draft.Owner.Subject.equal(request.Subject) {
			reply["syncError"] = "Plugin composer owner changed"
		} else {
			reply["draft"] = draft
		}
	}
	o.mu.Lock()
	current := !o.closed && o.page == page && o.selected.equal(request.Subject)
	if current {
		o.composerError, _ = reply["error"].(string)
	}
	o.mu.Unlock()
	if !current {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if wsjson.Write(ctx, page.socket, reply) != nil {
		o.disconnect(page)
	}
	o.notify()
}

func (o *Owner) ComposerError() string {
	if o == nil {
		return ""
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.composerError
}
