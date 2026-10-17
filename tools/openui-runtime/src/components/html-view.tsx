// Independently authored implementation of the public HtmlView parameters.
// Generated HTML runs only in a second opaque-origin sandbox, never in React's
// document. The native OpenUI preview's existing OS sandbox remains required.
import React, { useContext, useLayoutEffect, useMemo, useRef, useState } from "react";
import { defineComponent, useIsStreaming, type DefinedComponent } from "@openuidev/react-lang";
import { z } from "zod/v4";
import { byteLength, LIMITS } from "../bridge";
import { HtmlViewSupportContext } from "../capabilities";

const CHANNEL = "whale-htmlview-local-v1";
const MAX_HEIGHT = 2000; // Existing native OpenUI presentation limit.
const POLICY = "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data:; font-src data:; connect-src 'none'; worker-src 'none'; frame-src 'none'; object-src 'none'; media-src 'none'; base-uri 'none'; form-action 'none'; webrtc 'block'";
const DENIED_FEATURES = "camera 'none'; microphone 'none'; geolocation 'none'; clipboard-read 'none'; clipboard-write 'none'; display-capture 'none'; fullscreen 'none'; autoplay 'none'; payment 'none'; usb 'none'; serial 'none'; bluetooth 'none'; midi 'none'; web-share 'none'; xr-spatial-tracking 'none'; accelerometer 'none'; gyroscope 'none'; magnetometer 'none'";

// This small observer is our code, not part of the caller's HTML. It conveys
// presentation information only. Its output is untrusted and bounded again by
// the parent component; it never participates in the OpenUI resource/action API.
const observerScript = `(() => {
  'use strict';
  const send = parent.postMessage.bind(parent);
  const channel = '${CHANNEL}';
  const report = (type, extra = {}) => send({channel, type, ...extra}, '*');
  let pending = false, previous = -1, resize, mutations;
  const measure = () => {
    pending = false;
    const body = document.body;
    if (!body) return;
    const style = getComputedStyle(body), rect = body.getBoundingClientRect();
    const margin = Number.parseFloat(style.marginBottom) || 0;
    // The body box includes ordinary flow. Its scroll extent includes canvas,
    // positioned content and overflow without making the root viewport itself
    // the minimum height (which would prevent ordinary content from shrinking).
    const height = Math.ceil(Math.max(1, rect.bottom + scrollY + margin,
      rect.top + scrollY + body.scrollHeight + margin));
    if (Number.isFinite(height) && height !== previous) { previous = height; report('height', {height}); }
  };
  const schedule = () => { if (!pending) { pending = true; requestAnimationFrame(measure); } };
  addEventListener('error', () => report('script-error'));
  addEventListener('unhandledrejection', () => report('script-error'));
  addEventListener('securitypolicyviolation', event => {
    const directive = String(event.effectiveDirective || '').split('-')[0];
    if (['script','style','img','font','connect','worker','frame','object','media','base','form','webrtc'].includes(directive)) report('blocked', {directive});
  });
  const start = () => {
    if (!document.body) return;
    if (typeof ResizeObserver === 'function') { resize = new ResizeObserver(schedule); resize.observe(document.body); }
    mutations = new MutationObserver(schedule);
    mutations.observe(document.body, {childList:true, subtree:true, attributes:true, characterData:true});
    addEventListener('resize', schedule);
    addEventListener('load', schedule, true);
    document.fonts?.ready.then(schedule);
    schedule();
    report('ready');
  };
  addEventListener('pagehide', () => { resize?.disconnect(); mutations?.disconnect(); });
  if (document.readyState === 'loading') addEventListener('DOMContentLoaded', start, {once:true}); else start();
})();`;

function documentFor(html: string) {
  // The policy precedes all supplied markup. Later meta policies only restrict
  // it further. No HTML is inserted into the viewer DOM or rewritten as text.
  return `<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="${POLICY}"><meta name="referrer" content="no-referrer"><meta name="viewport" content="width=device-width,initial-scale=1"><script>${observerScript}</script></head>${html}`;
}

