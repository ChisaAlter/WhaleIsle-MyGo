// Package pad owns the native projection of the shipped dsh-pad Host service.
// Panel definitions and scheduled work remain owned by that service.
package pad

import "encoding/json"

type ModelSelection struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoningEffort,omitempty"`
}
type Source struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Command     string            `json:"command,omitempty"`
	URL         string            `json:"url,omitempty"`
	Method      string            `json:"method,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	Parse       string            `json:"parse"`
	IntervalSec int               `json:"intervalSec"`
	TimeoutMS   int               `json:"timeoutMs,omitempty"`
}
type AgentDefinition struct {
	Instruction  string `json:"instruction"`
	EveryMinutes int    `json:"everyMinutes"`
}
type Definition struct {
	Title   string           `json:"title"`
	Icon    string           `json:"icon,omitempty"`
	HTML    string           `json:"html"`
	CSS     string           `json:"css"`
	Script  string           `json:"script"`
	Height  json.RawMessage  `json:"height"`
	Sources []Source         `json:"sources"`
	Model   *ModelSelection  `json:"model,omitempty"`
	Agent   *AgentDefinition `json:"agent,omitempty"`
}
type Reading struct {
	OK         bool            `json:"ok"`
	Value      json.RawMessage `json:"value,omitempty"`
	Error      string          `json:"error,omitempty"`
	At         string          `json:"at"`
	DurationMS int             `json:"durationMs,omitempty"`
}
type Point struct {
	Value json.RawMessage `json:"value"`
	At    string          `json:"at"`
}
type AgentData struct {
	Value json.RawMessage `json:"value"`
	At    string          `json:"at"`
}
type Data struct {
	Sources map[string]Reading `json:"sources"`
	Agent   *AgentData         `json:"agent"`
}
type AgentRun struct {
	Status    string `json:"status"`
	LastRunAt string `json:"lastRunAt,omitempty"`
	NextRunAt string `json:"nextRunAt,omitempty"`
	Error     string `json:"error,omitempty"`
}
type Panel struct {
	ID            string             `json:"id"`
	Version       int                `json:"version"`
	Position      int                `json:"position"`
	Collapsed     bool               `json:"collapsed"`
	Span          int                `json:"span"`
	UpdatedAt     string             `json:"updatedAt"`
	Definition    Definition         `json:"definition"`
	CanUndo       bool               `json:"canUndo"`
	NeedsApproval bool               `json:"needsApproval"`
	Fingerprint   string             `json:"fingerprint"`
	AgentRun      *AgentRun          `json:"agentRun"`
	Data          Data               `json:"data"`
	History       map[string][]Point `json:"history,omitempty"`
}

func (p Panel) AwaitingData() bool {
	for _, s := range p.Definition.Sources {
		if _, ok := p.Data.Sources[s.ID]; !ok {
			return true
		}
	}
	return p.Definition.Agent != nil && p.Data.Agent == nil
}

type CatalogModel struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Description            string   `json:"description,omitempty"`
	ReasoningEfforts       []Effort `json:"reasoningEfforts,omitempty"`
	DefaultReasoningEffort string   `json:"defaultReasoningEffort,omitempty"`
}
type Effort struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Provider struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Models []CatalogModel `json:"models"`
}
type Catalog struct {
	Default   *ModelSelection `json:"default"`
	Providers []Provider      `json:"providers"`
}
type Action struct {
	Generation  uint64          `json:"-"`
	Op          string          `json:"op"`
	PanelID     string          `json:"panelId"`
	Fingerprint string          `json:"fingerprint,omitempty"`
	Collapsed   bool            `json:"collapsed"`
	Model       *ModelSelection `json:"model"`
}
type RunResult struct {
	OK    bool   `json:"ok"`
	Text  string `json:"text,omitempty"`
	Error string `json:"error,omitempty"`
}
type ModelAction struct {
	Request   uint64
	Busy      bool
	Succeeded bool
	Error     string
}
type Snapshot struct {
	Attached, Connected, Loaded bool
	Disabled                    bool
	Generation                  uint64
	Panels                      []Panel
	Layout                      Layout
	Catalog                     *Catalog
	ModelsLoading               bool
	ModelsError                 string
	ModelActions                map[string]ModelAction
	Error                       string
	Notice                      string
	PanelErrors                 map[string]string
	ActionErrors                map[string]string
}

func clone[T any](v T) T {
	raw, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(raw, &out)
	return out
}
