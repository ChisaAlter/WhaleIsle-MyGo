package settings

import (
	"image/color"
	"testing"

	"whaleisle/internal/desktop/providerdiscovery"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

func sampleDiscoveryCandidates() []providerdiscovery.Candidate {
	return []providerdiscovery.Candidate{
		{ID: "auto/gpt-6-astra", Name: "GPT-6 Astra · routing group", ContextWindow: 1000000, Picked: true},
		{ID: "auto/gpt-6-luna", Name: "GPT-6 Luna · routing group", ContextWindow: 1000000, Picked: true},
		{ID: "auto/gpt-6-sol", Name: "GPT-6 Sol · routing group", ContextWindow: 1000000, Picked: true},
		{ID: "auto/deepseek-v4.1-flash-1017", Name: "DeepSeek V4.1 Flash 1017 · routing group", ContextWindow: 1048576, Picked: true},
		{ID: "auto/claude-sonnet-5-5", Name: "Claude Sonnet 5.5 · routing group", ContextWindow: 1000000, Picked: true},
		{ID: "auto/claude-opus-5-5", Name: "Claude Opus 5.5 · routing group", ContextWindow: 1000000, Picked: true},
		{ID: "ai-apizn-com/gpt-6.2-sol", Name: "GPT-6.2 Sol · ai.apizn.com", ContextWindow: 922000, Picked: true},
		{ID: "claude-2/claude-haiku-5-5", Name: "Haiku 5.5 · Claude", ContextWindow: 1000000, Picked: true},
	}
}

func TestFormatCandidateTokensAndMatching(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, ""},
		{-1, ""},
		{512, "512"},
		{8192, "8K"},
		{128000, "128K"},
		{131072, "128K"},
		{922000, "922K"},
		{1000000, "1M"},
		{1048576, "1M"},
		{1500000, "1.5M"},
		{2097152, "2M"},
	}
	for _, tc := range cases {
		if got := formatCandidateTokens(tc.in); got != tc.want {
			t.Errorf("formatCandidateTokens(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}

	c := providerdiscovery.Candidate{ID: "claude-2/claude-haiku-5-5", Name: "Haiku 5.5 · Claude"}
	if !candidateMatches(c, "") || !candidateMatches(c, "haiku") || !candidateMatches(c, "claude 5.5") || !candidateMatches(c, "CLAUDE-2") {
		t.Fatal("expected candidateMatches to match case-insensitive multi-term queries")
	}
	if candidateMatches(c, "haiku opus") || candidateMatches(c, "gpt-6") {
		t.Fatal("unexpected match for non-matching query")
	}
}

func TestModelCandidateDialogLayoutButtonsAndScroll(t *testing.T) {
	zhForTest(t)
	state := &modelProbeState{candidates: sampleDiscoveryCandidates()}
	var rows []any
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(tk(c).Background).Children(func() {
			modelCandidateDialog(c, state, rows, false, true, func(next []any) { rows = next })
		})
	}, 1100, 800)
	tt.Frame()

	// Verify footer button hierarchy: Cancel (outline) to the left of Adopt (primary), both 32 DIP high.
	interactiveAt := func(r ui.Rect) ui.Rect {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		best := ui.Rect{}
		for _, box := range tt.Interactives() {
			if cx >= box.Rect.X && cx <= box.Rect.X+box.Rect.W && cy >= box.Rect.Y && cy <= box.Rect.Y+box.Rect.H {
				if best.W == 0 || box.Rect.W*box.Rect.H < best.W*best.H {
					best = box.Rect
				}
			}
		}
		return best
	}
	cancelText := buttonRect(tt.FindAll("取消"))
	adoptText := buttonRect(tt.FindAll("采纳选中模型"))
	cancelRect := interactiveAt(cancelText)
	adoptRect := interactiveAt(adoptText)
	if cancelRect.W == 0 || adoptRect.W == 0 {
		t.Fatalf("footer buttons missing: cancel=%+v adopt=%+v texts=%v", cancelRect, adoptRect, tt.Texts())
	}
	if cancelRect.X+cancelRect.W > adoptRect.X {
		t.Fatalf("expected secondary Cancel to the left of primary Adopt: cancel=%+v adopt=%+v", cancelRect, adoptRect)
	}
	if cancelRect.H != 32 || adoptRect.H != 32 {
		t.Fatalf("expected coss ButtonMD height 32, got cancel=%g adopt=%g", cancelRect.H, adoptRect.H)
	}

	// Verify primary Adopt button paints a distinct filled background vs outline Cancel button.
	samplePixel := func(r ui.Rect) color.RGBA {
		return tt.Image().RGBAAt(int(r.X+6), int(r.Y+r.H/2))
	}
	if samplePixel(adoptRect) == samplePixel(cancelRect) {
		t.Fatalf("primary Adopt and secondary Cancel share the same fill %v", samplePixel(adoptRect))
	}

	// Verify structured single-line rows: title, ID, and compact token badge do not wrap or overlap.
	titleNode := titleRect(tt.FindAll("DeepSeek V4.1 Flash 1017 · routing group"))
	idRect, okID := tt.Find("auto/deepseek-v4.1-flash-1017")
	if titleNode.W == 0 || !okID {
		t.Fatalf("structured candidate title/id missing: title=%+v id=%v texts=%v", titleNode, okID, tt.Texts())
	}
	if titleNode.H > design.TextSM.LineHeight+1 {
		t.Fatalf("candidate title wrapped unexpectedly: h=%g want <= %g", titleNode.H, design.TextSM.LineHeight)
	}
	if idRect.H > design.TextXS.LineHeight+1 {
		t.Fatalf("candidate ID wrapped unexpectedly: h=%g want <= %g", idRect.H, design.TextXS.LineHeight)
	}
	if idRect.Y <= titleNode.Y {
		t.Fatalf("expected ID beneath title: title=%+v id=%+v", titleNode, idRect)
	}
	if !tt.HasText("1M tokens") {
		t.Fatalf("expected compact token badge '1M tokens' in %v", tt.Texts())
	}
	for _, badgeRect := range tt.FindAll("1M tokens") {
		if badgeRect.H > 20 {
			t.Fatalf("token badge wrapped onto multiple lines: %+v", badgeRect)
		}
	}

	// Verify the 6th candidate sits fully above the footer without being half-sliced,
	// and scrolling brings the 8th candidate into view while keeping search & footer sticky.
	searchBefore, okSearch := tt.Find("搜索候选模型")
	if !okSearch {
		t.Fatalf("search input missing: %v", tt.Texts())
	}
	opusRect, okOpus := tt.Find("Claude Opus 5.5 · routing group")
	opusIDRect, okOpusID := tt.Find("auto/claude-opus-5-5")
	if !okOpus || !okOpusID {
		t.Fatalf("6th candidate missing: opus=%v id=%v", okOpus, okOpusID)
	}
	if opusIDRect.Y+opusIDRect.H >= cancelRect.Y {
		t.Fatalf("6th candidate overflows into footer: opusID=%+v footer=%+v", opusIDRect, cancelRect)
	}

	// Scroll the candidate list downward and check sticky header/search/footer.
	tt.Scroll(opusRect.X+opusRect.W/2, opusRect.Y, 0, 120)
	tt.Frame()
	if state.scroll.Y <= 0 {
		t.Fatalf("candidate list did not scroll: scroll=%+v", state.scroll)
	}
	searchAfter, _ := tt.Find("搜索候选模型")
	cancelAfter := interactiveAt(buttonRect(tt.FindAll("取消")))
	if searchAfter.Y != searchBefore.Y || cancelAfter.Y != cancelRect.Y {
		t.Fatalf("search or footer moved during list scroll: search %g->%g cancel %g->%g", searchBefore.Y, searchAfter.Y, cancelRect.Y, cancelAfter.Y)
	}

	// Audit cursors across all interactive controls in the dialog.
	assertInteractiveCursors(t, "model-picker-dialog", tt)
}

