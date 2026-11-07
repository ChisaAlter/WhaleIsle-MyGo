package sessioncost

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	cost "whaleisle/internal/store/sessioncost"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
)

func TestPreferencesIgnoreUncommittedSwitches(t *testing.T) {
	v := settings.ViewState{Namespaces: []settings.Namespace{{NS: "ui-conversation", Value: json.RawMessage(`{"statsLine":false,"sessionCost":true,"officialPeakValley":false}`)}}, SelectedNS: "ui-conversation", Drafts: map[string]string{"statsLine": "true", "sessionCost": "false", "officialPeakValley": "true"}, Dirty: true, Err: "save failed"}
	stats, show, official := preferences(v)
	if stats || !show || official {
		t.Fatalf("draft changed accepted preferences: %v %v %v", stats, show, official)
	}
	v.Namespaces[0].Value = json.RawMessage(`{"statsLine":true,"sessionCost":false,"officialPeakValley":true}`)
	stats, show, official = preferences(v)
	if !stats || show || !official {
		t.Fatal("accepted save did not update preferences")
	}
	v.Namespaces = nil
	stats, show, official = preferences(v)
	if !stats || show || official {
		t.Fatal("old Host preferences leaked")
	}
}

func TestSessionCostPopoverParityRowsZHAndEN(t *testing.T) {
	oldLocale := i18n.Locale()
	defer i18n.SetLocale(oldLocale)

	sources := []cost.Source{
		{
			ID:     "root",
			Origin: "main",
			Usage: &cost.Projection{
				Models: []cost.Route{
					{
						Provider: "deepseek",
						Model:    "deepseek-chat",
						Peak:     &cost.Bucket{Hit: 10000, Miss: 2000, Output: 1000},
						OffPeak:  &cost.Bucket{Hit: 5000, Miss: 0, Output: 500},
					},
					{
						Provider: "acme",
						Model:    "flat-model",
						Peak:     &cost.Bucket{Hit: 1000, Miss: 1000, Output: 500},
						OffPeak:  &cost.Bucket{},
					},
					{
						Provider: "custom",
						Model:    "unpriced-model",
						Peak:     &cost.Bucket{Hit: 0, Miss: 500, Output: 200},
						OffPeak:  &cost.Bucket{},
					},
				},
			},
		},
	}
	for i := 1; i <= 12; i++ {
		sources = append(sources, cost.Source{
			ID:     fmt.Sprintf("sub-%d", i),
			Name:   fmt.Sprintf("Agent-%d", i),
			Origin: "subagent",
			Parent: "root",
			Usage: &cost.Projection{
				Models: []cost.Route{
					{
						Provider: "deepseek",
						Model:    "deepseek-chat",
						Peak:     &cost.Bucket{Hit: 1000, Miss: 1000, Output: 500},
						OffPeak:  &cost.Bucket{},
					},
				},
			},
		})
	}

	billing := usage.BillingView{
		Loaded: true,
		Settings: usage.BillingSettings{
			PeakValleyEnabled: true,
			Prices: map[string]usage.ModelPrice{
				"deepseek/deepseek-chat": {
					PriceRates: usage.PriceRates{
						InputCacheHit:  0.2,
						InputCacheMiss: 2,
						Output:         3,
					},
					Idle: &usage.PriceRates{
						InputCacheHit:  0.1,
						InputCacheMiss: 1,
						Output:         1.5,
					},
				},
				"acme/flat-model": {
					PriceRates: usage.PriceRates{
						InputCacheHit:  0.5,
						InputCacheMiss: 4,
						Output:         8,
					},
					Flat: true,
				},
			},
		},
	}
	prefs := settings.ViewState{
		Namespaces: []settings.Namespace{
			{NS: "ui-conversation", Value: json.RawMessage(`{"statsLine":true,"sessionCost":true,"officialPeakValley":true}`)},
		},
	}

	renderTexts := func(locale string) string {
		i18n.SetLocale(locale)
		st := &State{SessionID: "root", Open: true, ChildrenOpen: true}
		m := Model{
			Snapshot: session.ViewState{
				SelectedID:       "root",
				SelectedProvider: "deepseek",
				SelectedModel:    "deepseek-chat",
				ClientReady:      true,
				CostSources:      sources,
			},
			Billing:     billing,
			Preferences: prefs,
			State:       st,
		}
		tt := ui.NewTester(func(c *ui.Context) {
			View(c, m, ui.RGB(0x81, 0x85, 0x8c))
		}, 900, 700)
		return strings.Join(tt.Texts(), "\n")
	}

	zhText := renderTexts(i18n.LocaleZH)
	for _, want := range []string{
		"会话累计费用",
		"deepseek/deepseek-chat · 费用 ¥",
		"高峰 命中 22K (61.1%) · 未命中 14K · 输出 7K",
		"空闲 命中 5K (100.0%) · 未命中 0 · 输出 500",
		"高峰：命中 0.2 / 未命中 2 / 输出 3（元/百万 tokens）",
		"空闲：命中 0.1 / 未命中 1 / 输出 1.5（元/百万 tokens）",
		"acme/flat-model · 费用 ¥",
		"价格：命中 0.5 / 未命中 4 / 输出 8（元/百万 tokens）",
		"custom/unpriced-model · 没有设置价格",
		"子代理 12 个 · 费用 ¥0.04 · 命中 12K (50.0%) · 未命中 12K · 输出 6K ▾",
		"还有 2 个",
		"在 设置 → 消耗统计 → 计费设置 中设置模型价格",
	} {
		if !strings.Contains(zhText, want) {
			t.Fatalf("zh sessioncost popover missing %q in:\n%s", want, zhText)
		}
	}

	enText := renderTexts(i18n.LocaleEN)
	for _, want := range []string{
		"Session cost",
		"deepseek/deepseek-chat · Cost ¥",
		"Peak hit 22K (61.1%) · miss 14K · output 7K",
		"Off-peak hit 5K (100.0%) · miss 0 · output 500",
		"Peak: hit 0.2 / miss 2 / output 3 (CNY per million tokens)",
		"Off-peak: hit 0.1 / miss 1 / output 1.5 (CNY per million tokens)",
		"acme/flat-model · Cost ¥",
		"Price: hit 0.5 / miss 4 / output 8 (CNY per million tokens)",
		"custom/unpriced-model · No price set",
		"Subagents (12) · Cost ¥0.04 · Hit 12K (50.0%) · miss 12K · output 6K ▾",
		"2 more",
		"Set model prices in Settings → Usage stats → Billing settings",
	} {
		if !strings.Contains(enText, want) {
			t.Fatalf("en sessioncost popover missing %q in:\n%s", want, enText)
		}
	}
}
