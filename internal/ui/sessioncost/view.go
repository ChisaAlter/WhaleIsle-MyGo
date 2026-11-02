// Package sessioncost renders Host-reported billing counters below the composer.
package sessioncost

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	cost "whaleisle/internal/store/sessioncost"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

var (
	costPrefsMemo struct {
		sync.Mutex
		raw                   string
		valid                 bool
		stats, show, official bool
	}
	costCalcMemo struct {
		sync.Mutex
		key        session.ProjectionKey
		selectedID string
		billing    usage.BillingSettings
		result     cost.Result
	}
)

type State struct {
	Open, ChildrenOpen bool
	SessionID          string
	NextTick           time.Time
}
type Model struct {
	Snapshot    session.ViewState
	Billing     usage.BillingView
	Preferences settings.ViewState
	State       *State
	OnPrices    func()
}

func preferences(v settings.ViewState) (stats, show, official bool) {
	stats = true
	var rawValue json.RawMessage
	found := false
	for _, ns := range v.Namespaces {
		if ns.NS == "ui-conversation" {
			rawValue = ns.Value
			found = true
			break
		}
	}
	if !found {
		return
	}
	rawStr := string(rawValue)
	costPrefsMemo.Lock()
	if costPrefsMemo.valid && costPrefsMemo.raw == rawStr {
		stats, show, official = costPrefsMemo.stats, costPrefsMemo.show, costPrefsMemo.official
		costPrefsMemo.Unlock()
		return
	}
	costPrefsMemo.Unlock()
	var fields map[string]json.RawMessage
	if json.Unmarshal(rawValue, &fields) != nil {
		return
	}
	_ = json.Unmarshal(fields["statsLine"], &stats)
	_ = json.Unmarshal(fields["sessionCost"], &show)
	_ = json.Unmarshal(fields["officialPeakValley"], &official)
	costPrefsMemo.Lock()
	costPrefsMemo.valid = true
	costPrefsMemo.raw = rawStr
	costPrefsMemo.stats, costPrefsMemo.show, costPrefsMemo.official = stats, show, official
	costPrefsMemo.Unlock()
	// Only accepted Host values affect display; failed or cancelled drafts stay local.

	return
}

func cachedCalculate(snap session.ViewState, billing usage.BillingSettings) cost.Result {
	projKey := snap.RosterProjectionKey()
	if !projKey.Valid() {
		return cost.Calculate(snap.SelectedID, snap.CostSources, billing)
	}
	costCalcMemo.Lock()
	if costCalcMemo.key == projKey && costCalcMemo.selectedID == snap.SelectedID &&
		costCalcMemo.billing.PeakValleyEnabled == billing.PeakValleyEnabled &&
		reflect.DeepEqual(costCalcMemo.billing.Prices, billing.Prices) {
		res := costCalcMemo.result
		costCalcMemo.Unlock()
		return res
	}
	costCalcMemo.Unlock()
	res := cost.Calculate(snap.SelectedID, snap.CostSources, billing)
	var pricesCopy map[string]usage.ModelPrice
	if billing.Prices != nil {
		pricesCopy = make(map[string]usage.ModelPrice, len(billing.Prices))
		for k, v := range billing.Prices {
			pricesCopy[k] = v
		}
	}
	costCalcMemo.Lock()
	costCalcMemo.key = projKey
	costCalcMemo.selectedID = snap.SelectedID
	costCalcMemo.billing = usage.BillingSettings{Prices: pricesCopy, PeakValleyEnabled: billing.PeakValleyEnabled}
	costCalcMemo.result = res
	costCalcMemo.Unlock()
	return res
}

// compactCount renders a token count the way the reference status line does:
// 812, 12.3K, 77.7M.
func compactCount(value float64) string {
	if value < 0 {
		value = 0
	}
	if value < 1000 {
		return fmt.Sprintf("%.0f", value)
	}
	scaled := func(f float64) string {
		if f >= 100 {
			return fmt.Sprintf("%.0f", f)
		}
		return strings.TrimSuffix(fmt.Sprintf("%.1f", f), ".0")
	}
	switch {
	case value < 1e6:
		return scaled(value/1e3) + "K"
	case value < 1e9:
		return scaled(value/1e6) + "M"
	default:
		return scaled(value/1e9) + "B"
	}
}

func money(r cost.Result) string {
	if !r.Available {
		return i18n.T(i18n.KeyComposerCostMoneyUnavailable)
	}
	if r.Priced == 0 {
		return i18n.T(i18n.KeyComposerCostNoPrices)
	}
	v := fmt.Sprintf("¥%.2f", r.Cents/100)
	if r.Unpriced > 0 {
		v += i18n.T(i18n.KeyComposerCostPartlyUnpriced)
	}
	return v
}