func TestModelCandidateDialogSearchEmptyAndSelectionFlow(t *testing.T) {
	zhForTest(t)
	state := &modelProbeState{candidates: sampleDiscoveryCandidates()}
	// Mark one model as already present in the provider catalog.
	existing := []any{map[string]any{"id": "auto/gpt-6-luna", "name": "GPT-6 Luna · routing group"}}
	state.candidates[1].Picked = false
	var adopted []any

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(tk(c).Background).Children(func() {
			modelCandidateDialog(c, state, existing, false, true, func(next []any) { adopted = next })
		})
	}, 1100, 800)
	tt.Frame()

	// Already-configured candidate displays the Added badge.
	if !tt.HasText("已添加") {
		t.Fatalf("expected '已添加' badge for existing model: %v", tt.Texts())
	}
	if !tt.HasText("共 8 项 · 已选 7 项") {
		t.Fatalf("expected initial summary count, got: %v", tt.Texts())
	}

	// Toggle select all -> selects the remaining unchecked candidate (8/8).
	if err := tt.Click("全选"); err != nil {
		t.Fatalf("click select all: %v", err)
	}
	if !tt.HasText("共 8 项 · 已选 8 项") || !tt.HasText("取消全选") {
		t.Fatalf("select all did not update state: %v", tt.Texts())
	}

	// Filter to Claude models (3 matches) and deselect only filtered matches.
	state.query = "claude"
	tt.Frame()
	if !tt.HasText("匹配 3 / 8 项 · 已选 8 项") {
		t.Fatalf("filtered summary mismatch: %v", tt.Texts())
	}
	if err := tt.Click("取消全选筛选"); err != nil {
		t.Fatalf("click deselect filtered: %v", err)
	}
	if !tt.HasText("匹配 3 / 8 项 · 已选 5 项") {
		t.Fatalf("deselecting filtered changed wrong candidates: %v", tt.Texts())
	}

	// Non-matching query renders empty state, disables select-all toggle, and clear search recovers.
	state.query = "no-such-model-xyz"
	tt.Frame()
	if !tt.HasText("没有匹配的候选模型") {
		t.Fatalf("empty search state missing: %v", tt.Texts())
	}
	// Clicking the disabled toggle button must not change selection.
	_ = tt.Click("全选筛选结果")
	pickedCount := 0
	for _, cand := range state.candidates {
		if cand.Picked {
			pickedCount++
		}
	}
	if pickedCount != 5 {
		t.Fatalf("disabled toggle mutated selection on empty search: picked=%d", pickedCount)
	}
	if err := tt.Click("清除搜索"); err != nil {
		t.Fatalf("click clear search: %v", err)
	}
	if state.query != "" || !tt.HasText("共 8 项 · 已选 5 项") {
		t.Fatalf("clear search did not restore full list: query=%q texts=%v", state.query, tt.Texts())
	}

	// Deselect everything -> Adopt button is disabled and clicking it does not submit.
	for i := range state.candidates {
		state.candidates[i].Picked = false
	}
	tt.Frame()
	_ = tt.Click("采纳选中模型")
	if adopted != nil || len(state.candidates) == 0 {
		t.Fatalf("adopt succeeded with 0 selected candidates: adopted=%v remaining=%d", adopted, len(state.candidates))
	}

	// Click a single candidate row directly to pick it, then adopt.
	if err := tt.Click("GPT-6 Astra · routing group"); err != nil {
		t.Fatalf("click candidate row: %v", err)
	}
	if !state.candidates[0].Picked {
		t.Fatal("clicking candidate row did not toggle Picked")
	}
	if err := tt.Click("采纳选中模型"); err != nil {
		t.Fatalf("click adopt: %v", err)
	}
	if len(adopted) != 2 {
		t.Fatalf("expected existing + 1 adopted model, got %d (%v)", len(adopted), adopted)
	}
	if len(state.candidates) != 0 || state.query != "" {
		t.Fatalf("dialog did not clear state after adoption: candidates=%d query=%q", len(state.candidates), state.query)
	}
}

