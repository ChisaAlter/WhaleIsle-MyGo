// Independently authored from the public component signatures, not upstream renderers.
import React, { useId, useState, type ReactNode } from 'react';
import { defineComponent, type DefinedComponent } from '@openuidev/react-lang';
import { z } from 'zod/v4';

const palette = ['#386cc9', '#188773', '#ad7226', '#8451bc', '#c15371', '#2484a6'];
const border = 'var(--oui-border, #d9e0e8)';
const muted = 'var(--oui-muted, #526379)';
const cellStyle = { padding: '8px 12px', borderBottom: `1px solid ${border}`, verticalAlign: 'top' } as const;
const chartSchema = z.object({
  // Key insertion order is the public positional calling convention.
  type: z.enum(['bar', 'line', 'area', 'pie', 'donut', 'scatter']),
  data: z.array(z.record(z.string(), z.union([z.string(), z.number()]))),
  x: z.string().optional(),
  series: z.array(z.string()).optional(),
  format: z.enum(['number', 'currency', 'percent']).optional(),
  currency: z.string().optional(),
  stacked: z.boolean().optional(),
  height: z.number().optional(),
});
type ChartProps = z.infer<typeof chartSchema>;
type Point = { value: number; bottom: number; top: number } | null;
type Prepared = {
  series: string[];
  labels: string[];
  xNumbers: number[];
  values: (number | null)[][];
  missing: number;
  xKey?: string;
};
type DatumAddress = { series: number; row: number };
type ChartInteraction = {
  tooltipId: string;
  focused: DatumAddress | null;
  active: DatumAddress | null;
  hover: (address: DatumAddress) => void;
  focus: (address: DatumAddress) => void;
  blur: () => void;
  dismiss: () => void;
};

function sameDatum(left: DatumAddress | null, right: DatumAddress): boolean {
  return left?.series === right.series && left.row === right.row;
}

function Marker({ address, label, interaction, children }: {
  address: DatumAddress; label: string; interaction: ChartInteraction; children: ReactNode;
}) {
  const focused = sameDatum(interaction.focused, address);
  return <g data-chart-marker="true" role="img" tabIndex={-1} aria-label={label} aria-describedby={sameDatum(interaction.active, address) ? interaction.tooltipId : undefined}
    onPointerEnter={() => interaction.hover(address)}
    onPointerDown={event => { event.currentTarget.focus(); interaction.focus(address); }}
    onFocus={() => interaction.focus(address)} onBlur={interaction.blur}
    style={{ cursor: 'crosshair', outline: focused ? '2px solid currentColor' : 'none', outlineOffset: 3 }}>
    {children}
  </g>;
}

function navigateMarkers(event: React.KeyboardEvent<SVGSVGElement>, interaction: ChartInteraction) {
  if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); interaction.dismiss(); return; }
  const keys = ['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'];
  if (!keys.includes(event.key)) return;
  const markers = Array.from(event.currentTarget.querySelectorAll<SVGGElement>('[data-chart-marker="true"]'));
  if (!markers.length) return;
  event.preventDefault();
  event.stopPropagation();
  const current = markers.indexOf(event.target as SVGGElement);
  const backward = event.key === 'ArrowLeft' || event.key === 'ArrowUp';
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? markers.length - 1
    : current === -1 ? (backward ? markers.length - 1 : 0)
    : (current + (backward ? -1 : 1) + markers.length) % markers.length;
  markers[next].focus();
}

