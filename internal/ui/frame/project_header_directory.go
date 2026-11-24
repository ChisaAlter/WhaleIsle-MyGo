package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/projects"
)

func projectHeaderDirectory(c *ui.Context, m Model, p projectflow.Project, ink ui.Color) ui.Element {
	button := ui.Button(c.Key("project-directory"), "").Label(paneLabel("打开项目目录", "Open project directory")).Tooltip(p.CanonicalWorkingDirectory).Size(28, 28).Padding(design.Sp(1.5)).BorderWidth(0).Radius(design.RadiusSM).Background(ui.Transparent)
	button.Children(func() { projects.HeaderIcon(c, "folder", 16, ink) })
	if button.Clicked() && m.Projects.OnProjectDirectoryFor != nil {
		m.Projects.OnProjectDirectoryFor(p.ID)
	}
	return button
}
