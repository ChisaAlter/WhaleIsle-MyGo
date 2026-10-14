package pet

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"
)

type GrowthStatus struct {
	Label, Detail, Message string
	Busy, CanFeed          bool
}

func (p *Companion) growthStatus() GrowthStatus {
	p.assetMu.RLock()
	busy, message := p.growth.status.Busy, p.growth.status.Message
	p.assetMu.RUnlock()
	status := GrowthStatus{Busy: busy, Message: message}
	if p.Store == nil {
		return status
	}
	snap, err := p.Store.PetGrowth(time.Now())
	if err != nil {
		status.Message = "成长账本不可用，未改动存档 / Ledger unavailable"
		return status
	}
	status.Label = fmt.Sprintf("Lv.%d %s · %d 成长点", snap.Level, snap.LevelName, snap.Points)
	status.Detail = fmt.Sprintf("本地粮 %d · 本次可喂 %d · 今日新增 %d", snap.Available, snap.NextFeed, snap.TodayUsed)
	status.CanFeed = !busy && snap.NextFeed > 0
	if status.Message == "" {
		status.Message = "刷新实际用量；首次仅建基线 / Local game only"
	}
	return status
}
func renderGrowthPage(page []byte, status GrowthStatus) []byte {
	return []byte(strings.NewReplacer("{{GROWTH_LABEL}}", html.EscapeString(status.Label), "{{GROWTH_DETAIL}}", html.EscapeString(status.Detail), "{{GROWTH_MESSAGE}}", html.EscapeString(status.Message)).Replace(string(page)))
}

// StartGrowth uses the same per-window cancellation/serial gate as explicit
// chat entry. The native main callback provides fixed app data/runtime paths.
func (p *Companion) StartGrowth() (context.Context, func(string, func() error) bool) {
	if p.win == nil || p.win.IsDestroyed() || !p.ready {
		return nil, nil
	}
	p.assetMu.Lock()
	ctx, serial := p.growth.begin()
	if ctx != nil {
		p.growth.status.Message = "读取本应用实际用量… / Reading local usage…"
	}
	p.assetMu.Unlock()
	if ctx == nil {
		return nil, nil
	}
	return ctx, func(message string, commit func() error) bool {
		p.assetMu.Lock()
		defer p.assetMu.Unlock()
		if p.growth.serial != serial || !p.growth.status.Busy {
			return false
		}
		if commit != nil {
			if ctx.Err() != nil {
				return p.growth.finish(serial, "读取已取消，未改动账本 / Cancelled")
			}
			if err := commit(); err != nil {
				message = "账本保存失败，粮与成长未改动 / Save failed; unchanged"
			}
		}
		return p.growth.finish(serial, message)
	}
}

// feedTokens is synchronous on the native UI thread and never launches a
// model or an account request. Empty bowls earn no points or care reward.
func (p *Companion) feedTokens() {
	if p.win == nil || p.win.IsDestroyed() || !p.ready || p.Store == nil || !p.Store.DesktopPet().Enabled {
		return
	}
	p.assetMu.Lock()
	defer p.assetMu.Unlock()
	if p.growth.status.Busy {
		return
	}
	fed, err := p.Store.FeedPetTokens(time.Now())
	if err != nil {
		p.growth.status.Message = "保存失败，粮与成长均未扣改 / Save failed; unchanged"
		return
	}
	if fed == 0 {
		p.growth.status.Message = "粮碗空了，先刷新实际用量 / No local food"
		return
	}
	_ = p.rest.wake(nil)
	p.growth.status.Message = fmt.Sprintf("已喂 %d 本地粮，成长 +%d / Fed", fed, fed)
}
