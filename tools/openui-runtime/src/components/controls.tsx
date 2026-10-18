// Independently authored from the published component signatures and the
// installed OpenUI SDK's public API. No application implementation is reused.
import React, { useContext, useId, useLayoutEffect, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { icons, type LucideIcon } from "lucide-react";
import { z } from "zod/v4";
import {
  action,
  defineComponent,
  FormNameContext,
  reactive,
  useFormName,
  useIsStreaming,
  useStateField,
  useTriggerAction,
  type ActionPlan,
  type DefinedComponent,
} from "@openuidev/react-lang";
import { HostActionsContext, ProgramRevisionContext } from "../capabilities";
import { getActionProgramRevision, prepareNativeActions, performNativeAction, type GestureEvent, type NativeAction } from "../native-actions";

const fieldStyle: CSSProperties = {
  boxSizing: "border-box",
  width: "100%",
  minWidth: 0,
  padding: "9px 11px",
  border: "1px solid color-mix(in srgb, currentColor 28%, transparent)",
  borderRadius: 8,
  background: "transparent",
  color: "inherit",
  font: "inherit",
};
const fieldGroupStyle: CSSProperties = { display: "grid", gap: 6, minWidth: 0 };
const choiceSchema = z.union([
  z.string(),
  z.object({ value: z.string(), label: z.string() }),
]);

function Field({ label, children }: { label?: string; children: ReactNode }) {
  return <label style={fieldGroupStyle}>
    {label && <span style={{ fontSize: "0.9em", fontWeight: 500 }}>{label}</span>}
    {children}
  </label>;
}

function controlIcon(name?: string) {
  if (!name) return null;
  const key = name.replace(/(^|-)([a-z0-9])/g, (_match, _prefix, char: string) => char.toUpperCase());
  const Icon = Object.prototype.hasOwnProperty.call(icons, key)
    ? (icons as Record<string, LucideIcon>)[key]
    : undefined;
  return Icon
    ? <Icon size={18} aria-hidden="true" focusable="false" />
    : <span style={{ fontSize: "0.75em" }} title={`Unknown Lucide icon: ${name}`}>Icon unavailable</span>;
}

type ControlAction = ActionPlan | { type?: string; params?: Record<string, unknown>; url?: string; context?: unknown } | null | undefined;
type ActionSegment = { kind: "sdk"; action: ControlAction } | { kind: "native" };
type BoundAction = { key: string; segments: ActionSegment[]; nativeActions: NativeAction[] };

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

// Snapshot the data, including later local steps, before any asynchronous work.
// Stable key order associates feedback with equal SDK props without serializing React nodes.
function snapshotAction(value: unknown, ancestors = new Set<object>()): unknown {
  if (value == null || ["string", "number", "boolean", "undefined"].includes(typeof value)) return value;
  if (typeof value !== "object" || ancestors.has(value)) throw new Error("Actions must contain acyclic data.");
  const prototype = Object.getPrototypeOf(value);
  if (!Array.isArray(value) && prototype !== Object.prototype && prototype !== null) throw new Error("Actions must contain plain data.");
  ancestors.add(value);
  try {
    if (Array.isArray(value)) return Object.freeze(value.map(item => snapshotAction(item, ancestors)));
    return Object.freeze(Object.fromEntries(Object.keys(value).sort().map(key => {
      const descriptor = Object.getOwnPropertyDescriptor(value, key)!;
      if (!("value" in descriptor)) throw new Error("Actions cannot contain accessors.");
      return [key, snapshotAction(descriptor.value, ancestors)];
    })));
  } finally { ancestors.delete(value); }
}

function actionKey(value: unknown): string {
  if (value === null) return "null";
  if (Array.isArray(value)) return `array[${value.map(actionKey).join(",")}]`;
  if (isRecord(value)) return `object{${Object.keys(value).map(key => `${JSON.stringify(key)}:${actionKey(value[key])}`).join(",")}}`;
  return `${typeof value}:${typeof value === "string" ? JSON.stringify(value) : String(value)}`;
}

// Validate expression structure without evaluating it: Set expressions must
// still be evaluated by the SDK at their original place in the local sequence.
function validExpression(node: unknown): boolean {
  if (!isRecord(node)) return false;
  const expressions = (value: unknown) => Array.isArray(value) && value.every(validExpression);
  switch (node.k) {
    case "Str": return typeof node.v === "string";
    case "Num": return typeof node.v === "number";
    case "Bool": return typeof node.v === "boolean";
    case "Null": return true;
    case "Ref": case "Ph": case "StateRef": return typeof node.n === "string";
    case "RuntimeRef": return typeof node.n === "string" && (node.refType === "query" || node.refType === "mutation");
    case "Arr": return expressions(node.els);
    case "Obj": return Array.isArray(node.entries) && node.entries.every(entry => Array.isArray(entry) && entry.length === 2 && typeof entry[0] === "string" && validExpression(entry[1]));
    case "Comp": return typeof node.name === "string" && expressions(node.args)
      && (node.mappedProps === undefined || (isRecord(node.mappedProps) && Object.values(node.mappedProps).every(validExpression)))
      && (node.fn === undefined || node.fn === true);
    case "BinOp": return typeof node.op === "string" && ["+", "-", "*", "/", "%", "==", "!=", ">", "<", ">=", "<=", "&&", "||"].includes(node.op) && validExpression(node.left) && validExpression(node.right);
    case "UnaryOp": return (node.op === "!" || node.op === "-") && validExpression(node.operand);
    case "Ternary": return validExpression(node.cond) && validExpression(node.then) && validExpression(node.else);
    case "Member": return validExpression(node.obj) && typeof node.field === "string";
    case "Index": return validExpression(node.obj) && validExpression(node.index);
    case "Assign": return typeof node.target === "string" && validExpression(node.value);
    default: return false;
  }
}

function bindControlAction(candidate: ControlAction): BoundAction {
  const snapshot = snapshotAction(candidate);
  const bound: BoundAction = { key: actionKey(snapshot), segments: [], nativeActions: [] };
  const native = (action: NativeAction) => {
    bound.nativeActions.push(Object.freeze(action));
    bound.segments.push({ kind: "native" });
  };
  if (snapshot == null) {
    bound.segments.push({ kind: "sdk", action: undefined });
    return bound;
  }
  if (!isRecord(snapshot)) throw new Error("This action is invalid.");
  if ("steps" in snapshot) {
    if (!Array.isArray(snapshot.steps)) throw new Error("This action has an invalid step list.");
    let local: ActionPlan["steps"] = [];
    const flush = () => {
      if (local.length) bound.segments.push({ kind: "sdk", action: { steps: local } });
      local = [];
    };
    for (const step of snapshot.steps) {
      if (!isRecord(step)) throw new Error("This action contains an invalid step.");
      switch (step.type) {
        case "set":
          if (typeof step.target !== "string" || !step.target || !validExpression(step.valueAST)) throw new Error("This Set step has an invalid target or expression.");
          break;
        case "reset":
          if (!Array.isArray(step.targets) || !step.targets.every(target => typeof target === "string" && !!target)) throw new Error("This Reset step has invalid targets.");
          break;
        case "continue_conversation":
          if (typeof step.message !== "string") throw new Error("This assistant step has an invalid message.");
          break;
        case "open_url":
          if (typeof step.url !== "string") throw new Error("Opening a link requires an explicit URL string.");
          flush(); native({ type: "open-url", url: step.url }); continue;
        case "custom_action":
          if (step.name !== "CopyToClipboard") throw new Error(`Action “${String(step.name ?? "unknown")}” is unavailable in this viewer.`);
          if (!isRecord(step.params) || typeof step.params.text !== "string") throw new Error("CopyToClipboard requires an explicit text string.");
          flush(); native({ type: "clipboard", text: step.params.text }); continue;
        default: throw new Error(`Action “${String(step.type ?? "unknown")}” is unavailable in this viewer.`);
      }
      local.push(step as ActionPlan["steps"][number]);
    }
    flush();
    // Preserve the real SDK's whole-plan behavior, including an empty plan.
    if (!bound.nativeActions.length) bound.segments = [{ kind: "sdk", action: snapshot as unknown as ActionPlan }];
    return bound;
  }
  if (snapshot.params !== undefined && !isRecord(snapshot.params)) throw new Error("This action has invalid parameters.");
  const params = isRecord(snapshot.params) ? snapshot.params : {};
  switch (snapshot.type) {
    case "continue_conversation":
      bound.segments.push({ kind: "sdk", action: snapshot as ControlAction }); break;
    case "open_url": {
      // These are the SDK's two explicit legacy URL locations. Conflicting
      // values are rejected rather than guessing which destination was meant.
      if (snapshot.url !== undefined && params.url !== undefined && snapshot.url !== params.url) throw new Error("This action contains conflicting URLs.");
      const url = snapshot.url ?? params.url;
      if (typeof url !== "string") throw new Error("Opening a link requires an explicit URL string.");
      native({ type: "open-url", url }); break;
    }
    case "CopyToClipboard":
      if (typeof params.text !== "string") throw new Error("CopyToClipboard requires an explicit text string.");
      native({ type: "clipboard", text: params.text }); break;
    default: throw new Error(`Action “${String(snapshot.type ?? "unknown")}” is unavailable in this viewer.`);
  }
  return bound;
}

export function useControlAction(label: string, candidate: ControlAction) {
  const trigger = useTriggerAction();
  const formName = useFormName();
  const streaming = useIsStreaming();
  const hostActions = useContext(HostActionsContext);
  const programRevision = useContext(ProgramRevisionContext);
  let bound: BoundAction | undefined;
  let blocked: string | undefined;
  try {
    bound = bindControlAction(candidate);
    if (bound.nativeActions.length && !hostActions) blocked = "Native clipboard and link actions require a supported host.";
  } catch (error) {
    blocked = error instanceof Error ? error.message : "This action is invalid.";
  }
  const identity = JSON.stringify([bound?.key, blocked, label, formName, streaming, hostActions]);
  const latestIdentity = useRef(identity);
  latestIdentity.current = identity;
  const scope = `${programRevision}:${streaming}:${hostActions}`;
  const latestScope = useRef(scope);
  latestScope.current = scope;
  const mounted = useRef(false);
  const active = useRef<symbol | undefined>(undefined);
  const [status, setStatus] = useState<{ scope: string; identity: string; pending: boolean; notice?: string }>();
  useLayoutEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; active.current = undefined; };
  }, []);
  useLayoutEffect(() => {
    // Only a real source/capability transition retires this component's run.
    // Set/Reset can change the reactive candidate or label without changing scope.
    setStatus(undefined);
    return () => { active.current = undefined; };
  }, [scope]);
  const pending = status?.scope === scope && status.pending;

  async function invoke(event?: GestureEvent) {
    if (!mounted.current || latestIdentity.current !== identity || getActionProgramRevision() !== programRevision || streaming || active.current || blocked || !bound) return;
    const run = Symbol("control action");
    active.current = run;
    const current = () => mounted.current && active.current === run && latestScope.current === scope && getActionProgramRevision() === programRevision;
    setStatus({ scope, identity, pending: true });
    try {
      // This is synchronous and precedes EVERY local write. It validates all
      // native payloads and binds them once to this actual source gesture. A
      // programmatic invoke() can still run local Reset, never a native action.
      const permit = bound.nativeActions.length ? prepareNativeActions(event, bound.nativeActions) : undefined;
      for (const segment of bound.segments) {
        if (!current()) return;
        if (segment.kind === "native") {
          if (!permit) throw new Error("This action has no native gesture permit.");
          await performNativeAction(permit);
        } else {
          // The SDK handles Set/Reset and assistant events. Its onAction callback
          // is not awaited, so native steps must never be delegated to the SDK.
          await trigger(label, formName, segment.action ?? undefined);
        }
      }
      if (current()) setStatus({ scope, identity, pending: false, notice: bound.nativeActions.length === 1 ? (bound.nativeActions[0].type === "clipboard" ? "Copied" : "Sent to browser") : bound.nativeActions.length ? "Host actions completed" : undefined });
    } catch (error) {
      // Failure stops all remaining steps. Completed local writes/native effects
      // are NOT rolled back. Unmount or source replacement cancels later steps;
      // reactive props never replace the invocation's immutable native payloads.
      // An already-dispatched host operation cannot be retracted.
      if (current()) setStatus({ scope, identity, pending: false, notice: error instanceof Error ? error.message : "The action could not be completed." });
    } finally {
      if (active.current === run) active.current = undefined;
    }
  }
  return { invoke, disabled: streaming || pending || !!blocked, notice: blocked ?? (status?.scope === scope && status.identity === identity ? status.notice : undefined) };
}