// bucketTotal is the sum the status line reports for a session.
func bucketTotal(b cost.Bucket) float64 { return b.Hit + b.Miss + b.Output }

// cacheHitPercent is empty until the bucket has prompt tokens to divide.
func cacheHitPercent(b cost.Bucket) string {
	if b.Hit+b.Miss <= 0 {
		return ""
	}
	return fmt.Sprintf("%.1f%%", 100*b.Hit/(b.Hit+b.Miss))
}

// tokensDetail is the panel's per-bucket breakdown.
func tokensDetail(b cost.Bucket) string {
	rate := ""
	if pct := cacheHitPercent(b); pct != "" {
		rate = " (" + pct + ")"
	}
	return fmt.Sprintf(i18n.T(i18n.KeyComposerCostTokensDetail),
		compactCount(b.Hit)+rate, compactCount(b.Miss), compactCount(b.Output))
}

// OwnsPerformanceStats reports that this package paints the composer's whole
// statistics line, including the decode-speed ("tok/s") and cache-hit clusters
// the conversation feature can also derive. The conversation's own performance
// row must stay off beside View while this is true.
const OwnsPerformanceStats = true

// dockLine is the composer's statistics strip: one centered row of compact
// clusters ("12 轮 280 步 · 237 tok/s", "77.7M tok · 缓存命中 99.7%"). It is the
// only place the composer reports those numbers; see OwnsPerformanceStats.
func dockLine(c *ui.Context, m Model, r cost.Result, stats bool, muted ui.Color) {
	var clusters []string
	if stats {
		memo := ui.Local(c.Root(), "dock-performance-history-derived", func() session.HistoryDerivedMemo { return session.HistoryDerivedMemo{} })
		perfTokens, perfMS, perfSteps := memo.RecordedPerformance(m.Snapshot)
		var parts []string
		if turns := len(m.Snapshot.TurnUsage); turns > 0 && perfSteps > 0 {
			parts = append(parts, fmt.Sprintf("%d %s %d %s", turns, i18n.T(i18n.KeyComposerDockTurns),
				perfSteps, i18n.T(i18n.KeyComposerDockSteps)))
		}
		if perfMS > 0 && perfTokens > 0 {
			parts = append(parts, fmt.Sprintf("%.0f %s", float64(perfTokens)*1000/float64(perfMS),
				i18n.T(i18n.KeyComposerDockSpeedUnit)))
		}
		if len(parts) > 0 {
			clusters = append(clusters, strings.Join(parts, " · "))
		}
	}
	if stats && r.Available {
		if total := bucketTotal(r.Total); total > 0 {
			parts := []string{fmt.Sprintf("%s %s", compactCount(total), i18n.T(i18n.KeyComposerDockTokenUnit))}
			if hit := cacheHitPercent(r.Total); hit != "" {
				parts = append(parts, fmt.Sprintf(i18n.T(i18n.KeyComposerDockCacheHit), hit))
			}
			clusters = append(clusters, strings.Join(parts, " · "))
		}
	}
	if len(clusters) == 0 {
		// Reference hides the strip until a turn has usage. Only an actual
		// unavailable cost read gets the fallback line.
		if stats && !r.Available {
			ui.Text(c, i18n.T(i18n.KeyComposerCostUnavailable)).FontSize(12).TextColor(muted).Padding(1, 8)
		}
		return
	}
	ui.Row(c).Key("composer-dock-stats").FillWidth().Justify(ui.Center).Gap(12).Children(func() {
		for _, text := range clusters {
			ui.Text(c, text).FontSize(12).TextColor(muted).Padding(1, 8)
		}
	})
}