function Tooltip({ id, props, data, address, display }: {
  id: string; props: ChartProps; data: Prepared; address: DatumAddress | null; display: (value: number) => string;
}) {
  // Store only row/series addresses in UI state. Always read values from the
  // current data, so streamed or state-driven updates cannot show stale numbers.
  const value = address ? data.values[address.series]?.[address.row] : undefined;
  if (!address || value === null || value === undefined) return <div id={id} role="tooltip" hidden />;
  let extra: ReactNode = null;
  if (props.stacked) {
    if (data.values.some(series => series[address.row] === null)) return <div id={id} role="tooltip" hidden />;
    const base = data.values.slice(0, address.series).reduce((sum, series) => {
      const previous = series[address.row];
      return previous !== null && (previous >= 0) === (value >= 0) ? sum + previous : sum;
    }, 0);
    const endpoint = base + value;
    if (!Number.isFinite(base) || !Number.isFinite(endpoint)) return <div id={id} role="tooltip" hidden />;
    extra = <div>Stack range: {String(base)} to {String(endpoint)}</div>;
  } else if (props.type === 'pie' || props.type === 'donut') {
    if (value <= 0) return <div id={id} role="tooltip" hidden />;
    const values = data.values[address.series] as number[];
    const maximum = Math.max(...values);
    const total = maximum === 0 ? 0 : values.reduce((sum, number) => sum + number / maximum, 0);
    extra = total > 0 ? <div>Share: {formatValue({ ...props, format: 'percent' })(value / maximum / total * 100)}</div> : <div>No nonzero total; a percentage is undefined.</div>;
  }
  return <div id={id} role="tooltip" style={{ marginTop: 8, padding: '9px 12px', border: `1px solid ${border}`, borderRadius: 8, background: 'var(--oui-surface, #f3f6fa)', fontSize: 12, fontVariantNumeric: 'tabular-nums', overflowWrap: 'anywhere' }}>
    <div><strong>{data.series[address.series]}</strong> · {data.xKey ?? 'Row'}: {data.labels[address.row]} · row {address.row + 1}</div>
    <div>Value: {display(value)}</div>
    <div style={{ color: muted }}>Exact supplied value: {String(value)}</div>
    {extra}
  </div>;
}

function Notice({ children, error = false }: { children: ReactNode; error?: boolean }) {
  return <div role={error ? 'alert' : 'status'} style={{ padding: 12, border: `1px solid ${error ? '#b43245' : border}`, borderRadius: 8, color: error ? '#b43245' : muted }}>{children}</div>;
}

function formatValue(props: ChartProps): (value: number) => string {
  if (props.format === 'currency') {
    const currency = props.currency ?? 'USD';
    if (!/^[A-Za-z]{3}$/.test(currency)) throw new Error('Chart currency must be a three-letter currency code, such as EUR.');
    const formatter = new Intl.NumberFormat(undefined, { style: 'currency', currency: currency.toUpperCase() });
    return value => formatter.format(value);
  }
  const formatter = new Intl.NumberFormat(undefined, { maximumSignificantDigits: 12 });
  const scientific = new Intl.NumberFormat(undefined, { notation: 'scientific', maximumSignificantDigits: 8 });
  return value => {
    const absolute = Math.abs(value);
    const number = absolute >= 1e12 || (absolute !== 0 && absolute < 1e-6) ? scientific.format(value) : formatter.format(value);
    return `${number}${props.format === 'percent' ? '%' : ''}`;
  };
}

function prepare(props: ChartProps): Prepared {
  const keys = [...new Set(props.data.flatMap(row => Object.keys(row)))];
  const numericKeys = keys.filter(key => props.data.some(row => typeof row[key] === 'number'));
  const xKey = props.x ?? (props.type === 'scatter'
    ? numericKeys[0]
    : keys.find(key => props.data.some(row => typeof row[key] === 'string')));
  const series = props.series ?? numericKeys.filter(key => key !== xKey);
  if (!series.length) throw new Error('Chart has no numeric value series. Supply at least one numeric column or name the series explicitly.');
  if (series.length > 32) throw new Error('Chart supports up to 32 value series. Reduce the number of series to display.');
  if (new Set(series).size !== series.length) throw new Error('Chart series names must be unique.');
  if (props.type === 'scatter' && xKey === undefined) throw new Error('Scatter charts require a numeric x column.');
  const labels: string[] = [];
  const xNumbers: number[] = [];
  for (const [index, row] of props.data.entries()) {
    if (xKey !== undefined && !Object.hasOwn(row, xKey)) throw new Error(`Chart row ${index + 1} has no x value for “${xKey}”.`);
    const x = xKey === undefined ? index + 1 : row[xKey];
    if (props.type === 'scatter' && (typeof x !== 'number' || !Number.isFinite(x))) throw new Error(`Scatter x column “${xKey}” must contain a finite number in every row.`);
    labels.push(String(x));
    xNumbers.push(typeof x === 'number' ? x : index + 1);
  }
  let missing = 0;
  const values = series.map(key => {
    if (!props.data.some(row => Object.hasOwn(row, key))) throw new Error(`Chart series “${key}” is absent from the data.`);
    return props.data.map((row, index) => {
      if (!Object.hasOwn(row, key)) { missing += 1; return null; }
      const value = row[key];
      if (typeof value !== 'number' || !Number.isFinite(value)) throw new Error(`Chart series “${key}”, row ${index + 1}, must be a finite number.`);
      return value;
    });
  });
  return { series, labels, xNumbers, values, missing, xKey };
}

