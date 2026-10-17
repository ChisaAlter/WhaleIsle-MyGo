// Independently authored, fixed YouTube native-seat transport. No network or
// browser-media API is used by this opaque viewer.
import type { GestureEvent } from './native-actions';

export type MediaRect = { x: number; y: number; width: number; height: number };
export type MediaGeometry = { rect: MediaRect; clip: MediaRect; viewport: { width: number; height: number }; visible: boolean };
export type MediaStatus = { phase: 'idle' | 'pending' | 'created' | 'hidden' | 'closed' | 'error'; message: string };
type Sender = (type: string, fields: Record<string, unknown>) => boolean;
type Job = { timer: ReturnType<typeof setTimeout>; complete: (error?: string, state?: string) => void };
const jobs = new Map<string, Job>();
const seats = new Set<{ remove(): void }>();
const removals = new Set<{ revision: number; stop(): void }>();
const usedGestures = new WeakSet<Event>();
let sender: Sender | undefined;
let sequence = 0;
let componentSequence = 0;
let context = { enabled: false, streaming: false, revision: 0 };
let closed = false;

export function configureMediaTransport(post: Sender) { sender = post; }
export function updateMediaContext(next: typeof context) {
  const invalidated = context.revision !== next.revision || !next.enabled || next.streaming;
  context = next;
  if (invalidated) for (const seat of [...seats]) seat.remove();
  if (invalidated) for (const removal of [...removals]) removal.stop();
}

function checkedGeometry(value: MediaGeometry): MediaGeometry {
  const { rect, clip, viewport } = value;
  if (!rect || !clip || !viewport || typeof value.visible !== 'boolean' ||
      ![rect.x, rect.y, rect.width, rect.height, clip.x, clip.y, clip.width, clip.height, viewport.width, viewport.height].every(Number.isFinite) ||
      viewport.width < 1 || viewport.height < 1 || viewport.width > 4096 || viewport.height > 4096 ||
      Math.abs(rect.x) > 1e6 || Math.abs(rect.y) > 1e6 || rect.width < 0 || rect.height < 0 || rect.width > 4096 || rect.height > 4096 ||
      clip.x < 0 || clip.y < 0 || clip.width < 0 || clip.height < 0 || clip.x + clip.width > viewport.width + 0.5 || clip.y + clip.height > viewport.height + 0.5 ||
      (clip.width > 0 && clip.height > 0 && (clip.x < rect.x - 0.5 || clip.y < rect.y - 0.5 || clip.x + clip.width > rect.x + rect.width + 0.5 || clip.y + clip.height > rect.y + rect.height + 0.5)) ||
      (value.visible && (clip.width <= 0 || clip.height <= 0))) throw new Error('The player rectangle is outside the supported native viewport.');
  return { rect: { ...rect }, clip: { ...clip }, viewport: { ...viewport }, visible: value.visible };
}

function trustedActivation(event: GestureEvent) {
  const native = event && 'nativeEvent' in event ? event.nativeEvent : event;
  if (!(native instanceof Event) || !native.isTrusted || !(native.target instanceof Node) || native.target.getRootNode() !== document ||
      !(native instanceof MouseEvent) || native.type !== 'click' || native.button !== 0 || usedGestures.has(native) ||
      (navigator.userActivation && !navigator.userActivation.isActive)) throw new Error('Loading YouTube requires a new real click or keyboard button activation.');
  usedGestures.add(native);
}

function dispatch(media: Record<string, unknown>, userGesture: boolean, complete: Job['complete']): string | undefined {
  if (!sender || closed || sequence >= Number.MAX_SAFE_INTEGER) { complete('The native media bridge is unavailable.'); return; }
  const current = ++sequence;
  const requestId = `media-${current}`;
  const timer = setTimeout(() => {
    if (!jobs.delete(requestId)) return;
    complete('The player Host did not confirm the request; playback availability is unknown.');
  }, 15000);
  jobs.set(requestId, { timer, complete });
  if (!sender('media', { sequence: current, requestId, userGesture, media })) {
    clearTimeout(timer); jobs.delete(requestId); complete('The player request could not be sent.'); return;
  }
  return requestId;
}
function forget(requestId?: string) {
  if (!requestId) return;
  const job = jobs.get(requestId);
  if (job) clearTimeout(job.timer);
  jobs.delete(requestId);
}

// A full existing event queue must not leave an orphaned native surface after
// React unmounts it. Retry only teardown, never player creation or playback.
// A source/capability teardown belongs to the Host and ends this cleanup loop.
function removeUntilAcknowledged(media: Record<string, unknown>, revision: number) {
  let requestId: string | undefined;
  let retry: ReturnType<typeof setTimeout> | undefined;
  let stopped = false;
  const valid = () => !closed && context.enabled && !context.streaming && context.revision === revision;
  const task = { revision, stop() {
    stopped = true; clearTimeout(retry); forget(requestId); removals.delete(task);
  } };
  const attempt = () => {
    if (stopped || !valid()) { task.stop(); return; }
    requestId = dispatch(media, false, (error, state) => {
      requestId = undefined;
      if (stopped || !valid() || (!error && state === 'closed')) { task.stop(); return; }
      retry = setTimeout(attempt, 250);
    });
  };
  removals.add(task);
  attempt();
}

