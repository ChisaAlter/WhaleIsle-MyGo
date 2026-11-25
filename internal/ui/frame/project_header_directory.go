package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/projects"
)

func projectHeaderDirectory(c *ui.Context, m Model, p projectflow.Project, ink ui.Color) ui.Element {
	button := coss.Button(c, "", coss.ButtonOptions{
		Variant: coss.ButtonGhost,
		Size:    coss.ButtonIconSM,
		Label:   paneLabel("打开项目目录", "Open project directory"),
		Key:     "project-directory",
		Content: func() { projects.HeaderIcon(c, "folder", 16, ink) },
	}).Tooltip(p.CanonicalWorkingDirectory).Padding(design.Sp(1.5)).Radius(design.RadiusSM)
	if button.Clicked() && m.Projects.OnProjectDirectoryFor != nil {
		m.Projects.OnProjectDirectoryFor(p.ID)
	}
	return button
}
