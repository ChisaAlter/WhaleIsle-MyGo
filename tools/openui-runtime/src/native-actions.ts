// Fixed, user-gesture-bound Host actions. No browser clipboard or navigation API.
export type NativeAction = { type: "clipboard"; text: string } | { type: "open-url"; url: string };
export type GestureEvent = Event | { nativeEvent: Event };
export type NativeActionPermit = { readonly __nativeActionPermit: unique symbol };
type Sender = (type: string, fields: Record<string, unknown>) => boolean;
type PermitState = { actions: NativeAction[]; next: number; revision: number; busy: boolean; closed: boolean };
type Pending = { requestId: string; action: NativeAction["type"]; permit: PermitState; resolve: () => void; reject: (error: Error) => void; timer?: ReturnType<typeof setTimeout> };

const permits = new WeakMap<NativeActionPermit, PermitState>();
const consumedEvents = new WeakSet<Event>();
const pending = new Map<string, Pending>();
const encoder = new TextEncoder();
let sender: Sender | undefined;
let enabled = false;
let streaming = false;
let revision = 0;
let contextEpoch = 0;
let sequence = 0;
let closed = false;

export function configureNativeActionTransport(post: Sender) { sender = post; }

function finish(job: Pending, error?: Error) {
  if (pending.get(job.requestId) !== job) return;
  pending.delete(job.requestId);
  clearTimeout(job.timer);
  job.permit.busy = false;
  if (error) { job.permit.closed = true; job.reject(error); }
  else job.resolve();
}

export function updateNativeActionContext(next: { enabled: boolean; streaming: boolean; revision: number }) {
  const invalidated = revision !== next.revision || !next.enabled || next.streaming;
  enabled = next.enabled;
  streaming = next.streaming;
  revision = next.revision;
  if (invalidated) contextEpoch += 1;
  if (invalidated) for (const job of [...pending.values()]) finish(job, new Error("The page changed before the Host action result arrived; the action may already have completed."));
}

function validateAction(input: NativeAction): NativeAction {
  if (!input || typeof input !== "object") throw new Error("Invalid Host action.");
  let value: NativeAction;
  if (input.type === "clipboard" && typeof input.text === "string") {
    if (input.text.includes("\0")) throw new Error("Clipboard text containing NUL cannot be copied without truncation on all supported hosts.");
    for (let index = 0; index < input.text.length; index += 1) {
      const unit = input.text.charCodeAt(index);
      if (unit >= 0xd800 && unit <= 0xdbff) {
        const next = input.text.charCodeAt(++index);
        if (!(next >= 0xdc00 && next <= 0xdfff)) throw new Error("Clipboard text contains an unpaired Unicode surrogate and cannot be preserved by the Host protocol.");
      } else if (unit >= 0xdc00 && unit <= 0xdfff) throw new Error("Clipboard text contains an unpaired Unicode surrogate and cannot be preserved by the Host protocol.");
    }
    value = { type: "clipboard", text: input.text };
  }
  else if (input.type === "open-url" && typeof input.url === "string") {
    if (/[\u0000-\u0020\u007f-\u009f]/.test(input.url)) throw new Error("A link must be an absolute HTTP(S) URL without whitespace or control characters.");
    let target: URL;
    try { target = new URL(input.url); } catch { throw new Error("A link must be an absolute HTTP(S) URL."); }
    if (!["http:", "https:"].includes(target.protocol) || !target.hostname || target.username || target.password) throw new Error("Only absolute HTTP(S) links without embedded credentials are supported.");
    value = { type: "open-url", url: target.href };
  } else throw new Error("Unsupported Host action or invalid payload.");
  // Reserve the longest safe sequence envelope. No source text is truncated.
  const envelope = { channel: "whale-openui", version: 1, instanceId: "0".repeat(64), ...value,
    requestId: `native-action-${Number.MAX_SAFE_INTEGER}`, sequence: Number.MAX_SAFE_INTEGER, userGesture: true };
  if (encoder.encode(JSON.stringify(envelope)).length > 65536) throw new Error("The complete Host action exceeds the existing 64 KiB event budget. Nothing was copied or opened.");
  return value;
}

export function prepareNativeActions(event: GestureEvent | undefined, actions: readonly NativeAction[]): NativeActionPermit {
  if (!sender || !enabled || streaming || closed) throw new Error("Host clipboard and external-link actions are unavailable in this preview.");
  const native = event && "nativeEvent" in event ? event.nativeEvent : event;
  if (!(native instanceof Event) || !native.isTrusted || !(native.target instanceof Node) || native.target.getRootNode() !== document) throw new Error("This action requires a real click or keyboard activation in the viewer.");
  const click = native.type === "click" && native instanceof MouseEvent && native.button === 0;
  const key = native.type === "keydown" && native instanceof KeyboardEvent && !native.repeat && (native.key === "Enter" || native.key === " ");
  if ((!click && !key) || consumedEvents.has(native)) throw new Error("This action requires a new click or keyboard activation.");
  // Legacy engines without UserActivation still provide immutable isTrusted.
  // No timer, postMessage or synthetic Event can create this invocation permit.
  if (navigator.userActivation && !navigator.userActivation.isActive) throw new Error("The user activation is no longer active.");
  if (!Array.isArray(actions) || actions.length === 0) throw new Error("No Host action was supplied.");
  const values = actions.map(validateAction);
  const permit = Object.freeze({}) as NativeActionPermit;
  permits.set(permit, { actions: values, next: 0, revision: contextEpoch, busy: false, closed: false });
  consumedEvents.add(native);
  return permit;
}

export function performNativeAction(permit: NativeActionPermit): Promise<void> {
  return new Promise((resolve, reject) => {
    const state = permits.get(permit);
    if (!state || state.closed || state.busy || state.revision !== contextEpoch || !enabled || streaming || closed || !sender) {
      reject(new Error("The user action is no longer available.")); return;
    }
    const action = state.actions[state.next];
    if (!action || sequence >= Number.MAX_SAFE_INTEGER) { state.closed = true; reject(new Error("The user action has already been consumed.")); return; }
    state.next += 1;
    state.busy = true;
    sequence += 1;
    const requestId = `native-action-${sequence}`;
    const job: Pending = { requestId, action: action.type, permit: state, resolve, reject };
    pending.set(requestId, job);
    job.timer = setTimeout(() => finish(job, new Error("The Host action result was not received in time; the action may already have completed.")), 15000);
    const { type, ...payload } = action;
    if (!sender(type, { ...payload, requestId, sequence, userGesture: true })) finish(job, new Error("The Host action request could not be sent."));
  });
}

// bridge.ts calls this only after exact parent source/channel/instance checks.
export function receiveNativeActionResponse(data: Record<string, unknown>) {
  if (typeof data.requestId !== "string") return;
  const job = pending.get(data.requestId);
  if (!job || data.action !== job.action || typeof data.ok !== "boolean") return;
  if (data.ok === true) { finish(job); return; }
  const message = typeof data.error === "string" && encoder.encode(data.error).length <= 4096 ? data.error : "The Host action was not completed.";
  finish(job, new Error(message || "The Host action was not completed."));
}

export function closeNativeActions() {
  closed = true;
  enabled = false;
  for (const job of [...pending.values()]) finish(job, new Error("The viewer closed before the Host action result arrived; the action may already have completed."));
}
