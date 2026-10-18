// Independently authored raster-map renderer. Geographic data and imagery come
// only from the fixed resource routes in the parent; this frame has no network.
import React, { useEffect, useId, useMemo, useRef, useState, type CSSProperties } from "react";
import { defineComponent, reactive, useIsStreaming, useStateField, type DefinedComponent } from "@openuidev/react-lang";
import { z } from "zod/v4";
import { requestResource } from "../resources";

const TILE_SIZE = 256;
const MAX_LATITUDE = 85.0511287798066;
const MIN_ZOOM = 0;
const MAX_ZOOM = 19;
const CACHE_BYTE_BUDGET = 32 * 1024 * 1024;
const tileCache = new Map<string, TileData>();
let tileCacheBytes = 0;

type View = { x: number; y: number; zoom: number };
type Size = { width: number; height: number };
type Point = { x: number; y: number };
type Tile = { key: string; x: number; y: number; zoom: number; left: number; top: number; size: number };
type Gesture = { ids: number[]; anchor: Point; center: Point; zoom: number; distance: number | null };
type TileData = { dataUrl: string; attribution: string; source: string };
type TileState = { data?: TileData; error?: string };
type RouteState = { key: string; points?: [number, number][]; error?: string; loading?: boolean };
type TileResponse = TileData & { z: number; x: number; y: number; style: string };
type RouteResponse = { points: unknown };

const markerSchema = z.object({
  label: z.string(),
  lat: z.number(),
  lng: z.number(),
  category: z.string().optional(),
  icon: z.string().optional(),
});
type Marker = z.infer<typeof markerSchema>;
type LocatedMarker = Marker & { index: number };

const MapMarker = defineComponent({
  name: "MapMarker",
  description: "A labeled map location; icon is an emoji, otherwise its position number is shown.",
  props: markerSchema,
  component: () => <span role="status">MapMarker must be placed inside a Map.</span>,
});

