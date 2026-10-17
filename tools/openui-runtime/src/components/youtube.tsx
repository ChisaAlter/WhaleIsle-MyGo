// Independently authored native-media placeholder. This document never embeds
// YouTube, fetches media, or navigates to a video URL.
import React, { useContext, useId, useLayoutEffect, useRef, useState, type CSSProperties } from "react";
import { defineComponent, useIsStreaming, type DefinedComponent } from "@openuidev/react-lang";
import { z } from "zod/v4";
import { MediaContext } from "../capabilities";
import { createYouTubeSeat, type MediaGeometry, type MediaStatus } from "../media";
import { parseYouTubeVideoId } from "../youtube-id";

type Rect = MediaGeometry["rect"];
type Viewport = MediaGeometry["viewport"];
type Seat = ReturnType<typeof createYouTubeSeat>;
type Measurement = { geometry: MediaGeometry; notice: string };
type Styles = Map<Element, CSSStyleDeclaration>;

const MAX_SIZE = 4096;
const MAX_COORDINATE = 1_000_000;
const EMPTY_RECT: Rect = { x: 0, y: 0, width: 0, height: 0 };
const INITIAL_STATUS: MediaStatus = { phase: "idle", message: "" };
const overlaySelector = '.oui-popover, [data-oui-popover-owner], .oui-celebration, [data-oui-overlay], dialog[open], [role="dialog"], [role="alertdialog"], [aria-modal="true"]';
const clipOverflow = new Set(["hidden", "clip", "scroll", "auto", "overlay"]);
const ink = "var(--oui-ink, #172337)";
const muted = "var(--oui-muted, #526379)";
const border = "var(--oui-border, #d9e0e8)";
const buttonStyle: CSSProperties = { marginTop: 8, padding: "7px 11px", border: `1px solid ${border}`, borderRadius: 7, background: "var(--oui-panel, #fff)", color: ink, font: "inherit" };

function styleOf(element: Element, styles: Styles): CSSStyleDeclaration {
  let style = styles.get(element);
  if (!style) { style = window.getComputedStyle(element); styles.set(element, style); }
  return style;
}

function rectOf(element: Element): Rect {
  const rect = element.getBoundingClientRect();
  return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
}

function finiteRect(rect: Rect): boolean {
  return [rect.x, rect.y, rect.width, rect.height].every(Number.isFinite) && rect.width >= 0 && rect.height >= 0;
}

function intersect(first: Rect, second: Rect, horizontal = true, vertical = true): Rect {
  const x = horizontal ? Math.max(first.x, second.x) : first.x;
  const y = vertical ? Math.max(first.y, second.y) : first.y;
  const right = horizontal ? Math.min(first.x + first.width, second.x + second.width) : first.x + first.width;
  const bottom = vertical ? Math.min(first.y + first.height, second.y + second.height) : first.y + first.height;
  return { x, y, width: Math.max(0, right - x), height: Math.max(0, bottom - y) };
}

// Axis-aligned scale and translation are representable by a native rectangle.
// A rotated, mirrored, or perspective-transformed area is not; showing its
// bounding box as a native rectangle could cover unrelated page content.
function rectangularTransform(style: CSSStyleDeclaration): boolean {
  if (style.transform !== "none") {
    try {
      const matrix = new DOMMatrixReadOnly(style.transform);
      if (!matrix.is2D || matrix.b !== 0 || matrix.c !== 0 || matrix.a <= 0 || matrix.d <= 0) return false;
    } catch { return false; }
  }
  if (style.rotate && style.rotate !== "none" && !/^(?:[xyz]\s+)?0(?:deg|rad|grad|turn)?$/.test(style.rotate)) return false;
  if (style.scale && style.scale !== "none" && style.scale.split(/\s+/).some(value => !Number.isFinite(Number(value)) || Number(value) <= 0)) return false;
  return true;
}

