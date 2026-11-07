package composer

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"whaleisle/internal/dsh/api"
)

// CommandAttachmentsUnsupportedNotice returns the localized notice when a slash
// command does not accept attachments (`command.attachmentsUnsupported` /
// `notice.attachmentsUnsupported`).
func CommandAttachmentsUnsupportedNotice(command string) string {
	name := strings.TrimPrefix(strings.TrimSpace(command), "/")
	if i := strings.IndexAny(name, " \t\r\n"); i >= 0 {
		name = name[:i]
	}
	return composerText(
		"/"+name+" 不接受附件，请先移除附件",
		"/"+name+" does not accept attachments; remove them first",
	)
}

func slashDescriptorMatchesToken(cmd api.ListResultItem2, token string) bool {
	if !strings.HasPrefix(token, "/") {
		return false
	}
	if "/"+cmd.Name == token || strings.EqualFold("/"+cmd.Name, token) {
		return true
	}
	if cmd.DefinitionId != nil {
		if label, _, ok := builtinSlashFace(*cmd.DefinitionId); ok && label != "" && ("/"+label == token || strings.EqualFold("/"+label, token)) {
			return true
		}
	}
	return false
}

func refuseCommandAttachments(m Model, text string) bool {
	if m.Editing || m.GroupMembers != nil || m.State == nil || len(m.State.Attachments) == 0 {
		return false
	}
	if claim := currentLeadingClaim(m); claim != nil {
		if !claim.attachments {
			m.State.Error = CommandAttachmentsUnsupportedNotice(claim.token)
			return true
		}
		return false
	}
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") || strings.Contains(trimmed, "\n") {
		return false
	}
	line := NormalizeCommandLine(trimmed)
	if isSkillOnlyInvocation(m, line) {
		return false
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return false
	}
	token := fields[0]
	bare := len(fields) == 1
	name := strings.TrimPrefix(token, "/")
	if name == "" {
		return false
	}
	if bare && (name == "file" || name == "feedback") && nativeCommandAvailable(m.Snapshot, name) {
		return false
	}
	for _, cmd := range m.Snapshot.SlashCommands {
		if !slashDescriptorMatchesToken(cmd, token) {
			continue
		}
		if cmd.Input != nil {
			if cmd.Input.Attachments != nil && *cmd.Input.Attachments {
				return false
			}
			m.State.Error = CommandAttachmentsUnsupportedNotice(token)
			return true
		}
		if bare {
			m.State.Error = CommandAttachmentsUnsupportedNotice(token)
			return true
		}
		return false
	}
	return false
}

// A claim decorates a real commands/list input descriptor. The command remains
// literal draft text, so the existing session-owned, non-detached command submit
// transaction revalidates its catalog and attachment policy at admission.
type leadingInputClaim struct {
	sessionID      string
	hostGeneration uint64
	token, hint    string
	attachments    bool
}

func currentLeadingClaim(m Model) *leadingInputClaim {
	state := suggestionState(m, m.TextInput)
	if state == nil || state.claim == nil || m.Draft == nil {
		return nil
	}
	claim := state.claim
	if m.Editing || claim.sessionID != m.Snapshot.SelectedID || claim.hostGeneration != m.Snapshot.ComposerHostGeneration || !m.Snapshot.ClientReady ||
		(!strings.HasPrefix(*m.Draft, claim.token) && *m.Draft != strings.TrimSuffix(claim.token, " ")) {
		state.claim = nil
		return nil
	}
	return claim
}

func claimLeadingSpace(client *DraftClient, m Model, ev ui.InputEvent) bool {
	if ev.Kind != ui.InputKeyDown || ev.Key != ui.KeySpace || ev.Mods != 0 || ev.Repeat || client == nil || client.composing() || m.Draft == nil ||
		m.Editing || m.GroupMembers != nil || !m.Snapshot.ClientReady || m.Snapshot.SlashCommandsLoading || m.Snapshot.SlashCommandsErr != "" ||
		m.State != nil && (m.State.Sending || m.State.Preparing > 0 || m.State.TargetID != "" && m.State.TargetID != m.Snapshot.SelectedID) {
		return false
	}
	selection := client.Selection().Range
	if selection.Start != selection.End {
		return false
	}
	caret := ui.UTF16ByteOffset(*m.Draft, selection.End)
	token := (*m.Draft)[:caret]
	if !strings.HasPrefix(token, "/") || strings.ContainsAny(token, " \t\n\r") {
		return false
	}
	// Only actual registered contributions suppress Host leading-input claims.
	// A permission/feedback decoration owns bare invocation, never Space/args.
	name := strings.TrimPrefix(token, "/")
	if (name == "file" || name == "model") && nativeCommandRegistered(m.Snapshot, name) {
		return false
	}
	for _, cmd := range m.Snapshot.SlashCommands {
		if !slashDescriptorMatchesToken(cmd, token) || cmd.Input == nil {
			continue
		}
		state := suggestionState(m, client)
		if state == nil {
			return false
		}
		client.ReplaceText(&ui.TextInputRange{Start: 0, End: selection.End}, token+" ")
		state.claim = &leadingInputClaim{sessionID: m.Snapshot.SelectedID, hostGeneration: m.Snapshot.ComposerHostGeneration,
			token: token + " ", hint: cmd.Input.Hint, attachments: cmd.Input.Attachments != nil && *cmd.Input.Attachments}
		activeSuggestion(m, client)
		dismissSuggestion(m, client)
		return true
	}
	return false
}

func paintLeadingClaimHint(c *ui.Context, m Model, muted ui.Color) {
	claim := currentLeadingClaim(m)
	if claim == nil || claim.hint == "" || !strings.HasPrefix(*m.Draft, claim.token) || strings.TrimSpace(strings.TrimPrefix(*m.Draft, claim.token)) != "" {
		return
	}
	ui.Text(c, claim.hint).Label("Command argument hint").FontSize(12).TextColor(muted).Padding(2, 8)
}
