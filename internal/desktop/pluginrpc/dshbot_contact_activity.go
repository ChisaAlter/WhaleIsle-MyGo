package pluginrpc

// Reuse the original room-member canonical identity mapping against the live
// session directory, without resolving or starting any member session.
func DshbotContactOwnsMember(room DshbotBotItem, sessionID string) bool {
	if room.Kind != "room" {
		return false
	}
	for _, botID := range room.MemberBotIDs {
		if dshbotRoomMemberIdentity(room.SessionID, botID, sessionID) {
			return true
		}
	}
	return false
}