function extent(values: number[], includeZero: boolean): [number, number] {
  let low = includeZero ? 0 : Infinity;
  let high = includeZero ? 0 : -Infinity;
  for (const value of values) { low = Math.min(low, value); high = Math.max(high, value); }
  if (!Number.isFinite(low) || !Number.isFinite(high)) return [-1, 1];
  if (low === high) {
    const pad = low === 0 ? 1 : Math.max(Number.MIN_VALUE, Math.abs(low) * 0.1);
    return [Number.isFinite(low - pad) ? low - pad : low, Number.isFinite(high + pad) ? high + pad : high];
  }
  return [low, high];
}

function interpolate(value: number, low: number, high: number): number {
  // Scaling all three terms avoids overflowing the subtraction for large signed data.
  const scale = Math.max(Math.abs(low), Math.abs(high), 1);
  return (value / scale - low / scale) / (high / scale - low / scale);
}

function tickValues([low, high]: [number, number]): number[] {
  return Array.from({ length: 5 }, (_, index) => low * (1 - index / 4) + high * index / 4);
}

function shortLabel(value: string): string {
  return value.length > 20 ? `${value.slice(0, 19)}…` : value;
}

function Legend({ names }: { names: string[] }) {
  return <ul aria-label="Chart legend" style={{ display: 'flex', flexWrap: 'wrap', gap: '6px 16px', listStyle: 'none', margin: '8px 0', padding: 0, fontSize: 12 }}>
    {names.map((name, index) => <li key={`${index}-${name}`} style={{ display: 'flex', alignItems: 'center', gap: 6 }}><span aria-hidden="true" style={{ width: 10, height: 10, flex: '0 0 10px', background: palette[index % palette.length], borderRadius: 2 }} />{name}</li>)}
  </ul>;
}

function SourceData({ data }: { data: Prepared }) {
  return <details style={{ marginTop: 10, fontSize: 12 }}>
    <summary style={{ cursor: 'pointer', color: muted }}>View chart data</summary>
    <div style={{ overflowX: 'auto', maxHeight: 320 }}><table style={{ width: '100%', borderCollapse: 'collapse', fontVariantNumeric: 'tabular-nums' }}>
      <caption style={{ textAlign: 'left', padding: 8 }}>Unformatted values supplied to this chart</caption>
      <thead><tr><th scope="col" style={{ ...cellStyle, textAlign: 'left' }}>{data.xKey ?? 'Row'}</th>{data.series.map(key => <th scope="col" key={key} style={{ ...cellStyle, textAlign: 'right' }}>{key}</th>)}</tr></thead>
      <tbody>{data.labels.map((label, row) => <tr key={row}><th scope="row" style={{ ...cellStyle, textAlign: 'left', fontWeight: 400 }}>{label}</th>{data.values.map((values, column) => <td key={column} style={{ ...cellStyle, textAlign: 'right' }}>{values[row] === null ? 'Missing' : String(values[row])}</td>)}</tr>)}</tbody>
    </table></div>
  </details>;
}

