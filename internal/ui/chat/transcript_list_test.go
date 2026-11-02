package chat

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func transcriptHistoryMessages(start, count int) []session.Message {
	messages := make([]session.Message, count)
	for i := range messages {
		n := start + i
		messages[i] = session.Message{ID: fmt.Sprintf("history-%03d", n), Seq: float64(n + 1), Role: "assistant", Kind: session.MessageKindText, Body: fmt.Sprintf("History row %03d", n)}
	}
	return messages
}

func transcriptHistoryFixture(t *testing.T, count int) (*ui.Tester, *Model, *int) {
	t.Helper()
	t.Setenv(paths.EnvUserData, t.TempDir())
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale("zh") })
	calls := new(int)
	m := &Model{
		List: &ui.ListState{FollowEnd: true}, Scroll: &ui.ScrollState{},
		Inspector:      &InspectorNavigation{SessionID: "history"},
		SuppressStatus: true, FullColumnViewport: true, ContentWidth: 520,
		ThemeID: theme.IDDeepseekLight, Preferences: settings.ChatPreferences{TranscriptView: "verbose"},
		Snapshot: session.ViewState{SelectedID: "history", PlanHistoryBinding: "fixture-host", ClientReady: true, HistoryHasMore: true, Messages: transcriptHistoryMessages(100, count)},
	}
	m.OnLoadEarlier = func() { *calls++; m.Snapshot.HistoryPageBusy = true }
	tt := ui.NewTester(func(c *ui.Context) { View(c, *m) }, 700, 360)
	tt.SetTime(time.Unix(9000, 0))
	return tt, m, calls
}

func historyButton(t *testing.T, tt *ui.Tester, busy bool) ui.Rect {
	t.Helper()
	label := i18n.T(i18n.KeyChatLoadEarlier)
	if busy {
		label = i18n.T(i18n.KeyChatLoadEarlierBusy)
	}
	box, ok := tt.Find(label)
	if !ok {
		t.Fatalf("history button %q not visible; labels=%v", label, tt.Texts())
	}
	return box
}

func TestTranscriptHistoryControlScrollsWithVirtualRowsAndLoadsOnce(t *testing.T) {
	tt, m, calls := transcriptHistoryFixture(t, 80)
	if *calls != 0 {
		t.Fatal("history fetched on mount")
	}
	// Start following the tail, then use actual wheel events into the middle.
	tt.Scroll(400, 200, 0, -900)
	if _, visible := tt.Find(i18n.T(i18n.KeyChatLoadEarlier)); visible {
		t.Fatal("history button fixed above the scrollport in the middle")
	}
	if m.Scroll.Y <= 0 || m.List.AtEnd() {
		t.Fatalf("fixture did not leave the tail: %+v", m.Scroll)
	}
	tt.Scroll(400, 200, 0, -100000)
	before := historyButton(t, tt, false)
	if first, _ := m.List.Visible(); first != 0 {
		t.Fatalf("loader is not the first virtual row: %d", first)
	}
	tt.Scroll(400, 200, 0, 6)
	after := historyButton(t, tt, false)
	if math.Abs(float64(before.Y-after.Y-6)) > .1 {
		t.Fatalf("button did not move with transcript: before=%+v after=%+v", before, after)
	}
	if *calls != 0 {
		t.Fatal("scrolling automatically fetched history")
	}
	tt.Scroll(400, 200, 0, -100)
	if err := tt.Click(i18n.T(i18n.KeyChatLoadEarlier)); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 {
		t.Fatalf("one click triggered %d loads", *calls)
	}
	busy := historyButton(t, tt, true)
	for range 3 {
		tt.ClickAt(busy.X+busy.W/2, busy.Y+busy.H/2)
		tt.Frame()
	}
	if *calls != 1 {
		t.Fatalf("busy clicks triggered another load: %d", *calls)
	}
}

