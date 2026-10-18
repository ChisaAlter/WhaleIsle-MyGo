// Package office implements the original dsh-im Office Connector control plane.
// It never opens an Office connection itself or persists Device Token drafts.
package office

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const ProtocolVersion = "office-harness.v1"

type Config struct {
	ProtocolVersion    string            `json:"protocolVersion"`
	BaseURL            string            `json:"baseUrl"`
	DeviceID           string            `json:"deviceId"`
	MaxConcurrency     int               `json:"maxConcurrency"`
	HeartbeatSeconds   int               `json:"heartbeatSeconds"`
	Workspaces         map[string]string `json:"workspaces"`
	InstructionPresets map[string]string `json:"instructionPresets"`
}
type Health struct {
	LastHeartbeatAt string `json:"lastHeartbeatAt"`
	LastEventType   string `json:"lastEventType"`
	Reconnects      int    `json:"reconnects"`
	JobsOffered     int    `json:"jobsOffered"`
	Jobs            struct {
		Running   int `json:"running"`
		Completed int `json:"completed"`
	} `json:"jobs"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
type Status struct {
	Configured      bool   `json:"configured"`
	Connected       bool   `json:"connected"`
	State           string `json:"state"`
	TokenConfigured bool   `json:"tokenConfigured"`
	Config          Config `json:"config"`
	Health          Health `json:"health"`
}

// DeviceToken is user-entered, memory-only and never populated from a response.
// It is omitted from JSON, diagnostic String output, and cleared after dispatch.
type Draft struct {
	BaseURL, DeviceID, MaxConcurrency, HeartbeatSeconds, Workspaces, InstructionPresets string
	DeviceToken                                                                         string `json:"-"`
}

func (d Draft) String() string   { return "Office draft (credential omitted)" }
func (d Draft) GoString() string { return d.String() }
func defaultDraft() Draft {
	return Draft{DeviceID: "local-harness", MaxConcurrency: "1", HeartbeatSeconds: "30"}
}
func draftFor(s Status) Draft {
	if !s.Configured {
		return defaultDraft()
	}
	c := s.Config
	return Draft{BaseURL: c.BaseURL, DeviceID: c.DeviceID, MaxConcurrency: strconv.Itoa(c.MaxConcurrency), HeartbeatSeconds: strconv.Itoa(c.HeartbeatSeconds), Workspaces: mapText(c.Workspaces), InstructionPresets: mapText(c.InstructionPresets)}
}
func mapText(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var lines []string
	for _, k := range keys {
		lines = append(lines, k+"="+m[k])
	}
	return strings.Join(lines, "\n")
}

var aliasPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)
var devicePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{2,63}$`)

func absolutePath(p string) bool {
	return filepath.IsAbs(p) || strings.HasPrefix(p, `\\`) || (len(p) >= 3 && ((p[0] >= 'a' && p[0] <= 'z') || (p[0] >= 'A' && p[0] <= 'Z')) && p[1] == ':' && (p[2] == '\\' || p[2] == '/'))
}
func parseMap(text string, workspace bool) (map[string]string, error) {
	out := map[string]string{}
	if len(text) > 512<<10 {
		return nil, errors.New("映射过长")
	}
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if !ok || !aliasPattern.MatchString(k) || v == "" || strings.ContainsRune(v, 0) {
			return nil, errors.New("映射每行须为 alias=value；alias 为 2–64 位小写字母、数字或连字符")
		}
		if _, duplicate := out[k]; duplicate {
			return nil, errors.New("映射 alias 不可重复")
		}
		if workspace && !absolutePath(v) {
			return nil, errors.New("Workspace 映射必须使用 Host 上的绝对路径")
		}
		if !workspace && len([]rune(v)) > 8000 {
			return nil, errors.New("Instruction Preset 不可超过 8000 字符")
		}
		out[k] = v
	}
	return out, nil
}
func normalizeBase(value string) (string, error) {
	u, e := url.Parse(strings.TrimSpace(value))
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return "", errors.New("Office URL 必须为不含账号、查询或片段的 HTTPS 地址")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	local := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return "", errors.New("Office URL 必须使用 HTTPS；HTTP 仅允许本机测试")
	}
	u.Path = ""
	u.RawPath = ""
	return strings.TrimRight(u.String(), "/"), nil
}
func configurePayload(d Draft, current Status) (map[string]any, error) {
	base, e := normalizeBase(d.BaseURL)
	if e != nil {
		return nil, e
	}
	id := strings.TrimSpace(d.DeviceID)
	if !devicePattern.MatchString(id) {
		return nil, errors.New("Device ID 须为 3–64 位字母、数字、下划线或连字符")
	}
	n, e := strconv.Atoi(strings.TrimSpace(d.MaxConcurrency))
	if e != nil || n < 1 || n > 4 {
		return nil, errors.New("最大并发须为 1–4 的整数")
	}
	h, e := strconv.Atoi(strings.TrimSpace(d.HeartbeatSeconds))
	if e != nil || h < 10 || h > 300 {
		return nil, errors.New("Heartbeat 须为 10–300 的整数秒")
	}
	work, e := parseMap(d.Workspaces, true)
	if e != nil {
		return nil, e
	}
	presets, e := parseMap(d.InstructionPresets, false)
	if e != nil {
		return nil, e
	}
	token := strings.TrimSpace(d.DeviceToken)
	if token != "" && len(token) < 32 {
		return nil, errors.New("Device Token 至少需要 32 个字符")
	}
	if token == "" && (!current.Configured || !current.TokenConfigured || base != current.Config.BaseURL || id != current.Config.DeviceID) {
		return nil, errors.New("新连接或更换 Office / Device ID 时请重新输入对应的 Device Token")
	}
	payload := map[string]any{"baseUrl": base, "deviceId": id, "maxConcurrency": n, "heartbeatSeconds": h, "workspaces": work, "instructionPresets": presets}
	if token != "" {
		payload["deviceToken"] = token
	}
	return payload, nil
}