func View(c *ui.Context, m Model, muted ui.Color) {
	if m.Snapshot.SelectedID == "" || m.State == nil {
		return
	}
	if m.State.SessionID != m.Snapshot.SelectedID {
		*m.State = State{SessionID: m.Snapshot.SelectedID}
	}
	stats, show, official := preferences(m.Preferences)
	if !stats && !show {
		return
	}
	r := cachedCalculate(m.Snapshot, m.Billing.Settings)
	// Keep the same 4 DIP bottom inset the Electron dock has; without it the
	// status line sits flush on the window edge at every viewport height.
	ui.Column(c).Key("composer-dock").FillWidth().Padding(0, 8, 4, 8).Gap(4).AlignItems(ui.Center).Children(func() {
		dockLine(c, m, r, stats, muted)
		if !show {
			return
		}
		if official || m.Snapshot.SelectedProvider == "deepseek" || m.Snapshot.SelectedProvider == "deepseek-official" {
			now := time.Now()
			peak, next := cost.Phase(now)
			if !now.Before(m.State.NextTick) {
				m.State.NextTick = now.Truncate(time.Second).Add(time.Second)
				services := c.Services()
				time.AfterFunc(time.Until(m.State.NextTick), services.Invalidate)
			}
			seconds := int(next.Sub(now).Seconds())
			if seconds < 0 {
				seconds = 0
			}
			phase := i18n.T(i18n.KeyComposerDockOffPeak)
			if peak {
				phase = i18n.T(i18n.KeyComposerDockPeak)
			}
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyComposerDockPeakSwitch), phase,
				fmt.Sprintf("%02d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60))).
				FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		label := i18n.T(i18n.KeyComposerCostEstimate) + " · " + money(r)
		if !m.Billing.Loaded && r.Available {
			label = i18n.T(i18n.KeyComposerCostEstimate) + " · " + i18n.T(i18n.KeyComposerCostPricesLoading)
		}
		if ui.Button(c, label).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			m.State.Open = !m.State.Open
		}
		if !m.Snapshot.ClientReady {
			ui.Text(c, i18n.T(i18n.KeyComposerCostOffline)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if r.Partial {
			ui.Text(c, i18n.T(i18n.KeyComposerCostPartial)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if !m.State.Open {
			return
		}
		ui.Scroll(c).Height(180).Children(func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyComposerCostNotABill)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if m.Billing.Err != "" {
					ui.Text(c, m.Billing.Err).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
				for _, row := range r.Rows {
					name := row.Provider + "/" + row.Model
					if name == "/" {
						name = i18n.T(i18n.KeyComposerCostUnknownRoute)
					}
					price := i18n.T(i18n.KeyComposerCostUnpriced)
					if row.Priced {
						price = fmt.Sprintf("¥%.2f", row.Cents/100)
					}
					ui.Text(c, name+" · "+price).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyComposerCostPeakPriceLine), tokensDetail(*row.Peak))).
						FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyComposerCostOffPeakPriceLine), tokensDetail(*row.OffPeak))).
						FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					if row.Priced {
						p, ok := m.Billing.Settings.Prices[row.Provider+"/"+row.Model]
						if !ok {
							p = m.Billing.Settings.Prices[row.Model]
						}
						ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyComposerCostPriceLine), p.InputCacheHit, p.InputCacheMiss, p.Output)).
							FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
						if !p.Flat && p.Idle != nil {
							ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyComposerCostOffPeakPriceLine),
								fmt.Sprintf("Hit %.6g · Miss %.6g · Output %.6g", p.Idle.InputCacheHit, p.Idle.InputCacheMiss, p.Idle.Output))).
								FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
						}
					}
				}
				if r.Children > 0 {
					if ui.Button(c, childSummary(m, r.Children)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
						m.State.ChildrenOpen = !m.State.ChildrenOpen
					}
					if m.State.ChildrenOpen {
						renderChildren(c, m, muted)
					}
				}
				if m.OnPrices != nil && ui.Button(c, i18n.T(i18n.KeyComposerCostGlobalBilling)).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
					m.OnPrices()
				}
			})
		})
	})
}

func renderChildren(c *ui.Context, m Model, muted ui.Color) {
	selected := map[string]bool{m.Snapshot.SelectedID: true}
	for changed := true; changed; {
		changed = false
		for _, s := range m.Snapshot.CostSources {
			if s.Origin == "subagent" && selected[s.Parent] && !selected[s.ID] {
				selected[s.ID] = true
				changed = true
			}
		}
	}
	seen := map[string]bool{}
	count := 0
	for _, s := range m.Snapshot.CostSources {
		if s.ID == m.Snapshot.SelectedID || !selected[s.ID] || seen[s.ID] || s.Usage == nil || len(s.Usage.Models) == 0 {
			continue
		}
		seen[s.ID] = true
		count++
		if count > 10 {
			continue
		}
		own := cost.Calculate(s.ID, []cost.Source{s}, m.Billing.Settings)
		name := s.Name
		if name == "" {
			name = s.ID
		}
		ui.Text(c, name+" · "+money(own)+" · "+tokensDetail(own.Total)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if count > 10 {
		ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyComposerCostMore), count-10)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
}

func childSummary(m Model, n int) string {
	sources := append([]cost.Source(nil), m.Snapshot.CostSources...)
	for i := range sources {
		if sources[i].ID == m.Snapshot.SelectedID {
			sources[i].Usage = &cost.Projection{Models: []cost.Route{}}
		}
	}
	r := cost.Calculate(m.Snapshot.SelectedID, sources, m.Billing.Settings)
	return fmt.Sprintf("%s · %s · %s ▾", fmt.Sprintf(i18n.T(i18n.KeyComposerCostSubagents), n), money(r), tokensDetail(r.Total))
}