func TestTranscriptHistoryPrependKeepsMessageAnchorAndRemoval(t *testing.T) {
	for _, exhausted := range []bool{false, true} {
		t.Run(fmt.Sprintf("exhausted=%v", exhausted), func(t *testing.T) {
			tt, m, calls := transcriptHistoryFixture(t, 60)
			tt.Scroll(400, 200, 0, -100000)
			before, ok := tt.Find("History row 100")
			if !ok {
				t.Fatal("first loaded message not visible")
			}
			if err := tt.Click(i18n.T(i18n.KeyChatLoadEarlier)); err != nil {
				t.Fatal(err)
			}
			m.Snapshot.Messages = append(transcriptHistoryMessages(80, 20), m.Snapshot.Messages...)
			m.Snapshot.HistoryPageBusy = false
			m.Snapshot.HistoryHasMore = !exhausted
			tt.Frame()
			after, ok := tt.Find("History row 100")
			if !ok || math.Abs(float64(after.Y-before.Y)) > .1 || *calls != 1 {
				t.Fatalf("prepend lost original visible anchor: before=%+v after=%+v visible=%v calls=%d", before, after, ok, *calls)
			}
			if m.List.AtEnd() {
				t.Fatal("history response resumed following the tail")
			}
			if exhausted {
				tt.Scroll(400, 200, 0, -100000)
				if _, visible := tt.Find(i18n.T(i18n.KeyChatLoadEarlier)); visible {
					t.Fatal("exhausted history retained the loader row")
				}
			}
		})
	}
}

func TestTranscriptHistoryFailureRetryAndEmptyLoadableList(t *testing.T) {
	tt, m, calls := transcriptHistoryFixture(t, 0)
	historyButton(t, tt, false)
	if err := tt.Click(i18n.T(i18n.KeyChatLoadEarlier)); err != nil {
		t.Fatal(err)
	}
	m.Snapshot.HistoryPageBusy = false
	m.Snapshot.HistoryPageErr = "Earlier page is unavailable"
	tt.Frame()
	if !tt.HasText(m.Snapshot.HistoryPageErr) {
		t.Fatal("page failure disappeared from scrollable history control")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatLoadEarlier)); err != nil {
		t.Fatal(err)
	}
	if *calls != 2 {
		t.Fatalf("failed page could not retry explicitly: calls=%d", *calls)
	}
	m.Snapshot.HistoryPageBusy = false
	m.Snapshot.HistoryHasMore = false
	m.Snapshot.HistoryPageErr = ""
	m.Snapshot.Messages = transcriptHistoryMessages(90, 1)
	tt.Frame()
	if _, visible := tt.Find(i18n.T(i18n.KeyChatLoadEarlier)); visible || !tt.HasText("History row 090") {
		t.Fatal("empty loadable transcript did not settle to its loaded message")
	}
}

func TestTranscriptHistoryFastPageWithoutBusySnapshotCanLoadAgain(t *testing.T) {
	tt, m, calls := transcriptHistoryFixture(t, 40)
	tt.Scroll(400, 200, 0, -100000)
	// Simulate a synchronous/cached completion whose busy transition is
	// coalesced before the next snapshot; no visible rows need be added.
	m.OnLoadEarlier = func() { *calls++ }
	for want := 1; want <= 2; want++ {
		if err := tt.Click(i18n.T(i18n.KeyChatLoadEarlier)); err != nil {
			t.Fatal(err)
		}
		if *calls != want {
			t.Fatalf("activation repeated or remained latched: calls=%d want=%d", *calls, want)
		}
		tt.Frame()
		if *calls != want {
			t.Fatal("idle frame repeated a completed page activation")
		}
	}
}

func TestTranscriptHistoryExhaustedEmptyPageRemovesLoaderAtContentStart(t *testing.T) {
	tt, m, _ := transcriptHistoryFixture(t, 50)
	tt.Scroll(400, 200, 0, -100000)
	if err := tt.Click(i18n.T(i18n.KeyChatLoadEarlier)); err != nil {
		t.Fatal(err)
	}
	m.Snapshot.HistoryPageBusy = false
	m.Snapshot.HistoryHasMore = false
	tt.Frame()
	if _, visible := tt.Find(i18n.T(i18n.KeyChatLoadEarlier)); visible {
		t.Fatal("empty final page left a ghost history control")
	}
	first, ok := tt.Find("History row 100")
	if !ok || m.Scroll.Y != 0 || math.Abs(float64(first.Y-m.Inspector.reveal.viewport.Y-12)) > .1 {
		t.Fatalf("removing loader exceeded content start: first=%+v scroll=%+v", first, m.Scroll)
	}
}

