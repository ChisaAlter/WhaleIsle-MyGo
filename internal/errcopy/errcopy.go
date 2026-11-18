// Package errcopy turns a failure into the copy a person reads: one localized
// sentence naming the cause and the next step. The raw error text never leaves
// the caller; it belongs in the log.
package errcopy

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"whaleisle/internal/ui/i18n"
)

// Cause is a family of failures the shell can explain.
type Cause int

const (
	CauseUnknown Cause = iota
	CauseCancelled
	CauseRuntimeMissing
	CausePortInUse
	CauseTimeout
	CauseNetwork
	CauseDiskFull
	CauseReadOnly
	CausePermission
	CauseConflict
	CauseNotFound
	CauseExited
	CauseRuntimeExited
	CauseKilled
	CausePageStopped
	CauseRevertConflict
)

const (
	keySeparator       = "errcopy.separator"
	keyUnknown         = "errcopy.unknown"
	keyCancelled       = "errcopy.cancelled"
	keyTimeout         = "errcopy.timeout"
	keyNetwork         = "errcopy.network"
	keyDiskFull        = "errcopy.disk_full"
	keyReadOnly        = "errcopy.read_only"
	keyPermission      = "errcopy.permission"
	keyConflict        = "errcopy.conflict"
	keyNotFound        = "errcopy.not_found"
	keyExited          = "errcopy.exited"
	keyRuntimeExited   = "errcopy.runtime_exited"
	keyRuntimeExitCode = "errcopy.runtime_exit_code"
	keyKilled          = "errcopy.killed"
	keyRuntimeMissing  = "errcopy.runtime_missing"
	keyPortInUse       = "errcopy.port_in_use"
	keyPageStopped     = "errcopy.page_stopped"
	keyRevertConflict  = "errcopy.revert_conflict"
)

// causes is ordered: an earlier family claims a message that a later pattern
// would also match, so the more specific explanation wins.
//
// The runtime-missing, port and process-exit entries cover the startup page's
// failures, which used to keep a table of their own and had drifted from this
// one. This table is the product's only classifier; the page prints these
// sentences.
//
// A runtime exit is its own family: the supervised desktop process dying is a
// different problem from a command that returned non-zero, and only the former
// earns "restart the app". The runtime's wrapper ("dsh web exited") is what
// tells them apart.
var causes = []struct {
	cause   Cause
	pattern *regexp.Regexp
}{
	{CauseRuntimeMissing, regexp.MustCompile(`(?i)spawn\s+\S+\s+ENOENT`)},
	{CausePortInUse, regexp.MustCompile(`(?i)EADDRINUSE|address already in use`)},
	{CauseCancelled, regexp.MustCompile(`(?i)context canceled|operation was canceled|cancelled by the caller`)},
	{CauseTimeout, regexp.MustCompile(`(?i)deadline exceeded|timed out|timeout`)},
	{CauseNetwork, regexp.MustCompile(`(?i)connection refused|connection reset|no such host|network is unreachable|network is down|dial tcp|broken pipe|tls handshake|unexpected eof`)},
	{CauseDiskFull, regexp.MustCompile(`(?i)no space left|disk is full|not enough space`)},
	{CauseReadOnly, regexp.MustCompile(`(?i)read-only file system|read only file system|write-protected`)},
	{CausePermission, regexp.MustCompile(`(?i)access is denied|permission denied|access denied|operation not permitted|\beacces\b|\beperm\b`)},
	{CauseConflict, regexp.MustCompile(`(?i)file changed on disk`)},
	{CauseNotFound, regexp.MustCompile(`(?i)no such file|cannot find|not found|does not exist|\benoent\b`)},
	{CauseRevertConflict, regexp.MustCompile(`(?i)revert_conflict`)},
	{CausePageStopped, regexp.MustCompile(`(?i)render-process-gone|renderer (?:stopped|crashed)|page (?:crashed|stopped expecting)`)},
	{CauseKilled, regexp.MustCompile(`(?i)signal:\s*killed`)},
	{CauseRuntimeExited, regexp.MustCompile(`(?i)dsh web exited|dsh 进程结束`)},
	{CauseExited, regexp.MustCompile(`(?i)exit status|exited with|process exited|signal:`)},
}

// exitCode is the code the desktop runtime reported when it exited. It is
// quoted inside a written sentence, so a person reads why startup stopped
// without reading the transport's "exit status N" line. Only a runtime exit
// names the code: any other command's status belongs in the log.
var exitCode = regexp.MustCompile(`(?i)exit status\s+(\d+)`)

