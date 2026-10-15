package browserpreview

import "errors"

func (b *Controller) SetWorkspaceOwner(owner func() WorkspaceOwner) { b.workspaceOwner = owner }

func (b *Controller) browserWorkspace() (browserWorkspace, error) {
	if b.workspaceOwner == nil {
		return browserWorkspace{}, errors.New("Browser 缺少可信的应用/Host归属")
	}
	return makeBrowserWorkspace(b.workspaceOwner())
}

func (t *Tab) browserWorkspaceLabel() string {
	if t == nil || t.workspace.key == "" {
		return "归属未确定"
	}
	return t.workspace.label
}