function Cartesian({ props, data, display, interaction }: { props: ChartProps; data: Prepared; display: (value: number) => string; interaction: ChartInteraction }) {
  const width = 680;
  const height = Math.max(160, props.height ?? 320);
  const left = 88, right = width - 22, top = 18, bottom = height - 55;
  const positive = Array(data.labels.length).fill(0) as number[];
  const negative = Array(data.labels.length).fill(0) as number[];
  // A missing member makes a stacked total unknown. Leave that entire stack as
  // a gap instead of implicitly substituting zero for the missing member.
  const incomplete = data.labels.map((_, row) => data.values.some(series => series[row] === null));
  const points: Point[][] = data.values.map(values => values.map((value, row) => {
    if (value === null || (props.stacked && incomplete[row])) return null;
    const totals = value >= 0 ? positive : negative;
    const base = props.stacked ? totals[row] : 0;
    const end = base + value;
    if (!Number.isFinite(end)) throw new Error('Stacked chart totals exceed the supported finite number range.');
    if (props.stacked) totals[row] = end;
    return { value, bottom: base, top: end };
  }));
  if (points.every(series => series.every(point => point === null))) return <Notice>No complete stacks are available to draw. Each row needs a value for every selected series.</Notice>;
  const yExtent = extent(points.flatMap(series => series.flatMap(point => point === null ? [] : props.stacked ? [point.bottom, point.top] : [point.value])), props.type === 'bar' || props.type === 'area');
  const xExtent = extent(data.xNumbers, false);
  const y = (value: number) => bottom - interpolate(value, ...yExtent) * (bottom - top);
  const categoryWidth = (right - left) / data.labels.length;
  const x = (row: number) => props.type === 'scatter'
    ? left + interpolate(data.xNumbers[row], ...xExtent) * (right - left)
    : props.type === 'bar'
      ? left + categoryWidth * (row + 0.5)
      : data.labels.length === 1 ? (left + right) / 2 : left + row / (data.labels.length - 1) * (right - left);
  const zeroY = yExtent[0] <= 0 && yExtent[1] >= 0 ? y(0) : bottom;
  const labelStride = Math.max(1, Math.ceil(data.labels.length / 8));
  const title = `${props.stacked ? 'Stacked ' : ''}${props.type} chart: ${data.series.join(', ')}`;
  return <div style={{ overflowX: 'auto' }}><svg role="group" tabIndex={0} aria-label={`${title}. Use arrow keys to inspect values; Escape hides details.`} onKeyDown={event => navigateMarkers(event, interaction)} viewBox={`0 0 ${width} ${height}`} style={{ display: 'block', width: '100%', minWidth: 360, height: props.height ?? 320, overflow: 'visible' }}>
    <title>{title}</title>
    <desc>{data.labels.length} data rows. Exact values are available in the expandable data table following the chart.</desc>
    {tickValues(yExtent).map((value, index) => <g key={`y-${index}`}><line x1={left} x2={right} y1={y(value)} y2={y(value)} stroke={border} /><text x={left - 8} y={y(value)} dy="0.35em" textAnchor="end" fontSize={11} fill={muted}>{display(value)}</text></g>)}
    <line x1={left} x2={left} y1={top} y2={bottom} stroke={muted} />
    <line x1={left} x2={right} y1={zeroY} y2={zeroY} stroke={muted} strokeWidth={1.25} />
    {props.type === 'scatter'
      ? tickValues(xExtent).map((value, index) => <text key={`x-${index}`} x={left + interpolate(value, ...xExtent) * (right - left)} y={bottom + 20} textAnchor="middle" fontSize={11} fill={muted}>{formatValue({ ...props, format: 'number' })(value)}</text>)
      : data.labels.map((label, row) => row % labelStride === 0 || row === data.labels.length - 1 ? <text key={`x-${row}`} x={x(row)} y={bottom + 20} textAnchor="middle" fontSize={11} fill={muted}><title>{label}</title>{shortLabel(label)}</text> : null)}
    {data.xKey !== undefined && <text x={(left + right) / 2} y={height - 7} textAnchor="middle" fontSize={11} fill={muted}>{data.xKey}</text>}
    {points.map((series, seriesIndex) => {
      const color = palette[seriesIndex % palette.length];
      const name = data.series[seriesIndex];
      if (props.type === 'bar') {
        const groupWidth = categoryWidth * 0.76;
        const barWidth = groupWidth / (props.stacked ? 1 : points.length);
        return <g key={name}>{series.map((point, row) => {
          if (point === null) return null;
          const start = x(row) - groupWidth / 2 + (props.stacked ? 0 : seriesIndex * barWidth);
          const span = Math.max(0.1, barWidth - Math.min(2, barWidth * 0.1));
          const label = `${data.labels[row]}, row ${row + 1} · ${name}: ${display(point.value)}. Exact value: ${String(point.value)}.${props.stacked ? ` Stack range: ${String(point.bottom)} to ${String(point.top)}.` : ''}`;
          return <Marker key={row} address={{ series: seriesIndex, row }} label={label} interaction={interaction}>
            {point.value === 0
              ? <><line x1={start} x2={start + span} y1={y(point.bottom)} y2={y(point.bottom)} stroke={color} strokeWidth={2} /><rect x={start} y={y(point.bottom) - 5} width={span} height={10} fill="transparent" stroke="none" /></>
              : <rect x={start} y={Math.min(y(point.bottom), y(point.top))} width={span} height={Math.abs(y(point.bottom) - y(point.top))} fill={color} />}
          </Marker>;
        })}</g>;
      }
      const segments: { point: Exclude<Point, null>; row: number }[][] = [];
      let segment: { point: Exclude<Point, null>; row: number }[] = [];
      for (const [row, point] of series.entries()) {
        if (point === null) { if (segment.length) segments.push(segment); segment = []; }
        else segment.push({ point, row });
      }
      if (segment.length) segments.push(segment);
      return <g key={name}>
        {props.type === 'area' && segments.map((part, index) => {
          const upper = part.map(({ point, row }) => `${x(row)},${y(point.top)}`).join(' L ');
          const lower = [...part].reverse().map(({ point, row }) => `${x(row)},${y(point.bottom)}`).join(' L ');
          return <path key={`area-${index}`} d={`M ${upper} L ${lower} Z`} fill={color} fillOpacity={props.stacked ? 0.55 : 0.2} />;
        })}
        {props.type !== 'scatter' && segments.map((part, index) => <polyline key={`line-${index}`} points={part.map(({ point, row }) => `${x(row)},${y(point.top)}`).join(' ')} fill="none" stroke={color} strokeWidth={2} strokeLinejoin="round" />)}
        {series.map((point, row) => point === null ? null : <Marker key={`point-${row}`} address={{ series: seriesIndex, row }} label={`${data.labels[row]}, row ${row + 1} · ${name}: ${display(point.value)}. Exact value: ${String(point.value)}.${props.stacked ? ` Stack range: ${String(point.bottom)} to ${String(point.top)}.` : ''}`} interaction={interaction}>
          <circle cx={x(row)} cy={y(point.top)} r={props.type === 'scatter' ? 4 : 2.5} fill={color} stroke="var(--oui-surface, white)" strokeWidth={1} />
          <circle cx={x(row)} cy={y(point.top)} r={9} fill="transparent" stroke="none" />
        </Marker>)}
      </g>;
    })}
  </svg></div>;
}