function buttonStyle(variant = "solid", color = "primary", size = "md", block = false): CSSProperties {
  const accent = color === "danger" ? "#b91c1c" : color === "success" ? "#166534" : "#1d4ed8";
  return {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    gap: 7,
    width: block ? "100%" : undefined,
    boxSizing: "border-box",
    border: variant === "outline" ? `1px solid ${accent}` : "1px solid transparent",
    borderRadius: 8,
    padding: size === "sm" ? "6px 10px" : size === "lg" ? "12px 18px" : "9px 14px",
    background: variant === "solid" ? accent : variant === "soft" ? `color-mix(in srgb, ${accent} 12%, transparent)` : "transparent",
    color: variant === "solid" ? "#fff" : accent,
    font: "inherit",
    fontSize: size === "sm" ? "0.85em" : size === "lg" ? "1.05em" : "0.95em",
    fontWeight: 600,
    cursor: "pointer",
  };
}

const Button = defineComponent({
  name: "Button",
  description: "An action button. Without an action, sends its label to the assistant.",
  // Property insertion order is the OpenUI positional calling convention.
  props: z.object({
    label: z.string(),
    action: action().optional(),
    variant: z.enum(["solid", "outline", "soft", "ghost"]).optional(),
    color: z.enum(["primary", "danger", "success"]).optional(),
    size: z.enum(["sm", "md", "lg"]).optional(),
    icon: z.string().optional(),
    disabled: z.boolean().optional(),
    block: z.boolean().optional(),
  }),
  component: ({ props }) => {
    const behavior = useControlAction(props.label, props.action);
    const disabled = props.disabled || behavior.disabled;
    return <span style={{ display: props.block ? "block" : "inline-grid", gap: 5 }}>
      <button type="button" disabled={disabled} onClick={behavior.invoke}
        title={behavior.notice}
        style={{ ...buttonStyle(props.variant, props.color, props.size, props.block), opacity: disabled ? 0.55 : 1, cursor: disabled ? "not-allowed" : "pointer" }}>
        {controlIcon(props.icon)}{props.label}
      </button>
      {behavior.notice && <span role="status" style={{ fontSize: "0.8em" }}>{behavior.notice}</span>}
    </span>;
  },
});

