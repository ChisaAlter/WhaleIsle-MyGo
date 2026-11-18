package main

import (
	"context"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/store/session"
)

// pluginDraftCreation retains one exact Creator admission across a lost receipt
// or a navigation change. The original conversation and its draft never become
// the destination of an asynchronous plugin-creation gesture.
type pluginDraftCreation struct {
	source *session.Store
	task   *session.CreatorTask
	prompt string
	route  mainRouteToken // guarded by mainState.mu
}

func pluginCreatorDraftPrompt(region string) string {
	switch region {
	case "leftbar":
		return resumeText("创建一个带左侧栏入口的 DSH 插件。", "Create a DSH plugin with a left sidebar entry.")
	case "rightbar":
		return resumeText("创建一个带右侧栏入口的 DSH 插件。", "Create a DSH plugin with a right sidebar entry.")
	default:
		return resumeText("创建一个 DSH 会话视图插件。", "Create a DSH conversation view plugin.")
	}
}

func (s *mainState) pluginDraftCurrent(op *pluginDraftCreation) bool {
	s.mu.Lock()
	current := op != nil && s.pluginDraftCreator == op && s.sessions == op.source && !s.quitting &&
		!s.projectWindowHidden && s.mainRouteCurrentLocked(op.route)
	s.mu.Unlock()
	return current
}

func (s *mainState) startCreatorPluginDraft(win *mygo.Window, sessions *session.Store, region string) {
	if win == nil || sessions == nil || !sessions.ClientReady() {
		s.notifyUser(win, hostNotReadyToast())
		return
	}
	route := s.captureMainRoute()
	s.mu.Lock()
	retained := s.pluginDraftCreator
	s.mu.Unlock()
	var previous session.CreatorTaskState
	if retained != nil {
		previous = retained.task.Snapshot()
		if previous.Busy {
			s.notifyUser(win, pluginCreatorText("正在准备 Creator 会话…", "Preparing the Creator conversation…"))
			return
		}
	}
	// An attempted creation has an exact Host identity even when its receipt was
	// lost. Repeated gestures locate/apply/open it; they never create another one.
	op := retained
	if op == nil || !previous.Attempted || previous.Opened {
		op = &pluginDraftCreation{source: sessions, prompt: pluginCreatorDraftPrompt(region), route: route}
		op.task = session.NewCreatorTask(sessions, func() bool { return s.pluginDraftCurrent(op) })
		s.mu.Lock()
		s.pluginDraftCreator = op
		s.mu.Unlock()
	} else if op.source != sessions {
		s.notifyUser(win, pluginCreatorText("请先在原 Host 核对保留的 Creator 会话：", "Check the retained Creator conversation on its original Host: ")+previous.SessionID)
		return
	} else {
		s.mu.Lock()
		op.route = route
		s.mu.Unlock()
	}
	op.task.ObserveVisible(true)
	if !previous.Attempted || previous.Opened {
		op.task.Prepare()
	}
	prepared := op.task.Snapshot()
	if !prepared.Review && !prepared.Attempted {
		s.notifyUser(win, prepared.Error)
		return
	}
	ticket := prepared.Ticket
	s.notifyUser(win, pluginCreatorText("正在准备 Creator 会话…", "Preparing the Creator conversation…"))
	go func() {
		ctx, cancel := context.WithTimeout(s.lifecycleContext(), 45*time.Second)
		defer cancel()
		automatic := !prepared.Attempted
		if automatic {
			op.task.Create(ctx, ticket)
		} else {
			if !prepared.Created {
				op.task.Locate(ctx, ticket)
			}
			located := op.task.Snapshot()
			if located.Created && !located.Applied {
				op.task.ApplyToExisting(ctx, ticket)
			}
		}
		win.Update(func() { s.finishCreatorPluginDraft(win, op, ticket, automatic) })
	}()
}

func (s *mainState) finishCreatorPluginDraft(win *mygo.Window, op *pluginDraftCreation, ticket uint64, automatic bool) {
	result := op.task.Snapshot()
	if !s.pluginDraftCurrent(op) {
		return
	}
	if !result.Created || !result.Applied || !op.task.Open(ticket, automatic) {
		message := result.Error
		if message == "" {
			message = pluginCreatorText("Creator 会话尚未就绪；再次选择创建入口可核对原目标。", "The Creator conversation is not ready; choose the create action again to check the retained target.")
		}
		if result.Attempted {
			message += " · " + result.SessionID
		}
		s.notifyUser(win, message)
		return
	}
	s.mu.Lock()
	// Open selected the confirmed fresh session. Append, rather than replacing,
	// so a locally restored draft for that identity remains intact as well.
	draft := s.composerDraftLocked(result.SessionID)
	if strings.TrimSpace(draft.Text) == "" {
		draft.Text = op.prompt
	} else {
		draft.Text = strings.TrimRight(draft.Text, "\n") + "\n\n" + op.prompt
	}
	draft.Client.Sync()
	s.agentPresetPick = "cordis"
	s.mu.Unlock()
	s.revealMainConversation()
	op.task.ObserveVisible(false)
}