func TestModelCandidateDialogDismissThemesAndSavingLock(t *testing.T) {
	zhForTest(t)
	// Test Escape dismissal and saving lock.
	saving := true
	state := &modelProbeState{candidates: sampleDiscoveryCandidates()[:3]}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(tk(c).Background).Children(func() {
			modelCandidateDialog(c, state, nil, saving, true, func([]any) {
				t.Fatal("change should not fire on cancel or while saving")
			})
		})
	}, 900, 700)
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if len(state.candidates) != 3 {
		t.Fatal("Escape closed the dialog while saving was in progress")
	}
	saving = false
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if len(state.candidates) != 0 {
		t.Fatal("Escape did not dismiss the dialog when idle")
	}

	// Test Light and Dark themes plus narrow viewport containment.
	for _, dark := range []bool{false, true} {
		state := &modelProbeState{candidates: sampleDiscoveryCandidates()}
		var wantPrimary, wantPopover ui.Color
		tt := ui.NewTester(func(c *ui.Context) {
			tokens := tk(c)
			wantPrimary, wantPopover = tokens.Primary, tokens.Popover
			ui.Box(c).Fill().Background(tokens.Background).Children(func() {
				modelCandidateDialog(c, state, nil, false, true, func([]any) {})
			})
		}, 420, 500)
		tt.SetDark(dark)
		tt.Frame()
		adopt := buttonRect(tt.FindAll("采纳选中模型"))
		cancel := buttonRect(tt.FindAll("取消"))
		if adopt.W == 0 || cancel.W == 0 {
			t.Fatalf("dark=%v narrow viewport missing buttons: adopt=%+v cancel=%+v", dark, adopt, cancel)
		}
		if adopt.X+adopt.W > 420 || cancel.X < 0 || adopt.Y+adopt.H > 500 {
			t.Fatalf("dark=%v footer buttons clipped viewport 420x500: cancel=%+v adopt=%+v", dark, cancel, adopt)
		}
		got := tt.Image().RGBAAt(int(adopt.X+6), int(adopt.Y+adopt.H/2))
		if got.R == wantPopover.R && got.G == wantPopover.G && got.B == wantPopover.B {
			t.Fatalf("dark=%v primary button blended into popover surface %v (want primary %v)", dark, got, wantPrimary)
		}
	}
}