function paintedBox(element: Element, viewport: Viewport, styles: Styles, nativeSeat = false): { rect: Rect; clip: Rect; visible: boolean; reason: string } {
  const rect = rectOf(element);
  let clip = intersect(rect, { x: 0, y: 0, ...viewport });
  if (!element.isConnected || !finiteRect(rect) || rect.width <= 0 || rect.height <= 0) return { rect, clip, visible: false, reason: "The video area has no visible size." };
  for (let ancestor: Element | null = element; ancestor; ancestor = ancestor.parentElement) {
    const style = styleOf(ancestor, styles);
    const opacity = Number.parseFloat(style.opacity);
    if (style.display === "none" || style.visibility === "hidden" || style.visibility === "collapse" || style.contentVisibility === "hidden" || opacity <= 0) {
      return { rect, clip, visible: false, reason: "The video area or one of its containers is hidden." };
    }
    if (nativeSeat && (ancestor.hasAttribute("inert") || ancestor.getAttribute("aria-hidden") === "true")) {
      return { rect, clip, visible: false, reason: "The video area is in an inactive container." };
    }
    if (nativeSeat && ((Number.isFinite(opacity) && opacity < 1) || !rectangularTransform(style))) {
      return { rect, clip, visible: false, reason: "The native video cannot safely follow this container's transparency or non-rectangular transform." };
    }
    // CSS clip paths and masks cannot be expressed by the rectangular bridge.
    if (nativeSeat && ((style.clipPath && style.clipPath !== "none") || (style.maskImage && style.maskImage !== "none"))) {
      return { rect, clip, visible: false, reason: "The native video cannot safely follow this container's non-rectangular clipping." };
    }
    if (ancestor === element) continue;
    const clipX = clipOverflow.has(style.overflowX);
    const clipY = clipOverflow.has(style.overflowY);
    if (!clipX && !clipY) continue;
    const bounds = rectOf(ancestor);
    if (!finiteRect(bounds)) return { rect, clip, visible: false, reason: "A video container has invalid geometry." };
    // Overflow clips at the padding box. client* excludes borders and scrollbars;
    // scale factors keep these measurements in getBoundingClientRect CSS units.
    let paddingBox = bounds;
    if (ancestor instanceof HTMLElement && ancestor.offsetWidth > 0 && ancestor.offsetHeight > 0) {
      const scaleX = bounds.width / ancestor.offsetWidth;
      const scaleY = bounds.height / ancestor.offsetHeight;
      paddingBox = { x: bounds.x + ancestor.clientLeft * scaleX, y: bounds.y + ancestor.clientTop * scaleY, width: ancestor.clientWidth * scaleX, height: ancestor.clientHeight * scaleY };
    }
    clip = intersect(clip, paddingBox, clipX, clipY);
  }
  return { rect, clip, visible: clip.width > 0 && clip.height > 0, reason: "The video area is outside its visible viewport or scroll container." };
}

function hasCoveringOverlay(element: HTMLElement, clip: Rect, viewport: Viewport, styles: Styles): boolean {
  // This is only the owned outer viewer DOM. HtmlView documents live in their
  // separate frames and cannot contribute nodes here. In addition to the known
  // Popover/Celebration portals, inspect computed fixed/sticky layers so a future
  // owned overlay does not silently end up beneath the native media surface.
  for (const candidate of document.body.querySelectorAll<Element>("*")) {
    if (candidate === element || element.contains(candidate) || candidate.contains(element)) continue;
    const style = styleOf(candidate, styles);
    if (!candidate.matches(overlaySelector) && style.position !== "fixed" && style.position !== "sticky") continue;
    const box = paintedBox(candidate, viewport, styles);
    if (!box.visible) continue;
    const overlap = intersect(clip, box.clip);
    if (overlap.width > 0 && overlap.height > 0) return true;
  }
  return false;
}

