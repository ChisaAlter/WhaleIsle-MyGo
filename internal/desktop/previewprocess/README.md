# Experimental AppContainer document helper

Status: source implementation with an explicit per-document product entry.
Static preview remains the default; no persistent preference or startup path
enables scripts. No Windows build or runtime claim is made by this change.
The creation/cleanup APIs below were not executed during development.

## Scope

The adversary supplies HTML/JavaScript. The parent, dedicated helper, installed
WebView2 binaries and Windows kernel are trusted. Native arbitrary-code
execution, compromised host binaries, administrator changes to firewall policy,
and WebView2 sandbox escape vulnerabilities are outside this feature's model.
This is not a general sandbox for hostile native executables.

The documented AppContainer/system-executable firewall exception is not, by
itself, a reachable HTML attack: a page would first need a native process-launch
primitive. This helper exposes none. The existing native policy rejects
navigation/external protocols, new windows, downloads, permissions and bindings;
the child iframe grants allow-scripts and allow-forms for local JavaScript
submit handlers. form-action none still prohibits form transmission; there is
no same-origin, popup, modal or top-navigation grant. No direct permitted webpage-to-system-EXE path was
identified in the bounded source review.

## Implemented path

1. Trusted code must explicitly call OpenExperimental with Options.Enabled=true
   and the path of the separately built trusted whaleisle-preview.exe.
   The zero value returns ErrDisabled before platform setup.
2. Query BFE/MpsSvc with read-only, non-administrator service rights. Create a
   fresh random per-session AppContainer profile with zero capabilities. Do not
   reuse profiles, grant network capabilities, install loopback exemptions,
   change firewall rules, start services or change global security descriptors.
3. Stage the helper and optional trusted sibling WebView2Loader.dll in the
   profile's own inherited-access directory. Provide an allowlisted environment;
   no account tokens, proxy settings, debugger pipes, MYGO_GENERATE or
   WEBVIEW2_* overrides are inherited.
4. Create a 32-process, 1 GiB, kill-on-close job with no breakaway flags. Bind it
   atomically using PROC_THREAD_ATTRIBUTE_JOB_LIST together with an empty
   SECURITY_CAPABILITIES capability list. Inherit only the three IPC handles.
   Create suspended; query the actual helper token and job before resuming.
5. The helper independently checks its token and job before initializing MyGo.
   It creates an isolated WebView2 and loads only about:blank. The original
   all-source resource filter, no-bridge policy, private UDF and deny handlers
   remain active.
6. The helper sends its native browser PID over inherited IPC. The parent
   enumerates the actual Windows job, including processes omitted from
   WebView2's GetProcessInfos (such as crashpad), and requires every current
   member to have the expected SID and zero capabilities. It verifies the
   helper and browser are members. Failure closes the job. Only after this
   gate does the parent send the bounded, already-authorized HTML snapshot.
7. The helper uses htmlpreview's existing opaque sandbox shell, with a separate
   exported markup constructor that grants no readiness. It loads exactly one
   snapshot. A second browser/tree check follows the loaded handshake.
8. Parent cancellation, helper exit, a token/tree check failure, startup timeout
   or the thirty-minute lease closes the job. A 500 ms observation detects
   unexpected drift; it is not the security boundary. OS token inheritance and
   the non-breakaway job establish containment before document delivery.
   Cleanup waits for the job to become empty before deleting its profile. If
   exit cannot be observed, the profile is retained and Close reports an error.

No token/job validation result updates global capture Readiness or Probe.
LocalHTMLURL, DocumentCSP and the ordinary static iframe remain unchanged.
The helper cannot open arbitrary files/URLs or request application services.
The IPC accepts one document, not a general command protocol.

## Build and integration

Build the separate helper for Windows amd64 or arm64. For example:
go build -ldflags="-H windowsgui" -o whaleisle-preview.exe ./cmd/whaleisle-preview

Package it only with trusted application binaries. The optional SDK loader must
be beside that helper. This implementation otherwise uses the installed
WebView2 runtime; fixed-version runtime packaging is not added here.

The HTML tab's "运行脚本（隔离实验）" button and a published visual reply's
"打开交互页" button are explicit per-document product callers. It keeps the static Host open while attempting a separate helper. Missing
helper, expired ownership or any containment/initialization error preserves the
static snapshot and reports the reason. There is an explicit stop button.

InteractiveSnapshot reuses the HTML bytes from the original saved-file read.
It checks the original directory and file identities, freezes declared JS/CSS/
image/font files under os.Root, then rechecks authorization and source identity.
Relative classic script files become bounded data URLs; inline scripts and
event handlers are retained only in this separate preparation mode. The
default prepareResources path continues to remove scripts and event handlers.
Runtime import/fetch dependencies, network, nested documents and evaluation
features forbidden by the existing CSP are not silently enabled or proxied.

