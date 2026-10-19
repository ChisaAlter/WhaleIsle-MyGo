package composer

import (
	"strings"
	"whaleisle/internal/store/session"
)

// HandleNativeCommand handles only the fixed contribution's bare invocation.
// Argument-bearing Host commands keep their existing input/submit transaction.
func (d *DraftState) HandleNativeCommand(s session.ViewState, action func(NativeCommandOwner, string) bool) bool {
	if d == nil || d.EditSeq > 0 {
		return false
	}
	text := strings.TrimSpace(d.Text)
	if len(strings.Fields(text)) != 1 || !strings.HasPrefix(text, "/") {
		return false
	}
	name := strings.TrimPrefix(text, "/")
	if !s.NativeContributions.Known {
		switch name {
		case "file", "model", "permission", "feedback":
			d.Error = s.NativeContributions.Error
			if d.Error == "" {
				d.Error = composerText("正在加载命令扩展…", "Loading command extensions…")
			}
			return true
		}
	}
	if !nativeCommandRegistered(s, name) {
		return false
	}
	if (name == "permission" || name == "feedback") && !s.SlashCommandsKnown {
		d.Error = s.SlashCommandsErr
		if d.Error == "" {
			d.Error = composerText("正在加载命令目录…", "Loading command catalog…")
		}
		return true
	}
	if !nativeCommandAvailable(s, name) {
		return false
	}
	if d.TargetID != "" && d.TargetID != s.SelectedID {
		d.Error = "Select the draft's session before using this command"
		return true
	}
	if d.Sending || d.Preparing > 0 {
		d.Error = composerText("请等待当前输入操作完成", "Wait for the current input operation")
		return true
	}
	segment := nativeCommandSegment{text: text, bare: true, client: d.Client}
	if name == "file" || name == "feedback" {
		if action == nil {
			d.Error = "Command action unavailable"
			return true
		}
		segment.consume(d)
		action(nativeCommandOwner(s, name), name)
		return true
	}
	if len(d.Attachments) > 0 {
		d.Error = "/" + name + " does not accept attachments; remove them before opening the picker"
		return true
	}
	d.openNativeCommand(s, name, segment)
	return true
}

// Retained for non-native callers; the live native submission uses the graph-
// bound HandleNativeCommand above. Opening a popup never consumes its input.
func (d *DraftState) HandleModelCommand(sessionID string) bool {
	if d == nil {
		return false
	}
	handled, err := false, ""
	if strings.TrimSpace(d.Text) == "/permission" {
		handled, err = d.PermissionCommand.Handle("/model", sessionID, d.TargetID, d.EditSeq > 0, len(d.Attachments))
		err = strings.ReplaceAll(strings.ReplaceAll(err, "/model", "/permission"), "a model", "permissions")
	} else {
		handled, err = d.ModelCommand.Handle(d.Text, sessionID, d.TargetID, d.EditSeq > 0, len(d.Attachments))
	}
	if handled {
		d.Error = err
	}
	return handled
}
