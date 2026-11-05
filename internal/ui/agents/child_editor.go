package agents

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"whaleisle/internal/desktop/clipboardimage"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func childEditor(c *ui.Context, m Model, v session.AgentNavigationView, bounds ...*ui.Rect) {
	owner := ui.Column(c.Key(fmt.Sprintf("child-editor:%p:%s:%v", m.Store, m.Snapshot.PlanHistoryBinding, v.Address))).FillWidth()
	if len(bounds) > 0 && bounds[0] != nil {
		owner.ObserveLayout(func(r ui.Rect) {
			if *bounds[0] != r {
				*bounds[0] = r
				c.Invalidate()
			}
		})
	}
	d := ui.Local(owner, "draft", func() *composer.DraftState { return composer.NewDraftState("") })
	chip := ui.Local(owner, "chip", func() composer.ChipEdit { return composer.ChipEdit{} })
	draft := *d
	if m.ChildDraft != nil {
		if retained := m.ChildDraft(m.Snapshot.PlanHistoryBinding, v.Address); retained != nil {
			draft = retained
		}
	}
	type flight struct {
		text              string
		receipt, revision uint64
		pending           bool
	}
	f := ui.Local(owner, "send-flight", func() flight { return flight{receipt: v.PromptReceipt} })
	if v.PromptReceipt != f.receipt {
		if f.pending && draft.Text == f.text && draft.Client.Revision() == f.revision {
			draft.Text = ""
			draft.Client.Sync()
			draft.Client.ClearHistory()
		}
		f.receipt = v.PromptReceipt
		f.pending = false
	}
	draft.Sending = v.ActionBusy
	stopModel := childStopModel(m, v, draft)
	childSnapshot := stopModel.Snapshot
	add := func(paths []string) { m.Store.AddAgentNavigationImages(v.Token, v.Address, paths, nil) }
	paste := func(explicit bool) {
		raw := mygo.Clipboard.ReadImage()
		if len(raw) == 0 {
			if explicit {
				draft.Error = agentText("剪贴板没有图片", "No image on the clipboard")
			}
			return
		}
		path, err := clipboardimage.StorePNG(filepath.Join(paths.UserDataDir(), "cache", "attachments"), raw)
		if err != nil {
			draft.Error = err.Error()
			return
		}
		add([]string{path})
	}
	owner.Children(func() {
		composer.ChildEditor(c, composer.ChildEditorModel{
			ThemeID: m.ThemeID, SystemDark: m.SystemDark, Snapshot: childSnapshot,
			OnQueryReferences: func(ctx context.Context, q string, quoted bool) ([]session.ReferenceCandidate, error) {
				return m.Store.QueryAgentReferences(ctx, v.Token, v.Address, q, quoted)
			},
			OnStop: stopModel.OnStop, StopBlocked: stopModel.StopBlocked,
			OnSendGesture: func(text string, accelerated bool) {
				f.text = text
				f.pending = true
				f.revision = draft.Client.Revision()
				m.Store.PromptAgentNavigationGesture(v.Token, v.Address, text, accelerated, m.BusyEnter)
			},
			Draft: draft, Chip: chip, Images: v.PromptImages, Disabled: (v.RetainedReadOnly && !v.RetainedPromptAllowed) || v.Busy || v.ActionBusy || !v.ParentAvailabilityKnown || !v.ParentAvailable,
			OnAddImages: add, OnPasteImage: paste,
			OnRemoveImage: func(path string) { m.Store.RemoveAgentNavigationImage(v.Token, v.Address, path) },
			OnPickImages: func() {
				selected, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Title: agentText("向子代理添加图片", "Add images for child"), Multiple: true, Filters: []mygo.FileFilter{{Name: "Images", Extensions: []string{"png", "jpg", "jpeg", "gif", "webp"}}}})
				m.Store.AddAgentNavigationImages(v.Token, v.Address, selected, err)
			},
			OnSend: func(text string) { m.Store.PromptAgentNavigationExplicit(v.Token, v.Address, text) },
			OnOpenReference: func(path string) {
				if m.OnOpenDocument != nil {
					m.OnOpenDocument(v.Token, v.Address, path)
				}
			},
		})
	})
}

func childStopModel(m Model, v session.AgentNavigationView, draft *composer.DraftState) composer.Model {
	view := m.Snapshot
	view.SelectedID = v.Address.ChildSessionID
	view.StopGeneration = v.Token
	view.StopTurnSeq = 0
	for _, record := range v.Process.Records {
		if record.Event.Type == "turn/start" {
			view.StopTurnSeq = max(view.StopTurnSeq, record.Event.Seq)
		}
	}
	for _, message := range v.Messages {
		view.StopTurnSeq = max(view.StopTurnSeq, message.TurnSeq)
		if message.EventType == "turn/start" {
			view.StopTurnSeq = max(view.StopTurnSeq, message.Seq)
		}
	}
	view.Items = []session.Item{{SessionID: v.Address.ChildSessionID, Running: v.Running}}
	blocked := v.ActionBusy || v.Busy
	if m.Interactions != nil {
		pending := m.Interactions.SnapshotForAgent(v.Address.ChildSessionID)
		for _, p := range pending.Pending {
			card, ok := pending.QuestionCards[p.EventID]
			if !ok || !card.Hidden && !card.PlanReviewHidden {
				blocked = true
				break
			}
		}
	}
	return composer.Model{Draft: &draft.Text, State: draft, TextInput: draft.Client, Editing: true, Child: true, Snapshot: view, StopBlocked: blocked, OnStop: func() {
		if v.RetainedReadOnly {
			m.Store.InterruptRetainedAgentExplicit(v.Token, v.Address)
		} else {
			m.Store.InterruptAgentNavigation(v.Token, []session.AgentSessionAddress{v.Address})
		}
	}}
}