function measure(element: HTMLDivElement): Measurement {
  const width = window.innerWidth;
  const height = window.innerHeight;
  const viewport = {
    width: Number.isFinite(width) ? Math.max(1, Math.min(MAX_SIZE, width)) : 1,
    height: Number.isFinite(height) ? Math.max(1, Math.min(MAX_SIZE, height)) : 1,
  };
  const empty = (notice: string): Measurement => ({ geometry: { rect: { ...EMPTY_RECT }, clip: { ...EMPTY_RECT }, viewport, visible: false }, notice });
  if (!Number.isFinite(width) || !Number.isFinite(height) || width < 1 || height < 1 || width > MAX_SIZE || height > MAX_SIZE) {
    return empty("The viewer is requesting that the native video stay hidden: its viewport must be between 1 and 4096 CSS pixels on each axis.");
  }
  const styles: Styles = new Map();
  const box = paintedBox(element, viewport, styles, true);
  if (!finiteRect(box.rect) || Math.abs(box.rect.x) > MAX_COORDINATE || Math.abs(box.rect.y) > MAX_COORDINATE || box.rect.width > MAX_SIZE || box.rect.height > MAX_SIZE) {
    return empty("The viewer is requesting that the native video stay hidden because its position or size is outside the supported geometry bounds.");
  }
  if (!box.visible) return empty(`The viewer is requesting that the native video stay hidden. ${box.reason}`);
  if (document.hidden) return { geometry: { rect: box.rect, clip: box.clip, viewport, visible: false }, notice: "The viewer is requesting that the native video stay hidden while this document is hidden." };
  if (hasCoveringOverlay(element, box.clip, viewport, styles)) {
    return { geometry: { rect: box.rect, clip: box.clip, viewport, visible: false }, notice: "The viewer is requesting that the native video stay hidden while an OpenUI overlay overlaps its area." };
  }
  return { geometry: { rect: box.rect, clip: box.clip, viewport, visible: true }, notice: "" };
}

function hostStatus(status: MediaStatus): string {
  const labels: Record<MediaStatus["phase"], string> = {
    idle: "YouTube has not been loaded.",
    pending: "The request for a native YouTube seat is pending.",
    created: "The Host reports that the native YouTube seat was created. This does not confirm playback.",
    hidden: "The Host reports that the native YouTube seat is hidden. Playback state is not available.",
    closed: "The Host reports that the native YouTube seat is closed.",
    error: "The native YouTube seat reported an error.",
  };
  return `${labels[status.phase]}${status.message ? ` ${status.message}` : ""}`;
}

function VideoArea({ areaRef }: { areaRef?: React.Ref<HTMLDivElement> }) {
  return <div ref={areaRef} className="oui-youtube-seat" role="region" aria-label="Reserved native YouTube video area" style={{ position: "relative", width: "100%", maxWidth: MAX_SIZE, aspectRatio: "16 / 9", minWidth: 0, overflow: "hidden", boxSizing: "border-box", border: `1px solid ${border}`, background: "var(--oui-surface, #f3f6fa)" }}>
    <span aria-hidden="true" style={{ position: "absolute", inset: 0, display: "grid", placeItems: "center", padding: 12, color: muted, textAlign: "center", fontSize: 13 }}>Native YouTube video area</span>
  </div>;
}

function UnavailableVideo({ title, message, error = false }: { title: string; message: string; error?: boolean }) {
  return <figure className="oui-youtube" style={{ margin: 0, minWidth: 0 }}>
    <VideoArea />
    <figcaption style={{ marginTop: 8, color: ink, overflowWrap: "anywhere" }}>{title || "YouTube video"}</figcaption>
    <div role={error ? "alert" : "status"} style={{ marginTop: 6, color: error ? "var(--oui-danger, #b43245)" : muted, fontSize: 13, lineHeight: 1.5 }}>{message}</div>
  </figure>;
}

