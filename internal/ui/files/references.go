package files

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/filelink"
)

func workspaceReference(m Model, path string) filelink.Reference {
	return filelink.Reference{BindingID: m.BindingID, SessionID: m.SessionID, WorkspaceRoot: m.WorkspaceRoot, Path: path}
}
func paintReferenceButtons(c *ui.Context, m Model, path string) {
	text, ok := filelink.Serialize(path)
	if !ok {
		return
	}
	if ui.Button(c, "@").Tooltip(label("引用整文件到对话", "Reference file in chat")).FontSize(11).Disabled(m.OnReferenceFile == nil || !m.CanAddToChat).Clicked() {
		if err := m.OnReferenceFile(workspaceReference(m, path)); err != nil {
			setFileStatus(m, err.Error())
		} else {
			setFileStatus(m, label("文件引用已插入对话草稿", "File reference inserted into chat draft"))
		}
	}
	if ui.Button(c, label("复制引用", "Copy reference")).FontSize(10).Clicked() {
		mygo.Clipboard.WriteText(text)
	}
}
