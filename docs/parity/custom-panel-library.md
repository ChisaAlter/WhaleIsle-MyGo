# N01/N02: native custom Panels

Implemented source path: a real `custom-panels` workspace dock resource renders the
per-Bot list, a reusable library, editor and preview. Existing dock routing retains
the resource identity across Files/Panel navigation. The library seat is a
singleton per workspace; individual user Panels live in its ordered list.

## User flow

1. Open a Bot's canonical chat and the right-side **Panels** tab.
2. Describe requirements and choose **Create with Bot**. This prepares a prompt
   in the existing composer, preserving its send/stream/tool-approval path. Review
   and send; continue discussing requirements with that Bot.
3. Choose **Import reply** for the latest completed assistant text, or paste the
   returned JSON. Review the native preview, then **Save**. A template and an
   independent instance for this Bot are saved together.
4. Another Bot's list can **Add copy** from the library. Changes to content,
   widget state, dimensions and list order affect only that Bot's copy.
5. **Edit** changes a local copy. **Edit template** / **Update library template**
   explicitly changes the reusable definition for future additions. Existing
   copies remain unchanged. **Remove** removes only this Bot's instance.

## Declarative contract

`internal/store/custompanel.Example` is a complete importable example. A definition
has title, width (240–1600), height (120–1600), columns (1–4), and ordered widgets.
Each widget has a unique id and kind: text, input, checkbox, progress, select or
action. Optional properties are label, text, options, value and fontSize (10–32).
JSON editing supports reordering widgets and changing all supported content and
layout values; list arrows reorder Panels, and dimension buttons adjust sizes.
Large Panel widths are constrained visually by the containing dock while their
preferred size remains persisted. Existing right-dock resizing remains available.

Actions only prepare text in the same Bot's composer; they do not send messages,
execute scripts, access files, call arbitrary Host methods, or expand permissions.
Preview has no interactive callbacks. This is a bounded native widget vocabulary,
not arbitrary HTML/CSS/JavaScript. Unsupported schemas are reported in the editor.

## Persistence and ownership

`userData/assistant/panels.json` has version 1, templates and deep-copied instances
keyed by the existing durable workspace Host namespace + Bot ID + definition ID.
The namespace is read from WorkspaceLayoutScope (supervised dsh home or remote
origin binding); changing a local Host listen port does not change Panel identity. The ordered instance array also
stores each instance's dimensions, widgets and string state map. Atomic local
replacement must succeed before in-memory data changes. A load error is visible;
there is no silent overwrite or ephemeral-success fallback. The canonical session
is matched against the current Host roster; unrelated chats and rooms are not
mapped to the last Bot selection. Callbacks recheck the live endpoint/client as well as this binding, and native UI
keys include Host and Bot.

## Verification boundaries

Focused source fixtures cover creation/reload, copy/state/size isolation, canonical
Bot binding and completed-reply import, and actual workspace dock rendering and
resource reuse. Tests/builds were not run by the implementer. Windows GUI and real
Bot conversation acceptance remain separate gates. No live message, service,
commit or push was performed by the implementer.