function polar(cx: number, cy: number, radius: number, angle: number): [number, number] {
  return [cx + Math.cos(angle) * radius, cy + Math.sin(angle) * radius];
}

function sector(start: number, end: number, inner: number): string {
  const outer = 92, cx = 120, cy = 110;
  const startOuter = polar(cx, cy, outer, start), endOuter = polar(cx, cy, outer, end);
  const big = end - start > Math.PI ? 1 : 0;
  if (!inner) return `M ${cx},${cy} L ${startOuter.join(',')} A ${outer},${outer} 0 ${big} 1 ${endOuter.join(',')} Z`;
  const startInner = polar(cx, cy, inner, start), endInner = polar(cx, cy, inner, end);
  return `M ${startOuter.join(',')} A ${outer},${outer} 0 ${big} 1 ${endOuter.join(',')} L ${endInner.join(',')} A ${inner},${inner} 0 ${big} 0 ${startInner.join(',')} Z`;
}

function Circular({ props, data, display, interaction }: { props: ChartProps; data: Prepared; display: (value: number) => string; interaction: ChartInteraction }) {
  if (data.values.some(values => values.some(value => value !== null && value < 0))) throw new Error(`${props.type === 'donut' ? 'Donut' : 'Pie'} charts cannot represent negative values. Choose bar, line, area, or scatter.`);
  if (data.missing) throw new Error('Pie and donut charts require a numeric value for every row in each selected series. Missing values are not converted to zero.');
  return <div style={{ display: 'flex', flexWrap: 'wrap', gap: 20 }}>{data.series.map((name, seriesIndex) => {
    const values = data.values[seriesIndex] as number[];
    const maximum = Math.max(...values);
    const scaledTotal = maximum === 0 ? 0 : values.reduce((sum, value) => sum + value / maximum, 0);
    let angle = -Math.PI / 2;
    const positiveCount = values.filter(value => value > 0).length;
    return <div key={name} style={{ flex: '1 1 220px', minWidth: 0 }}>
      <div style={{ textAlign: 'center', fontWeight: 600 }}>{name}</div>
      {scaledTotal === 0 ? <Notice>All values in “{name}” are zero. There are no nonzero slices to draw.</Notice> : <svg role="group" tabIndex={0} aria-label={`${props.type} chart for ${name}. Use arrow keys to inspect slices; Escape hides details.`} onKeyDown={event => navigateMarkers(event, interaction)} viewBox="0 0 240 220" style={{ display: 'block', width: '100%', height: props.height ?? 240 }}>
        <title>{`${props.type} chart for ${name}`}</title>
        <desc>Each slice is a supplied row. Values and percentages are listed below.</desc>
        {values.map((value, row) => {
          const share = value / maximum / scaledTotal;
          const start = angle;
          angle += share * Math.PI * 2;
          if (value === 0) return null;
          const color = palette[row % palette.length];
          const label = `${name} · ${data.labels[row]}, row ${row + 1}: ${display(value)} (${(share * 100).toFixed(1)}%). Exact value: ${String(value)}.`;
          return <Marker key={row} address={{ series: seriesIndex, row }} label={label} interaction={interaction}>
            {positiveCount === 1 ? props.type === 'donut'
              ? <circle cx={120} cy={110} r={68} fill="none" stroke={color} strokeWidth={48} />
              : <circle cx={120} cy={110} r={92} fill={color} />
              : <path d={sector(start, angle, props.type === 'donut' ? 44 : 0)} fill={color} stroke="var(--oui-surface, white)" strokeWidth={1} />}
          </Marker>;
        })}
      </svg>}
      <ul aria-label={`${name} slices`} style={{ listStyle: 'none', padding: 0, margin: '8px 0', fontSize: 12 }}>
        {values.map((value, row) => <li key={row} style={{ display: 'flex', gap: 7, alignItems: 'baseline', padding: '3px 0' }}><span aria-hidden="true" style={{ width: 9, height: 9, flex: '0 0 9px', background: palette[row % palette.length], borderRadius: 2 }} /><span style={{ flex: 1 }}>{data.labels[row]}</span><span style={{ fontVariantNumeric: 'tabular-nums' }}>{display(value)}{scaledTotal > 0 ? ` (${(value / maximum / scaledTotal * 100).toFixed(1)}%)` : ''}</span></li>)}
      </ul>
    </div>;
  })}</div>;
}

