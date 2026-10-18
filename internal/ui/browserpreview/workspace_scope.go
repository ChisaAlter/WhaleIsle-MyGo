package browserpreview

import "errors"

func (b *Controller) SetWorkspaceOwner(owner func() WorkspaceOwner) { b.workspaceOwner = owner }

func (b *Controller) browserWorkspace() (browserWorkspace, error) {
	if b.workspaceOwner == nil {
		return browserWorkspace{}, errors.New(browserChromeLabel("Browser 缺少可信的应用/Host归属", "Browser has no trusted application/Host owner"))
	}
	return makeBrowserWorkspace(b.workspaceOwner())
}

func (t *Tab) browserWorkspaceLabel() string {
	if t == nil || t.workspace.key == "" {
		return browserChromeLabel("归属未确定", "Owner undetermined")
	}
	return t.workspace.label
}