function wrap(value: number) { return ((value % 1) + 1) % 1; }
function nearestDelta(value: number) { return wrap(value + 0.5) - 0.5; }
function clamp(value: number, min: number, max: number) { return Math.min(max, Math.max(min, value)); }
function project(lat: number, lng: number): Point {
  const sinLatitude = Math.sin(clamp(lat, -MAX_LATITUDE, MAX_LATITUDE) * Math.PI / 180);
  return { x: (lng + 180) / 360, y: 0.5 - Math.log((1 + sinLatitude) / (1 - sinLatitude)) / (4 * Math.PI) };
}
function validLocation(lat: unknown, lng: unknown): boolean {
  return typeof lat === "number" && Number.isFinite(lat) && Math.abs(lat) <= MAX_LATITUDE
    && typeof lng === "number" && Number.isFinite(lng) && Math.abs(lng) <= 180;
}
function fitMarkers(markers: LocatedMarker[], size: Size): View {
  const points = markers.map(marker => project(marker.lat, marker.lng));
  if (!points.length) return { x: 0.5, y: 0.5, zoom: 0 };
  const xs = points.map(point => wrap(point.x)).sort((a, b) => a - b);
  let largestGap = -1;
  let start = xs[0];
  for (let index = 0; index < xs.length; index++) {
    const next = index + 1 < xs.length ? xs[index + 1] : xs[0] + 1;
    if (next - xs[index] > largestGap) {
      largestGap = next - xs[index];
      start = wrap(next);
    }
  }
  const unwrapped = xs.map(x => x < start ? x + 1 : x);
  let minX = Infinity;
  let maxX = -Infinity;
  let minY = Infinity;
  let maxY = -Infinity;
  for (const x of unwrapped) { minX = Math.min(minX, x); maxX = Math.max(maxX, x); }
  for (const point of points) { minY = Math.min(minY, point.y); maxY = Math.max(maxY, point.y); }
  const horizontalZoom = maxX === minX ? Infinity : Math.log2(Math.max(64, size.width - 96) / (TILE_SIZE * (maxX - minX)));
  const verticalZoom = maxY === minY ? Infinity : Math.log2(Math.max(64, size.height - 112) / (TILE_SIZE * (maxY - minY)));
  const zoom = clamp(Math.floor(Math.min(14, horizontalZoom, verticalZoom)), MIN_ZOOM, MAX_ZOOM);
  return { x: wrap((minX + maxX) / 2), y: clamp((minY + maxY) / 2, 0, 1), zoom };
}
function visibleTiles(view: View, size: Size, appearance: string, rasterZoom?: number | null): Tile[] {
  // Fractional view zoom scales authentic tiles; z/x/y sent to the parent stay
  // integers. Reuse the gesture's raster level while zooming in; step down while
  // zooming out so a wide view never enumerates millions of tiny high-z tiles.
  const zoom = clamp(Math.min(Math.floor(rasterZoom ?? view.zoom), Math.floor(view.zoom)), MIN_ZOOM, MAX_ZOOM);
  const count = 2 ** zoom;
  const displaySize = TILE_SIZE * 2 ** (view.zoom - zoom);
  const world = displaySize * count;
  const originX = view.x * world - size.width / 2;
  const originY = view.y * world - size.height / 2;
  const tiles: Tile[] = [];
  const minX = Math.floor(originX / displaySize);
  const maxX = Math.ceil((originX + size.width) / displaySize) - 1;
  const minY = Math.max(0, Math.floor(originY / displaySize));
  const maxY = Math.min(count - 1, Math.ceil((originY + size.height) / displaySize) - 1);
  for (let y = minY; y <= maxY; y++) {
    for (let x = minX; x <= maxX; x++) {
      const wrappedX = ((x % count) + count) % count;
      tiles.push({ key: `${appearance}/${zoom}/${wrappedX}/${y}`, x: wrappedX, y, zoom, left: x * displaySize - originX, top: y * displaySize - originY, size: displaySize });
    }
  }
  return tiles;
}
function cacheGet(key: string): TileData | undefined {
  const value = tileCache.get(key);
  if (value) { tileCache.delete(key); tileCache.set(key, value); }
  return value;
}
function cacheDelete(key: string) {
  const value = tileCache.get(key);
  if (value) { tileCacheBytes -= value.dataUrl.length * 2; tileCache.delete(key); }
}
function cachePut(key: string, value: TileData) {
  cacheDelete(key);
  const bytes = value.dataUrl.length * 2;
  // Evicted tiles are requested again when visible; no locations are discarded.
  if (bytes > CACHE_BYTE_BUDGET) return;
  while (tileCacheBytes + bytes > CACHE_BYTE_BUDGET && tileCache.size) cacheDelete(tileCache.keys().next().value!);
  tileCache.set(key, value);
  tileCacheBytes += bytes;
}
function validateTile(body: TileResponse, tile: Tile, appearance: string): TileData {
  if (!body || body.z !== tile.zoom || body.x !== tile.x || body.y !== tile.y || body.style !== appearance) throw new Error("Map tile response does not match its requested coordinates or style.");
  if (typeof body.dataUrl !== "string" || !/^data:image\/(?:png|jpeg|webp);base64,[A-Za-z0-9+/]+={0,2}$/.test(body.dataUrl)) throw new Error("Map tile response is not an embedded raster image.");
  if (typeof body.attribution !== "string" || !body.attribution.trim() || typeof body.source !== "string" || !body.source.trim()) throw new Error("Map tile response is missing its source attribution.");
  return { dataUrl: body.dataUrl, attribution: body.attribution, source: body.source };
}
function useAppearance() {
  const read = () => typeof document !== "undefined" && document.documentElement.dataset.appearance === "dark" ? "dark" : "light";
  const [appearance, setAppearance] = useState(read);
  useEffect(() => {
    const observer = new MutationObserver(() => setAppearance(read()));
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["data-appearance"] });
    return () => observer.disconnect();
  }, []);
  return appearance;
}
function useTiles(tiles: Tile[], appearance: string, dragging: boolean, retry: number) {
  const [states, setStates] = useState<Record<string, TileState>>({});
  const unique = Array.from(new Map(tiles.map(tile => [tile.key, tile])).values());
  const key = unique.map(tile => tile.key).sort().join("|");
  useEffect(() => {
    const controller = new AbortController();
    const current: Record<string, TileState> = {};
    const missing: Tile[] = [];
    for (const tile of unique) {
      const data = cacheGet(tile.key);
      if (data) current[tile.key] = { data };
      else missing.push(tile);
    }
    setStates(current);
    if (dragging || !missing.length) return () => controller.abort();
    let next = 0;
    // The small request pool bounds concurrent resource use, not map size.
    async function load() {
      while (!controller.signal.aborted && next < missing.length) {
        const tile = missing[next++];
        try {
          const body = await requestResource<TileResponse>("/api/map-tile", { z: String(tile.zoom), x: String(tile.x), y: String(tile.y), style: appearance }, controller.signal);
          const data = validateTile(body, tile, appearance);
          if (controller.signal.aborted) return;
          cachePut(tile.key, data);
          setStates(previous => ({ ...previous, [tile.key]: { data } }));
        } catch (error) {
          if (controller.signal.aborted) return;
          setStates(previous => ({ ...previous, [tile.key]: { error: error instanceof Error ? error.message : "Map tile request failed." } }));
        }
      }
    }
    const timer = setTimeout(() => { for (let index = 0; index < Math.min(4, missing.length); index++) void load(); }, 120);
    return () => { clearTimeout(timer); controller.abort(); };
  }, [key, appearance, dragging, retry]);
  function failImage(tileKey: string) {
    cacheDelete(tileKey);
    setStates(previous => ({ ...previous, [tileKey]: { error: "Map tile image could not be decoded." } }));
  }
  return { states, failImage };
}