// The Host exposes no revision/CAS. This is a local fingerprint of PUBLIC config,
// not a server revision; secret-only rotations and the preflight/write gap remain.
func fingerprint(s Status) [32]byte {
	raw, _ := json.Marshal(struct {
		Configured, TokenConfigured bool
		Config                      Config
	}{s.Configured, s.TokenConfigured, s.Config})
	return sha256.Sum256(raw)
}
func safeError(code string) string {
	switch code {
	case "invalid-device-token":
		return "AI Office 拒绝了 Device Token。"
	case "office-hook-unavailable":
		return "AI Office Hook 尚未上线或地址不正确。"
	case "office-protocol-mismatch":
		return "AI Office Connector 协议版本不兼容。"
	case "office-transport-failed":
		return "Host 暂时无法访问 AI Office。"
	default:
		return "AI Office 连接中断；请检查 Host 和 Office 状态。"
	}
}
func parseStatus(raw json.RawMessage) (Status, bool, error) {
	var outer struct {
		Snapshot json.RawMessage `json:"snapshot"`
		Tested   bool            `json:"tested"`
	}
	if json.Unmarshal(raw, &outer) != nil {
		return Status{}, false, errors.New("AI Office 响应无法识别")
	}
	tested := outer.Tested
	if len(outer.Snapshot) > 0 {
		raw = outer.Snapshot
	}
	var wire struct {
		SchemaVersion int   `json:"schemaVersion"`
		Configured    *bool `json:"configured"`
		Status
	}
	if json.Unmarshal(raw, &wire) != nil || wire.SchemaVersion != 1 || wire.Configured == nil {
		return Status{}, false, errors.New("AI Office 状态未确认，请刷新或使用插件页面")
	}
	s := wire.Status
	s.Configured = *wire.Configured
	if !s.Configured {
		return Status{State: "unconfigured"}, tested, nil
	}
	base, e := normalizeBase(s.Config.BaseURL)
	if e != nil || base != s.Config.BaseURL || !devicePattern.MatchString(s.Config.DeviceID) || s.Config.ProtocolVersion != ProtocolVersion {
		return Status{}, false, errors.New("AI Office 配置或协议不受支持，请使用插件页面")
	}
	if s.Config.Workspaces == nil {
		s.Config.Workspaces = map[string]string{}
	}
	if s.Config.InstructionPresets == nil {
		s.Config.InstructionPresets = map[string]string{}
	}
	// Validate the public snapshot without requiring or reading a credential.
	d := draftFor(s)
	d.DeviceToken = strings.Repeat("x", 32)
	if _, e = configurePayload(d, s); e != nil {
		return Status{}, false, errors.New("AI Office 返回了无效配置")
	}
	switch s.State {
	case "connected", "connecting", "reconnecting", "missing-token", "idle":
	default:
		s.State = "idle"
	}
	if _, e := time.Parse(time.RFC3339, s.Health.LastHeartbeatAt); e != nil {
		s.Health.LastHeartbeatAt = ""
	}
	switch s.Health.LastEventType {
	case "job.available", "job.cancelled", "ping", "connected", "heartbeat":
	default:
		s.Health.LastEventType = ""
	}
	if s.Health.Error != nil {
		msg := safeError(s.Health.Error.Code)
		code := s.Health.Error.Code
		switch code {
		case "invalid-device-token", "office-hook-unavailable", "office-protocol-mismatch", "office-transport-failed":
		default:
			code = "office-connection-failed"
		}
		s.Health.Error = &struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{code, msg}
	}
	return s, tested, nil
}
func StateLabel(s Status) string {
	if !s.Configured {
		return "尚未配置"
	}
	if s.Connected {
		return "已连接 Office"
	}
	switch s.State {
	case "connecting":
		return "配置已保存，正在连接"
	case "reconnecting":
		return "配置已保存，等待重连"
	case "missing-token":
		return "配置已保存，凭据缺失"
	default:
		return "配置已保存，Office 尚未连接"
	}
}
func HookURLs(base string) map[string]string {
	origin, e := normalizeBase(base)
	if e != nil {
		return nil
	}
	return map[string]string{"SSE": origin + "/api/harness/connector/stream", "Heartbeat": origin + "/api/harness/connector/heartbeat", "Job": origin + "/api/harness/connector/jobs/:id", "Result": origin + "/api/harness/connector/jobs/:id/result"}
}
