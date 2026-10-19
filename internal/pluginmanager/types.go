// Package pluginmanager owns the profile Plugins manager, independently of
// the desktop curated marketplace. Wire types follow the e353e8e13f Host
// packages/boot/{plugin-manager,plugin-market}/src/types.ts contracts.
package pluginmanager

import (
	"encoding/json"
	"sort"
	"strings"
)

type Text json.RawMessage

// LocalizedText is a JSON string or dictionary, not a base64 byte slice.
func (t Text) MarshalJSON() ([]byte, error)    { return json.RawMessage(t).MarshalJSON() }
func (t *Text) UnmarshalJSON(raw []byte) error { *t = append((*t)[:0], raw...); return nil }

func (t Text) Resolve(locale string) string {
	var literal string
	if json.Unmarshal(t, &literal) == nil {
		return literal
	}
	var dictionary map[string]string
	if json.Unmarshal(t, &dictionary) != nil {
		return ""
	}
	for _, key := range []string{locale, strings.Split(locale, "-")[0], "en", "zh"} {
		if dictionary[key] != "" {
			return dictionary[key]
		}
	}
	keys := make([]string, 0, len(dictionary))
	for key := range dictionary {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		return dictionary[key]
	}
	return ""
}

type Meta struct {
	Title       Text   `json:"title,omitempty"`
	Description Text   `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Error       string `json:"error,omitempty"`
}
type Incompatible struct {
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	RuntimeVersion string            `json:"runtimeVersion"`
	Requirements   map[string]string `json:"requirements"`
	// The bundled older Host names this field peers; preserve its diagnostics.
	Peers map[string]string `json:"peers,omitempty"`
}
type ManagementError struct {
	Code         string         `json:"code"`
	Diagnostic   string         `json:"diagnostic,omitempty"`
	Incompatible []Incompatible `json:"incompatible,omitempty"`
}
type InstallTarget struct {
	Spec    string `json:"spec"`
	Version string `json:"version"`
}
type BundleRow struct {
	RowID      string `json:"rowId"`
	ModuleName string `json:"moduleName"`
	Meta       *Meta  `json:"meta,omitempty"`
	EntryID    string `json:"entryId,omitempty"`
}
type Bundle struct {
	Name           string           `json:"name"`
	Official       bool             `json:"official"`
	Availability   string           `json:"availability"`
	InstallTarget  *InstallTarget   `json:"installTarget,omitempty"`
	Version        string           `json:"version,omitempty"`
	Meta           *Meta            `json:"meta,omitempty"`
	Description    string           `json:"description,omitempty"`
	Enabled        bool             `json:"enabled"`
	Installed      bool             `json:"installed"`
	Source         string           `json:"source,omitempty"`
	Optional       bool             `json:"optional"`
	Removable      bool             `json:"removable"`
	ReadOnlyReason string           `json:"readOnlyReason,omitempty"`
	Error          *ManagementError `json:"error,omitempty"`
	Rows           []BundleRow      `json:"rows"`
	Overrides      []string         `json:"overrides"`
}
type Plugin struct {
	EntryID        string `json:"entryId"`
	ModuleName     string `json:"moduleName"`
	Meta           *Meta  `json:"meta,omitempty"`
	Enabled        bool   `json:"enabled"`
	Phase          string `json:"fiberPhase"`
	PatchID        string `json:"patchId,omitempty"`
	ReadOnlyReason string `json:"readOnlyReason,omitempty"`
}
type Row struct {
	BundleRow
	Enabled               bool
	Phase, ReadOnlyReason string
}
type Package struct {
	Bundle
	Parts []Row
}

func (p Package) Title(locale string) string {
	if p.Meta != nil {
		if title := p.Meta.Title.Resolve(locale); title != "" {
			return title
		}
	}
	return ShortName(p.Name)
}
func (p Package) Summary(locale string) string {
	if p.Meta != nil {
		if description := p.Meta.Description.Resolve(locale); description != "" {
			return description
		}
	}
	return p.Description
}
func ShortName(name string) string {
	if strings.HasPrefix(name, "@") {
		if i := strings.IndexByte(name, '/'); i >= 0 {
			return name[i+1:]
		}
	}
	return name
}
func (r Row) Title(locale string) string {
	if r.Meta != nil {
		if title := r.Meta.Title.Resolve(locale); title != "" {
			return title
		}
	}
	if r.RowID != "" {
		return r.RowID
	}
	return r.ModuleName
}

type RegistryInfo struct {
	Registry           *string  `json:"registry"`
	FallbackRegistries []string `json:"fallbackRegistries"`
	Resolved           *string  `json:"resolved"`
}
type Inspection struct {
	Status      string    `json:"status"`
	Kind        string    `json:"kind,omitempty"`
	Name        string    `json:"name,omitempty"`
	Version     string    `json:"version,omitempty"`
	Description string    `json:"description,omitempty"`
	Bundle      *bool     `json:"bundle,omitempty"`
	Registry    *string   `json:"registry"`
	Host        string    `json:"host,omitempty"`
	Problem     string    `json:"problem,omitempty"`
	Reason      string    `json:"reason,omitempty"`
	Registries  []*string `json:"registries,omitempty"`
}
type PackageResult struct {
	ExitCode  *int   `json:"exitCode"`
	Output    string `json:"output"`
	Truncated bool   `json:"truncated"`
	LogPath   string `json:"logPath"`
	Kind      string `json:"kind,omitempty"`
	TimedOut  bool   `json:"timedOut,omitempty"`
}
type Change struct {
	Changed        bool             `json:"changed"`
	Application    string           `json:"application"`
	RestartReason  string           `json:"restartReason,omitempty"`
	Stage          string           `json:"stage"`
	Target         string           `json:"target"`
	Enabled        *bool            `json:"enabled,omitempty"`
	Error          *ManagementError `json:"error,omitempty"`
	Warnings       []string         `json:"warnings,omitempty"`
	PackageResult  *PackageResult   `json:"packageResult,omitempty"`
	Bundle         string           `json:"bundle,omitempty"`
	Version        string           `json:"version,omitempty"`
	PendingBuilds  []string         `json:"pendingBuilds,omitempty"`
	ApprovedBuilds []string         `json:"approvedBuilds,omitempty"`
	Registries     []*string        `json:"registries,omitempty"`
	FailedAt       string           `json:"failedAt,omitempty"`
}
type InstallOptions struct {
	Enabled        *bool    `json:"enabled,omitempty"`
	SaveExact      bool     `json:"saveExact,omitempty"`
	RequestID      string   `json:"requestId,omitempty"`
	ApprovedBuilds []string `json:"approvedBuilds,omitempty"`
	Registry       *string  `json:"registry,omitempty"`
}
type InstallLog struct {
	RequestID   string   `json:"requestId,omitempty"`
	JobID       string   `json:"jobId"`
	Argv        []string `json:"argv"`
	CWD         string   `json:"cwd"`
	Stream      string   `json:"stream"`
	Text        string   `json:"text"`
	ExitCode    *int     `json:"exitCode,omitempty"`
	ExitCodeSet bool     `json:"-"`
}

func (v *InstallLog) UnmarshalJSON(data []byte) error {
	type wire InstallLog
	var value wire
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*v = InstallLog(value)
	_, v.ExitCodeSet = fields["exitCode"]
	return nil
}

type InstallAttempt struct {
	Registry *string `json:"registry"`
	Index    int     `json:"index"`
	Total    int     `json:"total"`
}
type InstallProgress struct {
	RequestID string          `json:"requestId"`
	Phase     string          `json:"phase"`
	Attempt   *InstallAttempt `json:"attempt,omitempty"`
}
type InstallOutput struct{ Stream, Text string }
type InstallRun struct {
	JobID, Command, CWD, Output string
	Argv                        []string
	Chunks                      []InstallOutput
	ExitCode                    *int
	Settled, Truncated          bool
}

type Index struct {
	URL         string `json:"url"`
	Title       Text   `json:"title,omitempty"`
	Description Text   `json:"description,omitempty"`
	Error       string `json:"error,omitempty"`
}
type NamedMetadata struct {
	Title Text   `json:"title"`
	Order string `json:"order,omitempty"`
}
type MarketMetadata struct {
	Categories map[string]NamedMetadata `json:"categories"`
	Labels     map[string]NamedMetadata `json:"labels"`
	SortKeys   map[string]NamedMetadata `json:"sortKeys"`
}
type MarketSort struct {
	Kind      string `json:"kind"`
	ID        string `json:"id,omitempty"`
	Direction string `json:"direction,omitempty"`
}
type Query struct {
	Text      string     `json:"text,omitempty"`
	Category  string     `json:"category,omitempty"`
	Labels    []string   `json:"labels,omitempty"`
	Sort      MarketSort `json:"sort"`
	Offset    int        `json:"offset"`
	Limit     int        `json:"limit"`
	Refresh   bool       `json:"refresh,omitempty"`
	FetchedAt int64      `json:"fetchedAt,omitempty"`
}
type Compatibility struct {
	Compatible     bool              `json:"compatible"`
	RuntimeVersion string            `json:"runtimeVersion,omitempty"`
	Requirements   map[string]string `json:"requirements,omitempty"`
	Exempted       bool              `json:"exempted,omitempty"`
}
type MarketVersion struct {
	Version          string `json:"version"`
	PublishTimestamp int64  `json:"publishTimestamp"`
	Install          struct {
		Spec     string  `json:"spec"`
		Registry *string `json:"registry,omitempty"`
	} `json:"install"`
	Compatibility Compatibility `json:"compatibility"`
	Source        struct {
		Type     string `json:"type"`
		Registry string `json:"registry,omitempty"`
		URL      string `json:"url,omitempty"`
		Commit   string `json:"commit,omitempty"`
		Path     string `json:"path,omitempty"`
	} `json:"source"`
	Engines map[string]string `json:"engines"`
}
type MarketEntry struct {
	Name        string `json:"name"`
	Title       Text   `json:"title,omitempty"`
	Description Text   `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Author      string `json:"author,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
	Repository  *struct {
		URL       string `json:"url"`
		Directory string `json:"directory,omitempty"`
	} `json:"repository,omitempty"`
	Metadata struct {
		Category string             `json:"category,omitempty"`
		Keywords []string           `json:"keywords"`
		Labels   []string           `json:"labels"`
		SortKeys map[string]float64 `json:"sortKeys"`
	} `json:"metadata"`
	Versions       []MarketVersion `json:"versions"`
	DefaultVersion string          `json:"defaultVersion,omitempty"`
	Installation   struct {
		Status           string `json:"status"`
		InstalledVersion string `json:"installedVersion,omitempty"`
	} `json:"installation"`
}

func (e MarketEntry) Label(locale string) string {
	if title := e.Title.Resolve(locale); title != "" {
		return title
	}
	return ShortName(e.Name)
}

type SearchResult struct {
	URL         string         `json:"url"`
	Title       Text           `json:"title"`
	Description Text           `json:"description,omitempty"`
	Metadata    MarketMetadata `json:"metadata"`
	FetchedAt   int64          `json:"fetchedAt"`
	Entries     []MarketEntry  `json:"entries"`
	Total       int            `json:"total"`
	NextOffset  *int           `json:"nextOffset"`
}

// Subject mirrors plugins.item/bundle.config/row.config and detail slot owners.
type Subject struct {
	Kind, Name, RowID, ItemID string
	FromMore                  bool
}
type Confirmation struct{ Kind, Target, Description string }
type InstallState struct {
	Open                                                 bool
	Spec, Registry, Phase, Error, RequestID, Log         string
	RegistryInfo                                         *RegistryInfo
	Inspection                                           *Inspection
	Result                                               *Change
	SaveExact                                            bool
	Enabled                                              *bool
	ApprovedBuilds                                       []string
	Acknowledged, CancellationRequested, WaitingForStart bool
	Recovering                                           bool
	Runs                                                 []InstallRun
	Attempts                                             []InstallAttempt
}
type DiscoverState struct {
	Status, Error, Selected, SelectedEntry, Draft string
	Indexes                                       []Index
	Query                                         Query
	Listing                                       *SearchResult
	Loading, MoreLoading, SourcesOpen             bool
}
type State struct {
	Owner                      uint64
	Connected, Loading, Busy   bool
	Status, Error, Notice, Tab string
	Packages                   []Package
	View                       Subject
	Confirmation               *Confirmation
	Install                    InstallState
	Discover                   DiscoverState
}

var extensionOrder = []string{"@deepseek-ai/dsh-experimental-agent-team-profile", "@deepseek-ai/dsh-experimental-voice-input-bundle", "@deepseek-ai/dsh-experimental-auto-review", "@deepseek-ai/dsh-experimental-inspector-profile"}
var MoreBundles = []string{"@deepseek-ai/dsh-experimental-tool-worktree", "@deepseek-ai/dsh-experimental-cot-translation-bundle", "@deepseek-ai/dsh-experimental-terminal-bundle", "@deepseek-ai/dsh-experimental-session-search", "@deepseek-ai/dsh-experimental-session-titles-bundle", "@deepseek-ai/dsh-experimental-badge-skill-bundle", "@deepseek-ai/dsh-experimental-ralph-bundle", "@deepseek-ai/dsh-experimental-plugin-market-bundle", "@deepseek-ai/dsh-subagent-codex", "@deepseek-ai/dsh-subagent-claude-code"}

func Listed(p Package) bool {
	switch p.Name {
	case "@deepseek-ai/dsh-base", "@deepseek-ai/dsh-web-app", "@deepseek-ai/dsh-headless", "@deepseek-ai/dsh-sdk-app", "@deepseek-ai/dsh-acp-app", "@deepseek-ai/dsh-sdk-minimal":
		return false
	}
	return p.Installed || p.Official || p.Optional || p.Error != nil
}

func UpdateAvailable(p Package) bool {
	target := p.InstallTarget
	if !p.Installed || !p.Removable || p.Availability == "installation" || target == nil {
		return false
	}
	return p.Version != target.Version || ((strings.HasPrefix(target.Spec, "link:") || strings.HasPrefix(p.Source, "link:")) && p.Source != target.Spec)
}
func Pinned(p Package) bool {
	for _, name := range MoreBundles {
		if p.Name == name {
			return p.Enabled
		}
	}
	return true
}
func OfficialRank(name string) int {
	for i, n := range extensionOrder {
		if n == name {
			return i
		}
	}
	for i, n := range MoreBundles {
		if n == name {
			return len(extensionOrder) + 1 + i
		}
	}
	return len(extensionOrder)
}
