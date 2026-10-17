// Typed resource transport: all network activity remains in the Go parent.
export type ResourcePath = "/api/wiki" | "/api/image-search" | "/api/street-route" | "/api/map-tile";
type Sender = (type: string, fields: Record<string, unknown>) => boolean;
type Job = { requestId: string; path: ResourcePath; query: Record<string, string>; resolve: (value: unknown) => void; reject: (error: Error) => void; signal?: AbortSignal; abort: () => void; timer?: ReturnType<typeof setTimeout>; active: boolean; settled: boolean };
const queue: Job[] = [];
const pending = new Map<string, Job>();
let sender: Sender | undefined;
let serial = 0;
let stopped = false;

export function configureResourceTransport(post: Sender) { sender = post; pump(); }
function finish(job: Job, error?: Error, value?: unknown) {
  if (job.settled) return;
  job.settled = true;
  clearTimeout(job.timer);
  job.signal?.removeEventListener("abort", job.abort);
  pending.delete(job.requestId);
  const position = queue.indexOf(job); if (position >= 0) queue.splice(position, 1);
  if (error) job.reject(error); else job.resolve(value);
  pump();
}
function pump() {
  if (!sender || stopped) return;
  while (pending.size < 4 && queue.length) {
    const job = queue.shift()!;
    if (job.settled) continue;
    job.active = true;
    pending.set(job.requestId, job);
    job.timer = setTimeout(() => finish(job, new Error("OpenUI resource request timed out.")), 15000);
    if (!sender("api-request", { requestId: job.requestId, path: job.path, query: job.query })) {
      finish(job, new Error("OpenUI resource request could not be sent."));
    }
  }
}
export function requestResource<T = unknown>(path: ResourcePath, query: Record<string, string>, signal?: AbortSignal): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    if (stopped || signal?.aborted) { reject(new DOMException("Resource request cancelled.", "AbortError")); return; }
    if (!Object.values(query).every(value => typeof value === "string")) { reject(new Error("Invalid resource query.")); return; }
    const job: Job = { requestId: `owned-${++serial}`, path, query: { ...query }, resolve: resolve as (value: unknown) => void, reject,
      signal, active: false, settled: false, abort: () => {} };
    job.abort = () => finish(job, new DOMException("Resource request cancelled.", "AbortError"));
    signal?.addEventListener("abort", job.abort, { once: true });
    queue.push(job); pump();
  });
}
// Called only after bridge.ts validates the exact parent source and instance.
export function receiveResourceResponse(data: Record<string, unknown>) {
  if (typeof data.requestId !== "string" || !Number.isInteger(data.status)) return;
  const job = pending.get(data.requestId); if (!job) return;
  const status = data.status as number;
  if (status < 200 || status > 599) { finish(job, new Error("Invalid resource status.")); return; }
  try {
    if (new TextEncoder().encode(JSON.stringify(data.body)).length > 25 * 1024 * 1024) throw new Error("Resource response exceeds the host byte limit.");
    if (status >= 300) {
      const body = data.body as { error?: { message?: unknown } } | null;
      throw new Error(typeof body?.error?.message === "string" ? body.error.message : `Resource request failed (${status}).`);
    }
    finish(job, undefined, data.body);
  } catch (error) { finish(job, error instanceof Error ? error : new Error(String(error))); }
}
export function closeResources() {
  stopped = true;
  for (const job of [...pending.values(), ...queue]) finish(job, new DOMException("Viewer closed.", "AbortError"));
}