const controlStyle: CSSProperties = {
  border: "1px solid #9ca3af", borderRadius: 5, padding: "5px 9px", background: "#fff", color: "#111827",
  font: "inherit", fontSize: 14, minHeight: 32, cursor: "pointer",
};

const MapComponent = defineComponent({
  name: "Map",
  description: "A real raster street map fitted to its markers, with optional road route and bound selection.",
  props: z.object({
    markers: z.array(MapMarker.ref),
    route: z.boolean().optional(),
    selected: reactive(z.string()).optional(),
    height: z.number().optional(),
  }),
  component: ({ props, statementId }) => {
    const id = useId();
    const selected = useStateField(statementId ?? `map-${id}`, props.selected);
    const streaming = useIsStreaming();
    const appearance = useAppearance();
    const container = useRef<HTMLDivElement>(null);
    const [size, setSize] = useState<Size>({ width: 0, height: 0 });
    const [view, setView] = useState<View | null>(null);
    const [dragging, setDragging] = useState(false);
    const [wheeling, setWheeling] = useState(false);
    const [rasterZoom, setRasterZoom] = useState<number | null>(null);
    const [retry, setRetry] = useState(0);
    const pointers = useRef(new Map<number, Point>());
    const gesture = useRef<Gesture | null>(null);
    const gestureMoved = useRef(false);
    const suppressClickUntil = useRef(0);
    const wheelActive = useRef(false);
    const wheelTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
    const routeRequest = useRef<AbortController | null>(null);
    const [routeState, setRouteState] = useState<RouteState>({ key: "" });
    const height = typeof props.height === "number" && Number.isFinite(props.height) && props.height > 0 ? props.height : 360;
    const markers: LocatedMarker[] = [];
    let invalidMarkers = 0;
    for (const [index, node] of (props.markers ?? []).entries()) {
      if (node?.typeName === "MapMarker" && node.props && typeof node.props.label === "string" && validLocation(node.props.lat, node.props.lng)) markers.push({ ...node.props, index });
      else invalidMarkers++;
    }
    const markerKey = JSON.stringify(markers.map(marker => [marker.label, marker.lat, marker.lng]));
    const coords = markers.map(marker => `${marker.lng},${marker.lat}`).join(";");
    const routeKey = props.route ? coords : "";
    const fitted = useMemo(() => fitMarkers(markers, size), [markerKey, size.width, size.height]);
    const currentView = view ?? fitted;
    const viewRef = useRef(currentView);
    viewRef.current = currentView;
    const interaction = useRef({ size, streaming, hasMarkers: markers.length > 0, fitted });
    interaction.current = { size, streaming, hasMarkers: markers.length > 0, fitted };
    const world = TILE_SIZE * 2 ** currentView.zoom;
    const tiles = size.width > 0 && size.height > 0 && markers.length > 0 ? visibleTiles(currentView, size, appearance, rasterZoom) : [];
    const { states: tileStates, failImage } = useTiles(tiles, appearance, dragging || wheeling, retry);
    const uniqueTiles = Array.from(new Set(tiles.map(tile => tile.key)));
    const failures = uniqueTiles.filter(key => tileStates[key]?.error);
    const loaded = uniqueTiles.filter(key => tileStates[key]?.data);
    const loading = uniqueTiles.length > loaded.length + failures.length;
    const credits = Array.from(new Set(loaded.map(key => tileStates[key]!.data!.attribution)));
    const sources = Array.from(new Set(loaded.map(key => tileStates[key]!.data!.source)));
    const selectedMarker = markers.find(marker => marker.label === selected.value);
    const routeUnavailable = props.route && (invalidMarkers > 0 || markers.length < 2 || markers.length > 25)
      ? invalidMarkers > 0 ? "Road routing requires valid coordinates for every marker." : "The road-routing service requires between 2 and 25 markers."
      : undefined;

    useEffect(() => {
      const element = container.current;
      if (!element) return;
      const update = () => setSize(previous => element.clientWidth === previous.width && element.clientHeight === previous.height ? previous : { width: element.clientWidth, height: element.clientHeight });
      update();
      const observer = new ResizeObserver(update);
      observer.observe(element);
      return () => observer.disconnect();
    }, []);
    useEffect(() => {
      cancelInteractions();
      viewRef.current = interaction.current.fitted;
      setView(null);
    }, [markerKey]);
    useEffect(() => { if (streaming) cancelInteractions(); }, [streaming]);
    useEffect(() => { if (pointers.current.size) rebaseGesture(); }, [size.width, size.height]);
    useEffect(() => {
      const element = container.current;
      if (!element) return;
      const wheel = (event: WheelEvent) => {
        const context = interaction.current;
        // Ordinary scrolling belongs to the page. Modified scrolling (including
        // browser trackpad pinch events exposed as Ctrl+wheel) belongs to the map.
        if ((!event.ctrlKey && !event.metaKey) || context.streaming || !context.hasMarkers || !event.cancelable) return;
        event.preventDefault();
        event.stopPropagation();
        if (pointers.current.size || !Number.isFinite(event.deltaY) || event.deltaY === 0) return;
        const multiplier = event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? context.size.height : 1;
        const delta = clamp(event.deltaY * multiplier, -240, 240);
        const current = viewRef.current;
        const zoom = clamp(current.zoom - delta / 300, MIN_ZOOM, MAX_ZOOM);
        if (zoom === current.zoom) return;
        if (!wheelActive.current) {
          wheelActive.current = true;
          setWheeling(true);
          setRasterZoom(Math.floor(current.zoom));
        }
        const position = localPoint({ x: event.clientX, y: event.clientY });
        applyView(anchoredView(worldPoint(current, position), position, zoom));
        clearTimeout(wheelTimer.current);
        wheelTimer.current = setTimeout(() => {
          wheelActive.current = false;
          setWheeling(false);
          if (!pointers.current.size) setRasterZoom(null);
        }, 180);
      };
      const blur = () => cancelInteractions();
      const visibility = () => { if (document.hidden) cancelInteractions(); };
      element.addEventListener("wheel", wheel, { passive: false });
      window.addEventListener("blur", blur);
      document.addEventListener("visibilitychange", visibility);
      return () => {
        element.removeEventListener("wheel", wheel);
        window.removeEventListener("blur", blur);
        document.removeEventListener("visibilitychange", visibility);
        cancelInteractions(false);
      };
    }, []);
    useEffect(() => {
      routeRequest.current?.abort();
      if (!props.route || routeUnavailable || streaming || !routeKey) {
        setRouteState({ key: routeKey });
        return;
      }
      const controller = new AbortController();
      routeRequest.current = controller;
      setRouteState({ key: routeKey, loading: true });
      void requestResource<RouteResponse>("/api/street-route", { coords }, controller.signal).then(body => {
        if (controller.signal.aborted) return;
        if (!body || !Array.isArray(body.points) || body.points.length < 2 || !body.points.every(point => Array.isArray(point) && point.length === 2 && validLocation(point[0], point[1]))) throw new Error("The road-routing service returned no valid geographic path.");
        setRouteState({ key: routeKey, points: body.points as [number, number][] });
      }).catch(error => {
        if (!controller.signal.aborted) setRouteState({ key: routeKey, error: error instanceof Error ? error.message : "Road route request failed." });
      });
      return () => controller.abort();
    }, [routeKey, routeUnavailable, streaming, retry]);

    const routePoints = routeState.key === routeKey ? routeState.points : undefined;
    const routePath = useMemo(() => {
      let lastX: number | undefined;
      return (routePoints ?? []).map(([lat, lng]) => {
        const point = project(lat, lng);
        const x = lastX == null ? currentView.x + nearestDelta(point.x - currentView.x) : lastX + nearestDelta(point.x - lastX);
        lastX = x;
        return `${(x - currentView.x) * world + size.width / 2},${(point.y - currentView.y) * world + size.height / 2}`;
      }).join(" ");
    }, [routePoints, currentView.x, currentView.y, world, size.width, size.height]);
    const copies = Math.ceil(size.width / world) + 1;
    const copyOffsets = Array.from({ length: copies * 2 + 1 }, (_, index) => (index - copies) * world);

    function applyView(next: View) {
      // Event bursts can precede React's next render. Keep the next event anchored
      // to the view just applied, rather than to the previous rendered frame.
      viewRef.current = next;
      setView(next);
    }
    function localPoint(client: Point): Point {
      const element = container.current;
      if (!element) return { x: 0, y: 0 };
      const bounds = element.getBoundingClientRect();
      return { x: client.x - bounds.left - element.clientLeft, y: client.y - bounds.top - element.clientTop };
    }
    function worldPoint(current: View, position: Point): Point {
      const viewport = interaction.current.size;
      const scale = TILE_SIZE * 2 ** current.zoom;
      return { x: current.x + (position.x - viewport.width / 2) / scale, y: current.y + (position.y - viewport.height / 2) / scale };
    }
    function anchoredView(anchor: Point, position: Point, zoom: number): View {
      const viewport = interaction.current.size;
      const scale = TILE_SIZE * 2 ** zoom;
      return { x: wrap(anchor.x - (position.x - viewport.width / 2) / scale), y: clamp(anchor.y - (position.y - viewport.height / 2) / scale, 0, 1), zoom };
    }
    function gesturePosition(ids: number[]) {
      const first = localPoint(pointers.current.get(ids[0])!);
      if (ids.length === 1) return { center: first, distance: null };
      const second = localPoint(pointers.current.get(ids[1])!);
      return { center: { x: (first.x + second.x) / 2, y: (first.y + second.y) / 2 }, distance: Math.hypot(second.x - first.x, second.y - first.y) };
    }
    function rebaseGesture() {
      const ids = [...pointers.current.keys()].slice(0, 2);
      if (!ids.length) { gesture.current = null; return; }
      const position = gesturePosition(ids);
      const current = viewRef.current;
      gesture.current = { ids, anchor: worldPoint(current, position.center), center: position.center, zoom: current.zoom, distance: position.distance };
    }
    function releasePointer(pointerId: number) {
      const element = container.current;
      try { if (element?.hasPointerCapture(pointerId)) element.releasePointerCapture(pointerId); }
      catch { /* The browser may already have released a cancelled pointer. */ }
    }
    function cancelInteractions(updateState = true) {
      const ids = [...pointers.current.keys()];
      // Remove pointers before release; lostpointercapture can fire synchronously.
      pointers.current.clear();
      gesture.current = null;
      if (gestureMoved.current) suppressClickUntil.current = Date.now() + 400;
      gestureMoved.current = false;
      for (const pointerId of ids) releasePointer(pointerId);
      clearTimeout(wheelTimer.current);
      wheelActive.current = false;
      if (updateState) { setDragging(false); setWheeling(false); setRasterZoom(null); }
    }
    function finishPointer(pointerId: number) {
      if (!pointers.current.has(pointerId)) return;
      pointers.current.delete(pointerId);
      releasePointer(pointerId);
      if (pointers.current.size) {
        // A remaining finger starts from the current view, so releasing the
        // other finger cannot snap back to the pre-pinch pan origin.
        rebaseGesture();
      } else {
        gesture.current = null;
        if (gestureMoved.current) suppressClickUntil.current = Date.now() + 400;
        gestureMoved.current = false;
        setDragging(false);
        if (!wheelActive.current) setRasterZoom(null);
      }
    }
    function zoomBy(delta: number) {
      cancelInteractions();
      const current = viewRef.current;
      applyView({ ...current, zoom: clamp(current.zoom + delta, MIN_ZOOM, MAX_ZOOM) });
    }
    function moveBy(x: number, y: number) {
      cancelInteractions();
      const current = viewRef.current;
      const scale = TILE_SIZE * 2 ** current.zoom;
      applyView({ ...current, x: wrap(current.x + x / scale), y: clamp(current.y + y / scale, 0, 1) });
    }
    function fitLocations() {
      cancelInteractions();
      applyView(interaction.current.fitted);
    }

    return <div style={{ display: "grid", gap: 7, minWidth: 0 }}>
      {invalidMarkers > 0 && <div role="status">{invalidMarkers} map {invalidMarkers === 1 ? "marker has" : "markers have"} invalid coordinates or a latitude outside the supported Web Mercator range (±85.0511°).</div>}
      <div ref={container} role="region" aria-label="Interactive street map. Drag or use arrow keys to move. Pinch or hold Control or Command while scrolling to zoom. Use plus and minus to zoom. Home fits all markers."
        tabIndex={0}
        style={{ position: "relative", width: "100%", height, minHeight: 120, overflow: "hidden", borderRadius: 9, border: "1px solid #9ca3af", background: appearance === "dark" ? "#20242a" : "#f1f3f5", color: appearance === "dark" ? "#f3f4f6" : "#1f2937", touchAction: "none", cursor: dragging ? "grabbing" : "grab", userSelect: "none" }}
        onPointerDown={event => {
          if (!pointers.current.size) { suppressClickUntil.current = 0; gestureMoved.current = false; }
          if (streaming || !markers.length || event.button !== 0) return;
          const button = (event.target as HTMLElement).closest("button");
          const markerTouch = event.pointerType === "touch" && button?.hasAttribute("data-map-marker");
          if (button && !markerTouch) return;
          // A stationary one-finger marker tap keeps its native button click.
          // Capture transfers to the viewport only once it moves or joins a pinch.
          if (!markerTouch) {
            event.preventDefault();
            event.currentTarget.focus({ preventScroll: true });
          }
          if (!pointers.current.size) {
            clearTimeout(wheelTimer.current);
            wheelActive.current = false;
            setWheeling(false);
            setRasterZoom(Math.floor(viewRef.current.zoom));
          }
          pointers.current.set(event.pointerId, { x: event.clientX, y: event.clientY });
          if (pointers.current.size > 1) { gestureMoved.current = true; suppressClickUntil.current = Infinity; }
          if (!markerTouch || pointers.current.size > 1) {
            try { for (const pointerId of pointers.current.keys()) event.currentTarget.setPointerCapture(pointerId); }
            catch { cancelInteractions(); return; }
          }
          rebaseGesture();
          setDragging(true);
        }}
        onPointerMove={event => {
          if (!pointers.current.has(event.pointerId)) return;
          pointers.current.set(event.pointerId, { x: event.clientX, y: event.clientY });
          const active = gesture.current;
          if (!active || !active.ids.includes(event.pointerId)) return;
          const position = gesturePosition(active.ids);
          if (!gestureMoved.current && Math.hypot(position.center.x - active.center.x, position.center.y - active.center.y) < 4) return;
          gestureMoved.current = true;
          suppressClickUntil.current = Infinity;
          event.preventDefault();
          try { for (const pointerId of pointers.current.keys()) if (!event.currentTarget.hasPointerCapture(pointerId)) event.currentTarget.setPointerCapture(pointerId); }
          catch { cancelInteractions(); return; }
          if (active.distance != null && (active.distance === 0 || position.distance === 0)) { rebaseGesture(); return; }
          const zoom = active.distance != null && position.distance != null
            ? clamp(active.zoom + Math.log2(position.distance / active.distance), MIN_ZOOM, MAX_ZOOM)
            : active.zoom;
          applyView(anchoredView(active.anchor, position.center, zoom));
        }}
        onPointerUp={event => finishPointer(event.pointerId)} onPointerCancel={event => finishPointer(event.pointerId)}
        onLostPointerCapture={event => { if (event.target === event.currentTarget) finishPointer(event.pointerId); }}
        onClickCapture={event => {
          if (Date.now() < suppressClickUntil.current) { event.preventDefault(); event.stopPropagation(); }
        }}
        onKeyDown={event => {
          if (streaming || event.target !== event.currentTarget || !markers.length) return;
          if (event.key === "+" || event.key === "=") zoomBy(1);
          else if (event.key === "-") zoomBy(-1);
          else if (event.key === "Home") fitLocations();
          else if (event.key === "ArrowLeft") moveBy(-80, 0);
          else if (event.key === "ArrowRight") moveBy(80, 0);
          else if (event.key === "ArrowUp") moveBy(0, -80);
          else if (event.key === "ArrowDown") moveBy(0, 80);
          else return;
          event.preventDefault();
        }}>
        {tiles.map(tile => {
          const state = tileStates[tile.key];
          const position: CSSProperties = { position: "absolute", left: tile.left, top: tile.top, width: tile.size, height: tile.size, maxWidth: "none", pointerEvents: "none" };
          return state?.data ? <img key={`${tile.key}/${tile.left}/${tile.top}`} src={state.data.dataUrl} alt="" draggable={false} style={position} onError={() => failImage(tile.key)} />
            : state?.error ? <div key={`${tile.key}/${tile.left}/${tile.top}`} style={{ ...position, display: "grid", placeItems: "center", fontSize: 12, border: "1px dashed #9ca3af", boxSizing: "border-box" }}>Map tile unavailable</div> : null;
        })}
        {routePath && <svg width="100%" height="100%" style={{ position: "absolute", inset: 0, pointerEvents: "none" }} aria-label="Road route in marker order" role="img">
          {copyOffsets.map(offset => <g key={offset} transform={`translate(${offset} 0)`}><polyline points={routePath} fill="none" stroke="#fff" strokeWidth={7} strokeLinejoin="round" strokeLinecap="round" /><polyline points={routePath} fill="none" stroke="#2563eb" strokeWidth={4} strokeLinejoin="round" strokeLinecap="round" /></g>)}
        </svg>}
        {markers.flatMap(marker => {
          const point = project(marker.lat, marker.lng);
          const x = nearestDelta(point.x - currentView.x) * world + size.width / 2;
          const y = (point.y - currentView.y) * world + size.height / 2;
          return copyOffsets.filter(offset => x + offset >= -24 && x + offset <= size.width + 24 && y >= -24 && y <= size.height + 24).map(offset =>
            <button key={`${marker.index}/${offset}`} type="button" data-map-marker="true" disabled={streaming}
              aria-label={`${marker.index + 1}. ${marker.label}${marker.category ? `, ${marker.category}` : ""}`} aria-pressed={selected.value === marker.label}
              title={marker.label} onClick={() => selected.setValue(marker.label)}
              style={{ position: "absolute", left: x + offset, top: y, transform: "translate(-50%, -50%)", width: 32, height: 32, borderRadius: "50%", border: selected.value === marker.label ? "3px solid #111827" : "2px solid #fff", boxShadow: "0 1px 5px #0008", background: selected.value === marker.label ? "#fbbf24" : "#2563eb", color: selected.value === marker.label ? "#111827" : "#fff", font: "inherit", fontSize: 14, fontWeight: 700, padding: 0, cursor: "pointer" }}>
              {marker.icon || marker.index + 1}
            </button>);
        })}
        {!markers.length && <div role="status" style={{ position: "absolute", inset: 0, display: "grid", placeItems: "center", padding: 20, textAlign: "center" }}>{streaming ? "Waiting for map locations…" : "No valid map locations were supplied."}</div>}
        {markers.length > 0 && <div style={{ position: "absolute", top: 8, left: 8, display: "flex", gap: 4 }}>
          <button type="button" style={controlStyle} aria-label="Zoom in" disabled={streaming || currentView.zoom >= MAX_ZOOM} onClick={() => zoomBy(1)}>+</button>
          <button type="button" style={controlStyle} aria-label="Zoom out" disabled={streaming || currentView.zoom <= MIN_ZOOM} onClick={() => zoomBy(-1)}>−</button>
          <button type="button" style={controlStyle} disabled={streaming} onClick={fitLocations}>Fit locations</button>
        </div>}
        {loading && <span role="status" style={{ position: "absolute", top: 47, left: 8, padding: "3px 7px", fontSize: 12, background: "#fffe", color: "#111827", borderRadius: 4 }}>Loading street-map tiles…</span>}
        {credits.length > 0 && <div style={{ position: "absolute", right: 0, bottom: 0, left: 0, padding: "3px 7px", fontSize: 10, background: "#fffe", color: "#111827", cursor: "default", userSelect: "text" }}>
          {credits.join(" · ")}<span style={{ display: "block" }}>{sources.join(" · ")}</span>
        </div>}
      </div>
      {markers.length > 0 && <span style={{ fontSize: "0.75em", opacity: 0.8 }}>Drag to move; pinch or hold Ctrl/⌘ while scrolling to zoom. Ordinary scrolling moves the page.</span>}
      {failures.length > 0 && <div role="status" style={{ fontSize: "0.85em" }}>
        {failures.length} visible map {failures.length === 1 ? "tile is" : "tiles are"} unavailable. {tileStates[failures[0]]?.error}{" "}
        <button type="button" style={controlStyle} onClick={() => setRetry(value => value + 1)}>Retry map resources</button>
      </div>}
      {props.route && <div role="status" style={{ fontSize: "0.85em" }}>
        {routeUnavailable ?? (streaming ? "Road route will load when the response finishes." : routeState.key === routeKey && routeState.error ? `Road route unavailable: ${routeState.error}` : routeState.key === routeKey && routeState.loading ? "Loading the road route…" : routePoints ? "Road route follows the marker order." : "Road route is not available yet.")}
        {routeState.key === routeKey && routeState.error && <button type="button" style={{ ...controlStyle, marginLeft: 6 }} onClick={() => setRetry(value => value + 1)}>Retry route</button>}
      </div>}
      {selectedMarker && <div aria-live="polite" style={{ fontSize: "0.9em" }}>
        <strong>{selectedMarker.label}</strong>{selectedMarker.category ? ` · ${selectedMarker.category}` : ""}<span style={{ display: "block", opacity: 0.8 }}>{selectedMarker.lat}, {selectedMarker.lng}</span>
      </div>}
      {markers.length > 0 && <div role="group" aria-label="Map locations" style={{ display: "flex", flexWrap: "wrap", gap: 5 }}>
        {markers.map(marker => <button key={marker.index} type="button" aria-pressed={selected.value === marker.label} disabled={streaming}
          style={{ ...controlStyle, fontSize: 12, background: selected.value === marker.label ? "#dbeafe" : "#fff" }}
          onClick={() => { cancelInteractions(); selected.setValue(marker.label); const point = project(marker.lat, marker.lng); applyView({ ...viewRef.current, ...point }); }}>
          {marker.index + 1}. {marker.label}
        </button>)}
      </div>}
    </div>;
  },
});

export const mapComponents: DefinedComponent[] = [MapComponent, MapMarker];
