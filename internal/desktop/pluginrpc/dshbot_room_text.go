package pluginrpc

import (
	"fmt"
	"unicode/utf8"
)

// Rooms have name/description only. Their existing title must remain empty;
// workspace, avatar, model, member order and limits are not text fields.
type DshbotRoomTextValues struct{ Name, Description string }

func ParseDshbotRoomText(v DshbotRoomTextValues) (DshbotRoomTextValues, error) {
	if !utf8.ValidString(v.Name) || !utf8.ValidString(v.Description) {
		return v, fmt.Errorf("群文本含无效 Unicode / Invalid room text Unicode")
	}
	v.Name = trimBotMemory(v.Name)
	v.Description = trimBotMemory(v.Description)
	if v.Name == "" || WhaleTextLength(v.Name) > 120 {
		return v, fmt.Errorf("群名必填且最多 120 UTF-16 单元 / Room name required, maximum 120 UTF-16 units")
	}
	if WhaleTextLength(v.Description) > 12000 {
		return v, fmt.Errorf("群说明最多 12000 UTF-16 单元 / Room description maximum 12000 UTF-16 units")
	}
	// The Host owns locale-sensitive same-kind uniqueness; do not pretend Go
	// case folding is an equivalent preflight or silently choose another name.
	return v, nil
}
func DshbotRoomTextChanged(cmd DshbotRoomEditCommand) bool {
	return cmd.Values.Text != nil && (cmd.Values.Text.Name != cmd.Expected.Item.Name || cmd.Values.Text.Description != cmd.Expected.Item.Description)
}
