package settings

import (
	"strings"
	"testing"

	"whaleisle/internal/desktop/providerdiscovery"

	"github.com/egoist/mygo/ui"
)

func TestSP23HostDiscoveryAvailableWithoutEndpoint(t *testing.T) {
	for _, protocol := range []string{"", "google-generative-ai"} {
		t.Run(protocol, func(t *testing.T) {
			calls := 0
			row := providerdiscovery.Row{ID: "native", Name: "Native", Provider: "native", Namespace: "llm-pi-ai", Protocol: protocol, Writable: true, CredentialKnown: true, CredentialConfigured: true}
			var a *ProviderDiscoveryAccess
			a = &ProviderDiscoveryAccess{
				Snapshot: providerdiscovery.Snapshot{Connected: true, Ready: true, Ticket: 11, SelectedID: row.ID, Rows: []providerdiscovery.Row{row}, Protocol: protocol},
				OnDiscover: func(ticket uint64) {
					if ticket != 11 {
						t.Fatalf("wrong action ticket: %d", ticket)
					}
					calls++
				},
			}
			tt := ui.NewTester(func(c *ui.Context) { providerDiscoveryPane(c, a, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150)) }, 1200, 1800)
			text := strings.Join(tt.Texts(), "\n")
			if !strings.Contains(text, "Host · llm-pi-ai · native") || strings.Contains(text, "GET ") || strings.Contains(text, "this probe does not use it") {
				t.Fatalf("wrong Host discovery presentation: %s", text)
			}
			if err := tt.Click(providerCopy("通过 Host 探测模型", "Discover models through Host")); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatal("missing endpoint or native protocol blocked Host discovery")
			}
		})
	}
}

func TestProviderDiscoveryPaneCandidateFilterAndPick(t *testing.T) {
	zhForTest(t)
	row := providerdiscovery.Row{ID: "custom", Name: "Custom", Provider: "custom", Namespace: "llm-pi-ai", Protocol: "openai-completions", Writable: true}
	var picked []string
	adopted := 0
	var a *ProviderDiscoveryAccess
	a = &ProviderDiscoveryAccess{
		Snapshot: providerdiscovery.Snapshot{
			Connected:  true,
			Ready:      true,
			Ticket:     7,
			SelectedID: row.ID,
			Rows:       []providerdiscovery.Row{row},
			Protocol:   row.Protocol,
			Candidates: sampleDiscoveryCandidates()[:3],
		},
		OnEdit: func(_ uint64, field, value string) {
			if field == "query" {
				a.Snapshot.Query = value
			}
		},
		OnPick: func(_ uint64, id string, on bool) {
			picked = append(picked, id)
			for i := range a.Snapshot.Candidates {
				if a.Snapshot.Candidates[i].ID == id {
					a.Snapshot.Candidates[i].Picked = on
				}
			}
		},
		OnAdopt: func(ticket uint64) {
			if ticket == 7 {
				adopted++
			}
		},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(16).Children(func() {
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				providerDiscoveryPane(c, a, tk(c).Foreground, tk(c).MutedForeground)
			})
		})
	}, 900, 1400)
	tt.Frame()
	if !tt.HasText("共 3 项 · 已选 3 项") || !tt.HasText("1M tokens") {
		t.Fatalf("structured candidate summary/badges missing: %v", tt.Texts())
	}
	a.Snapshot.Query = "no-match"
	tt.Frame()
	if !tt.HasText("没有匹配的候选模型") {
		t.Fatalf("empty search state missing in providerDiscoveryPane: %v", tt.Texts())
	}
	if err := tt.Click("清除搜索"); err != nil {
		t.Fatalf("click clear search: %v", err)
	}
	if a.Snapshot.Query != "" {
		t.Fatalf("clear search did not reset query: %q", a.Snapshot.Query)
	}
	if err := tt.Click("取消全选"); err != nil {
		t.Fatalf("click deselect all: %v", err)
	}
	if len(picked) != 3 {
		t.Fatalf("expected 3 pick callbacks on deselect all, got %d (%v)", len(picked), picked)
	}
	// Adopt is disabled when 0 candidates are selected.
	_ = tt.Click("将选中项加入草稿")
	if adopted != 0 {
		t.Fatalf("adopt fired with 0 selected candidates: %d", adopted)
	}
	if err := tt.Click("GPT-6 Astra · routing group"); err != nil {
		t.Fatalf("click candidate row: %v", err)
	}
	if err := tt.Click("将选中项加入草稿"); err != nil {
		t.Fatalf("click adopt: %v", err)
	}
	if adopted != 1 {
		t.Fatalf("expected adopt to fire once, got %d", adopted)
	}
}
