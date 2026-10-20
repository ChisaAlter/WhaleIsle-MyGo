package pet

import (
	"fmt"
	"html"
	"strings"
	"time"
	"whaleisle/internal/store/storelocale"
)

type RestStatus struct {
	Sleeping, Dozing       bool
	Label, Action, Message string
}

func (p *Companion) restStatus() RestStatus {
	p.assetMu.RLock()
	defer p.assetMu.RUnlock()
	status := RestStatus{Sleeping: p.rest.sleeping, Dozing: p.rest.dozing, Label: storelocale.T(storelocale.KeyDesktopPetSleep), Action: "sleep-pet", Message: storelocale.T(storelocale.KeyDesktopPetAwake)}
	if status.Dozing {
		status.Message = storelocale.T(storelocale.KeyDesktopPetDozing)
	}
	if status.Sleeping {
		status.Label = storelocale.T(storelocale.KeyDesktopPetWake)
		status.Action = "wake-pet"
		status.Message = storelocale.T(storelocale.KeyDesktopPetResting)
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
			p.rest.message = storelocale.T(storelocale.KeyDesktopPetSaveFailedStillResting)
		}
	}
}
