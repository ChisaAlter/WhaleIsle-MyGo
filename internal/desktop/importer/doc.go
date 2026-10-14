// Package importer implements Electron-parity official-home import for the
// launcher「导入」tab: read-only scan (BuildCatalog) plus copy-only ApplyCatalog.
//
// ImportReport is a shallow probe of Electron userData artifacts.
// BuildCatalog mirrors scanImport against ~/.dsh (and ~/.agents/skills).
// ApplyCatalog copies selected sessions/skills/mcp/settings/presets/attachments
// into the desktop dsh-home; plugin reinstall is deferred (needs installPlugin).
package importer
