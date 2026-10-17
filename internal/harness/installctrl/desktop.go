package installctrl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// MarketplaceOptions are the query values accepted by GET
// /desktop/marketplace. They mirror the desktop-install-control.js contract.
type MarketplaceOptions struct {
	Query   string
	Refresh bool
}

// DesktopOps contains the main-process operations behind the authenticated
// /desktop/* control plane. Callbacks intentionally return the operation's
// JSON-shaped value as any: the native shell owns the payload schema and this
// loopback layer only validates the request envelope and forwards it. An
// operation error is returned as HTTP 500 with an auditable error string.
//
// InstallCatalog returns the same Result envelope as InstallFunc. The other
// callbacks may return map[string]any, a struct, or another JSON-marshalable
// value matching the original desktop callback's response.
type DesktopOps struct {
	State            func(context.Context) (any, error)
	ListMarketplace  func(context.Context, MarketplaceOptions) (any, error)
	ApplyConfig      func(context.Context, map[string]any) (any, error)
	PetSettings      func(context.Context) (any, error)
	ApplyPetSettings func(context.Context, map[string]any) (any, error)
	InstallCatalog   func(context.Context, string, InstallOpts) Result
	RemovePlugin     func(context.Context, string) (any, error)
	DisablePlugins   func(context.Context, []string) (any, error)
	EnablePlugin     func(context.Context, string) (any, error)
}

func (s *Server) handleDesktop(w http.ResponseWriter, r *http.Request) {
	// A nil DesktopOps value is deliberate: the harness can run without the
	// native shell. Report that boundary explicitly instead of claiming an
	// operation succeeded.
	if s.desktopEmpty() {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"ok": false, "error": "desktop control not configured",
		})
		return
	}

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/desktop/state":
		s.handleDesktopValue(w, r, s.opts.Desktop.State, "state")
	case r.Method == http.MethodGet && r.URL.Path == "/desktop/marketplace":
		if s.opts.Desktop.ListMarketplace == nil {
			s.handleDesktopValue(w, r, nil, "marketplace")
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		s.handleDesktopValue(w, r, func(ctx context.Context) (any, error) {
			return s.opts.Desktop.ListMarketplace(ctx, MarketplaceOptions{
				Query: q, Refresh: r.URL.Query().Get("refresh") == "1",
			})
		}, "marketplace")
	case r.Method == http.MethodPost && r.URL.Path == "/desktop/config":
		s.handleDesktopPatch(w, r, s.opts.Desktop.ApplyConfig, "config")
	case r.Method == http.MethodGet && r.URL.Path == "/desktop/pet-settings":
		s.handleDesktopValue(w, r, s.opts.Desktop.PetSettings, "pet settings")
	case r.Method == http.MethodPost && r.URL.Path == "/desktop/pet-settings":
		s.handleDesktopPatch(w, r, s.opts.Desktop.ApplyPetSettings, "pet settings")
	case r.Method == http.MethodPost && r.URL.Path == "/desktop/plugin":
		s.handleDesktopPlugin(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not found"})
	}
}

func (s *Server) desktopEmpty() bool {
	d := s.opts.Desktop
	return d.State == nil && d.ListMarketplace == nil && d.ApplyConfig == nil &&
		d.PetSettings == nil && d.ApplyPetSettings == nil && d.InstallCatalog == nil &&
		d.RemovePlugin == nil && d.DisablePlugins == nil && d.EnablePlugin == nil
}

