// Package importer implements Electron-parity official-home import for the
// launcher「导入」tab: read-only scan (BuildCatalog) plus selected-item ApplyCatalog.
//
// ImportReport is a shallow probe of Electron userData artifacts.
// BuildCatalog mirrors scanImport against ~/.dsh (and ~/.agents/skills).
// ApplyCatalog copies selected sessions/skills/mcp/settings/presets/attachments
// into the desktop dsh-home. Settings use raw whitelist blocks; MCP uses the
// official two-space servers sequence keyed by id. Unknown document shapes and
// duplicate keys fail closed. Unselected target blocks/records are preserved.
// Shared YAML and AGENTS.md are published with a private temporary file and
// rename, never truncating an existing destination on failure. This is per-file
// publication, not a multi-file transaction or protection against external path
// replacement races. Referenced credentials/OAuth state are not migrated;
// cross-section YAML anchors and alternate MCP directory layouts are unsupported.
// Supported GitHub/npm plugin specs are reinstalled by an injected marketplace
// installer; explicit build approval and cancellation belong to the caller.
package importer
