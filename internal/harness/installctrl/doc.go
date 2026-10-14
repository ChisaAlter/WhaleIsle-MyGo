// Package installctrl serves a loopback HTTP control plane for
// install_dsh_plugin: POST /install with Bearer auth. The harness child
// receives DSH_DESKTOP_INSTALL_URL and DSH_DESKTOP_INSTALL_TOKEN.
//
// PNPMInstall runs dsh plugin add into the web profile (pnpm under the hood),
// merges allowBuilds into pnpm-workspace.yaml, and surfaces needsAllowBuilds
// from install logs.
package installctrl
