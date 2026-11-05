package pet

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"
	"whaleisle/internal/petgrowth"
	"whaleisle/internal/store/storelocale"
)

type GrowthStatus struct {
	Label, Detail, Message string
	Busy, CanFeed          bool
	petgrowth.Snapshot
	Levels []growthTier
}

type growthTier struct {
	At          int64
	Name, Color string
}

var growthColors = []string{"#7fb3e8", "#4a90d9", "#3aa8c1", "#5b8def", "#9b6dd9", "#e85d75", "#f0785a", "#f09a4a", "#f5c04d", "#ffe98a"}

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
		status.Message = storelocale.T(storelocale.KeyDesktopPetLedgerUnavailable)
		return status
	}
	status.Label = fmt.Sprintf(storelocale.T(storelocale.KeyStorePetLvDSDGrowthPoints), snap.Level, snap.LevelName, snap.Points)
	status.Snapshot = snap
	for i, tier := range petgrowth.Levels {
		status.Levels = append(status.Levels, growthTier{tier.At, tier.Name, growthColors[i]})
	}
	status.Detail = fmt.Sprintf(storelocale.T(storelocale.KeyStorePetLocalFoodDFeedableNowDAddedTodayD), snap.Available, snap.NextFeed, snap.TodayUsed)
	// An empty displayed bowl may have new unsettled usage. A deliberate click
	// must still be able to run the original pre-feed refresh.
	status.CanFeed = !busy
	if status.Message == "" {
		status.Message = storelocale.T(storelocale.KeyDesktopPetLocalGameOnly)
	}
	return status
}
func renderGrowthPage(page []byte, status GrowthStatus) []byte {
	return []byte(strings.NewReplacer("{{GROWTH_LABEL}}", html.EscapeString(status.Label), "{{GROWTH_DETAIL}}", html.EscapeString(status.Detail), "{{GROWTH_MESSAGE}}", html.EscapeString(status.Message)).Replace(string(page)))
}

// StartGrowth uses the same per-window cancellation/serial gate as explicit
// chat entry. The native main callback provides fixed app data/runtime paths.
func (p *Companion) StartGrowth() (context.Context, func(string, func() error) bool) {
	return p.StartGrowthRefresh(false)
}

func (p *Companion) StartGrowthRefresh(automatic bool) (context.Context, func(string, func() error) bool) {
	if p.win == nil || p.win.IsDestroyed() || !p.win.IsVisible() || !p.ready || p.Store == nil || !p.Store.DesktopPet().Enabled {
		return nil, nil
	}
	p.assetMu.Lock()
	ctx, serial := p.growth.begin()
	if ctx != nil {
		p.growthAutomatic = automatic
		p.growth.status.Message = storelocale.T(storelocale.KeyDesktopPetReadingLocalUsage)
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
		p.growthAutomatic = false
		if ctx.Err() != nil {
			p.growth.finish(serial, storelocale.T(storelocale.KeyDesktopPetCancelled))
			return false
		}
		if commit != nil {
			if err := commit(); err != nil {
				message = storelocale.T(storelocale.KeyDesktopPetSaveFailedUnchanged)
			}
		}
		return p.growth.finish(serial, message)
	}
}

// FeedTokensAfterRefresh is reached only after an explicit feed request and
// an accepted scan completion. Timer refreshes never invoke it.
func (p *Companion) FeedTokensAfterRefresh(fresh bool) { p.consumePetTokens(fresh) }

// feedTokens retains the direct internal entry for existing local callers.
func (p *Companion) feedTokens() { p.consumePetTokens(true) }

// consumePetTokens is synchronous on the native UI thread and never launches a
// model or an account request. Empty bowls earn no points or care reward.
func (p *Companion) consumePetTokens(fresh bool) {
	if p.win == nil || p.win.IsDestroyed() || !p.win.IsVisible() || !p.ready || p.Store == nil || !p.Store.DesktopPet().Enabled {
		return
	}
	p.assetMu.Lock()
	defer p.assetMu.Unlock()
	if p.growth.status.Busy || p.feeding {
		return
	}
	before, _ := p.Store.PetGrowth(time.Now())
	fed, err := p.Store.FeedPetTokens(time.Now())
	if err != nil {
		p.growth.status.Message = storelocale.T(storelocale.KeyDesktopPetSaveFailedUnchanged2)
		return
	}
	if fed == 0 {
		p.growth.status.Message = storelocale.T(storelocale.KeyDesktopPetNoLocalFood)
		return
	}
	after, _ := p.Store.PetGrowth(time.Now())
	level := 0
	if after.Level > before.Level {
		level = after.Level
	}
	p.careAction = newCareAnimation("token", "", level)
	_ = p.rest.wake(nil)
	p.growth.status.Message = fmt.Sprintf(storelocale.T(storelocale.KeyDesktopPetFedLocalFood), fed, fed)
	if !fresh {
		p.growth.status.Message += storelocale.T(storelocale.KeyDesktopPetSavedFoodOnly)
	}
}
