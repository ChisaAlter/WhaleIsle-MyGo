package pet

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/petcare"
)

type CareStatus struct {
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
	busy, message := p.feeding, p.feedMessage
	p.assetMu.RUnlock()
	if message == "" {
		message = "不读取内容，不修改文件 / Files stay untouched"
	}
	level := 1
	for _, threshold := range []float64{40, 120, 260, 520} {
		if stats.Affection >= threshold {
			level++
		}
	}
	return CareStatus{Satiety: stats.Satiety, AffectionLevel: level, Label: fmt.Sprintf("饱食 / Satiety %.0f · 心情 / Mood %.0f · 亲密 / Bond %.0f", stats.Satiety, stats.Mood, stats.Affection), Busy: busy, Message: message}
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
	p.feedMessage = "选择一个文件投喂 / Choose one file"
	p.assetMu.Unlock()
	defer func() {
		if p.win == w {
			p.assetMu.Lock()
			p.feeding = false
			p.assetMu.Unlock()
		}
	}()
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: w, Title: "投喂一次（不读取或修改文件） / Feed once; file stays untouched"})
	// Native dialogs can pump close/settings events. A replacement pet must not
	// receive the previous instance's selection or its pending care award.
	if p.win != w || w.IsDestroyed() || !p.ready || !p.Store.DesktopPet().Enabled {
		return
	}
	message := ""
	fed := false
	if err != nil {
		message = "文件选择失败 / File selection failed"
	} else if len(paths) > 0 {
		if err = p.Store.FeedPetFile(time.Now()); err != nil {
			message = "保存失败，未计入状态 / Save failed; not applied"
		} else {
			fed = true
			message = "已投喂：饱食 +2，心情 +1（上限 100） / Fed"
		}
	}
	p.assetMu.Lock()
	p.feeding = false
	if fed {
		_ = p.rest.wake(nil)
	}
	p.feedMessage = message
	p.assetMu.Unlock()
}
