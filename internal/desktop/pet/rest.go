package pet

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// restState belongs to one pet page/window, never config.json. A duplicate
// sleep/wake URL is idempotent; only an actual explicit wake charges care.
type restState struct {
	sleeping bool
	message  string
}

func (r *restState) sleep() { r.sleeping = true; r.message = "" }
func (r *restState) wake(commit func() error) error {
	if !r.sleeping {
		return nil
	}
	if commit != nil {
		if err := commit(); err != nil {
			return err
		}
	}
	r.sleeping = false
	r.message = ""
	return nil
}

type RestStatus struct {
	Sleeping               bool
	Label, Action, Message string
}

func (p *Companion) restStatus() RestStatus {
	p.assetMu.RLock()
	defer p.assetMu.RUnlock()
	status := RestStatus{Sleeping: p.rest.sleeping, Label: "睡觉 / Sleep", Action: "sleep-pet", Message: "清醒 / Awake"}
	if status.Sleeping {
		status.Label = "叫醒 / Wake"
		status.Action = "wake-pet"
		status.Message = "休眠中 / Resting"
	}
	if p.rest.message != "" {
		status.Message = p.rest.message
	}
	return status
}
func renderRestPage(page []byte, rest RestStatus) []byte {
	return []byte(strings.NewReplacer("{{REST_SLEEPING}}", fmt.Sprint(rest.Sleeping), "{{REST_ACTION}}", rest.Action, "{{REST_LABEL}}", html.EscapeString(rest.Label), "{{REST_MESSAGE}}", html.EscapeString(rest.Message)).Replace(string(page)))
}
func (p *Companion) restPet(action string) {
	if p.win == nil || p.win.IsDestroyed() || !p.ready || p.Store == nil || !p.Store.DesktopPet().Enabled {
		return
	}
	p.assetMu.Lock()
	defer p.assetMu.Unlock()
	if p.feeding {
		return
	}
	switch action {
	case "sleep-pet":
		p.rest.sleep()
	case "wake-contact":
		_ = p.rest.wake(nil) // original cursor contact: no care penalty
	case "wake-pet":
		if err := p.rest.wake(func() error { return p.Store.WakePet(time.Now()) }); err != nil {
			p.rest.message = "唤醒保存失败，仍保持休眠 / Save failed; still resting"
		}
	}
}
