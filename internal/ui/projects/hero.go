package projects

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// HeroOption is the installed New Session Project option: a small action in
// the Hero options row, not another directory picker or replacement composer.
func HeroOption(c *ui.Context, m Model, snap session.ViewState) {
	if !m.Listing.Available || snap.SelectedCwd == "" || sameHeroDirectory(snap.SelectedCwd, snap.ScratchCwd) || !sameHeroDirectory(snap.SelectedCwd, m.HeroWorkspace) || m.OnCreateFromDraft == nil {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	ink := chrome.Color(colors.Muted)
	hint := tr("以当前目录开启 Project，已输入的内容会一起带过去", "Start a Project in this directory; your draft comes along")
	if m.HeroExisting {
		hint = tr("打开此目录的 Project，已输入的内容会一起带过去", "Open this directory’s Project; your draft comes along")
	}
	if m.HeroDirectoryReason != "" {
		hint = directoryRefusalText(m.HeroDirectoryReason)
	}
	// The row shares the composer stack envelope (card + 16-DIP clearance per
	// side); the card's content inset adds its own 16-DIP padding, so the label
	// lines up with the card's inner controls at +32, not the envelope edge.
	row := ui.Row(c).Key("project-hero-option").Gap(6).AlignItems(ui.Center).Padding(0, 0, 0, 32).Tooltip(hint)
	row.Children(func() {
		button := coss.Button(c, "", coss.ButtonOptions{
			Variant:  coss.ButtonGhost,
			Size:     coss.ButtonSM,
			Label:    "Project",
			Disabled: m.Busy || m.HeroDirectoryReason != "",
			Content: func() {
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					projectIcon(c, "checklist", 16, ink, false)
					ui.Text(c, "Project").FontSize(theme.InterfaceSize(13)).TextColor(ink)
				})
			},
		}).Tooltip(hint)
		if button.Clicked() {
			m.OnCreateFromDraft()
		}
	})
	widgets.Alert(c, widgets.AlertOptions{ID: "project-hero-create", Message: m.HeroError})
}

func sameHeroDirectory(a, b string) bool {
	return strings.EqualFold(strings.TrimRight(strings.ReplaceAll(a, `\`, "/"), "/"), strings.TrimRight(strings.ReplaceAll(b, `\`, "/"), "/"))
}

func directoryRefusalText(reason string) string {
	switch reason {
	case "missing":
		return tr("找不到这个目录。", "This folder cannot be found.")
	case "not-directory":
		return tr("这个路径不是文件夹。", "This path is not a folder.")
	case "root":
		return tr("磁盘根目录不能作为 Project 目录，请选择具体的项目文件夹。", "A drive root cannot be a Project folder. Choose a specific project folder.")
	case "home":
		return tr("用户主目录不能作为 Project 目录，请选择其中具体的项目文件夹。", "Your home folder cannot be a Project folder. Choose a specific project folder inside it.")
	case "system-data":
		return tr("系统配置目录（如 AppData、.config、.ssh）不能作为 Project 目录，请选择你的项目文件夹。", "System configuration folders cannot be Project folders. Choose your project folder.")
	case "app-data":
		return tr("这是 鲸屿 MyGo 自己的数据目录，不能作为 Project 目录，请选择你的项目文件夹。", "This is Whale Isle MyGo’s own data folder and cannot be a Project folder. Choose your project folder.")
	default:
		return tr("此目录不能作为 Project 目录。", "This directory cannot host a Project.")
	}
}
