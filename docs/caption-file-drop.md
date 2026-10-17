# CP08 main caption file drop

The ready main chat's native caption now shares the existing CE/CP06 attachment
intake and navigation rail owner gates. Its parent-session target is disabled
when a child conversation replaces the main view, or an exclusive page/settings
is active. Existing native UI modal hit testing and innermost file-target routing
remain in force. Only actual OS file payloads enter this path; directory rejection
and local-path versus upload handling remain in the composer intake.

This adds a target to the caption's own ancestry, not the whole window. Resource
docks, child asides and WebViews remain outside it. Internal Dock drag payloads
are not attachments. Source inspection establishes that MyGo ui/drop.go selects
one innermost target and content.go calls OnFileDrop only after native content
rejects the event.

A global fallback is intentionally not installed. FileDropEvent has Paths/X/Y
but no source surface or owner. Window.filesDropped also uses the same event for
DOM drops. An unconsumed resource or child blank area cannot safely be inferred
to belong to the parent's draft. Thus arbitrary unowned areas still differ from
the original document-wide listener. A future full solution needs authoritative
surface/region ownership, not guessed coordinates or a universal fallback.

Focused checks: synthetic caption rendering/internal-Dock rejection plus existing
navigation main-child owner and stale/exclusive draft gates pass in frame and main
packages. No real Finder/Explorer drop or macOS/Windows caption acceptance was run.
