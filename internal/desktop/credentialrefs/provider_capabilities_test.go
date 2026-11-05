package credentialrefs

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"whaleisle/internal/dsh/api"
)

type capabilityHost struct {
	*fakeHost
	described []string
	omit      map[string]bool
}

func (h *capabilityHost) Call(ctx context.Context, method string, args, out any) error {
	if err := h.fakeHost.Call(ctx, method, args, out); err != nil {
		return err
	}
	if method == "credentials/describe" {
		h.described = append(h.described, args.(map[string]any)["refs"].([]string)...)
		states := *out.(*map[string]credentialInfo)
		for ref, state := range states {
			if h.omit[ref] {
				delete(states, ref)
			} else {
				state.Source = "environment"
				states[ref] = state
			}
		}
	}
	return nil
}

func providerCapabilitiesHost() *capabilityHost {
	h := &capabilityHost{fakeHost: fixtureHost()}
	values := map[string]any{}
	h.providers = nil
	add := func(route string, auth any, endpoint []string) {
		values[route] = map[string]any{}
		p := map[string]any{"provider": route, "displayName": route, "settingsNs": "llm-pi-ai", "settingsPath": []string{"providers", route}}
		if auth != nil {
			p["nativeAuth"] = auth
		}
		if endpoint != nil {
			p["baseURLEnv"] = endpoint
		}
		h.providers = append(h.providers, p)
	}
	add("openai", map[string]any{"env": []string{"OPENAI_API_KEY"}, "oauth": false}, nil)
	add("azure", map[string]any{"env": []string{"AZURE_OPENAI_API_KEY", "AZURE_OPENAI_BASE_URL"}, "oauth": false}, []string{"AZURE_OPENAI_BASE_URL", "AZURE_OPENAI_RESOURCE_NAME"})
	add("anthropic", map[string]any{"env": []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_OAUTH_TOKEN", "ANTHROPIC_API_KEY"}, "oauth": true}, nil)
	add("openai-codex", map[string]any{"env": []string{}, "oauth": true}, nil)
	add("custom", nil, nil)
	h.settings["namespaces"].([]any)[0].(map[string]any)["value"] = map[string]any{"providers": values}
	h.configured["ANTHROPIC_AUTH_TOKEN"] = true
	return h
}

func capabilityRow(t *testing.T, c *Controller, route string) Row {
	t.Helper()
	for _, row := range c.Snapshot().Rows {
		if row.Provider == route {
			return row
		}
	}
	t.Fatalf("provider %s missing", route)
	return Row{}
}

func TestProviderCapabilitiesNativeMetadataIsReadOnlyAndOrdered(t *testing.T) {
	h := providerCapabilitiesHost()
	h.omit = map[string]bool{"ANTHROPIC_API_KEY": true}
	var c Controller
	c.Attach(h)
	c.Refresh(context.Background())
	if s := c.Snapshot(); !s.Ready || s.Err != "" {
		t.Fatalf("load failed: %+v", s)
	}
	r := capabilityRow(t, &c, "anthropic")
	want := []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_OAUTH_TOKEN", "ANTHROPIC_API_KEY"}
	if r.NamedReference || !r.BindReference || !slices.Equal(r.NativeAuth.Env, want) || len(r.NativeChecks) != 3 {
		t.Fatalf("native metadata lost: %+v", r)
	}
	for i, check := range r.NativeChecks {
		if check.Ref != want[i] {
			t.Fatal("native authentication order changed")
		}
	}
	if !r.NativeChecks[0].Configured || r.NativeChecks[0].Source != "environment" || r.NativeChecks[2].Known {
		t.Fatalf("presence/unknown/source lost: %+v", r.NativeChecks)
	}
	azure := capabilityRow(t, &c, "azure")
	if len(azure.NativeChecks) != 1 || azure.NativeChecks[0].Ref != "AZURE_OPENAI_API_KEY" || slices.Contains(h.described, "AZURE_OPENAI_BASE_URL") || slices.Contains(h.described, "AZURE_OPENAI_RESOURCE_NAME") {
		t.Fatal("endpoint variables were offered as authentication references")
	}
	if slices.Contains(h.described, "OPENAI_CODEX_API_KEY") || len(h.writes) != 0 {
		t.Fatal("read introduced a key target or wrote configuration")
	}
	// Callers cannot mutate the next view's metadata through a snapshot.
	r.NativeAuth.Env[0] = "MODIFIED"
	r.NativeChecks[0].Ref = "MODIFIED"
	azure.BaseURLEnv[0] = "MODIFIED"
	if next := capabilityRow(t, &c, "anthropic"); next.NativeAuth.Env[0] != want[0] || next.NativeChecks[0].Ref != want[0] {
		t.Fatal("snapshot aliased authentication metadata")
	}
	if capabilityRow(t, &c, "azure").BaseURLEnv[0] != "AZURE_OPENAI_BASE_URL" {
		t.Fatal("snapshot aliased endpoint metadata")
	}
}

func TestProviderCapabilitiesKeylessAndNamedReferencesPreserveConditions(t *testing.T) {
	for _, named := range []string{"omitted", "", "EXPLICIT_KEY"} {
		t.Run(named, func(t *testing.T) {
			h := providerCapabilitiesHost()
			profiles := h.settings["namespaces"].([]any)[0].(map[string]any)["value"].(map[string]any)["providers"].(map[string]any)
			if named != "omitted" {
				profiles["openai-codex"].(map[string]any)["apiKeyEnv"] = named
			}
			var c Controller
			c.Attach(h)
			c.Refresh(context.Background())
			r := capabilityRow(t, &c, "openai-codex")
			c.Select(r.ID)
			s := c.Snapshot()
			c.Edit(s.Ticket, "synthetic-key")
			c.Save(context.Background(), s.Ticket)
			if named == "EXPLICIT_KEY" {
				if r.KeylessNativeAuth() || !r.NamedReference || !r.Writable || r.BindReference || len(h.writes) != 1 || h.writes[0]["ref"] != named {
					t.Fatal("existing explicit reference became unusable or rebound")
				}
			} else if !r.KeylessNativeAuth() || r.Writable || len(h.writes) != 0 {
				t.Fatal("OAuth-only route admitted an API-key write")
			}
		})
	}
}

func TestProviderCapabilitiesNamedReferenceWinsOverNativeChecks(t *testing.T) {
	h := providerCapabilitiesHost()
	profiles := h.settings["namespaces"].([]any)[0].(map[string]any)["value"].(map[string]any)["providers"].(map[string]any)
	profiles["anthropic"].(map[string]any)["apiKeyEnv"] = "CHOSEN_KEY"
	var c Controller
	c.Attach(h)
	c.Refresh(context.Background())
	r := capabilityRow(t, &c, "anthropic")
	if !r.NamedReference || r.Ref != "CHOSEN_KEY" || len(r.NativeChecks) != 0 || slices.Contains(h.described, "ANTHROPIC_AUTH_TOKEN") || slices.Contains(h.described, "ANTHROPIC_OAUTH_TOKEN") {
		t.Fatal("native metadata overrode the named credential")
	}
}

func TestProviderCapabilitiesChangedNativeMetadataRevokesKeyWrite(t *testing.T) {
	h := providerCapabilitiesHost()
	var codex map[string]any
	for _, p := range h.providers {
		if p["provider"] == "openai-codex" {
			codex = p
			delete(p, "nativeAuth")
		}
	}
	var c Controller
	c.Attach(h)
	c.Refresh(context.Background())
	r := capabilityRow(t, &c, "openai-codex")
	c.Select(r.ID)
	s := c.Snapshot()
	c.Edit(s.Ticket, "synthetic-key")
	codex["nativeAuth"] = map[string]any{"env": []string{}, "oauth": true}
	c.Save(context.Background(), s.Ticket)
	if len(h.writes) != 0 || c.Snapshot().Err == "" {
		t.Fatal("fresh OAuth-only capability did not revoke a prepared key write")
	}
}

func TestProviderCapabilitiesCustomAndOrdinaryKeysKeepMinimalWrites(t *testing.T) {
	for _, route := range []string{"openai", "custom"} {
		t.Run(route, func(t *testing.T) {
			h := providerCapabilitiesHost()
			var c Controller
			c.Attach(h)
			c.Refresh(context.Background())
			r := capabilityRow(t, &c, route)
			c.Select(r.ID)
			s := c.Snapshot()
			c.Edit(s.Ticket, " ")
			c.Save(context.Background(), s.Ticket)
			if len(h.writes) != 0 {
				t.Fatal("blank input created a reference")
			}
			c.Edit(s.Ticket, "synthetic-key")
			c.Save(context.Background(), s.Ticket)
			if len(h.writes) != 2 || h.writes[0]["expectedRevision"] != float64(7) || h.writes[1]["ref"] != deriveRef(route) || c.Snapshot().Err != "" {
				t.Fatalf("ordinary reference binding changed: %+v", c.Snapshot())
			}
		})
	}
}

func TestProviderCapabilitiesPublishedOptionalCodec(t *testing.T) {
	var directory api.ListConfigurableProvidersResult
	if err := json.Unmarshal([]byte(`[{"provider":"minimal","displayName":"Minimal","settingsNs":"llm-pi-ai","settingsPath":[]},{"provider":"empty","displayName":"Empty","settingsNs":"llm-pi-ai","settingsPath":[],"defaultBaseURL":"","baseURLEnv":[],"nativeAuth":{"env":[],"oauth":false}},{"provider":"full","displayName":"Full","settingsNs":"llm-pi-ai","settingsPath":[],"defaultBaseURL":"https://example.invalid/v1","baseURLEnv":["REGION","RESOURCE"],"nativeAuth":{"env":["PRIMARY","SECONDARY"],"oauth":true}}]`), &directory); err != nil {
		t.Fatal(err)
	}
	if directory[0].DefaultBaseURL != nil || directory[0].NativeAuth != nil || directory[1].DefaultBaseURL == nil || *directory[1].DefaultBaseURL != "" || directory[1].NativeAuth == nil || directory[1].NativeAuth.Oauth || !slices.Equal(directory[2].NativeAuth.Env, []string{"PRIMARY", "SECONDARY"}) || !slices.Equal(directory[2].BaseURLEnv, []string{"REGION", "RESOURCE"}) {
		t.Fatal("optional provider capability wire was not retained")
	}
}
