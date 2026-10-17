// Package installctrl serves a loopback HTTP control plane for
// install_dsh_plugin: POST /install and the authenticated /desktop/* routes
// used by dsh-whale's desktop management tools. The harness child receives
// DSH_DESKTOP_INSTALL_URL and DSH_DESKTOP_INSTALL_TOKEN.
//
// PNPMInstall runs dsh plugin add into the web profile (pnpm under the hood),
// merges allowBuilds into pnpm-workspace.yaml, and surfaces needsAllowBuilds
// from install logs. DesktopOps keeps main-process behavior injectable so the
// control layer can report an explicit not-configured boundary when a native
// platform implementation is unavailable.
package installctrl
