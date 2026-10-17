package agents

import (
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"whaleisle/internal/desktop/clipboardimage"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

func childEditor(c *ui.Context, m Model, v session.AgentNavigationView) {
	owner := ui.Column(c.Key(fmt.Sprintf("child-editor:%p:%s:%v", m.Store, m.Snapshot.PlanHistoryBinding, v.Address))).FillWidth()
	d := ui.Local(owner, "draft", func() *composer.DraftState { return composer.NewDraftState("") })
	chip := ui.Local(owner, "chip", func() composer.ChipEdit { return composer.ChipEdit{} })
	draft := *d
	draft.Sending = v.ActionBusy
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
			Draft: draft, Chip: chip, Images: v.PromptImages, Disabled: v.Busy || v.ActionBusy || !v.ParentAvailabilityKnown || !v.ParentAvailable,
			OnAddImages: add, OnPasteImage: paste,
			OnRemoveImage: func(path string) { m.Store.RemoveAgentNavigationImage(v.Token, v.Address, path) },
			OnPickImages: func() {
				selected, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Title: agentText("向子代理添加图片", "Add images for child"), Multiple: true, Filters: []mygo.FileFilter{{Name: "Images", Extensions: []string{"png", "jpg", "jpeg", "gif", "webp"}}}})
				m.Store.AddAgentNavigationImages(v.Token, v.Address, selected, err)
			},
			OnSend: func(text string) { m.Store.PromptAgentNavigation(v.Token, v.Address, text) },
			OnOpenReference: func(path string) {
				if m.OnOpenDocument != nil {
					m.OnOpenDocument(v.Token, v.Address, path)
				}
			},
		})
	})
}
