// Independently authored shell; no open-intelligent-ui application code.
import React, { Component, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { createStreamingParser, Renderer, type ActionEvent, type OpenUIError, type ParseResult } from "@openuidev/react-lang";
import { compatibleLibrary } from "./library";
import { configurePersistence } from "./state-adapter";
import { closeResources, configureResourceTransport } from "./resources";
import "./viewer.css";
import { connect, byteLength, LIMITS, type RenderMessage } from "./bridge";
import { HtmlViewSupportContext, HostActionsContext, ProgramRevisionContext } from "./capabilities";
import { configureNativeActionTransport, updateNativeActionContext, closeNativeActions } from "./native-actions";

const mount = document.getElementById("viewer")!;
const fatal = document.getElementById("fatal")!;
const root = createRoot(mount);
let transport: ReturnType<typeof connect>;
let currentProgram = "";
let currentStreaming = false;
let generation = 0;
let hydrated = false;
let lastHostError = "";
let errorSerial = 0;
let hardErrorGeneration = -1;

function report(message: string, code = "runtime-error") {
  errorSerial += 1;
  // SDK/parser errors have explicit current-result checks below. Browser,
  // persistence and capability failures cannot be cleared by a successful parse.
  if (!["sdk-error", "program-error", "unsupported-provider"].includes(code)) hardErrorGeneration = generation;
  // Bound by UTF-8 bytes rather than JavaScript UTF-16 code units.
  let text = String(message);
  while (byteLength(text) > LIMITS.error) text = text.slice(0, Math.floor(text.length * 0.9));
  fatal.textContent = text;
  fatal.hidden = false;
  const key = `${generation}\n${code}\n${text}`;
  if (key !== lastHostError) {
    lastHostError = key;
    transport?.post("error", { message: text, code });
  }
}

class Boundary extends Component<{ version: number; children: React.ReactNode }, { version: number; failed: boolean }> {
  state = { version: this.props.version, failed: false };
  static getDerivedStateFromError() { return { failed: true }; }
  static getDerivedStateFromProps(props: { version: number }, state: { version: number }) {
    return props.version !== state.version ? { version: props.version, failed: false } : null;
  }
  componentDidCatch(error: Error) { if (this.props.version === generation) report(error.message, "render-error"); }
  render() { return this.state.failed ? null : this.props.children; }
}

function Viewer({ data, version }: { data: RenderMessage; version: number }) {
  const parser = useMemo(() => createStreamingParser(compatibleLibrary.toJSONSchema(), compatibleLibrary.root), []);
  const errors = useRef<OpenUIError[]>([]);
  const lastResolved = useRef("");
  const lastParsed = useRef<{ version: number; result: ParseResult | null } | null>(null);
  const [renderError, setRenderError] = useState("");
  const revision = useRef(0);
  const parsed = useMemo(() => {
    revision.current += 1;
    lastResolved.current = "";
    try { return { result: parser.set(data.program, !data.streaming), error: "" }; }
    catch (error) { return { result: null, error: error instanceof Error ? error.message : String(error) }; }
  }, [parser, data.program, data.streaming]);
  const result = parsed.result;
  const blockedTools = Boolean(result?.queryStatements.length || result?.mutationStatements.length);
  const parseErrors = result?.meta.errors ?? [];
  const completedError = !data.streaming && ((!result?.root && !result?.rootExpression) || result.meta.incomplete || result.meta.unresolved.length > 0);
  const sourceError = parsed.error || (blockedTools ? "Query and Mutation providers are unavailable in this isolated viewer." : "") ||
    parseErrors.map(error => `${error.code}: ${error.message}`).join("\n") ||
    (completedError ? "The OpenUI program is incomplete or has no usable root." : "");

  useEffect(() => {
    if (generation === version && sourceError) report(sourceError, blockedTools ? "unsupported-provider" : "program-error");
  }, [data.program, data.streaming, sourceError, blockedTools, version]);

  const resolveParsed = useCallback((actual: ParseResult | null) => {
    const currentRevision = revision.current;
    const observedErrorSerial = errorSerial;
    if (generation !== version || data.streaming || sourceError || (!actual?.root && !actual?.rootExpression) || actual.meta.incomplete || actual.meta.unresolved.length || actual.meta.errors.length) return;
    requestAnimationFrame(() => {
      if (generation !== version || revision.current !== currentRevision || errorSerial !== observedErrorSerial ||
          hardErrorGeneration === version || errors.current.length || currentProgram !== data.program) return;
      // This confirms parsing and the SDK commit only. Independently loaded
      // resource components retain their own visible loading/failure states.
      if (lastResolved.current !== data.program && byteLength(data.program) <= LIMITS.resolved) {
        if (transport.post("resolved", { program: data.program })) {
          lastResolved.current = data.program;
          lastHostError = "";
          fatal.hidden = true;
        }
      }
    });
  }, [data.program, data.streaming, sourceError, version]);

  const onErrors = useCallback((next: OpenUIError[]) => {
    if (generation !== version) return;
    errors.current = next;
    if (next.length) {
      const message = next.map(error => `${error.code}: ${error.message}`).join("\n");
      setRenderError(message);
      report(message, "sdk-error");
      lastResolved.current = "";
    } else {
      setRenderError("");
      // A state edit can cure an evaluation error without producing a new
      // parse result, so an explicit SDK clearance must reconsider resolution.
      if (lastParsed.current?.version === version) resolveParsed(lastParsed.current.result);
    }
  }, [resolveParsed, version]);

  const onAction = useCallback((event: ActionEvent) => {
    if (data.streaming || generation !== version) return;
    if (event.type !== "continue_conversation") {
      report(`Action ${event.type} is unavailable in this isolated viewer.`, "unsupported-action");
      return;
    }
    const message = event.humanFriendlyMessage;
    if (typeof message !== "string" || !message.trim() || byteLength(message) > LIMITS.message) {
      report("The OpenUI assistant action is empty or exceeds 16 KiB.", "invalid-action"); return;
    }
    transport.post("assistant", { message, ...(event.formState ? { formState: event.formState } : {}),
      ...(event.params?.context !== undefined ? { context: event.params.context } : {}) });
  }, [data.streaming, version]);

  const onParsed = useCallback((actual: ParseResult | null) => {
    if (generation !== version) return;
    lastParsed.current = { version, result: actual };
    resolveParsed(actual);
  }, [resolveParsed, version]);

  return <main>
    <div className="runtime-label">OpenUI · independent compatible components</div>
    {sourceError && <div className="viewer-error" role="alert">{sourceError}</div>}
    {renderError && <div className="viewer-error" role="alert">{renderError}</div>}
    <div hidden={Boolean(!data.streaming && (sourceError || renderError))}>
      {!blockedTools && !parsed.error && <Boundary version={version}>
        <HtmlViewSupportContext.Provider value={data.htmlView}>
        <HostActionsContext.Provider value={data.hostActions}>
        <ProgramRevisionContext.Provider value={version}>
        <Renderer response={data.program} library={compatibleLibrary} isStreaming={data.streaming}
          onAction={onAction}
          onError={onErrors} onParseResult={onParsed} publishObservability={false} />
        </ProgramRevisionContext.Provider>
        </HostActionsContext.Provider>
        </HtmlViewSupportContext.Provider>
      </Boundary>}
    </div>
  </main>;
}

// Unmediated browser clipboard/link behavior remains unavailable. Our owned
// Host-action controls use typed requests and never match these browser paths.
document.addEventListener("click", event => {
  const target = event.target instanceof Element ? event.target : null;
  if (target?.closest(".openui-code-block-copy-button, a[href]")) {
    event.preventDefault();
    event.stopImmediatePropagation();
    report("Unmediated browser clipboard and in-frame links are unavailable. Use the owned Host action controls.", "unsupported-capability");
  }
}, true);
document.addEventListener("submit", event => event.preventDefault(), true);
document.addEventListener("securitypolicyviolation", event => {
  if (event.effectiveDirective === "img-src") report("External images are blocked. This viewer can display embedded data images only.", "image-unavailable");
});
window.addEventListener("error", event => report(event.message, "browser-error"));
window.addEventListener("unhandledrejection", event => report(event.reason instanceof Error ? event.reason.message : String(event.reason), "unhandled-error"));

transport = connect(data => {
  if (currentProgram !== data.program || currentStreaming !== data.streaming) generation += 1;
  currentProgram = data.program;
  currentStreaming = data.streaming;
  updateNativeActionContext({ enabled: data.hostActions, streaming: data.streaming, revision: generation });
  if (!hydrated) {
    try {
      configurePersistence({ initialState: data.state, onState: state => transport.post("state", { state }), onError: message => report(message, "state-error") });
      hydrated = true;
    } catch (error) {
      report(error instanceof Error ? error.message : String(error), "state-error");
      return;
    }
  }
  document.documentElement.style.colorScheme = data.theme.appearance;
  document.documentElement.dataset.appearance = data.theme.appearance;
  root.render(<Viewer data={data} version={generation} />);
}, message => report(message, "bridge-error"));
configureResourceTransport(transport.post);
configureNativeActionTransport(transport.post);

let queuedHeight = false;
let lastHeight = 0;
const observer = new ResizeObserver(() => {
  if (queuedHeight) return;
  queuedHeight = true;
  requestAnimationFrame(() => {
    queuedHeight = false;
    const height = Math.max(80, Math.min(2000, Math.ceil(document.body.getBoundingClientRect().height)));
    if (height !== lastHeight) { lastHeight = height; transport.post("height", { height }); }
  });
});
observer.observe(document.body);
window.addEventListener("pagehide", () => { generation += 1; observer.disconnect(); closeResources(); closeNativeActions(); transport.close(); root.unmount(); }, { once: true });
