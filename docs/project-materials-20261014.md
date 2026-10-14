# Native Project notes and preferences
Fixed source: upstream-90465b67/vendor/dsh-project/client/client.js:908-920 and lib/service.js:869-875,1044-1047.

- Calls actual store/read and store/write with projectId/path/text.
- notes.md edits only userText; generatedText displayed read-only. Separator insertion rejected. preferences.md edits text.
- Explicit Save only; no messages, task launch, or invented RPC.
- Maximum 256000 UTF-8 bytes. Missing editable notes fields, Host errors and archived writes fail visibly.
- Drafts cached separately by Project ID and material path within current Host attachment. Back retains drafts; reload disabled while dirty; explicit discard restores saved text. Failed save retains draft.
- Archived/archiving/non-ready Project is read-only. Server remains mutation authority.
- Focused local mock fixture: TestProjectMaterialEditing passed. No real Host, project, session or model used.
- Integration: four new Go files plus integration.patch for project_flow.go and projects/view.go. apply --check passed. Windows compile delegated to integration owner.