func init() {
	i18n.Register(keySeparator, "：", ": ")
	i18n.Register(keyUnknown,
		"操作没有完成。请重试；若仍然失败，请查看日志。",
		"The operation did not finish. Try again; if it keeps failing, check the log.")
	i18n.Register(keyCancelled,
		"操作已取消，需要时可以重试。",
		"The operation was cancelled. Try again when you are ready.")
	i18n.Register(keyTimeout,
		"操作超时，网络或服务没有及时响应。请重试；详情见日志。",
		"The operation timed out; the network or the service did not answer in time. Try again, and check the log for details.")
	i18n.Register(keyNetwork,
		"网络连接不可用。请检查网络后重试；详情见日志。",
		"The network is unavailable. Check your connection and try again; the log has the details.")
	i18n.Register(keyDiskFull,
		"磁盘空间不足。请清理空间后重试；详情见日志。",
		"There is not enough disk space. Free some space and try again; the log has the details.")
	i18n.Register(keyReadOnly,
		"该位置不可写入。请换一个位置，或调整权限后重试；详情见日志。",
		"That location cannot be written. Choose another one or fix its permissions, then try again; the log has the details.")
	i18n.Register(keyPermission,
		"没有权限访问所需资源，请检查安全软件拦截或系统权限后重试；详情见日志。",
		"Access to a required resource was denied. Check security software or system permissions, then try again; the log has the details.")
	i18n.Register(keyConflict,
		"文件已在磁盘上改变，草稿已保留。请再次保存以覆盖当前文件；详情见日志。",
		"The file changed on disk; your draft was kept. Save again to overwrite the current file; the log has the details.")
	i18n.Register(keyNotFound,
		"找不到需要的文件或位置。请确认它仍然存在后重试；详情见日志。",
		"The file or location is missing. Confirm it still exists and try again; the log has the details.")
	i18n.Register(keyExited,
		"有一步操作没有完成就退出了，请重试；若仍然失败，请查看日志。",
		"A step exited before it finished. Retry, or check the log if it keeps failing.")
	i18n.Register(keyRuntimeExited,
		"桌面运行时已退出，可重试或到日志里查看原因。",
		"The desktop runtime exited; retry, or check the log for the cause.")
	i18n.Register(keyRuntimeExitCode,
		"桌面运行时已退出（退出码 %s），可重试或到日志里查看原因。",
		"The desktop runtime exited (code %s); retry, or check the log for the cause.")
	i18n.Register(keyKilled,
		"桌面运行时被系统结束，可重试或到日志里查看原因。",
		"The desktop runtime was stopped by the system; retry, or check the log for the cause.")
	i18n.Register(keyRuntimeMissing,
		"桌面运行时文件缺失，请重新安装桌面端。",
		"A desktop runtime file is missing; reinstall the desktop app.")
	i18n.Register(keyRevertConflict,
		"这项改动在读取后又发生了变化，请刷新后再试。",
		"This change moved again after it was read. Refresh, then try again.")
	i18n.Register(keyPageStopped,
		"页面渲染进程已停止，请重新载入页面；详情见日志。",
		"The page's renderer stopped. Reload the page; the log has the details.")
	i18n.Register(keyPortInUse,
		"启动所需端口被其他程序占用，请关闭占用程序后重试。",
		"Another program holds the port this start needs; close it and retry.")
}

// Classify reports which family err belongs to.
func Classify(err error) Cause {
	if err == nil {
		return CauseUnknown
	}
	switch {
	case errors.Is(err, context.Canceled):
		return CauseCancelled
	case errors.Is(err, context.DeadlineExceeded):
		return CauseTimeout
	}
	return classifyText(err.Error())
}

func classifyText(raw string) Cause {
	for _, entry := range causes {
		if entry.pattern.MatchString(raw) {
			return entry.cause
		}
	}
	return CauseUnknown
}

// Sentence returns the localized sentence for err, or "" when err is nil or
// carries no text.
func Sentence(err error) string {
	if err == nil {
		return ""
	}
	return SentenceText(err.Error())
}

// SentenceText is Sentence for a carrier that only kept the message.
func SentenceText(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	cause := classifyText(raw)
	if cause == CauseRuntimeExited {
		if match := exitCode.FindStringSubmatch(raw); match != nil {
			return fmt.Sprintf(i18n.T(keyRuntimeExitCode), match[1])
		}
	}
	return i18n.T(keyFor(cause))
}

// Under puts the caller's own description of what stopped in front of the
// cause, so a person sees which action failed as well as why.
func Under(lead string, err error) string {
	lead = strings.TrimRight(strings.TrimSpace(lead), ":：")
	sentence := Sentence(err)
	switch {
	case lead == "":
		return sentence
	case sentence == "":
		return lead
	default:
		return lead + i18n.T(keySeparator) + sentence
	}
}

func keyFor(cause Cause) string {
	switch cause {
	case CauseCancelled:
		return keyCancelled
	case CauseTimeout:
		return keyTimeout
	case CauseNetwork:
		return keyNetwork
	case CauseDiskFull:
		return keyDiskFull
	case CauseReadOnly:
		return keyReadOnly
	case CausePermission:
		return keyPermission
	case CauseConflict:
		return keyConflict
	case CauseNotFound:
		return keyNotFound
	case CauseExited:
		return keyExited
	case CauseRuntimeExited:
		return keyRuntimeExited
	case CauseKilled:
		return keyKilled
	case CauseRuntimeMissing:
		return keyRuntimeMissing
	case CausePortInUse:
		return keyPortInUse
	case CausePageStopped:
		return keyPageStopped
	case CauseRevertConflict:
		return keyRevertConflict
	default:
		return keyUnknown
	}
}