func TestTranscriptHistoryPartialMessageAnchorAcrossPrepend(t *testing.T) {
	tt, m, _ := transcriptHistoryFixture(t, 50)
	tt.Scroll(400, 200, 0, -100000)
	tt.Scroll(400, 200, 0, 53)
	before := m.Inspector.reveal.rows["history-100"]
	if before.H <= 0 {
		t.Fatal("first row not partially visible")
	}
	// Another explicit paging entry can publish a page while this reader is
	// partway through its first message; the ordinary keyed anchor owns it.
	m.OnLoadEarlier()
	tt.Frame()
	m.Snapshot.Messages = append(transcriptHistoryMessages(80, 20), m.Snapshot.Messages...)
	m.Snapshot.HistoryPageBusy = false
	tt.Frame()
	if after := m.Inspector.reveal.rows["history-100"]; after != before {
		t.Fatalf("normal keyed partial anchor moved: before=%+v after=%+v", before, after)
	}
}

func TestTranscriptHistoryPrependAfterReaderScrollKeepsCurrentAnchor(t *testing.T) {
	tt, m, _ := transcriptHistoryFixture(t, 80)
	tt.Scroll(400, 200, 0, -100000)
	if err := tt.Click(i18n.T(i18n.KeyChatLoadEarlier)); err != nil {
		t.Fatal(err)
	}
	tt.Scroll(400, 200, 0, 500)
	id, before := "", ui.Rect{}
	for _, msg := range m.Snapshot.Messages {
		if box := m.Inspector.reveal.rows[msg.ID]; box.H > 10 && box.Y > 10 && box.Y+box.H < 350 {
			id, before = msg.ID, box
			break
		}
	}
	if id == "" {
		t.Fatal("no interior visible row")
	}
	m.Snapshot.Messages = append(transcriptHistoryMessages(80, 20), m.Snapshot.Messages...)
	m.Snapshot.HistoryPageBusy = false
	tt.Frame()
	if after := m.Inspector.reveal.rows[id]; math.Abs(float64(before.Y-after.Y)) > .1 || after.H == 0 {
		t.Fatalf("response overrode reader scroll: %s before=%+v after=%+v", id, before, after)
	}
}

func TestTranscriptHistoryViewportResizeAndDuplicateIDs(t *testing.T) {
	tt, m, _ := transcriptHistoryFixture(t, 60)
	// Snapshots are immutable. Replace the window before changing its IDs so
	// projection caches see the new source, as they do for Host publications.
	m.Snapshot.Messages = append([]session.Message(nil), m.Snapshot.Messages...)
	// Same ID, separate durable occurrences must keep unique virtual keys.
	m.Snapshot.Messages[0].ID = "duplicate"
	m.Snapshot.Messages[1].ID = "duplicate"
	tt.Scroll(400, 200, 0, -100000)
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeyChatLoadEarlier)); err != nil {
		t.Fatal(err)
	}
	tt.SetSize(510, 450)
	before, _ := tt.Find("History row 100")
	m.Snapshot.Messages = append(transcriptHistoryMessages(80, 20), m.Snapshot.Messages...)
	m.Snapshot.HistoryPageBusy = false
	tt.Frame()
	after, ok := tt.Find("History row 100")
	if !ok || math.Abs(float64(after.Y-before.Y)) > .1 {
		t.Fatalf("resize/prepend moved anchor: before=%+v after=%+v", before, after)
	}
	keys := transcriptMessageKeys(*m, m.Snapshot.Messages)
	seen := map[transcriptMessageKey]bool{}
	for _, key := range keys {
		if seen[key] {
			t.Fatalf("duplicate virtual key=%+v", key)
		}
		seen[key] = true
	}
}

func TestTranscriptHistoryKeysDistinguishIdenticalOccurrencesAndKeepLoadedTail(t *testing.T) {
	m := Model{Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "host"}}
	message := session.Message{ID: "duplicate", Seq: 8, Kind: session.MessageKindText, Role: "assistant"}
	loaded := []session.Message{message, message}
	before := transcriptMessageKeys(m, loaded)
	after := transcriptMessageKeys(m, append([]session.Message{message}, loaded...))
	if before[0] == before[1] || before[0] != after[1] || before[1] != after[2] {
		t.Fatalf("prepend renamed duplicate occurrences: before=%+v after=%+v", before, after)
	}
	m.Snapshot.SelectedID = "other"
	if transcriptMessageKeys(m, loaded)[0] == before[0] {
		t.Fatal("another session reused the old virtual row identity")
	}
}

