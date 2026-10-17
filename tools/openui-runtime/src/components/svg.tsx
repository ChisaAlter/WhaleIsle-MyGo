// Independently authored drawing renderers for the published positional OpenUI API.
// Only the SDK evaluates expressions; this module receives resolved prop values.
import React, { createContext, useCallback, useContext, useEffect, useId, useState, type ReactNode } from 'react';
import { defineComponent, type DefinedComponent } from '@openuidev/react-lang';
import { z } from 'zod/v4';

const ink = 'var(--oui-ink, #172337)';
const names = new Set(['Svg', 'Circle', 'Ellipse', 'Rect', 'Line', 'Path', 'Polyline', 'Polygon', 'G', 'SvgText']);
const maxMagnitude = 1_000_000;
const maxGeometryLength = 100_000;
const numberToken = /^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?/;
const lengthSchema = z.union([z.number(), z.string()]);
const reportContext = createContext<((id: string, error: string | null) => void) | null>(null);

function invalid(field: string, detail: string): never {
  throw new Error(`${field}: ${detail}`);
}

function finite(value: unknown, field: string, min = -maxMagnitude, max = maxMagnitude): number {
  if (typeof value !== 'number' || !Number.isFinite(value) || value < min || value > max) {
    return invalid(field, `expected a finite number between ${min} and ${max}`);
  }
  return value;
}

function optionalNumber(value: unknown, field: string, min = 0, max = maxMagnitude): number | undefined {
  return value == null ? undefined : finite(value, field, min, max);
}

function boundedString(value: unknown, field: string, max = maxGeometryLength): string {
  if (typeof value !== 'string') return invalid(field, 'expected text');
  if (value.length > max) return invalid(field, `maximum length is ${max} characters`);
  return value;
}

function length(value: unknown, field: string): string | number | undefined {
  if (value == null) return undefined;
  if (typeof value === 'number') return finite(value, field, 0);
  const text = boundedString(value, field, 100).trim();
  if (text === 'auto') return text;
  if (/^(?:\d+(?:\.\d*)?|\.\d+)(?:px|%|em|rem|ex|ch|cap|ic|lh|rlh|vw|vh|vi|vb|vmin|vmax|svw|svh|lvw|lvh|dvw|dvh|cm|mm|q|in|pt|pc)?$/i.test(text)) {
    finite(Number.parseFloat(text), field, 0);
    // SVG interprets an unqualified numeric string in user units.
    return Number.isFinite(Number(text)) ? Number(text) : text;
  }
  return invalid(field, 'use a non-negative number, a simple CSS length, or auto');
}

const palette: Record<string, string> = {
  ink, muted: 'var(--oui-muted, #526379)', faint: 'var(--oui-faint, #7b899b)',
  surface: 'var(--oui-surface, #f3f6fa)', info: '#245fba', success: '#18734d',
  warning: '#936015', danger: '#b43245', purple: '#7945b7', pink: '#b5357a',
  chart1: '#386cc9', chart2: '#188773', chart3: '#ad7226', chart4: '#8451bc',
  chart5: '#c15371', chart6: '#2484a6',
};
const namedColors = new Set(('aliceblue antiquewhite aqua aquamarine azure beige bisque black blanchedalmond blue blueviolet brown burlywood cadetblue chartreuse chocolate coral cornflowerblue cornsilk crimson cyan darkblue darkcyan darkgoldenrod darkgray darkgreen darkgrey darkkhaki darkmagenta darkolivegreen darkorange darkorchid darkred darksalmon darkseagreen darkslateblue darkslategray darkslategrey darkturquoise darkviolet deeppink deepskyblue dimgray dimgrey dodgerblue firebrick floralwhite forestgreen fuchsia gainsboro ghostwhite gold goldenrod gray green greenyellow grey honeydew hotpink indianred indigo ivory khaki lavender lavenderblush lawngreen lemonchiffon lightblue lightcoral lightcyan lightgoldenrodyellow lightgray lightgreen lightgrey lightpink lightsalmon lightseagreen lightskyblue lightslategray lightslategrey lightsteelblue lightyellow lime limegreen linen magenta maroon mediumaquamarine mediumblue mediumorchid mediumpurple mediumseagreen mediumslateblue mediumspringgreen mediumturquoise mediumvioletred midnightblue mintcream mistyrose moccasin navajowhite navy oldlace olive olivedrab orange orangered orchid palegoldenrod palegreen paleturquoise palevioletred papayawhip peachpuff peru pink plum powderblue purple rebeccapurple red rosybrown royalblue saddlebrown salmon sandybrown seagreen seashell sienna silver skyblue slateblue slategray slategrey snow springgreen steelblue tan teal thistle tomato turquoise violet wheat white whitesmoke yellow yellowgreen transparent currentcolor').split(' '));