const IconButton = defineComponent({
  name: "IconButton",
  description: "A round icon action button with an accessible label.",
  props: z.object({
    icon: z.string(),
    action: action().optional(),
    label: z.string().optional(),
    variant: z.enum(["outline", "ghost", "solid"]).optional(),
  }),
  component: ({ props }) => {
    const label = props.label || props.icon;
    const behavior = useControlAction(label, props.action);
    return <span style={{ display: "inline-grid", gap: 5 }}>
      <button type="button" aria-label={label} disabled={behavior.disabled}
        title={behavior.notice || label} onClick={behavior.invoke}
        style={{ ...buttonStyle(props.variant ?? "outline"), borderRadius: "50%", width: 38, height: 38, padding: 8, opacity: behavior.disabled ? 0.55 : 1 }}>
        {controlIcon(props.icon)}
      </button>
      {behavior.notice && <span role="status" style={{ fontSize: "0.8em" }}>{behavior.notice}</span>}
    </span>;
  },
});

const Input = defineComponent({
  name: "Input",
  description: "A reactive text or number input. Numeric edits store numbers; an empty field stores an empty string.",
  props: z.object({
    value: reactive(z.union([z.string(), z.number()])),
    label: z.string().optional(),
    type: z.enum(["text", "number"]).optional(),
    placeholder: z.string().optional(),
  }),
  component: ({ props, statementId }) => {
    const id = useId();
    const field = useStateField(statementId ?? id, props.value);
    const streaming = useIsStreaming();
    return <Field label={props.label}>
      <input id={id} type={props.type ?? "text"} value={field.value ?? ""}
        aria-label={props.label || props.placeholder || (props.type === "number" ? "Number" : "Text")}
        disabled={streaming} placeholder={props.placeholder} style={fieldStyle}
        onChange={(event) => field.setValue(props.type === "number"
          ? Number.isFinite(event.currentTarget.valueAsNumber) ? event.currentTarget.valueAsNumber : ""
          : event.currentTarget.value)} />
    </Field>;
  },
});