function ChartView({ props }: { props: ChartProps }) {
  const tooltipId = useId();
  const [hovered, setHovered] = useState<DatumAddress | null>(null);
  const [focused, setFocused] = useState<DatumAddress | null>(null);
  const [dismissed, setDismissed] = useState(false);
  const interaction: ChartInteraction = {
    tooltipId, focused, active: dismissed ? null : hovered ?? focused,
    hover: address => { setHovered(address); setDismissed(false); },
    focus: address => { setFocused(address); setHovered(null); setDismissed(false); },
    blur: () => setFocused(null),
    dismiss: () => setDismissed(true),
  };
  const validated = chartSchema.safeParse(props);
  if (!validated.success) return <Notice error>Invalid Chart input: {validated.error.issues.map(issue => `${issue.path.join('.') || 'Chart'}: ${issue.message}`).join('; ')}</Notice>;
  props = validated.data;
  if (!props.data.length) return <Notice>No chart data was supplied.</Notice>;
  if (props.data.length > 5000) return <Notice error>Chart supports up to 5,000 rows. Filter or aggregate the data before displaying it.</Notice>;
  if (props.height !== undefined && props.height <= 0) return <Notice error>Chart height must be a positive number.</Notice>;
  if (props.stacked && props.type !== 'bar' && props.type !== 'line' && props.type !== 'area') return <Notice error>The public Chart contract does not specify how stacked {props.type} should be drawn. This combination is not implemented in this viewer.</Notice>;
  try {
    const data = prepare(props);
    const display = formatValue(props);
    // Call the pure drawing functions here so invalid data produces a local error.
    const circular = props.type === 'pie' || props.type === 'donut';
    const drawing = circular ? Circular({ props, data, display, interaction }) : Cartesian({ props, data, display, interaction });
    const hasMarks = circular ? data.values.some(values => values.some(value => value !== null && value > 0))
      : props.stacked ? data.labels.some((_, row) => data.values.every(values => values[row] !== null))
      : data.values.some(values => values.some(value => value !== null));
    return <figure onPointerLeave={() => setHovered(null)} onKeyDown={event => { if (event.key === 'Escape') interaction.dismiss(); }} style={{ margin: 0, width: '100%', minWidth: 0 }}>
      {drawing}
      {!circular && <Legend names={data.series} />}
      {hasMarks && <p style={{ fontSize: 11, color: muted, margin: '6px 0' }}>Hover a mark for its values, or focus a chart and use arrow keys. Home/End jump to the first/last mark; Escape hides details.</p>}
      <Tooltip id={tooltipId} props={props} data={data} address={interaction.active} display={display} />
      {data.missing > 0 && <p role="status" style={{ fontSize: 12, color: muted, margin: '8px 0' }}>{data.missing} missing value{data.missing === 1 ? ' is' : 's are'} shown as gaps, not zeros.{props.stacked ? ' Rows with incomplete stacks are omitted from the drawing; supplied values remain in the data table.' : ''}</p>}
      <SourceData data={data} />
    </figure>;
  } catch (error) {
    return <Notice error>{error instanceof Error ? error.message : 'Chart data could not be displayed.'}</Notice>;
  }
}