func (s *Server) handleDesktopValue(w http.ResponseWriter, r *http.Request, fn func(context.Context) (any, error), name string) {
	if fn == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{
			"ok": false, "error": "desktop " + name + " operation not configured",
		})
		return
	}
	value, err := fn(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) handleDesktopPatch(w http.ResponseWriter, r *http.Request, fn func(context.Context, map[string]any) (any, error), name string) {
	payload, err := readJSONObject(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if fn == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{
			"ok": false, "error": "desktop " + name + " operation not configured",
		})
		return
	}
	patch, err := rawObject(payload["patch"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "patch must be an object"})
		return
	}
	if patch == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "patch must be an object"})
		return
	}
	value, err := fn(r.Context(), patch)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) handleDesktopPlugin(w http.ResponseWriter, r *http.Request) {
	payload, err := readJSONObject(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	action := strings.TrimSpace(rawString(payload["action"]))
	allowBuilds, err := rawStringSlice(payload["allowBuilds"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "allowBuilds contains an invalid package key"})
		return
	}
	normalized := NormalizeAllowBuilds(allowBuilds)
	if normalized == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "allowBuilds contains an invalid package key"})
		return
	}

	switch action {
	case "install":
		id := strings.TrimSpace(rawString(payload["id"]))
		spec := strings.TrimSpace(rawString(payload["spec"]))
		if id == "" && spec == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing catalog id or install spec"})
			return
		}
		if spec != "" && !IsValidGithubSpec(spec) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "only github:owner/repo[#ref] specs are supported"})
			return
		}
		var result Result
		if id != "" {
			if s.opts.Desktop.InstallCatalog == nil {
				writeJSON(w, http.StatusNotImplemented, map[string]any{
					"ok": false, "error": "desktop catalog install operation not configured",
				})
				return
			}
			result = s.opts.Desktop.InstallCatalog(r.Context(), id, InstallOpts{AllowBuilds: normalized})
		} else {
			install := s.opts.Install
			if install == nil {
				result = Result{OK: false, Spec: spec, Error: "install handler not configured"}
			} else {
				result = install(spec, InstallOpts{AllowBuilds: normalized})
			}
		}
		result = normalizeResult(result, specOrID(spec, id))
		body := map[string]any{
			"ok":               result.OK,
			"needsAllowBuilds": result.NeedsAllowBuilds,
			"allowBuilds":      result.AllowBuilds,
			"spec":             result.Spec,
			"error":            result.Error,
			"log":              result.Log,
			"restarting":       result.OK && !result.NeedsAllowBuilds,
		}
		writeJSON(w, http.StatusOK, body)
		if result.OK && !result.NeedsAllowBuilds {
			s.scheduleRestart()
		}
	case "remove":
		name := strings.TrimSpace(rawString(payload["name"]))
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing plugin name"})
			return
		}
		if s.opts.Desktop.RemovePlugin == nil {
			writeJSON(w, http.StatusNotImplemented, map[string]any{
				"ok": false, "error": "desktop plugin removal operation not configured",
			})
			return
		}
		value, err := s.opts.Desktop.RemovePlugin(r.Context(), name)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		ok := jsonValueOK(value)
		writeJSON(w, http.StatusOK, withJSONFields(value, map[string]any{"ok": ok, "restarting": ok}))
		if ok {
			s.scheduleRestart()
		}
	case "disable", "enable":
		if action == "disable" {
			names, err := pluginNames(payload)
			if err != nil || len(names) == 0 {
				writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing plugin name"})
				return
			}
			if s.opts.Desktop.DisablePlugins == nil {
				writeJSON(w, http.StatusNotImplemented, map[string]any{
					"ok": false, "error": "desktop plugin disable operation not configured",
				})
				return
			}
			value, err := s.opts.Desktop.DisablePlugins(r.Context(), names)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, value)
			if jsonValueOK(value) && !jsonValueBool(value, "harnessRestarted") {
				s.scheduleRestart()
			}
			return
		}
		name := strings.TrimSpace(rawString(payload["name"]))
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "missing plugin name"})
			return
		}
		if s.opts.Desktop.EnablePlugin == nil {
			writeJSON(w, http.StatusNotImplemented, map[string]any{
				"ok": false, "error": "desktop plugin enable operation not configured",
			})
			return
		}
		value, err := s.opts.Desktop.EnablePlugin(r.Context(), name)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, value)
		if jsonValueOK(value) && !jsonValueBool(value, "harnessRestarted") {
			s.scheduleRestart()
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": fmt.Sprintf("unknown action: %s", action)})
	}
}

func readJSONObject(r *http.Request) (map[string]json.RawMessage, error) {
	body, err := readBody(r)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return map[string]json.RawMessage{}, nil
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil || payload == nil {
		return nil, errorsInvalidJSON{}
	}
	return payload, nil
}

type errorsInvalidJSON struct{}

func (errorsInvalidJSON) Error() string { return "invalid json" }

func rawObject(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, fmt.Errorf("patch must be an object")
	}
	return value, nil
}

func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return value
}

func rawStringSlice(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var value []string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func pluginNames(payload map[string]json.RawMessage) ([]string, error) {
	if raw := payload["names"]; len(raw) > 0 && string(raw) != "null" {
		var names []string
		if err := json.Unmarshal(raw, &names); err != nil {
			return nil, err
		}
		out := make([]string, 0, len(names))
		for _, name := range names {
			if name = strings.TrimSpace(name); name != "" {
				out = append(out, name)
			}
		}
		return out, nil
	}
	if name := strings.TrimSpace(rawString(payload["name"])); name != "" {
		return []string{name}, nil
	}
	return nil, nil
}

func specOrID(spec, id string) string {
	if spec != "" {
		return spec
	}
	return id
}

func jsonValueOK(value any) bool {
	if result, ok := value.(Result); ok {
		return result.OK
	}
	data, err := json.Marshal(value)
	if err != nil {
		return false
	}
	var envelope struct {
		OK bool `json:"ok"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return false
	}
	return envelope.OK
}

func jsonValueBool(value any, key string) bool {
	data, err := json.Marshal(value)
	if err != nil {
		return false
	}
	var envelope map[string]any
	if json.Unmarshal(data, &envelope) != nil {
		return false
	}
	flag, _ := envelope[key].(bool)
	return flag
}

func withJSONFields(value any, fields map[string]any) any {
	data, err := json.Marshal(value)
	if err != nil {
		out := map[string]any{}
		for k, v := range fields {
			out[k] = v
		}
		return out
	}
	var out map[string]any
	if json.Unmarshal(data, &out) != nil || out == nil {
		out = map[string]any{}
	}
	for k, v := range fields {
		out[k] = v
	}
	return out
}