func TestTranscriptHistoryMessageActionsAndInspectorKeepMessageCoordinates(t *testing.T) {
	tt, m, _ := transcriptHistoryFixture(t, 50)
	m.Snapshot.Messages = append(m.Snapshot.Messages,
		session.Message{ID: "final-answer", SourceMessageID: "final-answer", Kind: session.MessageKindText, Role: "assistant", Seq: 200, Body: "The final answer"},
		session.Message{ID: "final-end", Kind: session.MessageKindText, Role: "status", EventType: "turn/end", Seq: 201})
	tt.Frame()
	actions, ok := tt.Find("Assistant message actions")
	if !ok {
		t.Fatal("assistant tail actions were shifted to a different message")
	}
	tt.ClickAt(actions.X+14, actions.Y+14)
	if tt.Clipboard() != "The final answer" {
		t.Fatalf("assistant copy used wrong source: %q", tt.Clipboard())
	}
	// Locate an offscreen occurrence through the actual Inspector animation.
	m.Inspector.Target = &InspectorTarget{ID: "history-120"}
	tt.Frame()
	tt.SetTime(time.Unix(9000, 0).Add(2 * time.Second))
	tt.Frame()
	box := m.Inspector.reveal.rows["history-120"]
	if box.H <= 0 || m.Inspector.Highlight != "history-120" {
		t.Fatalf("inspector landed on shifted row: highlight=%q box=%+v", m.Inspector.Highlight, box)
	}
	center := box.Y + box.H/2
	view := m.Inspector.reveal.viewport
	if math.Abs(float64(center-(view.Y+view.H/2))) > 20 {
		t.Fatalf("inspector centered the neighbor instead of target: target=%+v viewport=%+v", box, view)
	}
	m.Inspector.CancelReveal()
	m.Inspector.Picking = true
	tt.Frame()
	pick, _, ok := m.Inspector.pickAt(box.X+20, box.Y+box.H/2)
	if !ok || pick.Message.ID != "history-120" {
		t.Fatalf("inspector geometry maps to wrong occurrence: %+v", pick)
	}
}

func TestTranscriptHistoryNavigationMapsFoldedSourceToVisibleGroup(t *testing.T) {
	tt, m, _ := transcriptHistoryFixture(t, 60)
	m.Preferences.TranscriptView = "standard"
	m.Snapshot.Messages = append(m.Snapshot.Messages[:20], append([]session.Message{
		{ID: "fold-a", Kind: session.MessageKindThinking, Role: "assistant", Seq: 500, Body: "First reasoning"},
		{ID: "fold-b", Kind: session.MessageKindThinking, Role: "assistant", Seq: 501, Body: "Second reasoning"},
	}, m.Snapshot.Messages[20:]...)...)
	tt.Frame()
	if !ScrollToMessageSeq(*m, 501, ui.Start) {
		t.Fatal("folded reasoning target was silently dropped")
	}
	tt.Frame()
	messages, controls := projectTranscript(visibleChatMessages(projectChatMessages(m.Snapshot)), *m)
	group := ""
	for _, msg := range messages {
		if control, ok := controls[msg.ID]; ok {
			for _, id := range control.members {
				if id == "fold-b" {
					group = msg.ID
				}
			}
		}
	}
	if group == "" || m.Inspector.reveal.rows[group].H <= 0 {
		t.Fatalf("folded source did not land on visible group %q", group)
	}
}

func TestTranscriptHistoryNavigationResolvesProjectedRowsWithAndWithoutPrefix(t *testing.T) {
	tt, m, _ := transcriptHistoryFixture(t, 60)
	m.Snapshot.Messages = append([]session.Message{{ID: "hidden-system", Kind: session.MessageKindSystemPrompt, Seq: .5}}, m.Snapshot.Messages...)
	for _, hasMore := range []bool{true, false} {
		m.Snapshot.HistoryHasMore = hasMore
		tt.Frame()
		if !ScrollToMessageSeq(*m, 121, ui.Start) {
			t.Fatal("loaded target not resolved")
		}
		tt.Frame()
		box, ok := tt.Find("History row 120")
		if !ok || math.Abs(float64(box.Y-m.Inspector.reveal.viewport.Y-12)) > .1 {
			t.Fatalf("history prefix=%v target landed on wrong row: box=%+v viewport=%+v", hasMore, box, m.Inspector.reveal.viewport)
		}
	}
	before := m.Scroll.Y
	if ScrollToMessageSeq(*m, .5, ui.Start) || ScrollToMessageSeq(*m, 99999, ui.Start) {
		t.Fatal("hidden or missing message queued a misleading landing")
	}
	tt.Frame()
	if m.Scroll.Y != before {
		t.Fatal("missing target moved transcript")
	}
}
