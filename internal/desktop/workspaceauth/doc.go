// Package workspaceauth validates configured boot workspace paths before the
// harness starts or settings persist them. Rules mirror Electron
// reference/electron/src/main/workspace-authority.js (volume roots and
// high-risk trust roots) plus config load expectations (existing directory).
package workspaceauth
