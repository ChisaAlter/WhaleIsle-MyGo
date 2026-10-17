// Independently authored WhaleIsle-MyGo protocol adapter.
// The parent remains responsible for instance ownership and native actions.
import { receiveResourceResponse } from "./resources";
export const LIMITS = { program: 262144, state: 49152, event: 65536, message: 16384, error: 4096, resolved: 32768 };
const encoder = new TextEncoder();
export const byteLength = (value: string) => encoder.encode(value).length;
export const isObject = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === "object" && !Array.isArray(value);

export interface RenderMessage {
  program: string;
  streaming: boolean;
  state: Record<string, unknown>;
  theme: { appearance: "light" | "dark" };
}

export function connect(receive: (value: RenderMessage) => void, fail: (message: string) => void) {
  const match = /^whale-openui:([a-fA-F0-9]{64})$/.exec(window.name);
  const instanceId = match?.[1];
  let closed = false;
  const post = (type: string, fields: Record<string, unknown> = {}) => {
    if (closed || !instanceId || window.parent === window) return false;
    const message = { ...fields, channel: "whale-openui", version: 1, instanceId, type };
    try {
      if (byteLength(JSON.stringify(message)) > LIMITS.event) {
        fail("The OpenUI event exceeds the host's size limit.");
        return false;
      }
      window.parent.postMessage(message, "*");
      return true;
    } catch {
      fail("The OpenUI event cannot be serialized.");
      return false;
    }
  };
  const listener = (event: MessageEvent) => {
    const data = event.data;
    if (closed || event.source !== window.parent || !isObject(data) ||
        data.channel !== "whale-openui" || data.version !== 1 || data.instanceId !== instanceId) return;
    if (data.type === "api-response") { receiveResourceResponse(data); return; }
    // Unknown messages cannot change state.
    if (data.type !== "render") return;
    if (typeof data.program !== "string" || !data.program.trim() || byteLength(data.program) > LIMITS.program ||
        typeof data.streaming !== "boolean" || !isObject(data.state) || !isObject(data.theme) ||
        !["light", "dark"].includes(String(data.theme.appearance))) {
      fail("Invalid OpenUI render message.");
      return;
    }
    try {
      if (byteLength(JSON.stringify(data.state)) > LIMITS.state) {
        fail("The OpenUI saved state exceeds the host's size limit.");
        return;
      }
    } catch {
      fail("Invalid OpenUI saved state.");
      return;
    }
    receive({ program: data.program, streaming: data.streaming, state: data.state,
      theme: { appearance: data.theme.appearance as "light" | "dark" } });
  };
  if (!instanceId || window.parent === window) {
    fail("This viewer must be opened by the isolated WhaleIsle-MyGo preview helper.");
  } else {
    window.addEventListener("message", listener);
    post("ready", { runtime: "whaleisle-independent-col-v1" });
  }
  return { post, close() { closed = true; window.removeEventListener("message", listener); } };
}