function paint(value: unknown, field: string): string | undefined {
  if (value == null) return undefined;
  const text = boundedString(value, field, 256).trim();
  if (Object.hasOwn(palette, text)) return palette[text];
  if (text === 'none' || namedColors.has(text.toLowerCase())) return text;
  if (/^#(?:[\da-f]{3}|[\da-f]{4}|[\da-f]{6}|[\da-f]{8})$/i.test(text)) return text;
  // No arbitrary CSS functions, variables, URL paint servers, or external references.
  const numericColor = /^(?:rgb|rgba|hsl|hsla|hwb|lab|lch|oklab|oklch)\([\d\s.,%/+\-eE]*(?:deg|grad|rad|turn)?[\d\s.,%/+\-eE]*\)$/i.test(text);
  const colorSpace = /^color\((?:srgb|srgb-linear|display-p3|a98-rgb|prophoto-rgb|rec2020|xyz|xyz-d50|xyz-d65)\s+[\d\s.%/+\-eE]+\)$/i.test(text);
  if ((numericColor || colorSpace) && typeof CSS !== 'undefined' && CSS.supports('color', text)) return text;
  return invalid(field, 'unsupported color; use a named, hex, or numeric CSS color without URL references');
}

type PaintProps = { fill?: unknown; stroke?: unknown; strokeWidth?: unknown; opacity?: unknown };
function appearance(props: PaintProps) {
  return {
    fill: paint(props.fill, 'fill'), stroke: paint(props.stroke, 'stroke'),
    strokeWidth: optionalNumber(props.strokeWidth, 'strokeWidth'),
    opacity: optionalNumber(props.opacity, 'opacity', 0, 1),
  };
}

// A bounded scanner accepts SVG numeric notation, including exponent notation.
class Numbers {
  index = 0;
  constructor(readonly text: string, readonly field: string) {}
  skip() { while (/[\s,]/.test(this.text[this.index] || '') && this.index < this.text.length) this.index++; }
  done() { this.skip(); return this.index >= this.text.length; }
  read(): number {
    this.skip();
    const token = this.text.slice(this.index).match(numberToken)?.[0];
    if (!token) return invalid(this.field, 'invalid numeric geometry');
    this.index += token.length;
    return finite(Number(token), this.field);
  }
  flag(): number {
    this.skip();
    const flag = this.text[this.index++];
    if (flag !== '0' && flag !== '1') return invalid(this.field, 'arc flags must be 0 or 1');
    return Number(flag);
  }
}

function numbers(value: unknown, field: string): number[] {
  const scanner = new Numbers(boundedString(value, field), field);
  const result: number[] = [];
  while (!scanner.done()) {
    if (result.length >= 20_000) return invalid(field, 'too many coordinates');
    result.push(scanner.read());
  }
  return result;
}

function viewBox(value: unknown): string {
  const box = numbers(value, 'viewBox');
  if (box.length !== 4 || box[2] <= 0 || box[3] <= 0) return invalid('viewBox', 'expected minX minY width height with positive width and height');
  return box.join(' ');
}

function points(value: unknown): string {
  const coords = numbers(value, 'points');
  if (coords.length % 2) return invalid('points', 'expected x,y coordinate pairs');
  return coords.join(' ');
}

function path(value: unknown): string {
  const text = boundedString(value, 'd');
  const scanner = new Numbers(text, 'd');
  const output: string[] = [];
  const counts: Record<string, number> = { m: 2, l: 2, h: 1, v: 1, c: 6, s: 4, q: 4, t: 2, a: 7 };
  let command = '';
  while (!scanner.done()) {
    if (output.length >= 10_000) return invalid('d', 'too many path segments');
    const next = text[scanner.index];
    if (/[a-z]/i.test(next)) {
      if (!/[MmLlHhVvCcSsQqTtAaZz]/.test(next)) return invalid('d', 'unsupported path command');
      command = next;
      scanner.index++;
    } else if (!command) return invalid('d', 'a path command is required');
    const lower = command.toLowerCase();
    if (!output.length && lower !== 'm') return invalid('d', 'the first path command must be M or m');
    if (lower === 'z') {
      output.push(command);
      command = '';
      continue;
    }
    const values: number[] = [];
    for (let i = 0; i < counts[lower]; i++) values.push(lower === 'a' && (i === 3 || i === 4) ? scanner.flag() : scanner.read());
    if (lower === 'a' && (values[0] < 0 || values[1] < 0)) return invalid('d', 'arc radii must be non-negative');
    output.push(`${command} ${values.join(' ')}`);
    // Further coordinate pairs following moveto are implicit line segments.
    if (command === 'M') command = 'L';
    else if (command === 'm') command = 'l';
  }
  return output.join(' ');
}

function transform(value: unknown): string | undefined {
  if (value == null) return undefined;
  let remaining = boundedString(value, 'transform', 10_000).trim();
  const output: string[] = [];
  const arities: Record<string, number[]> = { matrix: [6], translate: [1, 2], scale: [1, 2], rotate: [1, 3], skewX: [1], skewY: [1] };
  while (remaining) {
    if (output.length >= 256) return invalid('transform', 'too many transforms');
    const match = /^([a-zA-Z]+)\s*\(([^()]*)\)/.exec(remaining);
    if (!match || !Object.hasOwn(arities, match[1])) return invalid('transform', 'use matrix, translate, scale, rotate, skewX, or skewY');
    const values = numbers(match[2], 'transform');
    if (!arities[match[1]].includes(values.length)) return invalid('transform', `wrong number of parameters for ${match[1]}`);
    output.push(`${match[1]}(${values.join(' ')})`);
    remaining = remaining.slice(match[0].length).replace(/^[\s,]+/, '');
  }
  return output.length ? output.join(' ') : undefined;
}

function DrawingError({ message }: { message: string }) {
  const report = useContext(reportContext);
  const id = useId();
  useEffect(() => {
    if (!report) return;
    report(id, message);
    return () => report(id, null);
  }, [report, id, message]);
  // The outer HTML summary remains visible even inside an invisible/transformed G.
  return report
    ? <g role="alert" aria-label={message}><title>{message}</title><text x={4} y={16} fontSize={12} fill="#b43245">{message}</text></g>
    : <div className="oui-svg-error" role="alert" style={{ color: '#b43245', padding: 12, border: '1px solid currentColor', borderRadius: 6 }}>{message}</div>;
}

function failure(name: string, error: unknown): ReactNode {
  return <DrawingError message={`${name}: ${error instanceof Error ? error.message : 'unsupported drawing input'}`} />;
}

function Shape({ name, draw }: { name: string; draw: () => ReactNode }) {
  const report = useContext(reportContext);
  if (!report) return <DrawingError message={`${name} must be inside Svg.`} />;
  try { return draw(); } catch (error) { return failure(name, error); }
}

function children(value: unknown, renderNode: (value: unknown) => ReactNode, depth = 0): ReactNode {
  if (value == null || typeof value === 'boolean') return null;
  if (Array.isArray(value)) {
    if (depth > 32 || value.length > 10_000) return <DrawingError message="Svg: drawing children exceed the supported size." />;
    return value.map((child, index) => <React.Fragment key={index}>{children(child, renderNode, depth + 1)}</React.Fragment>);
  }
  if (typeof value === 'object' && 'type' in value && value.type === 'element' && 'typeName' in value && typeof value.typeName === 'string' && names.has(value.typeName)) {
    return renderNode(value);
  }
  return <DrawingError message="Svg: children must be drawing components. Use SvgText for labels." />;
}

const Svg = defineComponent({
  name: 'Svg',
  props: z.object({ children: z.any(), viewBox: z.string(), width: lengthSchema.optional(), height: lengthSchema.optional() }),
  description: 'Responsive native SVG drawing with evaluated shape children and a numeric viewBox.',
  component: ({ props, renderNode }) => {
    const inheritedReport = useContext(reportContext);
    const [errors, setErrors] = useState<Record<string, string>>({});
    const report = useCallback((id: string, error: string | null) => {
      setErrors((previous) => {
        if (error == null ? !Object.hasOwn(previous, id) : previous[id] === error) return previous;
        const next = { ...previous };
        if (error == null) delete next[id]; else next[id] = error;
        return next;
      });
    }, []);
    try {
      const box = viewBox(props.viewBox);
      const width = length(props.width, 'width') ?? '100%';
      const height = length(props.height, 'height');
      const drawing = <svg className="oui-svg" xmlns="http://www.w3.org/2000/svg" viewBox={box} width={width} height={height} preserveAspectRatio="xMidYMid meet" role="img" aria-label="Drawing" fill={ink} style={{ display: 'block', width, height: height ?? 'auto', maxWidth: '100%', color: ink, overflow: 'hidden' }}>
        {children(props.children, renderNode)}
      </svg>;
      if (inheritedReport) return drawing;
      const messages = [...new Set(Object.values(errors))];
      return <div className="oui-svg-container" style={{ minWidth: 0, maxWidth: '100%' }}>
        <reportContext.Provider value={report}>{drawing}</reportContext.Provider>
        {messages.length > 0 && <div className="oui-svg-errors" role="alert" style={{ color: '#b43245', padding: 12, border: '1px solid currentColor', borderRadius: 6, marginTop: 8 }}>
          {messages.map((message) => <div key={message}>{message}</div>)}
        </div>}
      </div>;
    } catch (error) { return failure('Svg', error); }
  },
});

// Property insertion order below is the public positional argument order.
const Circle = defineComponent({
  name: 'Circle',
  props: z.object({ cx: z.number(), cy: z.number(), r: z.number(), fill: z.string().optional(), stroke: z.string().optional(), strokeWidth: z.number().optional(), opacity: z.number().optional() }),
  description: 'Circle in SVG user coordinates.',
  component: ({ props }) => <Shape name="Circle" draw={() => <circle cx={finite(props.cx, 'cx')} cy={finite(props.cy, 'cy')} r={finite(props.r, 'r', 0)} {...appearance(props)} />} />,
});

const Ellipse = defineComponent({
  name: 'Ellipse',
  props: z.object({ cx: z.number(), cy: z.number(), rx: z.number(), ry: z.number(), fill: z.string().optional(), stroke: z.string().optional(), strokeWidth: z.number().optional(), opacity: z.number().optional() }),
  description: 'Ellipse with horizontal and vertical radii.',
  component: ({ props }) => <Shape name="Ellipse" draw={() => <ellipse cx={finite(props.cx, 'cx')} cy={finite(props.cy, 'cy')} rx={finite(props.rx, 'rx', 0)} ry={finite(props.ry, 'ry', 0)} {...appearance(props)} />} />,
});

const Rect = defineComponent({
  name: 'Rect',
  props: z.object({ x: z.number(), y: z.number(), width: z.number(), height: z.number(), fill: z.string().optional(), stroke: z.string().optional(), strokeWidth: z.number().optional(), opacity: z.number().optional(), rx: z.number().optional() }),
  description: 'Rectangle with an optional rounded-corner radius.',
  component: ({ props }) => <Shape name="Rect" draw={() => <rect x={finite(props.x, 'x')} y={finite(props.y, 'y')} width={finite(props.width, 'width', 0)} height={finite(props.height, 'height', 0)} rx={optionalNumber(props.rx, 'rx')} {...appearance(props)} />} />,
});

const Line = defineComponent({
  name: 'Line',
  props: z.object({ x1: z.number(), y1: z.number(), x2: z.number(), y2: z.number(), stroke: z.string().optional(), strokeWidth: z.number().optional(), opacity: z.number().optional() }),
  description: 'Line segment between two points.',
  component: ({ props }) => <Shape name="Line" draw={() => <line x1={finite(props.x1, 'x1')} y1={finite(props.y1, 'y1')} x2={finite(props.x2, 'x2')} y2={finite(props.y2, 'y2')} {...appearance(props)} stroke={paint(props.stroke, 'stroke') ?? ink} />} />,
});

const Path = defineComponent({
  name: 'Path',
  props: z.object({ d: z.string(), fill: z.string().optional(), stroke: z.string().optional(), strokeWidth: z.number().optional(), opacity: z.number().optional() }),
  description: 'Native SVG path supporting standard move, line, curve, arc, and close commands.',
  component: ({ props }) => <Shape name="Path" draw={() => <path d={path(props.d)} {...appearance(props)} />} />,
});

const Polyline = defineComponent({
  name: 'Polyline',
  props: z.object({ points: z.string(), stroke: z.string().optional(), strokeWidth: z.number().optional(), fill: z.string().optional(), opacity: z.number().optional() }),
  description: 'Connected open line segments from x,y coordinate pairs.',
  component: ({ props }) => <Shape name="Polyline" draw={() => <polyline points={points(props.points)} {...appearance(props)} stroke={paint(props.stroke, 'stroke') ?? ink} fill={paint(props.fill, 'fill') ?? 'none'} />} />,
});

const Polygon = defineComponent({
  name: 'Polygon',
  props: z.object({ points: z.string(), fill: z.string().optional(), stroke: z.string().optional(), strokeWidth: z.number().optional(), opacity: z.number().optional() }),
  description: 'Closed polygon from x,y coordinate pairs.',
  component: ({ props }) => <Shape name="Polygon" draw={() => <polygon points={points(props.points)} {...appearance(props)} />} />,
});

const G = defineComponent({
  name: 'G',
  props: z.object({ children: z.any(), transform: z.string().optional(), opacity: z.number().optional() }),
  description: 'SVG drawing group with a numeric SVG transform and opacity.',
  component: ({ props, renderNode }) => <Shape name="G" draw={() => <g transform={transform(props.transform)} opacity={optionalNumber(props.opacity, 'opacity', 0, 1)}>{children(props.children, renderNode)}</g>} />,
});

const SvgText = defineComponent({
  name: 'SvgText',
  props: z.object({ text: z.string(), x: z.number(), y: z.number(), size: z.number().optional(), fill: z.string().optional(), anchor: z.enum(['start', 'middle', 'end']).optional(), weight: z.enum(['normal', 'medium', 'semibold']).optional() }),
  description: 'Escaped SVG text with positioning, font size, color, alignment, and weight.',
  component: ({ props }) => <Shape name="SvgText" draw={() => {
    if (props.anchor != null && !['start', 'middle', 'end'].includes(props.anchor)) return invalid('anchor', 'expected start, middle, or end');
    if (props.weight != null && !['normal', 'medium', 'semibold'].includes(props.weight)) return invalid('weight', 'expected normal, medium, or semibold');
    return <text x={finite(props.x, 'x')} y={finite(props.y, 'y')} fontSize={optionalNumber(props.size, 'size') ?? 14} fill={paint(props.fill, 'fill')} textAnchor={props.anchor ?? 'start'} fontWeight={props.weight === 'semibold' ? 600 : props.weight === 'medium' ? 500 : 400} fontFamily="inherit" xmlSpace="preserve">{boundedString(props.text, 'text')}</text>;
  }} />,
});

export const svgComponents: DefinedComponent<any>[] = [Svg, Circle, Ellipse, Rect, Line, Path, Polyline, Polygon, G, SvgText];
