// Package errcopy turns a failure into the copy a person reads: one localized
// sentence naming the cause and the next step. The raw error text never leaves
// the caller; it belongs in the log.
package errcopy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/ui/i18n"
)

// Cause is a family of failures the shell can explain.
type Cause int

const (
	CauseUnknown Cause = iota
	CauseCancelled
	CauseRuntimeMissing
	CauseRuntimeMismatch
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
	CauseMissingCredential
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
	keyRuntimeMismatch = "errcopy.runtime_mismatch"
	keyPortInUse       = "errcopy.port_in_use"
	keyPageStopped     = "errcopy.page_stopped"
	keyRevertConflict  = "errcopy.revert_conflict"
	keyNoCredential    = "errcopy.no_credential"
	keyRejectedRequest = "errcopy.rejected_request"
	keyRejectedAuth    = "errcopy.rejected_auth"
	keyRejectedRate    = "errcopy.rejected_rate"
	keyRejectedServer  = "errcopy.rejected_server"
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
	// Two programs name these failures, so each family covers both spellings:
	// Go's exec/syscall errors, and the platform's own message when it is the one
	// reporting (Windows winsock has no errno in the text).
	{CauseRuntimeMissing, regexp.MustCompile(`(?i)spawn\s+\S+\s+ENOENT|executable file not found|exec: "\S+": not found|vendored harness (?:not found|not built|missing)|Host runtime is missing|missing its bundled Node|Node\.js [0-9][0-9.+]* is required`)},
	// A runtime that is present but not the one this app was built against is a
	// different problem with a different next step: reinstalling, not repairing
	// a path. Both the provenance receipt and the session reader report it.
	{CauseRuntimeMismatch, regexp.MustCompile(`(?i)compatible Host runtime|Host session reader is |cannot verify the built Host session event registry|differs from the verified Host session runtime`)},
	{CausePortInUse, regexp.MustCompile(`(?i)EADDRINUSE|address already in use|only one usage of each socket address|no free port in `)},
	// A provider route without a credential is the one failure a person fixes in
	// Settings, so it claims its own family ahead of the generic ones: the Host
	// names it after the route ("no API key for provider route \"deepseek-official\"")
	// and the route, the provider and the environment variable are all noise to
	// the reader. The stable code carries the same cause without the wording.
	{CauseMissingCredential, regexp.MustCompile(`(?i)no api key|missing api key|api key is not (?:set|configured)|no api credential|credentials? (?:are )?not configured|missing_credential|missing_api_key|no_api_key|未配置\s*api\s*key|api\s*key\s*未配置`)},
	{CauseCancelled, regexp.MustCompile(`(?i)context canceled|operation was canceled|cancelled by the caller`)},
	{CauseTimeout, regexp.MustCompile(`(?i)deadline exceeded|timed out|timeout`)},
	{CauseNetwork, regexp.MustCompile(`(?i)connection refused|connection reset|no such host|network is unreachable|network is down|dial tcp|broken pipe|tls handshake|unexpected eof|actively refused|connection attempt failed|wsarecv|wsaconnect|remote host closed`)},
	{CauseDiskFull, regexp.MustCompile(`(?i)no space left|disk is full|not enough space`)},
	{CauseReadOnly, regexp.MustCompile(`(?i)read-only file system|read only file system|write-protected`)},
	{CausePermission, regexp.MustCompile(`(?i)access is denied|permission denied|access denied|operation not permitted|\beacces\b|\beperm\b`)},
	{CauseConflict, regexp.MustCompile(`(?i)file changed on disk`)},
	{CauseNotFound, regexp.MustCompile(`(?i)no such file|cannot find|not found|does not exist|\benoent\b`)},
	{CauseRevertConflict, regexp.MustCompile(`(?i)revert_conflict`)},
	{CausePageStopped, regexp.MustCompile(`(?i)render-process-gone|renderer (?:stopped|crashed)|page (?:crashed|stopped expecting)`)},
	{CauseKilled, regexp.MustCompile(`(?i)signal:\s*killed`)},
	// A runtime that is gone shows up as its own exit, and — while another layer
	// is the one reporting — as the carrier to it dying. The exit sentence is the
	// honest next step for both; only a live Host's network failure is different.
	{CauseRuntimeExited, regexp.MustCompile(`(?i)dsh web exited|dsh 进程结束|carrier lost|dsh mux: closed|dsh mux: not connected|host bootstrap (?:unavailable|could not be loaded)|invalid host boot document`)},
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
		"桌面运行时文件缺失，请重新安装桌面端后重试；详情见日志。",
		"A desktop runtime file is missing. Reinstall the desktop app, then try again; the log has the details.")
	i18n.Register(keyRuntimeMismatch,
		"桌面运行时与当前版本不匹配，请重新安装桌面端后重试；详情见日志。",
		"The desktop runtime does not match this app version. Reinstall the desktop app, then try again; the log has the details.")
	i18n.Register(keyRevertConflict,
		"这项改动在读取后又发生了变化，请刷新后再试。",
		"This change moved again after it was read. Refresh, then try again.")
	i18n.Register(keyPageStopped,
		"页面渲染进程已停止，请重新载入页面；详情见日志。",
		"The page's renderer stopped. Reload the page; the log has the details.")
	i18n.Register(keyPortInUse,
		"启动所需端口被其他程序占用，请关闭占用程序后重试。",
		"Another program holds the port this start needs; close it and retry.")
	i18n.Register(keyNoCredential,
		"未配置模型凭据。请在 设置 → 模型 里填写 API Key 后重试。",
		"No model credentials are set. Add an API key in Settings → Models, then try again.")
	// A rejection is the service answering, not the network failing: the status
	// is the one fact a person needs, and it is quoted inside the sentence so the
	// row never shows a bare HTTP_404 chip. A credential rejection, a rate limit
	// and a service fault each need a different next step.
	i18n.Register(keyRejectedRequest,
		"模型服务拒绝了这次请求（%s）。请检查模型名称与地址，或重试；连续失败可查看日志。",
		"The model service rejected this request (%s). Check the model name and address, or retry; if it keeps failing, check the log.")
	i18n.Register(keyRejectedAuth,
		"模型服务拒绝了这次请求（%s）：凭据无效或没有权限。请在 设置 → 模型 里检查 API Key 后重试。",
		"The model service rejected this request (%s): the credential is invalid or lacks permission. Check the API key in Settings → Models, then try again.")
	i18n.Register(keyRejectedRate,
		"模型服务拒绝了这次请求（%s）：请求太多，已被限流。请稍后重试；连续失败可查看日志。",
		"The model service rejected this request (%s): too many requests, so it was rate limited. Try again later; check the log if it keeps failing.")
	i18n.Register(keyRejectedServer,
		"模型服务这次出错了（%s）。请稍后重试；连续失败可查看日志。",
		"The model service failed this request (%s). Try again later; check the log if it keeps failing.")
}

