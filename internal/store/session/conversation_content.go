package session

// HasConversationContent distinguishes the transcript from configuration-command
// audit records. A permission/model command does not start the first turn, so it
// must not retire the welcome surface or prevent carrying its unsent draft.
func (s ViewState) HasConversationContent() bool {
	for _, message := range s.Messages {
		if message.Kind != MessageKindCommand {
			return true
		}
	}
	return false
}
