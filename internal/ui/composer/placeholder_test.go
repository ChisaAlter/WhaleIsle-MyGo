package composer

import (
	"testing"
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