// codeCauses maps the Host's stable failure codes onto causes, so one cause does
// not depend on which provider route happens to be configured: the code is the
// Host's own classification, and the message beside it is written per route.
var codeCauses = map[string]Cause{
	"MISSING_CREDENTIAL": CauseMissingCredential,
	"MISSING_API_KEY":    CauseMissingCredential,
	"NO_API_KEY":         CauseMissingCredential,
	"NO_CREDENTIAL":      CauseMissingCredential,
}

// ClassifyCode reports which family a failure belongs to, preferring the Host's
// stable code and falling back to the message for carriers that keep only text.
func ClassifyCode(code, raw string) Cause {
	if cause, ok := codeCauses[strings.ToUpper(strings.TrimSpace(code))]; ok {
		return cause
	}
	return classifyText(raw)
}

// providerEnvelope is the shape of a provider route's own failure text, the
// string the Host builds for its logs: `llm-<route>: …`. It is never copy, so a
// surface that paints a string it did not compose can ask before showing it.
var providerEnvelope = regexp.MustCompile(`(?i)^\s*llm-[a-z0-9._-]+\s*:`)

// ProviderEnvelope reports whether raw is a provider route's failure text
// rather than something written for a person.
func ProviderEnvelope(raw string) bool {
	return providerEnvelope.MatchString(raw)
}

// Classify reports which family err belongs to. Typed carriers from
// internal/dsh answer first; the message text is only the fallback for
// errors that carry no structure.
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
	var transport *dsh.TransportError
	if errors.As(err, &transport) {
		var ne net.Error
		if errors.As(transport.Err, &ne) && ne.Timeout() {
			return CauseTimeout
		}
		return CauseNetwork
	}
	if errors.Is(err, dsh.ErrEndpointNotReady) {
		// The Host answered that a lazy service is still activating; the
		// honest next step is the retry the timeout sentence offers.
		return CauseTimeout
	}
	var rpcErr *dsh.Error
	if errors.As(err, &rpcErr) {
		if cause, ok := codeCauses[strings.ToUpper(strings.TrimSpace(rpcErr.Code))]; ok {
			return cause
		}
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
	return sentenceFor(classifyText(raw), raw)
}