const TextArea = defineComponent({
  name: "TextArea",
  description: "A reactive multiline text field.",
  props: z.object({
    value: reactive(z.string()),
    label: z.string().optional(),
    placeholder: z.string().optional(),
    rows: z.number().optional(),
  }),
  component: ({ props, statementId }) => {
    const id = useId();
    const field = useStateField(statementId ?? id, props.value);
    const streaming = useIsStreaming();
    return <Field label={props.label}>
      <textarea id={id} value={field.value ?? ""} disabled={streaming}
        aria-label={props.label || props.placeholder || "Multiline text"}
        rows={Math.max(1, Math.min(50, Math.round(props.rows ?? 4)))}
        placeholder={props.placeholder} style={{ ...fieldStyle, resize: "vertical" }}
        onChange={(event) => field.setValue(event.currentTarget.value)} />
    </Field>;
  },
});

const Select = defineComponent({
  name: "Select",
  description: "A reactive dropdown of string choices or value/label objects.",
  props: z.object({
    value: reactive(z.string()),
    options: z.array(choiceSchema),
    label: z.string().optional(),
    placeholder: z.string().optional(),
  }),
  component: ({ props, statementId }) => {
    const id = useId();
    const field = useStateField(statementId ?? id, props.value);
    const streaming = useIsStreaming();
    const choices = (props.options ?? []).map((option) => typeof option === "string" ? { value: option, label: option } : option);
    const currentValue = typeof field.value === "string" ? field.value : "";
    const hasCurrentOption = choices.some((option) => option.value === currentValue);
    return <Field label={props.label}>
      <select id={id} value={currentValue} disabled={streaming}
        aria-label={props.label || props.placeholder || "Select an option"}
        style={fieldStyle} onChange={(event) => field.setValue(event.currentTarget.value)}>
        {(!hasCurrentOption || (props.placeholder && !choices.some((option) => option.value === ""))) &&
          <option value={hasCurrentOption ? "" : currentValue} disabled>{!hasCurrentOption && currentValue ? `Unknown option: ${currentValue}` : props.placeholder ?? "Select an option"}</option>}
        {choices.map((option, index) => <option key={`${option.value}:${index}`} value={option.value}>{option.label}</option>)}
      </select>
    </Field>;
  },
});

