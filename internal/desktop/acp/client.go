package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"whaleisle/internal/desktop/pluginrpc"
)

type Field struct {
	Name   string `json:"name"`
	Key    string `json:"key"`
	Label  string `json:"label"`
	Secret bool   `json:"secret"`
}
type Method struct {
	ID          string  `json:"id"`
	Label       string  `json:"label"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Fields      []Field `json:"fields"`
}
type Interaction struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	URL             string  `json:"url"`
	UserCode        string  `json:"userCode"`
	Output          string  `json:"output"`
	RequiresConsent bool    `json:"requiresConsent"`
	AcceptsCallback bool    `json:"acceptsCallback"`
	Fields          []Field `json:"fields"`
}
type Flow struct {
	FlowID           string       `json:"flowId"`
	Phase            string       `json:"phase"`
	Message          string       `json:"message"`
	AuthorizationURL string       `json:"authorizationUrl"`
	Interaction      *Interaction `json:"interaction"`
}

func (f *Flow) Active() bool {
	return f != nil && f.FlowID != "" && f.Phase != "succeeded" && f.Phase != "failed" && f.Phase != "cancelled" && f.Phase != "idle"
}

type Account struct {
	Authenticated    bool     `json:"authenticated"`
	AvailableMethods []Method `json:"availableMethods"`
}
type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Agent struct {
	ID                 string                     `json:"id"`
	Name               string                     `json:"name"`
	ProviderID         string                     `json:"providerId"`
	Command            string                     `json:"command"`
	Args               []string                   `json:"args"`
	Env                map[string]string          `json:"env"`
	ProfileDir         string                     `json:"profileDir"`
	PermissionPolicy   string                     `json:"permissionPolicy"`
	AuthMethod         string                     `json:"authMethod"`
	RegistrySource     string                     `json:"registrySource"`
	RegistryAgentID    string                     `json:"registryAgentId"`
	Enabled            bool                       `json:"enabled"`
	Builtin            bool                       `json:"builtin"`
	BinaryFound        bool                       `json:"binaryFound"`
	ActiveSessionCount int                        `json:"activeSessionCount"`
	Account            *Account                   `json:"account"`
	AuthFlow           *Flow                      `json:"authFlow"`
	ConfiguredModels   json.RawMessage            `json:"configuredModels"`
	FavoriteModels     []string                   `json:"favoriteModels"`
	HiddenModels       []string                   `json:"hiddenModels"`
	AllModels          []Model                    `json:"allModels"`
	Raw                map[string]json.RawMessage `json:"-"`
}

func (a *Agent) UnmarshalJSON(data []byte) error {
	type plain Agent
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*a = Agent(p)
	return json.Unmarshal(data, &a.Raw)
}
func (a Agent) Patch(fields map[string]any) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for k, v := range a.Raw {
		out[k] = append(json.RawMessage(nil), v...)
	}
	if a.ConfiguredModels != nil {
		out["models"] = append(json.RawMessage(nil), a.ConfiguredModels...)
	}
	for k, v := range fields {
		out[k], _ = json.Marshal(v)
	}
	return out
}

type Session struct {
	SessionID    string `json:"sessionId"`
	AgentID      string `json:"agentId"`
	AcpSessionID string `json:"acpSessionId"`
	Cwd          string `json:"cwd"`
	CurrentModel string `json:"currentModel"`
}
type Catalog struct {
	Agents         []Agent   `json:"agents"`
	ActiveSessions []Session `json:"activeSessions"`
}
type RegistryAgent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
type Client struct{ RPC *pluginrpc.Client }

func (c Client) Catalog(ctx context.Context) (Catalog, error) {
	var v Catalog
	err := c.RPC.Call(ctx, "dsh-acp-agents", "catalog", map[string]any{}, &v)
	return v, err
}
func (c Client) Action(ctx context.Context, method string, payload any) (Catalog, string, error) {
	var raw json.RawMessage
	if err := c.RPC.Call(ctx, "dsh-acp-agents", method, payload, &raw); err != nil {
		return Catalog{}, "", err
	}
	var wrapper struct {
		Catalog   *Catalog `json:"catalog"`
		OK        *bool    `json:"ok"`
		Sent      *bool    `json:"sent"`
		Message   string   `json:"message"`
		Error     string   `json:"error"`
		CreatedID string   `json:"createdId"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return Catalog{}, "", err
	}
	if wrapper.Sent != nil && !*wrapper.Sent {
		return Catalog{}, "", fmt.Errorf("登录步骤已过期，请刷新后重试。")
	}
	if wrapper.OK != nil && !*wrapper.OK {
		return Catalog{}, "", fmt.Errorf("ACP 握手未通过：%s %s", wrapper.Message, wrapper.Error)
	}
	if wrapper.Catalog != nil {
		return *wrapper.Catalog, wrapper.CreatedID, nil
	}
	var out Catalog
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, "", err
	}
	return out, "", nil
}
func (c Client) Search(ctx context.Context, query string) ([]RegistryAgent, error) {
	var out struct {
		Agents []RegistryAgent `json:"agents"`
	}
	err := c.RPC.Call(ctx, "dsh-acp-agents", "agent/registry-search", map[string]any{"query": query, "limit": 100}, &out)
	return out.Agents, err
}
