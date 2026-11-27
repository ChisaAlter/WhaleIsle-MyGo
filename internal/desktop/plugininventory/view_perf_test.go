package plugininventory

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"whaleisle/internal/dsh/api"
)

func largeInventoryWire(tb testing.TB, globalCount, presetCount, rowsPerPreset int) api.PluginInventorySnapshot {
	tb.Helper()
	entries := make([]map[string]any, 0, globalCount)
	for i := 0; i < globalCount; i++ {
		var title any = map[string]string{
			"en": fmt.Sprintf("Global Plugin %d", i),
			"zh": fmt.Sprintf("全局插件 %d", i),
		}
		if i%5 == 0 {
			title = fmt.Sprintf("@deepseek-ai/dsh-host-global-%d", i)
		}
		entries = append(entries, map[string]any{
			"entryId":    fmt.Sprintf("include:global-%d", i),
			"moduleName": fmt.Sprintf("@deepseek-ai/dsh-host-global-%d", i),
			"enabled":    i%4 != 0,
			"fiberPhase": "active",
			"meta": map[string]any{
				"title": title,
				"description": map[string]string{
					"en": fmt.Sprintf("Description for global plugin %d", i),
					"zh": fmt.Sprintf("全局插件 %d 的详细功能说明", i),
				},
			},
		})
	}
	presets := make([]map[string]any, 0, presetCount)
	for p := 0; p < presetCount; p++ {
		rows := make([]map[string]any, 0, rowsPerPreset)
		for r := 0; r < rowsPerPreset; r++ {
			rows = append(rows, map[string]any{
				"entryId":    fmt.Sprintf("include:session-%d-%d", p, r),
				"moduleName": fmt.Sprintf("@deepseek-ai/dsh-host-session-%d-%d", p, r),
				"enabled":    true,
				"fiberPhase": "active",
				"meta": map[string]any{
					"title": map[string]string{
						"en": fmt.Sprintf("Session Plugin %d-%d", p, r),
						"zh": fmt.Sprintf("会话插件 %d-%d", p, r),
					},
					"description": map[string]string{
						"en": fmt.Sprintf("Session description %d-%d", p, r),
						"zh": fmt.Sprintf("会话插件 %d-%d 的功能说明", p, r),
					},
				},
			})
		}
		presets = append(presets, map[string]any{
			"id":        fmt.Sprintf("preset-%d", p),
			"name":      fmt.Sprintf("Preset %d", p),
			"isDefault": p == 0,
			"rows":      rows,
		})
	}
	raw, err := json.Marshal(map[string]any{
		"managementAvailable": true,
		"entries":             entries,
		"agentPresets":        presets,
	})
	if err != nil {
		tb.Fatal(err)
	}
	var wire api.PluginInventorySnapshot
	if err := json.Unmarshal(raw, &wire); err != nil {
		tb.Fatal(err)
	}
	return wire
}

func TestPluginInventoryViewCachedZeroAlloc(t *testing.T) {
	wire := largeInventoryWire(t, 40, 4, 35)
	var c Controller
	c.Attach(fixtureCaller(func(context.Context, string, any, any) error { return nil }))
	host := c.Snapshot().Value.Host
	c.Accept(host, 1, wire)

	snap := c.Snapshot().Value
	v := snap.View("zh-CN")
	if len(v.Global) != 40 || len(v.Presets) != 1 || len(v.Presets[0].Rows) != 35 {
		t.Fatalf("unexpected view sizes: global=%d presets=%d", len(v.Global), len(v.Presets))
	}
	if v.Global[0].Title != "global-0" || v.Global[1].Title != "全局插件 1" {
		t.Fatalf("unexpected resolved titles: [0]=%q [1]=%q", v.Global[0].Title, v.Global[1].Title)
	}

	// Repeated render frames on the same snapshot must hit the view cache with 0 allocations.
	if allocs := testing.AllocsPerRun(200, func() {
		got := snap.View("zh-CN")
		if len(got.Presets) != 1 {
			t.Fatal("cached view lost preset")
		}
	}); allocs != 0 {
		t.Fatalf("cached Snapshot.View allocated %g times/run, want 0", allocs)
	}

	// Toggling expanded or global-open must not invalidate the cached View projection.
	c.ToggleExpanded(host, v.Presets[0].Rows[0].Key)
	c.ToggleGlobal(host)
	snapAfterToggle := c.Snapshot().Value
	if allocs := testing.AllocsPerRun(200, func() {
		_ = snapAfterToggle.View("zh-CN")
	}); allocs != 0 {
		t.Fatalf("Snapshot.View after ToggleExpanded/ToggleGlobal allocated %g times/run, want 0", allocs)
	}

	// Switching query or locale updates results accurately and caches the new key.
	c.SetQuery(host, "会话插件 0-12")
	snapSearch := c.Snapshot().Value
	gotSearch := snapSearch.View("zh-CN")
	if !gotSearch.Searching || len(gotSearch.Presets) != 1 || len(gotSearch.Presets[0].Rows) != 1 || gotSearch.Presets[0].Rows[0].Title != "会话插件 0-12" {
		t.Fatalf("search view mismatch: %+v", gotSearch)
	}
	if allocs := testing.AllocsPerRun(200, func() {
		_ = snapSearch.View("zh-CN")
	}); allocs != 0 {
		t.Fatalf("cached search Snapshot.View allocated %g times/run, want 0", allocs)
	}

	// Uncached query rebuilds use pre-parsed localized fields without json.Unmarshal blow-up.
	uncached := snap
	uncached.views = nil
	uncachedAllocs := testing.AllocsPerRun(50, func() {
		_ = uncached.View("zh-CN")
	})
	// Previously 180 entries * 3 json.Unmarshal calls generated >1500 allocations/call.
	if uncachedAllocs > 40 {
		t.Fatalf("uncached pre-parsed Snapshot.View allocated %g times/run (json.Unmarshal regression?)", uncachedAllocs)
	}
}

func BenchmarkPluginInventoryViewCached(b *testing.B) {
	wire := largeInventoryWire(b, 40, 4, 35)
	var c Controller
	c.Attach(fixtureCaller(func(context.Context, string, any, any) error { return nil }))
	c.Accept(c.Snapshot().Value.Host, 1, wire)
	snap := c.Snapshot().Value
	_ = snap.View("zh-CN")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = snap.View("zh-CN")
	}
}

func BenchmarkPluginInventoryViewUncached(b *testing.B) {
	wire := largeInventoryWire(b, 40, 4, 35)
	var c Controller
	c.Attach(fixtureCaller(func(context.Context, string, any, any) error { return nil }))
	c.Accept(c.Snapshot().Value.Host, 1, wire)
	snap := c.Snapshot().Value
	snap.views = nil

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = snap.View("zh-CN")
	}
}
