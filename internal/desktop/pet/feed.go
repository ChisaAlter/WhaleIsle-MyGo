package pet

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/petcare"
	"whaleisle/internal/store/storelocale"
)

type CareStatus struct {
	Action         careAnimation
	Label          string
	Busy           bool
	Message        string
	Satiety        float64
	AffectionLevel int
}

func (p *Companion) careStatus() CareStatus {
	stats := petcare.Snapshot(nil, time.Now())
	if p.Store != nil {
		stats = p.Store.PetStats(time.Now())
	}
	p.assetMu.RLock()
	busy, message, action := p.feeding, p.feedMessage, p.careAction
	p.assetMu.RUnlock()
	if message == "" {
		message = storelocale.T(storelocale.KeyDesktopPetFilesStayUntouched)
	}
	level := 1
	for _, threshold := range []float64{40, 120, 260, 520} {
		if stats.Affection >= threshold {
			level++
		}
	}
	return CareStatus{Action: action, Satiety: stats.Satiety, AffectionLevel: level, Label: fmt.Sprintf(storelocale.T(storelocale.KeyDesktopPetCareLabel), stats.Satiety, stats.Mood, stats.Affection), Busy: busy, Message: message}
}
func renderCarePage(page []byte, care CareStatus) []byte {
	return []byte(strings.NewReplacer("{{CARE_LABEL}}", html.EscapeString(care.Label), "{{CARE_MESSAGE}}", html.EscapeString(care.Message)).Replace(string(page)))
}

// feedFile is invoked only by the exact native navigation action. The picker
// confirms a deliberate file meal; its returned paths are never read or stored.
func (p *Companion) feedFile() {
	w := p.win
	if w == nil || w.IsDestroyed() || !p.ready || p.Store == nil {
		return
	}
	p.assetMu.Lock()
	if p.feeding {
		p.assetMu.Unlock()
		return
	}
	p.feeding = true
	p.feedMessage = storelocale.T(storelocale.KeyDesktopPetChooseOneFile)
	p.assetMu.Unlock()
	defer func() {
		if p.win == w {
			p.assetMu.Lock()
			p.feeding = false
			p.assetMu.Unlock()
		}
	}()
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: w, Title: storelocale.T(storelocale.KeyDesktopPetFeedOnceFileStaysUntouched)})
	// Native dialogs can pump close/settings events. A replacement pet must not
	// receive the previous instance's selection or its pending care award.
	if p.win != w || w.IsDestroyed() || !p.ready || !p.Store.DesktopPet().Enabled {
		return
	}
	message := ""
	fed := false
	if err != nil {
		message = storelocale.T(storelocale.KeyDesktopPetFileSelectionFailed)
	} else if len(paths) > 0 {
		if err = p.Store.FeedPetFile(time.Now()); err != nil {
			message = storelocale.T(storelocale.KeyDesktopPetSaveFailedNotApplied)
		} else {
			fed = true
			message = storelocale.T(storelocale.KeyDesktopPetFed)
		}
	}
	p.assetMu.Lock()
	p.feeding = false
	if fed {
		p.careAction = newCareAnimation("file", "", 0)
		_ = p.rest.wake(nil)
	}
	p.feedMessage = message
	p.assetMu.Unlock()
}