export function createYouTubeSeat(options: { videoId: string; title: string; revision: number; onStatus: (status: MediaStatus) => void }) {
  let componentId = '';
  let geometry: MediaGeometry | undefined;
  let active = false;
  let removed = false;
  let requestId: string | undefined;
  let sentGeometry = '';
  let lastStatus = '';
  const valid = /^[A-Za-z0-9_-]{11}$/.test(options.videoId) && new TextEncoder().encode(options.title).length <= 512;
  const say = (phase: MediaStatus['phase'], message: string) => {
    const key = `${phase}:${message}`;
    if (!removed && key !== lastStatus) { lastStatus = key; options.onStatus({ phase, message }); }
  };
  const allowed = () => !removed && !closed && context.enabled && !context.streaming && context.revision === options.revision;
  const payload = (operation: 'upsert' | 'remove', id = componentId) => ({ operation, componentId: id, revision: options.revision,
    videoId: options.videoId, title: options.title, ...geometry! });
  const retire = () => {
    forget(requestId); requestId = undefined;
    if (active && geometry) removeUntilAcknowledged(payload('remove'), options.revision);
    active = false;
  };
  const failure = (message: string) => {
    retire();
    // Host tombstones removed identities. A deliberate retry gets a fresh ID,
    // never resurrecting a late upsert for an old component identity.
    componentId = '';
    sentGeometry = '';
    say('error', message);
  };
  const send = (userGesture: boolean) => {
    if (!allowed() || !geometry || requestId || !active) return;
    const encoded = JSON.stringify(geometry);
    if (!userGesture && encoded === sentGeometry) return;
    sentGeometry = encoded;
    if (userGesture) say('pending', 'Requesting an isolated YouTube player…');
    // dispatch can reject synchronously; do not overwrite failure's fresh ID.
    let rejected = false;
    const id = dispatch(payload('upsert'), userGesture, (error, state) => {
      requestId = undefined;
      if (removed) return;
      if (error) { rejected = true; failure(error); return; }
      if (!allowed()) { retire(); return; }
      if (state === 'hidden') say('hidden', 'Player hidden while outside the visible area or behind an overlay.');
      else if (state === 'native-view-created') say('created', 'Isolated player view created. YouTube controls playback and video availability.');
      else { failure('The Host returned an unexpected player state.'); return; }
      send(false);
    });
    if (!rejected) requestId = id;
  };
  const seat = {
    setGeometry(next: MediaGeometry) {
      if (removed) return;
      const hideImmediately = active && geometry?.visible && !next.visible;
      try { geometry = checkedGeometry(next); } catch (error) { failure(error instanceof Error ? error.message : 'Invalid player geometry.'); return; }
      // Hiding for an overlay/unmount must not wait behind an older geometry
      // acknowledgement. Higher sequence wins; the old response is retired.
      if (hideImmediately && requestId) { forget(requestId); requestId = undefined; }
      if (active) send(false);
    },
    activate(event: GestureEvent) {
      if (!allowed() || !valid) { say('error', !valid ? 'The video ID or title exceeds the supported player contract.' : 'Isolated YouTube playback is unavailable in this Host.'); return; }
      if (active) return;
      try {
        if (!geometry?.visible) throw new Error('The player must be visible and unobstructed before loading.');
        trustedActivation(event);
        componentId = `youtube-${++componentSequence}`;
        active = true;
        send(true);
      } catch (error) { say('error', error instanceof Error ? error.message : 'The player could not be activated.'); }
    },
    remove() {
      if (removed) return;
      retire(); removed = true; seats.delete(seat);
    },
  };
  seats.add(seat);
  return seat;
}

// bridge.ts invokes this only for the exact parent/channel/instance.
export function receiveMediaResponse(data: Record<string, unknown>) {
  if (typeof data.requestId !== 'string' || typeof data.ok !== 'boolean') return;
  const job = jobs.get(data.requestId);
  if (!job) return;
  forget(data.requestId);
  if (!data.ok) {
    const error = typeof data.error === 'string' && new TextEncoder().encode(data.error).length <= 4096 ? data.error : 'The Host could not create the isolated player.';
    job.complete(error); return;
  }
  if (!['native-view-created', 'hidden', 'closed'].includes(String(data.state))) { job.complete('The Host returned an invalid media acknowledgement.'); return; }
  job.complete(undefined, data.state as string);
}
export function closeMedia() {
  for (const seat of [...seats]) seat.remove();
  closed = true;
  for (const removal of [...removals]) removal.stop();
  for (const id of [...jobs.keys()]) forget(id);
}