const Checkbox = defineComponent({
  name: "Checkbox",
  description: "A reactive checkbox; optionally strike the checked label through.",
  props: z.object({
    checked: reactive(z.boolean()),
    label: z.string(),
    lineThrough: z.boolean().optional(),
  }),
  component: ({ props, statementId }) => {
    const id = useId();
    const field = useStateField(statementId ?? id, props.checked);
    const streaming = useIsStreaming();
    return <label style={{ display: "flex", alignItems: "center", gap: 9 }}>
      <input id={id} type="checkbox" checked={field.value === true} disabled={streaming}
        onChange={(event) => field.setValue(event.currentTarget.checked)} />
      <span style={{ textDecoration: props.lineThrough && field.value === true ? "line-through" : undefined }}>{props.label}</span>
    </label>;
  },
});

const Slider = defineComponent({
  name: "Slider",
  description: "A reactive range input with a value label and optional marks.",
  props: z.object({
    value: reactive(z.number()),
    min: z.number(),
    max: z.number(),
    step: z.number().optional(),
    label: z.string().optional(),
    unit: z.string().optional(),
    marks: z.array(z.object({ value: z.number(), label: z.string() })).optional(),
  }),
  component: ({ props, statementId }) => {
    const id = useId();
    const field = useStateField(statementId ?? id, props.value);
    const streaming = useIsStreaming();
    const validRange = Number.isFinite(props.min) && Number.isFinite(props.max) && props.max > props.min && (props.step == null || (Number.isFinite(props.step) && props.step > 0));
    const numericValue = typeof field.value === "number" && Number.isFinite(field.value);
    const validValue = numericValue && field.value >= props.min && field.value <= props.max;
    const formatted = numericValue ? (props.unit === "$" ? `$${field.value}` : `${field.value}${props.unit ?? ""}`) : "No value";
    const marks = (props.marks ?? []).filter((mark) => mark.value >= props.min && mark.value <= props.max);
    return <div style={fieldGroupStyle}>
      <label htmlFor={id} style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
        <span>{props.label ?? "Value"}</span><output htmlFor={id}>{formatted}</output>
      </label>
      <input id={id} type="range" min={props.min} max={props.max} step={props.step ?? 1}
        value={validValue ? field.value : props.min} aria-valuetext={formatted}
        disabled={streaming || !validRange || !validValue}
        list={marks.length ? `${id}-marks` : undefined} style={{ width: "100%", margin: "4px 0" }}
        onChange={(event) => { if (Number.isFinite(event.currentTarget.valueAsNumber)) field.setValue(event.currentTarget.valueAsNumber); }} />
      {marks.length > 0 && <datalist id={`${id}-marks`}>{marks.map((mark, index) => <option key={index} value={mark.value} label={mark.label} />)}</datalist>}
      {marks.length > 0 && validRange && <div aria-hidden="true" style={{ position: "relative", minHeight: "1.5em", fontSize: "0.8em" }}>{marks.map((mark, index) =>
        <span key={index} style={{ position: "absolute", left: `${100 * (mark.value - props.min) / (props.max - props.min)}%`, transform: mark.value === props.min ? undefined : mark.value === props.max ? "translateX(-100%)" : "translateX(-50%)" }}>{mark.label}</span>)}</div>}
      {(!validRange || !validValue) && <span role="status" style={{ fontSize: "0.8em" }}>{!validRange ? "Slider requires a valid increasing range and a positive step." : !numericValue ? "Slider requires a numeric bound value." : "Slider value is outside its declared range."}</span>}
    </div>;
  },
});

