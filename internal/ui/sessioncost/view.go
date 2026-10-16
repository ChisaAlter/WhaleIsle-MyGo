// Package sessioncost renders Host-reported billing counters below the composer.
package sessioncost

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"time"
	"whaleisle/internal/store/session"
	cost "whaleisle/internal/store/sessioncost"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/store/usage"
	fonttheme "whaleisle/internal/ui/theme"
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
	for _, ns := range v.Namespaces {
		if ns.NS != "ui-conversation" {
			continue
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(ns.Value, &fields) != nil {
			continue
		}
		_ = json.Unmarshal(fields["statsLine"], &stats)
		_ = json.Unmarshal(fields["sessionCost"], &show)
		_ = json.Unmarshal(fields["officialPeakValley"], &official)
	}
	// The generic settings form owns pending edits; reflect its current switches.
	if v.SelectedNS == "ui-conversation" {
		for key, value := range v.Drafts {
			switch key {
			case "statsLine":
				if value == "true" || value == "false" {
					stats = value == "true"
				}
			case "sessionCost":
				if value == "true" || value == "false" {
					show = value == "true"
				}
			case "officialPeakValley":
				if value == "true" || value == "false" {
					official = value == "true"
				}
			}
		}
	}
	return
}
func money(r cost.Result) string {
	if !r.Available {
		return "Host 未提供计费用量 / Billing usage unavailable"
	}
	if r.Priced == 0 {
		return "没有设置价格 / No priced routes"
	}
	v := fmt.Sprintf("¥%.2f", r.Cents/100)
	if r.Unpriced > 0 {
		v += "（部分模型未定价 / partly unpriced）"
	}
	return v
}
func tokens(b cost.Bucket) string {
	rate := ""
	if b.Hit+b.Miss > 0 {
		rate = fmt.Sprintf(" (%.1f%%)", 100*b.Hit/(b.Hit+b.Miss))
	}
	return fmt.Sprintf("命中 / Hit %.0f%s · 未命中 / Miss %.0f · 输出 / Output %.0f", b.Hit, rate, b.Miss, b.Output)
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
	r := cost.Calculate(m.Snapshot.SelectedID, m.Snapshot.CostSources, m.Billing.Settings)
	ui.Column(c).FillWidth().Gap(4).Padding(4, 8, 4, 8).Children(func() {
		if stats && settings.ChatPreferencesFrom(m.Preferences).PerformanceUsage != "compact" {
			text := "Host 未提供计费 Token / Billed tokens unavailable"
			if r.Available {
				text = "含子代理计费 Token / Billed tokens incl. subagents · " + tokens(r.Total)
			}
			ui.Text(c, text).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
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
			phase := "空闲 / Off-peak"
			if peak {
				phase = "高峰 / Peak"
			}
			ui.Text(c, fmt.Sprintf("北京时间 %s · 距切换 / Switch in %02d:%02d:%02d", phase, seconds/3600, seconds/60%60, seconds%60)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		label := "会话累计费用 / Session estimate · " + money(r)
		if !m.Billing.Loaded && r.Available {
			label = "会话累计费用 / Session estimate · 价格未加载 / Prices not loaded"
		}
		if ui.Button(c, label).FontSize(fonttheme.InterfaceSize(11)).Clicked() {
			m.State.Open = !m.State.Open
		}
		if !m.Snapshot.ClientReady {
			ui.Text(c, "离线缓存 / Offline cached usage").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if r.Partial {
			ui.Text(c, "仅包含已上报用量的会话及子代理；缺失部分未计入 / Reported sessions and subagents only; missing usage excluded").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if !m.State.Open {
			return
		}
		ui.Scroll(c).Height(180).Children(func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				ui.Text(c, "基于全局配置估算，非账单 / Estimate from global prices, not a bill").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				if m.Billing.Err != "" {
					ui.Text(c, m.Billing.Err).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				}
				for _, row := range r.Rows {
					name := row.Provider + "/" + row.Model
					if name == "/" {
						name = "未知路由 / Unknown route"
					}
					price := "没有设置价格 / Unpriced"
					if row.Priced {
						price = fmt.Sprintf("¥%.2f", row.Cents/100)
					}
					ui.Text(c, name+" · "+price).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					ui.Text(c, "高峰 / Peak · "+tokens(*row.Peak)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					ui.Text(c, "空闲 / Off-peak · "+tokens(*row.OffPeak)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
					if row.Priced {
						p, ok := m.Billing.Settings.Prices[row.Provider+"/"+row.Model]
						if !ok {
							p = m.Billing.Settings.Prices[row.Model]
						}
						ui.Text(c, fmt.Sprintf("CNY / 1M · Hit %.6g · Miss %.6g · Output %.6g", p.InputCacheHit, p.InputCacheMiss, p.Output)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
						if !p.Flat && p.Idle != nil {
							ui.Text(c, fmt.Sprintf("Off-peak CNY / 1M · Hit %.6g · Miss %.6g · Output %.6g", p.Idle.InputCacheHit, p.Idle.InputCacheMiss, p.Idle.Output)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
				if m.OnPrices != nil && ui.Button(c, "消耗统计中的计费设置 / Global billing settings").FontSize(fonttheme.InterfaceSize(11)).Clicked() {
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
		ui.Text(c, name+" · "+money(own)+" · "+tokens(own.Total)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if count > 10 {
		ui.Text(c, fmt.Sprintf("还有 / More: %d", count-10)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
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
	return fmt.Sprintf("子代理 / Subagents %d · %s · %s ▾", n, money(r), tokens(r.Total))
}