func TestModelListDiscoveryFetchTriggerAndCancel(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	probe := &modelProbeState{}
	stopped := false
	var deliver func([]providerdiscovery.Candidate, error)
	m := Model{
		Models: &ModelsAccess{
			configurationProbe: probe,
			ProviderDiscovery: &ProviderDiscoveryAccess{
				OnProbeEditor: func(_ providerdiscovery.EditorProbe, done func([]providerdiscovery.Candidate, error)) func() {
					deliver = done
					return func() { stopped = true }
				},
			},
		},
	}
	s := store.ViewState{
		SelectedNS:  "llm-pi-ai",
		ClientReady: true,
		Writable:    true,
		Drafts:      map[string]string{"providers": `{"custom":{"baseURL":"https://api.example/v1","api":"openai-completions"}}`},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Children(func() {
			modelListDiscovery(c, m, s, "custom", nil, func([]any) {}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 700)
	if err := tt.Click("Fetch available models"); err != nil {
		t.Fatal(err)
	}
	if !probe.busy || deliver == nil {
		t.Fatal("fetch did not start")
	}
	if err := tt.Click("Cancel fetch"); err != nil {
		t.Fatal(err)
	}
	if probe.busy || !stopped {
		t.Fatalf("cancel fetch did not stop probe: busy=%v stopped=%v", probe.busy, stopped)
	}
}
