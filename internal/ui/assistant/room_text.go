package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/ui/i18n"
)

func roomTextEffects() string {
	return i18n.T(i18n.KeyW4AssistantAGroupHasNoSeparateTitleAnEmptyExist)
}

func roomTextOptional(s string) string {
	if s == "" {
		return i18n.T(i18n.KeyMiscAssistantEmpty2)
	}
	return s
}
func roomEditTextReview(cmd pluginrpc.DshbotRoomEditCommand) string {
	if cmd.Values.Text == nil {
		return ""
	}
	v := cmd.Values.Text
	return fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantRoomTextDiff), cmd.Expected.Item.Name, v.Name, roomTextOptional(cmd.Expected.Item.Description), roomTextOptional(v.Description), roomTextEffects())
}
func roomEditTextInputs(c *ui.Context, m RoomEditModel, disabled bool) {
	if m.State.Draft == nil {
		return
	}
	d := m.State.Draft
	name := d.Name
	ui.TextInput(c.Key("room-edit-name"), &name).Label(i18n.T(i18n.KeyMiscAssistantRoomName120Utf16)).FillWidth().Disabled(disabled).OnChange(func() {
		if m.OnChange != nil {
			m.OnChange("name", name)
		}
	})
	description := d.Description
	ui.TextArea(c.Key("room-edit-description"), &description).Label(i18n.T(i18n.KeyMiscAssistantCompleteRoomDescription12000Utf16)).Lines(3, 10).FillWidth().Disabled(disabled).OnChange(func() {
		if m.OnChange != nil {
			m.OnChange("description", description)
		}
	})
}
