// Independently authored from the public Pressable/Timer component contracts.
import React, { useEffect, useId, useRef, useState, type CSSProperties } from "react";
import { action, defineComponent, isReactiveAssign, reactive, useIsStreaming, useStateField, type ActionPlan, type DefinedComponent } from "@openuidev/react-lang";
import { z } from "zod/v4";
import { useControlAction } from "./controls";

const radii: Record<string, number> = { sm: 4, md: 8, lg: 12, xl: 20, full: 9999 };
const backgrounds: Record<string, string> = {
  ink: "var(--oui-ink, #172337)", muted: "var(--oui-muted, #526379)",
  faint: "var(--oui-faint, #7b899b)", surface: "var(--oui-surface, #f3f6fa)",
  info: "#eaf2ff", success: "#e8f6ed", warning: "#fff4df", danger: "#ffedf0",
  purple: "#f2ebff", pink: "#ffedf6", chart1: "#386cc9", chart2: "#188773",
  chart3: "#ad7226", chart4: "#8451bc", chart5: "#c15371", chart6: "#2484a6",
};
const buttonStyle: CSSProperties = {
  border: "1px solid var(--oui-border, #9ca3af)", borderRadius: 7,
  padding: "6px 12px", background: "var(--oui-surface, #f3f6fa)",
  color: "inherit", font: "inherit", fontSize: "0.9em", cursor: "pointer",
};

const Pressable = defineComponent({
  name: "Pressable",
  description: "Clickable content with optional selected styling and a local or assistant action.",
  props: z.object({
    children: z.array(z.any()),
    action: action().optional(),
    selected: z.boolean().optional(),
    padding: z.number().optional(),
    background: z.string().optional(),
    radius: z.union([z.enum(["sm", "md", "lg", "xl", "full"]), z.number()]).optional(),
  }),
  component: ({ props, renderNode }) => {
    const content = useRef<HTMLDivElement>(null);
    const [label, setLabel] = useState("");
    useEffect(() => { setLabel(content.current?.innerText.trim() ?? ""); }, [props.children]);
    const behavior = useControlAction(label, props.action);
    const missingLabel = props.action == null && !label;
    const disabled = behavior.disabled || missingLabel;
    const radius = typeof props.radius === "number" ? Math.max(0, props.radius) : radii[props.radius ?? "md"];
    const background = props.background && Object.hasOwn(backgrounds, props.background) ? backgrounds[props.background] : props.background;
    const invalidColor = !!background && typeof CSS !== "undefined" && !CSS.supports("color", background);
    const notice = behavior.notice ?? (missingLabel ? "Clickable content without readable text needs an explicit action." : invalidColor ? `Unsupported background color: ${props.background}` : undefined);
    return <div style={{ display: "grid", gap: 5, minWidth: 0 }}>
      <div ref={content} role="button" tabIndex={disabled ? -1 : 0} aria-disabled={disabled} aria-pressed={props.selected}
        style={{ minWidth: 0, border: props.selected ? "2px solid #2563eb" : "2px solid transparent", borderRadius: radius,
          padding: Math.max(0, props.padding ?? 3) * 4, backgroundColor: invalidColor ? undefined : background,
          cursor: disabled ? "not-allowed" : "pointer", opacity: disabled ? 0.6 : 1 }}
        onClick={event => {
          const interactive = (event.target as Element).closest("button,a,input,select,textarea,[role=button]");
          if (!disabled && (!interactive || interactive === event.currentTarget)) void behavior.invoke();
        }}
        onKeyDown={event => {
          if (disabled || event.target !== event.currentTarget || (event.key !== "Enter" && event.key !== " ")) return;
          event.preventDefault();
          void behavior.invoke();
        }}>
        {renderNode(props.children)}
      </div>
      {notice && <span role="status" style={{ fontSize: "0.8em" }}>{notice}</span>}
    </div>;
  },
});

type Clock = { startedAt: number; initialSeconds: number; lastPublished: number };
function validSeconds(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value) && value >= 0 && Number.isSafeInteger(Math.round(value * 1000));
}
function timerText(value: number): string {
  const seconds = Math.ceil(value);
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const remainder = String(seconds % 60).padStart(2, "0");
  return hours ? `${hours}:${String(minutes).padStart(2, "0")}:${remainder}` : `${minutes}:${remainder}`;
}