const Chart = defineComponent({
  name: 'Chart', props: chartSchema,
  description: 'Data charts: bar, line, area, pie, donut, and scatter. Hover or focus real data marks; arrow keys inspect exact values. Stacking is available for bar, line, and area. The public contract does not specify stacked pie, donut, or scatter semantics; these combinations are not implemented. Pie and donut render one chart per selected series. Missing values are gaps; negative pie or donut values are rejected.',
  component: ({ props }) => <ChartView props={props} />,
});

function tableCell(value: unknown, renderNode: (value: unknown) => ReactNode): ReactNode {
  if (value === null || value === undefined) return null;
  if (typeof value === 'string' || typeof value === 'number') return String(value);
  if (typeof value === 'boolean') return value ? 'true' : 'false';
  if (Array.isArray(value)) return value.map((item, index) => <React.Fragment key={index}>{tableCell(item, renderNode)}</React.Fragment>);
  if (typeof value === 'object' && 'type' in value && value.type === 'element' && 'typeName' in value && typeof value.typeName === 'string') return renderNode(value);
  try { return JSON.stringify(value); } catch { return '[Value could not be displayed]'; }
}

const tableSchema = z.object({ columns: z.array(z.string()), rows: z.array(z.array(z.any())) });
const Table = defineComponent({
  name: 'Table', props: tableSchema,
  description: 'Each row lists its cells in column order. Component cells render inline and numeric columns align right.',
  component: ({ props, renderNode }) => {
    const validated = tableSchema.safeParse(props);
    if (!validated.success) return <Notice error>Table requires a list of column names and a list of row arrays.</Notice>;
    const { columns, rows } = validated.data;
    if (!columns.length) return <Notice error>Table requires at least one column.</Notice>;
    if (rows.some(row => row.length > columns.length)) return <Notice error>A table row contains more cells than the declared columns. Add the missing column labels.</Notice>;
    if (rows.length * columns.length > 50000) return <Notice error>Table supports up to 50,000 cells. Reduce the supplied rows or columns.</Notice>;
    const numeric = columns.map((_, index) => {
      const populated = rows.map(row => row[index]).filter(value => value !== null && value !== undefined);
      return populated.length > 0 && populated.every(value => typeof value === 'number');
    });
    return <div style={{ width: '100%', overflowX: 'auto', border: `1px solid ${border}`, borderRadius: 8 }}><table style={{ width: '100%', borderCollapse: 'collapse', fontVariantNumeric: 'tabular-nums' }}>
      <thead><tr>{columns.map((column, index) => <th key={index} scope="col" style={{ ...cellStyle, textAlign: numeric[index] ? 'right' : 'left', background: 'var(--oui-surface, #f3f6fa)', fontWeight: 600 }}>{column}</th>)}</tr></thead>
      <tbody>{rows.length === 0 ? <tr><td colSpan={columns.length} style={{ ...cellStyle, color: muted }}>No rows to display.</td></tr> : rows.map((row, rowIndex) => <tr key={rowIndex}>{columns.map((_, column) => <td key={column} style={{ ...cellStyle, textAlign: numeric[column] ? 'right' : 'left' }}>{tableCell(row[column], renderNode)}</td>)}</tr>)}</tbody>
    </table></div>;
  },
});

// Sparkline has no signature in the pinned public contract and is not invented here.
export const chartComponents: DefinedComponent[] = [Chart, Table];
