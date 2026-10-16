# Standalone native launcher

`cmd/whaleisle-launcher` is a separate Windows GUI executable managing the
registered native desktop. Build using `scripts/package-launcher.ps1`. It does
not bundle or start a Host in the launcher's process, and installs no registry
entries for itself. Distribute its own `dist/whaleisle-launcher` directory.

The launcher's home surface starts/stops the installed desktop. Its Versions
surface uses the existing pinned native release catalog, package selection,
staging, checksums and installer cache. Missing installations require packages
including the runtime. Installer launch uses the same final hash/size check as
the full desktop, waits for completion and adopts the registered target version.
The launcher remains alive. Cancellation/failed installation never reports success.

Discovery reads the current-user native uninstall registration and requires
`NativeRuntime=WhaleIsle-MyGo`, product name, install location, uninstall command
shape, version and regular non-symlink executable. A legacy Electron registration
is never treated as native. Process discovery compares full executable paths.

Stop launches the verified installed executable with a bounded stop command
and its own executable identity. The matching desktop's single-instance handler
routes it through existing saved-session, import exclusion and task-protected quit.
It does not force-kill processes. Older desktop versions lacking the command can
be closed manually; timeout is an error, not a successful stop. The command cannot
cold-start a stopped desktop. Different installations sharing an app singleton
must refuse commands bearing a different executable identity.

Installation requires the desktop stopped first. Lifecycle operations share the
importer's cross-process maintenance lock and refuse unresolved import journals.
The installer runs with its normal visible confirmation flow; the launcher never
passes silent flags. Once an installer starts, closing is blocked until it exits
so the maintenance lock is not released mid-install. No launcher process or
installer has been executed as part of implementing this source.

Settings, components, plugin management and import belong to the installed
desktop; the slim UI directs users there. No second copy of those services or
authentication state is created.

Deferred verification: registry identity rejection, process-image matching,
protected-stop refusal/success, no-install cold stop, missing desktop installation,
update/downgrade/reinstall completion and registration adoption, Windows GUI and
release package integration. Injected controller fixtures are provided separately.