const Timer = defineComponent({
  name: "Timer",
  description: "A start/pause/reset countdown that updates its bound seconds and stops at zero.",
  props: z.object({ seconds: reactive(z.number()), label: z.string().optional() }),
  component: ({ props, statementId }) => {
    const id = useId();
    const field = useStateField(statementId ?? `timer-${id}`, props.seconds);
    const streaming = useIsStreaming();
    const valid = validSeconds(field.value);
    const [running, setRunning] = useState(false);
    const clock = useRef<Clock | null>(null);
    const latestField = useRef(field);
    latestField.current = field;
    const [initial, setInitial] = useState<number | undefined>(undefined);
    const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
    const resetTarget = isReactiveAssign(props.seconds) ? props.seconds.target : undefined;
    const resetPlan: ActionPlan = { steps: resetTarget ? [{ type: "reset", targets: [resetTarget] }] : [] };
    const resetAction = useControlAction("", resetPlan);

    useEffect(() => {
      if (!streaming && valid && initial === undefined) setInitial(field.value);
    }, [streaming, valid, field.value, initial]);

    useEffect(() => {
      if (!running || streaming || !valid || field.value <= 0) {
        if (running) setRunning(false);
        clock.current = null;
        return;
      }
      // Keep a wall-clock anchor through our own writes. An external bound-value
      // change starts a new countdown from that new value.
      if (!clock.current || clock.current.lastPublished !== field.value) {
        clock.current = { startedAt: Date.now(), initialSeconds: field.value, lastPublished: field.value };
      }
      const activeClock = clock.current;
      let mounted = true;
      const tick = () => {
        if (!mounted || clock.current !== activeClock || latestField.current.value !== activeClock.lastPublished) return;
        const elapsed = Math.max(0, Date.now() - activeClock.startedAt);
        const duration = activeClock.initialSeconds * 1000;
        const remaining = elapsed >= duration ? 0 : Math.max(0, activeClock.initialSeconds - Math.floor(elapsed / 1000));
        if (remaining !== activeClock.lastPublished) {
          activeClock.lastPublished = remaining;
          latestField.current.setValue(remaining);
        }
        if (remaining === 0) { clock.current = null; setRunning(false); return; }
        const untilNextSecond = 1000 - elapsed % 1000;
        timer.current = setTimeout(tick, Math.max(1, Math.min(untilNextSecond, duration - elapsed)));
      };
      tick();
      return () => { mounted = false; clearTimeout(timer.current); };
    }, [running, streaming, valid, field.value]);

    function pause() {
      const activeClock = clock.current;
      clock.current = null;
      clearTimeout(timer.current);
      setRunning(false);
      if (activeClock && activeClock.lastPublished === latestField.current.value) {
        const elapsed = Math.max(0, Date.now() - activeClock.startedAt);
        // Preserve sub-second time when paused rather than adding it back when
        // resumed. Display rounds upward; the binding remains a genuine number.
        const remaining = Math.max(0, Math.round(activeClock.initialSeconds * 1000 - elapsed) / 1000);
        latestField.current.setValue(remaining);
      }
    }
    function reset() {
      clock.current = null;
      clearTimeout(timer.current);
      setRunning(false);
      if (resetTarget) void resetAction.invoke();
      else if (initial !== undefined) latestField.current.setValue(initial);
    }
    return <div style={{ display: "grid", gap: 9 }}>
      {props.label && <span style={{ fontWeight: 500 }}>{props.label}</span>}
      <output role="timer" aria-label={props.label ?? "Countdown"} aria-live="off" style={{ fontSize: "1.6em", fontVariantNumeric: "tabular-nums" }}>{valid ? timerText(field.value) : "Invalid duration"}</output>
      <div style={{ display: "flex", gap: 7, flexWrap: "wrap" }}>
        <button type="button" style={buttonStyle} disabled={streaming || !valid || (!running && field.value <= 0)}
          onClick={() => running ? pause() : setRunning(true)}>{running ? "Pause" : "Start"}</button>
        <button type="button" style={buttonStyle} disabled={streaming || resetAction.disabled || (!resetTarget && initial === undefined)} onClick={reset}>Reset</button>
      </div>
      {!valid && <span role="status">Timer seconds must be a non-negative finite duration representable in milliseconds.</span>}
      {resetAction.notice && <span role="status">{resetAction.notice}</span>}
      <span style={{ fontSize: "0.75em", opacity: 0.75 }}>Updates catch up after a suspended tab resumes. Closing this view stops the timer; it does not provide a background alarm.</span>
    </div>;
  },
});

export const extraControlComponents: DefinedComponent[] = [Pressable, Timer];
