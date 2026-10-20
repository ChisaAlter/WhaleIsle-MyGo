package main

import (
	"context"
	"time"
	"whaleisle/internal/desktop/botnotify"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

func applyBotReadView(contacts []botnotify.ContactActivity, view pluginrpc.DshbotSettingsView) {
	for i := range contacts {
		for _, p := range view.Value.Items {
			if p.ID == contacts[i].ID && p.SessionID == contacts[i].SessionID {
				contacts[i].ReadInitialized = p.ReadInitialized
				if p.LastSeenSeq > contacts[i].LastSeenSeq {
					contacts[i].LastSeenSeq = p.LastSeenSeq
				}
				break
			}
		}
	}
}

func (s *mainState) initializeBotReadState(ctx context.Context, run *botNotificationRun, snapshot *botnotify.Snapshot) error {
	s.mu.Lock()
	sessions := s.sessions
	s.mu.Unlock()
	sid := ""
	if sessions != nil {
		sid = sessions.Snapshot().SelectedID
	}
	candidates := botnotify.ReadCandidates(snapshot.Contacts, sid)
	if len(candidates) == 0 {
		return nil
	}
	var authority struct {
		Writable bool `json:"writable"`
	}
	if err := run.client.Call(ctx, "settings/describe", nil, &authority); err != nil {
		return err
	}
	if !authority.Writable || !s.botNotificationCurrent(run) {
		return nil
	}
	marks := make([]pluginrpc.DshbotReadMark, 0, len(candidates))
	for _, a := range candidates {
		marks = append(marks, pluginrpc.DshbotReadMark{ID: a.ID, ThroughSeq: a.ActivitySeq})
	}
	view, err := s.pluginRPCClient(run.endpoint).DshbotMarkRead(ctx, marks)
	if err == nil {
		applyBotReadView(snapshot.Contacts, view)
		snapshot.Revision = view.Revision
		run.window.Update(func() {
			if !s.botNotificationCurrent(run) {
				return
			}
			s.mu.Lock()
			roster := s.dshbotRoster
			s.mu.Unlock()
			if roster != nil {
				roster.SetFromDescribeForHost(pluginrpc.DshbotDescribeResult{View: view}, false, run.endpoint.Origin)
			}
		})
	}
	return err
}

// Called only after canonical navigation succeeds. This reads the Host's actual
// activity seq rather than a transcript index or timestamp.
func (s *mainState) markOpenedBotRead(ep harness.Endpoint, client *dsh.Client, id, sid string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		current := func() bool {
			s.dshMu.Lock()
			same := s.dshClient == client
			s.dshMu.Unlock()
			s.mu.Lock()
			sessions := s.sessions
			ready := s.bootStatus.Endpoint == ep
			s.mu.Unlock()
			return same && ready && sessions != nil && sessions.Snapshot().SelectedID == sid
		}
		rpc := s.pluginRPCClient(ep)
		if !current() {
			return
		}
		activity, err := rpc.DshbotActivityTyped(ctx, []string{id})
		if err != nil {
			return
		}
		var authority struct {
			Writable bool `json:"writable"`
		}
		if client.Call(ctx, "settings/describe", nil, &authority) != nil || !authority.Writable || !current() {
			return
		}
		for _, a := range activity.Entries {
			if a.BotID == id && a.SessionID == sid && a.Error == "" && a.ActivitySeq >= 0 {
				_, err = rpc.DshbotMarkRead(ctx, []pluginrpc.DshbotReadMark{{ID: id, ThroughSeq: a.ActivitySeq}})
				if err != nil {
					s.mu.Lock()
					asst := s.assistant
					win := s.win
					s.mu.Unlock()
					if current() && asst != nil {
						asst.SetLastAction(i18n.T(i18n.KeyCmdConversationOpenedReadMarkerFailed), err.Error())
						if win != nil {
							win.Invalidate()
						}
					}
				}
				return
			}
		}
	}()
}