function NativeVideo({ videoId, title, revision }: { videoId: string; title: string; revision: number }) {
  const area = useRef<HTMLDivElement>(null);
  const seat = useRef<Seat | null>(null);
  const live = useRef(false);
  const epoch = useRef(0);
  const active = useRef(false);
  const frame = useRef<number | null>(null);
  const [status, setStatus] = useState<MediaStatus>(INITIAL_STATUS);
  const [requested, setRequested] = useState(false);
  const [notice, setNotice] = useState("");
  const statusId = useId();

  const stop = () => {
    active.current = false;
    if (frame.current !== null) cancelAnimationFrame(frame.current);
    frame.current = null;
  };
  const sync = (handle = seat.current) => {
    if (!live.current || !handle || !area.current) return;
    const measured = measure(area.current);
    handle.setGeometry(measured.geometry);
    setNotice(previous => previous === measured.notice ? previous : measured.notice);
  };
  const start = () => {
    if (active.current) return;
    active.current = true;
    const tick = () => {
      frame.current = null;
      if (!live.current || !active.current) return;
      sync();
      if (active.current) frame.current = requestAnimationFrame(tick);
    };
    frame.current = requestAnimationFrame(tick);
  };
  const retire = () => {
    stop();
    epoch.current += 1;
    const previous = seat.current;
    seat.current = null;
    previous?.remove();
  };

  useLayoutEffect(() => {
    live.current = true;
    const onVisibility = () => { if (active.current) sync(); };
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      live.current = false;
      document.removeEventListener("visibilitychange", onVisibility);
      // Layout cleanup retires the handle even when creation is still pending.
      // Its epoch also prevents late Host replies from changing the next seat.
      retire();
    };
  }, []);

  const load = (event: React.MouseEvent<HTMLButtonElement>) => {
    if (!live.current || active.current || requested || status.phase === "pending" || status.phase === "created" || status.phase === "hidden") return;
    try {
      retire();
      const ownEpoch = epoch.current;
      const handle = createYouTubeSeat({ videoId, title, revision, onStatus: next => {
        if (!live.current || ownEpoch !== epoch.current) return;
        setStatus(next);
        if (next.phase === "error" || next.phase === "closed") {
          stop();
          setRequested(false);
          setNotice("");
        }
      } });
      seat.current = handle;
      setRequested(true);
      // Both operations stay in this original trusted click stack. The transport
      // owns the isTrusted/userActivation check; no gesture is synthesized here.
      sync(handle);
      start();
      handle.activate(event);
    } catch (error) {
      retire();
      setRequested(false);
      setNotice("");
      setStatus({ phase: "error", message: error instanceof Error ? error.message : "The native YouTube request could not be prepared." });
    }
  };

  const busy = requested || status.phase === "pending" || status.phase === "created" || status.phase === "hidden";
  return <figure className="oui-youtube" style={{ margin: 0, minWidth: 0 }}>
    <VideoArea areaRef={area} />
    <figcaption style={{ marginTop: 8, color: ink, overflowWrap: "anywhere" }}>{title || "YouTube video"}</figcaption>
    <button type="button" onClick={load} disabled={busy} aria-describedby={statusId} style={{ ...buttonStyle, cursor: busy ? "default" : "pointer", opacity: busy ? 0.65 : 1 }}>Load YouTube video</button>
    <div id={statusId} role={status.phase === "error" ? "alert" : "status"} aria-live={status.phase === "error" ? "assertive" : "polite"} aria-atomic="true" style={{ marginTop: 6, color: status.phase === "error" ? "var(--oui-danger, #b43245)" : muted, fontSize: 13, lineHeight: 1.5, overflowWrap: "anywhere" }}>
      {hostStatus(status)}
    </div>
    {notice && <div role="note" style={{ marginTop: 6, color: muted, fontSize: 12, lineHeight: 1.5 }}>{notice}</div>}
  </figure>;
}

const YouTube = defineComponent({
  name: "YouTube",
  props: z.object({ video: z.string(), title: z.string().optional() }),
  description: "An explicitly loaded, Host-owned native YouTube seat. The isolated viewer itself never embeds or fetches video; Host creation status does not confirm playback.",
  component: ({ props }) => {
    const support = useContext(MediaContext);
    const streaming = useIsStreaming();
    const title = props.title ?? "YouTube video";
    if (title.length > 512 || new TextEncoder().encode(title).byteLength > 512) {
      return <UnavailableVideo title="YouTube video" error message="YouTube title exceeds the 512-byte UTF-8 limit. Supply a shorter title; this title has not been sent to the Host." />;
    }
    if (streaming) return <UnavailableVideo title={title} message="Waiting for the complete OpenUI response before YouTube can be loaded." />;
    const parsed = parseYouTubeVideoId(props.video);
    if (parsed.videoId === undefined) return <UnavailableVideo title={title} error message={parsed.error} />;
    if (!support.enabled) return <UnavailableVideo title={title} message="YouTube is unavailable: this Host has not enabled native YouTube media." />;
    // Changing identity, caption, or trusted Host revision unmounts the old seat
    // before the replacement can be activated, including in-flight requests.
    return <NativeVideo key={JSON.stringify([parsed.videoId, title, support.revision])} videoId={parsed.videoId} title={title} revision={support.revision} />;
  },
});

export const mediaComponents: DefinedComponent<any>[] = [YouTube];