function HtmlDocument({ html, height, title }: { html: string; height?: number; title?: string }) {
  const frame = useRef<HTMLIFrameElement>(null);
  const [measuredHeight, setMeasuredHeight] = useState(360);
  const [clipped, setClipped] = useState(false);
  const [ready, setReady] = useState(false);
  const [scriptError, setScriptError] = useState(false);
  const [blocked, setBlocked] = useState<string[]>([]);
  const source = useMemo(() => documentFor(html), [html]);

  useLayoutEffect(() => {
    const receive = (event: MessageEvent) => {
      if (event.source !== frame.current?.contentWindow || event.origin !== "null") return;
      const value = event.data;
      if (!value || typeof value !== "object" || Array.isArray(value) || value.channel !== CHANNEL) return;
      if (value.type === "height" && typeof value.height === "number" && Number.isFinite(value.height) && value.height > 0) {
        setMeasuredHeight(Math.min(MAX_HEIGHT, Math.max(1, Math.ceil(value.height))));
        setClipped(value.height > MAX_HEIGHT);
      } else if (value.type === "ready") setReady(true);
      else if (value.type === "script-error") setScriptError(true);
      else if (value.type === "blocked" && ["script", "style", "img", "font", "connect", "worker", "frame", "object", "media", "base", "form", "webrtc"].includes(value.directive)) {
        setBlocked(previous => previous.includes(value.directive) ? previous : [...previous, value.directive]);
      }
    };
    window.addEventListener("message", receive);
    return () => window.removeEventListener("message", receive);
  }, []);

  return <section style={{ display: "grid", gap: 6, minWidth: 0 }} aria-label={title || "Interactive HTML content"}>
    <iframe ref={frame} srcDoc={source} title={title || "Interactive HTML content"}
      sandbox="allow-scripts" referrerPolicy="no-referrer" allow={DENIED_FEATURES}
      style={{ display: "block", width: "100%", height: height ?? measuredHeight, border: "1px solid var(--oui-border, #d9e0e8)", borderRadius: 8, background: "#fff", colorScheme: "light" }} />
    <div role="note" style={{ fontSize: 12, color: "var(--oui-muted, #526379)" }}>Local HTML/CSS/JavaScript. External scripts, network, media, navigation and native actions are unavailable.</div>
    {!ready && <div role="status">Loading isolated HTML content…</div>}
    {height == null && clipped && <div role="status">Content exceeds the existing 2,000px preview height; scroll inside the HTML view.</div>}
    {scriptError && <div role="alert">The embedded HTML reported a JavaScript error.</div>}
    {blocked.length > 0 && <div role="status">The embedded HTML requested capabilities blocked by its policy: {blocked.join(", ")}.</div>}
  </section>;
}

const HtmlView = defineComponent({
  name: "HtmlView",
  props: z.object({ html: z.string(), height: z.number().optional(), title: z.string().optional() }),
  description: "An isolated local HTML/CSS/inline JavaScript page for games and canvas. Content auto-sizes unless height is set. No CDN, network, media, host actions or persistent state.",
  component: ({ props }) => {
    const streaming = useIsStreaming();
    const supported = useContext(HtmlViewSupportContext);
    if (props.height != null && (!Number.isFinite(props.height) || props.height < 1 || props.height > MAX_HEIGHT)) return <div role="alert">HtmlView height must be between 1 and 2,000 pixels, within the existing preview limit.</div>;
    if (byteLength(props.html) > LIMITS.program) return <div role="alert">HtmlView content exceeds the existing 256 KiB OpenUI program budget.</div>;
    if (streaming) return <div role="status">Waiting for the complete HTML content before running it…</div>;
    if (!supported) return <div role="alert">HtmlView is unavailable: this preview host has not enabled isolated nested HTML support. Other OpenUI components remain available.</div>;
    // New HTML means a new opaque document and a new contentWindow. Same HTML
    // survives unrelated React/state updates. Streaming and removal destroy it.
    return <HtmlDocument key={props.html} html={props.html} height={props.height} title={props.title} />;
  },
});

export const htmlComponents: DefinedComponent<any>[] = [HtmlView];
