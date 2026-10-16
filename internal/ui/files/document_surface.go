package files

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// DocumentSurface is an exact-Host child owned by one native file tab. The live
// Cordis document owner supplies registry subscriptions, selection and renderers.
// It never receives native bindings or another tab's read/draft state.
type DocumentSurface interface {
	Alive() bool
	DocumentViewerRecommended() bool
	SetViewport(*mygo.Rectangle)
	SetVisible(bool)
	Reload()
	Close() error
}

type documentSurfaceState struct {
	surface   DocumentSurface
	enabled   bool
	choice    int // 0 automatic, 1 native, 2 original document owner
	attempted bool
}

func (t *FileTab) closeDocumentSurface() {
	if t.document.surface != nil {
		_ = t.document.surface.Close()
		t.document.surface = nil
	}
	t.document.enabled = false
	t.document.attempted = false
}

func (t *Tabs) hideDocumentSurfaces() {
	if t == nil {
		return
	}
	for _, workspace := range t.Workspaces {
		if workspace != nil {
			for _, tab := range workspace.Open {
				if tab.document.surface != nil {
					tab.document.surface.SetVisible(false)
				}
			}
		}
	}
}

func syncDocumentSurfaces(m Model) {
	if m.Tabs == nil {
		return
	}
	active := selectedTab(m)
	for _, workspace := range m.Tabs.Workspaces {
		if workspace != nil {
			for _, tab := range workspace.Open {
				if tab.document.surface != nil && (tab != active || !tab.document.enabled || tab.Editing) {
					tab.document.surface.SetVisible(false)
				}
			}
		}
	}
}

func paintDocumentViewerButton(c *ui.Context, m Model) {
	tab := selectedTab(m)
	if tab == nil || m.OpenDocumentViewer == nil {
		return
	}
	if !tab.document.attempted && !tab.Editing {
		tab.document.attempted = true
		surface, err := m.OpenDocumentViewer(tab.Path)
		if err == nil {
			tab.document.surface = surface
		}
	}
	tab.document.enabled = documentSurfaceEnabled(tab.document)
	if !tab.document.enabled && tab.document.surface != nil {
		tab.document.surface.SetVisible(false)
	}
	title := label("选择预览器", "Choose viewer")
	if tab.document.enabled {
		title = label("返回原生预览", "Native preview")
	}
	if ui.Button(c, title).Disabled(tab.Editing).Clicked() {
		if tab.document.enabled {
			tab.document.choice = 1
			tab.document.enabled = false
			if tab.document.surface != nil {
				tab.document.surface.SetVisible(false)
			}
			return
		}
		if tab.document.surface == nil || !tab.document.surface.Alive() {
			surface, err := m.OpenDocumentViewer(tab.Path)
			if err != nil {
				setFileStatus(m, err.Error())
				return
			}
			tab.document.surface = surface
		}
		tab.document.choice = 2
		tab.document.enabled = true
	}
	if tab.document.choice != 0 && ui.Button(c, label("自动选择", "Auto viewer")).Clicked() {
		tab.document.choice = 0
		tab.document.enabled = documentSurfaceEnabled(tab.document)
	}

}

func paintDocumentSurface(c *ui.Context, m Model) bool {
	tab := selectedTab(m)
	if tab == nil || !tab.document.enabled || tab.Editing {
		return false
	}
	surface := tab.document.surface
	if surface == nil || !surface.Alive() {
		tab.document.enabled = false
		setFileStatus(m, label("预览器已关闭，请重新打开。", "Viewer closed; choose a viewer again."))
		return false
	}
	box := ui.Column(c).Key("file-document-viewer").Fill().MinWidth(0).MinHeight(0)
	bounds := box.Bounds()
	surface.SetViewport(&mygo.Rectangle{X: int(bounds.X), Y: int(bounds.Y), Width: int(bounds.W), Height: int(bounds.H)})
	surface.SetVisible(bounds.W > 0 && bounds.H > 0)
	box.Children(func() { ui.Text(c, label("正在载入资源预览器…", "Loading resource viewer…")) })
	return true
}

func documentSurfaceEnabled(state documentSurfaceState) bool {
	if state.surface == nil || !state.surface.Alive() {
		return false
	}
	return state.choice == 2 || (state.choice == 0 && state.surface.DocumentViewerRecommended())
}
