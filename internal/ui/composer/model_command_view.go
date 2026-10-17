package composer

import (
	"github.com/egoist/mygo/ui"
	"strings"
)

func handleModelCommand(m Model) bool {
	if m.Editing || m.Draft == nil || (!strings.EqualFold(strings.TrimSpace(*m.Draft), "/model") && !strings.EqualFold(strings.TrimSpace(*m.Draft), "/permission")) {
		return false
	}
	// The native composer always supplies State. A stateless preview must not
	// accidentally dispatch a UI-only command to a model or remote command API.
	if m.State == nil {
		return true
	}
	return m.State.HandleModelCommand(m.Snapshot.SelectedID)
}

func paintModelCommand(c *ui.Context, m Model, panel, border, fg, muted ui.Color) {
	ui.Column(c.Key("local-model-picker:" + m.Snapshot.SelectedID)).FillWidth().Padding(12).Gap(8).
		Background(panel).BorderColor(border).BorderWidth(1).Radius(12).Children(func() {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Text(c, "模型与思考强度 / Model and reasoning effort").TextColor(fg).FontSize(13).Grow(1)
			if ui.Button(c, "关闭 / Close").Clicked() {
				m.State.ModelCommand.Open = false
			}
		})
		// Both entry points read the same confirmed provider/model/effort choices
		// and invoke the same session-bound callbacks and error/retry surface.
		paintModelPicker(c, m, muted, 28)
	})
}
