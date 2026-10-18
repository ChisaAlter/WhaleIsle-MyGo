# Persistent navigation for main panels

The shared workspace/sidebar shell now stays present while the native market,
schedule or assistant replaces the center conversation. These are direct panel
switches: users can choose another panel or an existing session without first
pressing Back. Selecting the current session also dismisses the panel. New
session and folder actions return to the conversation surface; no prompt is sent.

This follows the supplied Electron source:
- `ui-layout/src/client/AppFrame.tsx`: `MainContent` changes only the main slot;
  sidebar and main are independently mounted siblings (51–58, 389–395, 452–466).
- `ui-workspace/src/client/navigation.ts`: opening a session reveals the main
  conversation and clears selected panel (226–227, 559).
- `ui-sidebar/src/client/SidebarRoot.tsx`: settings intentionally uses a fixed
  full-viewport descendant overlay (158–161). Settings retains its fullscreen
  behavior here; PDF/browser/plugin guests, usage, IM and remote surfaces retain
  their previous independent layout and security boundaries.

`frame.NavigationShell` reuses the existing full sidebar, including workspace,
Project and session actions. Chat uses the same shell without a second sidebar.
The three panels keep their existing stores/controllers rather than recreating
state on navigation. Session draft objects, market filters/details/install state,
schedule editor drafts and assistant draft/history are not reset. Shared sidebar
collapse/resize/search/rename state is written back even when a panel returns
early; chat/right/terminal geometry still uses the existing column solver.

Project file references from the persistent rail route through the existing
chat file callback. A one-shot request is bound to its initiating session store,
selection and Host generation/origin/session cookie; late references are dropped.
The existing callback continues to own file confinement, dirty editor tabs, PDF
preview and Host fallback. This adds no file protocol or guest permissions.

Verification: one temporary synthetic fixture exercised market → schedule →
assistant → another session, selecting the already selected session, return to
market with filter/tab retained, both draft identities/text retained, rail
collapse/restore, side-by-side geometry, fullscreen bypass and stale Project
file handoff. Only a temporary loopback mock and headless UI inputs were used;
no screenshots, real Host, model, user files or account operations. Windows
compilation belongs to the integration owner. Native Windows interaction and
visual acceptance remain unverified.
