# Native file-list paste

The shipped DraftClient paste command first reads MyGo Clipboard.ReadFiles, which
requests typed FileList/URIList representations. Windows exposes CF_HDROP via its
native clipboard/OLE mapping; macOS exposes NSPasteboard public.file-url and native
file-list aliases. No plain clipboard text is interpreted as a filesystem path.
No clipboard polling or real clipboard access was used for development tests.

Native file items go through the existing OnAttachFiles callback and CP06 local
path/reference versus upload classification. They do not also insert a duplicate
image or clipboard text representation. Plain text and image-only pastes retain
their existing path. Clipboard read errors consume the command with an error.

Both file paste and OS file drop validate the complete batch before attachment:
absolute paths, at most 128 entries, existing regular files only, deduplication.
Directories, symlinks resolving to directories and special files are rejected.
Explicit folder-picker and workspace-reference actions remain separate and
unchanged. Sending/preparing draft ownership gates remain in force. Attachment
upload content/race checks still belong to the existing attachment intake.

Synthetic tests create only temporary fixture files and typed transfer.Data.
They cover PDF-named file intake, text-only paths not being guessed, clipboard
read failure, mixed directory/file rejection and a sending draft. Platform
cross-compilation does not establish native Finder/Explorer clipboard behavior.
The legacy TextArea fallback is not extended; main uses the native DraftClient.