const SegmentedControl = defineComponent({
  name: "SegmentedControl",
  description: "Reactive short choices arranged in a row.",
  props: z.object({
    value: reactive(z.string()),
    options: z.array(choiceSchema),
    block: z.boolean().optional(),
  }),
  component: ({ props, statementId }) => {
    const id = useId();
    const field = useStateField(statementId ?? id, props.value);
    const streaming = useIsStreaming();
    return <div role="group" aria-label="Choose an option" style={{ display: props.block ? "flex" : "inline-flex", gap: 4, flexWrap: "wrap" }}>
      {(props.options ?? []).map((option, index) => {
        const choice = typeof option === "string" ? { value: option, label: option } : option;
        const selected = choice.value === field.value;
        return <button key={`${choice.value}:${index}`} type="button" aria-pressed={selected} disabled={streaming}
          style={{ ...buttonStyle(selected ? "solid" : "outline", "primary", "sm"), flex: props.block ? "1 1 0" : undefined }}
          onClick={() => field.setValue(choice.value)}>{choice.label}</button>;
      })}
    </div>;
  },
});

const RadioGroup = defineComponent({
  name: "RadioGroup",
  description: "A reactive single choice displayed as a list, pills, or cards.",
  props: z.object({
    value: reactive(z.string()),
    options: z.array(z.union([z.string(), z.object({ value: z.string(), label: z.string(), description: z.string().optional() })])),
    label: z.string().optional(),
    variant: z.enum(["list", "pills", "cards"]).optional(),
  }),
  component: ({ props, statementId }) => {
    const id = useId();
    const field = useStateField(statementId ?? id, props.value);
    const streaming = useIsStreaming();
    return <fieldset disabled={streaming} style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}>
      {props.label && <legend style={{ marginBottom: 7, fontWeight: 500 }}>{props.label}</legend>}
      <div style={{ display: props.variant === "pills" ? "flex" : "grid", flexWrap: "wrap", gap: 8 }}>
        {(props.options ?? []).map((option, index) => {
          const choice = typeof option === "string" ? { value: option, label: option, description: undefined } : option;
          const selected = field.value === choice.value;
          return <label key={`${choice.value}:${index}`} style={{ display: "flex", gap: 8, alignItems: "baseline", padding: props.variant === "cards" ? 12 : props.variant === "pills" ? "6px 10px" : 0, borderRadius: props.variant === "pills" ? 999 : 8, border: props.variant === "cards" || props.variant === "pills" ? `1px solid ${selected ? "#1d4ed8" : "color-mix(in srgb, currentColor 25%, transparent)"}` : undefined }}>
            <input type="radio" name={id} value={choice.value} checked={selected} onChange={() => field.setValue(choice.value)} />
            <span><span>{choice.label}</span>{choice.description && <span style={{ display: "block", opacity: 0.75, fontSize: "0.85em", marginTop: 4 }}>{choice.description}</span>}</span>
          </label>;
        })}
      </div>
    </fieldset>;
  },
});