The per-tab operation owns the Session. Parent cancellation, hide/tab switch,
tab close, document expiry and owner/session/cwd/Host revocation stop it.
Document ownership is checked even while the main window is not painting.
The helper receives bytes only; it never receives a workspace path or a URL.

scripts/package.ps1 builds the separate Windows helper alongside the main EXE.
NSIS requires that exact companion and removes it on uninstall; neither build
nor install runs it. The offline-candidate path supports explicit previewHelper
plus previewHelperSHA256 inputs. Omitting it keeps old candidates static-only,
rather than discovering or trusting another executable automatically.

Published visual replies use only session/readVisualReply after verifying the
exact durable native/PTC publication. Their attachment bytes are already prepared
by the Host (including data-URL images); this path does not resolve filesystem,
relative, file: or remote resource URLs. It supplies only those bytes, dimensions
and initial appearance to the helper. Close, another read, stop, session or Host
replacement revoke the owning read and cancel the helper. The window supports
controls, local form handlers and scrolling beyond the static capture limit.
It follows the opening appearance; live app-theme synchronization and an inline
embedded interactive chat surface are not implemented by this one-document IPC.

Remaining engineering work:
- Compile both Windows architectures and run the synthetic protocol tests.
- Confirm supported WebView2 builds preserve the strict same-SID/zero-capability
  condition for their complete process collection. A runtime that adds or
  changes capabilities/SIDs currently fails closed; no permissive exception
  has been invented.
- Validate successful runtime behavior separately; that does not block other
  modules. The dynamic-preview parity item remains open.

## ABI evidence (source checked, not a Windows execution claim)

The BrowserProcessID getter uses original ICoreWebView2 slot 37. This was
verified by enumerating all fields, including private usize placeholders, of
Microsoft's repr(C) ICoreWebView2_Vtbl in windows-rs commit
c4932863208881f0cf9270471864d94761d09cd0. The IUnknown base contributes three
function pointers; BrowserProcessId is the 35th subsequent member (zero-based
slot 37). It is not inferred from a gap in our vendored constants.

- Complete official vtable declaration starts at line 555; BrowserProcessId
  is at line 657:
  https://github.com/microsoft/windows-rs/blob/c4932863208881f0cf9270471864d94761d09cd0/crates/libs/webview/src/bindings.rs#L555-L660
- Official IUnknown base layout, three members:
  https://github.com/microsoft/windows-rs/blob/c4932863208881f0cf9270471864d94761d09cd0/crates/libs/core/src/unknown.rs#L16-L24
- Official getter signature, HRESULT with UINT32 output:
  https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2?view=webview2-1.0.3537.50#get_browserprocessid

scripts/verify-preview-webview2-abi.ps1 is an additional reproducible header
check against the pinned official NuGet SDK. It only reads/downloads SDK
source, without creating an AppContainer or starting WebView2. That script
was not executed in this environment; the public Microsoft vtable above
was directly retrieved and counted.

## Primary references

- Microsoft: implementing an AppContainer, SECURITY_CAPABILITIES and
  CreateProcess:
  https://learn.microsoft.com/en-us/windows/win32/secauthz/implementing-an-appcontainer
- Microsoft: AppContainer inheritance in legacy applications:
  https://learn.microsoft.com/en-us/windows/win32/secauthz/appcontainer-for-legacy-applications-
- Microsoft: JOB_LIST, HANDLE_LIST, SECURITY_CAPABILITIES:
  https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute
- Microsoft: job membership, breakaway and kill-on-close:
  https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects
- Microsoft: nested jobs:
  https://learn.microsoft.com/en-us/windows/win32/procthread/nested-jobs
- Microsoft: AppContainer network isolation:
  https://learn.microsoft.com/en-us/windows/security/operating-system-security/network-security/windows-firewall/troubleshooting-uwp-firewall
- WebView2 maintainer: AppContainer is supported; arbitrary restricted-token
  hosts are not equivalent:
  https://github.com/MicrosoftEdge/WebView2Feedback/issues/4850#issuecomment-2412609705
- Microsoft: WebView2 process collection and UDF ownership:
  https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/process-model
- Microsoft: GetProcessInfos excludes crashpad:
  https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2environment8
- Google Project Zero: native process-launch/WFP executable exception analysis:
  https://projectzero.google/2021/08/understanding-network-access-windows-app.html
