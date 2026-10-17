// Independently authored persistence adapter over the official MIT Store API.
// This module owns one viewer instance, not one store per streamed revision.
// Configure once before mounting Renderer. Do not also pass Renderer.initialState
// or Renderer.onStateUpdate: SDK hydration replay cannot be distinguished from a
// user write at the public Store.set boundary.
import { createStore as createOfficialStore } from "@whaleisle/official-lang-core";
import { byteLength, isObject, LIMITS } from "./bridge";

type Store = ReturnType<typeof createOfficialStore>;
type Snapshot = Record<string, unknown>;

export interface PersistenceOptions {
  initialState: Snapshot;
  // Return false if the host did not accept the snapshot. A later write retries.
  onState: (state: Snapshot) => boolean | void;
  onError?: (message: string) => void;
}

let options: PersistenceOptions | undefined;
let activeOwner: symbol | undefined;
let retained: Snapshot = {};
let acknowledgedJSON = "{}";
let lastError = "";

function fail(message: string) {
  if (message === lastError) return;
  lastError = message;
  // A reporting failure must not break SDK state changes or recurse into itself.
  try { options?.onError?.(message); } catch { /* The host owns error reporting. */ }
}

function encode(state: Snapshot): string {
  const ancestors = new Set<object>();
  let remaining = LIMITS.state;
  const charge = (size: number) => {
    remaining -= size;
    if (remaining < 0) throw new Error("OpenUI state exceeds 48 KiB; this change was not persisted.");
  };
  const text = (value: string) => {
    // UTF-16 length is a cheap lower bound before allocating UTF-8 bytes.
    if (value.length > remaining) charge(value.length);
    charge(byteLength(value) + 2);
  };
  const check = (value: unknown, depth = 0): void => {
    if (depth > 128) throw new Error("OpenUI state exceeds the supported nesting depth.");
    if (value === null) { charge(4); return; }
    if (typeof value === "string") { text(value); return; }
    if (typeof value === "boolean") { charge(value ? 4 : 5); return; }
    if (typeof value === "number" && Number.isFinite(value)) { charge(String(value).length); return; }
    if (typeof value !== "object" || value === null) throw new Error("OpenUI state contains a value that JSON cannot preserve.");
    if (ancestors.has(value)) throw new Error("OpenUI state contains a circular value.");
    const prototype = Object.getPrototypeOf(value);
    if (!Array.isArray(value) && prototype !== Object.prototype && prototype !== null) {
      throw new Error("OpenUI state contains an unsupported object.");
    }
    charge(2);
    ancestors.add(value);
    if (Array.isArray(value)) {
      // Holes and undefined entries must not silently change into persisted nulls.
      for (let index = 0; index < value.length; index += 1) { if (index) charge(1); check(value[index], depth + 1); }
    } else {
      // The official form API uses optional undefined object metadata (for
      // example componentType). JSON intentionally omits those properties.
      let count = 0;
      for (const [name, item] of Object.entries(value)) if (item !== undefined) {
        if (count++) charge(1);
        text(name); charge(1); check(item, depth + 1);
      }
    }
    ancestors.delete(value);
  };
  check(state);
  const json = JSON.stringify(state);
  if (byteLength(json) > LIMITS.state) throw new Error("OpenUI state exceeds 48 KiB; this change was not persisted.");
  return json;
}

export function configurePersistence(configuration: PersistenceOptions): void {
  if (options) throw new Error("OpenUI persistence was already configured for this viewer instance.");
  if (!isObject(configuration.initialState) || typeof configuration.onState !== "function") {
    throw new Error("Invalid OpenUI persistence configuration.");
  }
  // Validate and detach the entire initial snapshot before any store can expose it.
  // Both reactive bindings and form keys are hydrated together, without callbacks.
  const json = encode(configuration.initialState);
  retained = JSON.parse(json) as Snapshot;
  acknowledgedJSON = json;
  options = { ...configuration, initialState: retained };
}

export function createPersistentStore(): Store {
  if (!options) throw new Error("Configure OpenUI persistence before mounting the renderer.");
  const owner = Symbol("OpenUI viewer store");
  const backing = createOfficialStore();
  const dirty = new Set(Object.keys(retained));
  let disposed = false;
  let initialized = false;
  // Official initialize treats persisted keys as touched. Passing all keys in one
  // call also prevents half-hydrated snapshots containing only form or $ keys.
  backing.initialize({}, retained);
  activeOwner = owner;

  const ownsPersistence = () => !disposed && activeOwner === owner;
  const dirtySnapshot = (): Snapshot => Object.fromEntries(
    [...dirty].map(name => [name, backing.get(name)]),
  );
  const keepCurrent = () => {
    if (ownsPersistence()) retained = dirtySnapshot();
  };
  const emit = () => {
    if (!ownsPersistence()) return;
    // Retain the live dirty state even when it exceeds the host's persistence
    // bound; a remount must not revert an edit merely because it could not save.
    keepCurrent();
    let json: string;
    try { json = encode(retained); }
    catch (error) { fail(error instanceof Error ? error.message : String(error)); return; }
    lastError = "";
    if (json === acknowledgedJSON) return;
    try {
      // Never give the host mutable references into the SDK's live store.
      const accepted = options!.onState(JSON.parse(json) as Snapshot);
      if (accepted !== false && ownsPersistence()) acknowledgedJSON = json;
    } catch {
      fail("The OpenUI state could not be sent to the host.");
    }
  };

  return {
    get: name => backing.get(name),
    getSnapshot: () => backing.getSnapshot(),
    subscribe: listener => disposed ? () => {} : backing.subscribe(listener),
    set(name, value, settings) {
      if (disposed) return;
      // Record intent before equality checks inside the official store. A user
      // setting the existing default still makes the key touched; @Reset uses
      // pristine:true and makes the key eligible for future streamed defaults.
      if (settings?.pristine) dirty.delete(name);
      else dirty.add(name);
      backing.set(name, value, settings);
      emit();
    },
    initialize(defaults, persisted) {
      if (disposed) return;
      // Accommodate the public Store initializer once. Later declarations may
      // update pristine defaults, but repeated persisted input is never replayed.
      const firstHydration: Snapshot = {};
      if (!initialized) {
        for (const name of Object.keys(persisted)) {
          if (dirty.has(name)) continue;
          Object.defineProperty(firstHydration, name, { value: persisted[name], enumerable: true });
          dirty.add(name);
        }
        initialized = true;
      }
      backing.initialize(defaults, firstHydration);
      keepCurrent();
      // Initialization is deliberately silent. Pristine defaults are not saved.
    },
    dispose() {
      if (disposed) return;
      keepCurrent();
      disposed = true;
      if (activeOwner === owner) activeOwner = undefined;
      backing.dispose();
    },
  };
}