const DatePicker = defineComponent({
  name: "DatePicker",
  description: "A reactive date input using YYYY-MM-DD values.",
  props: z.object({
    value: reactive(z.string()),
    label: z.string().optional(),
    min: z.string().optional(),
    max: z.string().optional(),
  }),
  component: ({ props, statementId }) => {
    const id = useId();
    const field = useStateField(statementId ?? id, props.value);
    const streaming = useIsStreaming();
    return <Field label={props.label}>
      <input id={id} type="date" value={field.value ?? ""} min={props.min} max={props.max}
        disabled={streaming} aria-label={props.label || "Date"} style={fieldStyle}
        onChange={(event) => field.setValue(event.currentTarget.value)} />
    </Field>;
  },
});

const Form = defineComponent({
  name: "Form",
  description: "Groups fields and passes their current state with assistant actions.",
  props: z.object({ children: z.array(z.any()) }),
  component: ({ props, renderNode, statementId }) => {
    const id = useId();
    const formName = statementId ?? `form-${id}`;
    // A group avoids nested HTML forms and accidental browser navigation. The
    // SDK owns the form context and includes current reactive state in actions.
    return <FormNameContext.Provider value={formName}>
      <div role="group" aria-label="Form" style={{ display: "grid", gap: 14 }}>
        {renderNode(props.children)}
      </div>
    </FormNameContext.Provider>;
  },
});

export const controlComponents: DefinedComponent[] = [
  Button, IconButton, Input, TextArea, Select, Checkbox, Slider,
  SegmentedControl, RadioGroup, DatePicker, Form,
];
