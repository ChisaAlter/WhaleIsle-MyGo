// Package hoststate turns the supervisor's status into the lines the window
// shows while the local service is not ready. One report per state, shared by
// the startup page, the workbench strip, the sidebar and the composer, so no
// surface can blame the workspace for a service that is not running.
package hoststate

import (
	"whaleisle/internal/errcopy"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

// Phase is what the local service is doing, as every surface reports it.
type Phase int

const (
	NotStarted Phase = iota
	Starting
	Ready
	Failed
)

// Report is one Host state in the words the page, the sidebar and the composer
// show. Every field is a sentence written for a person: what happened and what
// to do next. The raw failure stays in the log drawer.
type Report struct {
	Phase Phase
	// Line is the startup page's status line and the workbench strip's sentence.
	Line string
	// Note replaces the sidebar's empty-list line while the service is not ready.
	Note string
	// Placeholder is the composer's placeholder while the service is not ready.
	Placeholder string
	// Action is the leading button's label; empty unless the person must start
	// or restart the service.
	Action string
}

// Of reads a supervisor status into the report every surface paints.
func Of(st harness.Status) Report {
	if st.RecoveryCancelled {
		// A cancelled automatic recovery is a failure that stays put until the
		// person retries, whatever phase the last attempt reached.
		return Report{
			Phase:       Failed,
			Line:        text("已取消自动恢复，可手动重试。", "Automatic recovery cancelled. You can retry manually."),
			Note:        text("本地服务启动失败，会话暂时无法显示", "The local service failed to start, so sessions stay hidden for now"),
			Placeholder: text("本地服务启动失败，重试成功后可开始", "The local service failed to start; retry to begin"),
			Action:      text("重试", "Retry"),
		}
	}
	switch PhaseOf(st) {
	case Ready:
		return Report{Phase: Ready}
	case Starting:
		return Report{
			Phase:       Starting,
			Line:        text("正在启动本地服务，请稍候。", "Starting the local service. This takes a moment."),
			Note:        text("正在启动本地服务，会话稍后出现", "Starting the local service; sessions appear shortly"),
			Placeholder: text("正在启动本地服务，请稍候", "Starting the local service; this takes a moment"),
		}
	case Failed:
		return Report{
			Phase:       Failed,
			Line:        failureLine(st),
			Note:        text("本地服务启动失败，会话暂时无法显示", "The local service failed to start, so sessions stay hidden for now"),
			Placeholder: text("本地服务启动失败，重试成功后可开始", "The local service failed to start; retry to begin"),
			Action:      failureAction(st),
		}
	default:
		return Report{
			Phase:       NotStarted,
			Line:        text("本地服务没有运行。点击“启动”即可开始使用。", "The local service is not running. Select Start to begin."),
			Note:        text("本地服务启动后，会话会显示在这里", "Sessions appear here once the local service starts"),
			Placeholder: text("启动本地服务后即可开始", "Start the local service to begin"),
			Action:      text("启动", "Start"),
		}
	}
}

// PhaseOf names the state a supervisor status is in for the window.
func PhaseOf(st harness.Status) Phase {
	switch st.State {
	case harness.Ready:
		return Ready
	case harness.Starting:
		return Starting
	case harness.Failed:
		return Failed
	}
	return NotStarted
}

// failureLine is the failure's own sentence: the family errcopy classifies,
// which already names the next step, or the recovery state that has its own.
func failureLine(st harness.Status) string {
	if st.RuntimeFailure {
		return text("桌面端意外退出，可立即重启；原因见日志。",
			"The desktop app exited unexpectedly. Restart now, or check the log.")
	}
	// A classified failure says what it was; an unclassified one must not fall
	// back to a sentence about an operation nobody named.
	if st.Err != nil && errcopy.Classify(st.Err) != errcopy.CauseUnknown {
		if line := errcopy.Sentence(st.Err); line != "" {
			return line
		}
	}
	return text("本地服务没有启动成功，请重试；原因见日志。",
		"The local service did not start. Retry, and check the log for the cause.")
}

// failureAction restarts a service that was ready and then exited, and retries
// one that never came up. Both are the same supervisor call.
func failureAction(st harness.Status) string {
	if st.RuntimeFailure {
		return text("立即重启", "Restart now")
	}
	return text("重试", "Retry")
}

func text(zh, en string) string {
	return i18n.Pick(zh, en)
}
