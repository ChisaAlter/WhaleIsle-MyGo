package dshbot

import "whaleisle/internal/desktop/pluginrpc"

type ContactActivity struct {
	Active     bool
	ActivityAt int64
	Preview    string
}

func ContactActivityAt(row pluginrpc.DshbotBotItem, live ContactActivity) int64 {
	at := int64(pluginrpc.DshbotContactNumber(row, "updatedAt"))
	if live.ActivityAt > at {
		at = live.ActivityAt
	}
	return at
}

// e9 contactActivityMatches: active first; recent excludes active and means
// a timestamp within seven days; idle is the remaining nonactive contacts.
func ContactActivityMatches(row pluginrpc.DshbotBotItem, live ContactActivity, filter string, now int64) bool {
	if filter == "" || filter == "all" {
		return true
	}
	if filter == "active" {
		return live.Active
	}
	if live.Active {
		return false
	}
	at := ContactActivityAt(row, live)
	recent := at > 0 && now-at <= 7*24*60*60*1000
	if filter == "recent" {
		return recent
	}
	return !recent
}