// SentenceFor is SentenceText for a carrier that also kept the Host's stable
// code, which names the cause even when the message is written per route or left
// out entirely.
func SentenceFor(code, raw string) string {
	if cause, ok := codeCauses[strings.ToUpper(strings.TrimSpace(code))]; ok {
		return sentenceFor(cause, raw)
	}
	if status, ok := HTTPStatus(code, raw); ok {
		return rejectedSentence(status)
	}
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	return sentenceFor(classifyText(raw), raw)
}

// CauseSentenceIn renders the sentence for an already-classified failure in
// the given locale ("" uses the active catalog). The stable code still wins
// over the cause, matching SentenceFor's order: a carrier that kept only the
// classification reproduces the same sentence.
func CauseSentenceIn(locale string, cause Cause, code string) string {
	if c, ok := codeCauses[strings.ToUpper(strings.TrimSpace(code))]; ok {
		cause = c
	}
	if status, ok := HTTPStatus(code, ""); ok {
		return rejectedSentenceIn(locale, status)
	}
	return lookupT(locale, keyFor(cause))
}

func lookupT(locale, key string) string {
	if locale == "" {
		return i18n.T(key)
	}
	return i18n.Lookup(locale, key)
}

func sentenceFor(cause Cause, raw string) string {
	if cause == CauseRuntimeExited {
		if match := exitCode.FindStringSubmatch(raw); match != nil {
			return fmt.Sprintf(i18n.T(keyRuntimeExitCode), match[1])
		}
	}
	return i18n.T(keyFor(cause))
}

// httpStatus is the status a Host writes into a failure code or message:
// HTTP_404, HTTP 429. It is a code, not a sentence, so a surface that shows the
// status quotes it inside one.
var httpStatus = regexp.MustCompile(`(?i)\bHTTP[_ ](\d{3})\b`)

// HTTPStatus reports the status a failure carries, preferring the stable code
// over the message, so a caller can tell a rejection from a transport failure.
func HTTPStatus(code, raw string) (string, bool) {
	for _, text := range []string{code, raw} {
		if match := httpStatus.FindStringSubmatch(text); match != nil {
			return match[1], true
		}
	}
	return "", false
}

// rejectedSentence picks the next step the status calls for.
func rejectedSentence(status string) string {
	return rejectedSentenceIn("", status)
}

func rejectedSentenceIn(locale, status string) string {
	switch {
	case status == "401" || status == "403":
		return fmt.Sprintf(lookupT(locale, keyRejectedAuth), status)
	case status == "429":
		return fmt.Sprintf(lookupT(locale, keyRejectedRate), status)
	case strings.HasPrefix(status, "5"):
		return fmt.Sprintf(lookupT(locale, keyRejectedServer), status)
	default:
		return fmt.Sprintf(lookupT(locale, keyRejectedRequest), status)
	}
}

// Under puts the caller's own description of what stopped in front of the
// cause, so a person sees which action failed as well as why.
func Under(lead string, err error) string {
	lead = strings.TrimRight(strings.TrimSpace(lead), ":：")
	if err == nil {
		return lead
	}
	// An unclassified failure has nothing to name, so the caller's action is the
	// subject of the sentence rather than a prefix on "an operation failed".
	if classifyText(err.Error()) == CauseUnknown {
		return underUnknown(lead, err)
	}
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

// UnderText is Under for a carrier that only kept the message.
func UnderText(lead, raw string) string {
	if strings.TrimSpace(raw) == "" {
		return strings.TrimRight(strings.TrimSpace(lead), ":：")
	}
	return Under(lead, errors.New(raw))
}

func underUnknown(lead string, err error) string {
	if lead == "" {
		return Sentence(err)
	}
	return i18n.Pick(lead+"没有完成。请重试；若仍然失败，请查看日志。",
		lead+" did not finish. Try again; if it keeps failing, check the log.")
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
	case CauseRuntimeMismatch:
		return keyRuntimeMismatch
	case CausePortInUse:
		return keyPortInUse
	case CausePageStopped:
		return keyPageStopped
	case CauseRevertConflict:
		return keyRevertConflict
	case CauseMissingCredential:
		return keyNoCredential
	default:
		return keyUnknown
	}
}
