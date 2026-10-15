# Desktop pet action contracts

## Source and scope

Compared the original desktop pet menus and `src/main/desktop-live2d.js` at WhaleIsle `90465b67ec16765631d54c195af19d9ff3c3a770` with the native pet handlers. This module repairs concrete action contracts; it does not change the renderer/GPU pipeline or claim pixel parity.

## Implemented

- **Hide** persists the native companion's disabled state, matching the original panel opt-out. It preserves the saved skin/THA4 choice and other drafts. A failed configuration write keeps the pet visible with an error. Closing a window through the OS remains temporary.
- **Settings** raises the main window on the existing local Pet pane, including offline, without selecting another conversation.
- **Look** is a manual, one-shot action. Before capture, it requires an enabled Whale assistant, a ready Host, and the explicit configured vision route still present in the current image-capable model catalog. Every click asks for approval identifying the provider/model/Host and warning that the screenshot may contain private information and may be retained in Whale chat.
- The Windows capture reads only the display containing the existing pet window. It creates an in-memory image of at most 768×480 pixels, encoded JPEG quality 70. No screenshot file, clipboard entry, or screenshot log is created. The existing authenticated `dsh-whale/pet/look` endpoint owns model execution and conversation retention.
- Look has one pending request, a four-second capture cooldown, a 60-second request deadline, and cancellation when its pet window closes/recreates. Before sending, it rechecks the pet window, Host generation/origin/cookie, selected conversation, configured provider/model and current capability. An obsolete capture is discarded.
- Replies are escaped on initial HTML render and assigned with `textContent` on refresh. The pet page receives no Host credentials, screenshot data, arbitrary IPC or native capture capability.

Existing feed-file, token bowl/feeding, pat, sleep/wake, local skin selection, selected-task status, and independent persistent Whale conversation navigation remain in place. The native chat action opens the main persistent conversation; this is not a claim that the original compact overlay chat/model-picker surface is reproduced.

## Verification boundaries

Added regression source for Hide persistence/conflicts and exact local actions; Look cooldown/cancellation, synthetic JPEG size/color, configured vision-route gating, and fake-Host RPC routing/redirect rejection. Tests use synthetic pixels and an in-process HTTP test server only. Windows cross-compilation does not prove runtime screenshot correctness, GPU output, monitor placement, native-dialog focus or real model behavior.

No user screenshot, real Host request, real vision-model invocation or desktop-pet/GPU runtime acceptance was performed for this module. Those remain interactive acceptance gates.
