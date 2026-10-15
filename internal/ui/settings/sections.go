package settings

import (
	"sort"
	"strings"
)

// SectionID is one of the fifteen harness settings areas from docs/plan.md §3.2,
// plus SectionOther for namespaces that do not map to a known section.
type SectionID string

const (
	SectionAccount         SectionID = "account"
	SectionGeneral         SectionID = "general"
	SectionAppearance      SectionID = "appearance"
	SectionInterface       SectionID = "interface"
	SectionModels          SectionID = "models"
	SectionPlugins         SectionID = "plugins"
	SectionSkills          SectionID = "skills"
	SectionMarket          SectionID = "market"
	SectionRemote          SectionID = "remote"
	SectionMCP             SectionID = "mcp"
	SectionAgentPresets    SectionID = "agent-presets"
	SectionPet             SectionID = "pet"
	SectionAbout           SectionID = "about"
	SectionUsageStats      SectionID = "usage-stats"
	SectionRemoteWorkspace SectionID = "remote-workspace"
	SectionAgentLoop       SectionID = "agent-loop"
	SectionWebSearch       SectionID = "web-search"
	SectionSubagent        SectionID = "subagent"
	SectionShell           SectionID = "shell"
	SectionOther           SectionID = "other"
)

// planSectionOrder is the left-nav order for harness namespaces (desktop is separate).
var planSectionOrder = []SectionID{
	SectionAccount,
	SectionGeneral,
	SectionAppearance,
	SectionInterface,
	SectionModels,
	SectionPlugins,
	SectionSkills,
	SectionMarket,
	SectionRemote,
	SectionMCP,
	SectionAgentPresets,
	SectionPet,
	SectionAbout,
	SectionUsageStats,
	SectionRemoteWorkspace,
	SectionAgentLoop,
	SectionWebSearch,
	SectionSubagent,
	SectionShell,
	SectionOther,
}

// namespaceToSection maps settings/describe namespace ids to a plan section.
// Unknown namespaces fall through to SectionOther.
var namespaceToSection = map[string]SectionID{
	"agent-loop":                        SectionAgentLoop,
	"web-search-deepseek":               SectionWebSearch,
	"session-log-deepseek":              SectionGeneral,
	"subagent":                          SectionSubagent,
	"subagent-model-selection-settings": SectionSubagent,
	"bash-sandbox":                      SectionShell,
	"pwsh-sandbox":                      SectionShell,
	"bash-local":                        SectionShell,
	"pwsh-local":                        SectionShell,
	"llm-deepseek":                      SectionModels,
	"llm-deepseek-api-key":              SectionModels,
	"llm-deepseek-account":              SectionModels,
	"llm-pi-ai":                         SectionModels,
	"llm-vision-fallback":               SectionModels,
	"account":                           SectionAccount,
	"general":                           SectionGeneral,
	"appearance":                        SectionAppearance,
	"theme":                             SectionAppearance,
	"interface":                         SectionInterface,
	"ui":                                SectionInterface,
	"models":                            SectionModels,
	"model":                             SectionModels,
	"llm":                               SectionModels,
	"plugins":                           SectionPlugins,
	"plugin":                            SectionPlugins,
	"skills":                            SectionSkills,
	"skill":                             SectionSkills,
	"market":                            SectionMarket,
	"ui-settings-market":                SectionMarket,
	"remote":                            SectionRemote,
	"mcp":                               SectionMCP,
	"agent-presets":                     SectionAgentPresets,
	"agentPresets":                      SectionAgentPresets,
	"agentPreset":                       SectionAgentPresets,
	"pet":                               SectionPet,
	"live2d":                            SectionPet,
	"about":                             SectionAbout,
	"usage-stats":                       SectionUsageStats,
	"usageStats":                        SectionUsageStats,
	"usage":                             SectionUsageStats,
	"remote-workspace":                  SectionRemoteWorkspace,
	"remoteWorkspace":                   SectionRemoteWorkspace,
}

// SectionForNamespace returns the plan section for a harness settings namespace.
func SectionForNamespace(ns string) SectionID {
	ns = strings.TrimSpace(ns)
	if ns == "" {
		return SectionOther
	}
	if id, ok := namespaceToSection[ns]; ok {
		return id
	}
	lower := strings.ToLower(ns)
	if id, ok := namespaceToSection[lower]; ok {
		return id
	}
	compact := strings.ReplaceAll(lower, "-", "")
	if id, ok := namespaceToSection[compact]; ok {
		return id
	}
	return SectionOther
}

// NavGroup is one section header and its namespace buttons in the left nav.
type NavGroup struct {
	Section SectionID
	Items   []string // namespace ids, sorted
}

// GroupNamespaces buckets namespace names into plan sections in display order.
// Empty sections are omitted except SectionOther when it has items.
func GroupNamespaces(namespaces []string) []NavGroup {
	buckets := map[SectionID][]string{}
	for _, ns := range namespaces {
		ns = strings.TrimSpace(ns)
		if ns == "" {
			continue
		}
		sec := SectionForNamespace(ns)
		buckets[sec] = append(buckets[sec], ns)
	}
	var out []NavGroup
	for _, sec := range planSectionOrder {
		items := buckets[sec]
		if len(items) == 0 {
			continue
		}
		sort.Strings(items)
		out = append(out, NavGroup{Section: sec, Items: items})
	}
	return out
}
