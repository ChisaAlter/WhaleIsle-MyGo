package pluginrpc

import (
	"errors"
	"unicode/utf8"
	"whaleisle/internal/store/storelocale"
)

// Rooms have name/description only. Their existing title must remain empty;
// workspace, avatar, model, member order and limits are not text fields.
type DshbotRoomTextValues struct{ Name, Description string }

func ParseDshbotRoomText(v DshbotRoomTextValues) (DshbotRoomTextValues, error) {
	if !utf8.ValidString(v.Name) || !utf8.ValidString(v.Description) {
		return v, errors.New(storelocale.T(storelocale.KeyDesktopPluginrpcInvalidRoomTextUnicode))
	}
	v.Name = trimBotMemory(v.Name)
	v.Description = trimBotMemory(v.Description)
	if v.Name == "" || WhaleTextLength(v.Name) > 120 {
		return v, errors.New(storelocale.T(storelocale.KeyDesktopPluginrpcRoomNameRequiredMaximum120Utf16Un))
	}
	if WhaleTextLength(v.Description) > 12000 {
		return v, errors.New(storelocale.T(storelocale.KeyDesktopPluginrpcRoomDescriptionMaximum12000Utf16Un))
	}
	// The Host owns locale-sensitive same-kind uniqueness; do not pretend Go
	// case folding is an equivalent preflight or silently choose another name.
	return v, nil
}
func DshbotRoomTextChanged(cmd DshbotRoomEditCommand) bool {
	return cmd.Values.Text != nil && (cmd.Values.Text.Name != cmd.Expected.Item.Name || cmd.Values.Text.Description != cmd.Expected.Item.Description)
}
