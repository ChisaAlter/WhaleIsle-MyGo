package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestComposerPlanPlaceholderUsesLiveEffectiveTarget(t *testing.T) {
	m := Model{State: NewDraftState(""), Snapshot: session.ViewState{ClientReady: true, SelectedID: "s", PlanModeControl: session.PlanModeControl{Present: true, Effective: true}}}
	if got := composerPlaceholder(m); got != i18n.T(i18n.KeyComposerPlaceholderPlan) {
		t.Fatal(got)
	}
	m.Snapshot.PlanModeControl.Effective = false
	if got := composerPlaceholder(m); got != i18n.T(i18n.KeyComposerPlaceholder) {
		t.Fatal("pending off did not restore default", got)
	}
	m.Snapshot.PlanModeControl.Effective = true
	m.Hero = true
	if composerPlaceholder(m) != i18n.T(i18n.KeyComposerPlaceholderHero) {
		t.Fatal("plan replaced hero")
	}
	m.Hero = false
	m.Snapshot.ClientReady = false
	if composerPlaceholder(m) == i18n.T(i18n.KeyComposerPlaceholderPlan) {
		t.Fatal("disconnected state leaked plan hint")
	}
	m.Snapshot.ClientReady = true
	m.Snapshot.PlanBoard.Queue = []session.PlanQueueItem{{ID: "q"}}
	m.Snapshot.PlanBoard.QueueActions = session.QueueActionState{Mutable: true, Running: true}
	if composerPlaceholder(m) != i18n.T(i18n.KeyComposerPlaceholderSteerQueue) {
		t.Fatal("plan hid available queue action")
	}
	m.State.Preparing = 1
	if composerPlaceholder(m) != i18n.T(i18n.KeyComposerPlaceholderPlan) {
		t.Fatal("unavailable queue action advertised")
	}
}

func TestComposerPlaceholderParityVariantsAndNotices(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		i18n.SetLocale(locale)
		if got := composerPlaceholder(Model{Inert: true, Hero: true}); got != WorkspacePlaceholder() {
			t.Fatalf("locale %s workspace placeholder=%q want %q", locale, got, WorkspacePlaceholder())
		}
		if got := composerPlaceholder(Model{Disabled: true}); got != UnavailablePlaceholder() {
			t.Fatalf("locale %s unavailable placeholder=%q want %q", locale, got, UnavailablePlaceholder())
		}
		if got := composerPlaceholder(Model{ParentOffline: true}); got != ParentOfflinePlaceholder() {
			t.Fatalf("locale %s parentOffline placeholder=%q want %q", locale, got, ParentOfflinePlaceholder())
		}
		continuableOffline := Model{
			Snapshot: session.ViewState{
				SelectedID: "child",
				Items: []session.Item{
					{SessionID: "parent", AgentAvailable: false},
					{SessionID: "child", ParentID: "parent", Origin: "subagent", SubagentIdentity: &session.SessionSubagentIdentity{Mode: "continuable"}},
				},
			},
		}
		if got := composerPlaceholder(continuableOffline); got != ParentOfflinePlaceholder() {
			t.Fatalf("locale %s continuable child with offline parent placeholder=%q want %q", locale, got, ParentOfflinePlaceholder())
		}
		continuableOnline := continuableOffline
		continuableOnline.Snapshot.Items[0].AgentAvailable = true
		if got := composerPlaceholder(continuableOnline); got == ParentOfflinePlaceholder() {
			t.Fatalf("locale %s online parent still showed parentOffline placeholder", locale)
		}
		managed := Model{
			Snapshot: session.ViewState{
				ClientReady: true,
				SelectedID:  "m",
				Items: []session.Item{
					{SessionID: "m", Presentation: &session.SessionPresentation{Composer: "managed"}},
				},
			},
		}
		if got := composerPlaceholder(managed); got != ManagedPlaceholder() {
			t.Fatalf("locale %s managed placeholder=%q want %q", locale, got, ManagedPlaceholder())
		}
		if HeroHeadline() != i18n.T(i18n.KeyChatHeroSlogan) || HeroBadge() != i18n.T(i18n.KeyChatHeroPreview) {
			t.Fatalf("locale %s hero headline/badge mismatch: %q / %q", locale, HeroHeadline(), HeroBadge())
		}
		if got := FormatComposerError("rpc error: session/writer-held: busy"); got != SessionInUseNotice() {
			t.Fatalf("locale %s writer-held notice=%q want %q", locale, got, SessionInUseNotice())
		}
	}
}

func TestParentOfflineBlocksSendWhileKeepingStopAndOccupiedNotice(t *testing.T) {
	d := NewDraftState("")
	d.Error = "session/writer-held"
	stopped, sent := 0, 0
	m := Model{
		State:              d,
		Draft:              &d.Text,
		TextInput:          d.Client,
		UseTextInputClient: true,
		ParentOffline:      true,
		Snapshot: session.ViewState{
			ClientReady: true,
			SelectedID:  "child",
			PromptBusy:  true,
			Items:       []session.Item{{SessionID: "child", Origin: "subagent", Running: true}},
		},
		OnSend: func(string) { sent++ },
		OnStop: func() { stopped++ },
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 700, 300)
	if !tt.HasText(ParentOfflinePlaceholder()) {
		t.Fatalf("parentOffline placeholder missing: %v", tt.Texts())
	}
	if !tt.HasText(SessionInUseNotice()) {
		t.Fatalf("session/writer-held notice not translated: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyComposerStop)); err != nil || stopped != 1 || sent != 0 {
		t.Fatalf("parentOffline should allow stopping active run: stopped=%d sent=%d err=%v", stopped, sent, err)
	}

	childDraft := NewDraftState("")
	childDraft.Error = "session/writer-held"
	childTT := ui.NewTester(func(c *ui.Context) {
		ChildEditor(c, ChildEditorModel{Draft: childDraft, Disabled: true})
	}, 600, 260)
	if !childTT.HasText(ParentOfflinePlaceholder()) {
		t.Fatalf("disabled ChildEditor did not render parentOffline placeholder: %v", childTT.Texts())
	}
	if !childTT.HasText(SessionInUseNotice()) {
		t.Fatalf("ChildEditor did not translate session/writer-held notice: %v", childTT.Texts())
	}
}
