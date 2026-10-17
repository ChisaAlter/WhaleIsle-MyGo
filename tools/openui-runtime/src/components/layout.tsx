// Independently authored compatibility renderers; only the public DSL contract is reused.
import React, { type CSSProperties } from 'react';
import { defineComponent, type DefinedComponent } from '@openuidev/react-lang';
import { z } from 'zod/v4';

const alignment = z.enum(['start', 'center', 'end', 'stretch']);
const justification = z.enum(['start', 'center', 'end', 'between']);
const length = z.union([z.number(), z.string()]);
const radius = z.union([z.enum(['sm', 'md', 'lg', 'xl', 'full']), z.number()]);
const borderColor = 'var(--oui-border, #d9e0e8)';
const surfaceColor = 'var(--oui-surface, #f3f6fa)';

function bounded(value: unknown, fallback: number, ceiling = 10000): number {
  return typeof value === 'number' && Number.isFinite(value)
    ? Math.min(ceiling, Math.max(0, value))
    : fallback;
}

// Public gap/padding values are four-pixel steps, not pixel lengths.
function spacing(value: unknown, fallback: number): number {
  return bounded(value, fallback, 250) * 4;
}

function safeLength(value: unknown): CSSProperties['width'] {
  if (typeof value === 'number') return bounded(value, 0);
  if (typeof value !== 'string') return undefined;
  const text = value.trim();
  if (/^(?:auto|min-content|max-content|fit-content|stretch)$/.test(text)) return text;
  if (/^(?:\d+(?:\.\d+)?|\.\d+)(?:px|%|em|rem|vh|vw|vmin|vmax|svh|svw|dvh|dvw|ch|ex)$/.test(text)) {
    return Number.parseFloat(text) <= 10000 ? text : undefined;
  }
  return undefined;
}

function safeColor(value: unknown, background = false): string | undefined {
  if (typeof value !== 'string') return undefined;
  const colors: Record<string, string> = {
    ink: 'var(--oui-ink, #172337)', muted: 'var(--oui-muted, #526379)',
    faint: 'var(--oui-faint, #7b899b)', surface: surfaceColor,
    info: '#245fba', success: '#18734d', warning: '#936015', danger: '#b43245',
    purple: '#7945b7', pink: '#b5357a', chart1: '#386cc9', chart2: '#188773',
    chart3: '#ad7226', chart4: '#8451bc', chart5: '#c15371', chart6: '#2484a6',
  };
  const tints: Record<string, string> = {
    info: '#eaf2ff', success: '#e8f6ed', warning: '#fff4df', danger: '#ffedf0',
    purple: '#f2ebff', pink: '#ffedf6',
  };
  const text = value.trim();
  if (background && Object.hasOwn(tints, text)) return tints[text];
  if (Object.hasOwn(colors, text)) return colors[text];
  if (/^#[\da-f]{3,4}$|^#[\da-f]{6}$|^#[\da-f]{8}$/i.test(text)) return text;
  if (/^[a-z]+$/i.test(text)) return text;
  if (/^(?:rgb|rgba|hsl|hsla|hwb|lab|lch|oklab|oklch)\([\d\s.,%/+\-]+\)$/i.test(text)) return text;
  return undefined;
}

function safeRadius(value: unknown): number | undefined {
  if (typeof value === 'number') return bounded(value, 0);
  return typeof value === 'string'
    ? ({ sm: 4, md: 8, lg: 12, xl: 20, full: 9999 } as Record<string, number>)[value]
    : undefined;
}

function align(value: unknown, fallback: CSSProperties['alignItems']): CSSProperties['alignItems'] {
  if (value === 'start') return 'flex-start';
  if (value === 'end') return 'flex-end';
  if (value === 'center' || value === 'stretch') return value;
  return fallback;
}

function justify(value: unknown): CSSProperties['justifyContent'] {
  if (value === 'center') return 'center';
  if (value === 'end') return 'flex-end';
  if (value === 'between') return 'space-between';
  return 'flex-start';
}

const Col = defineComponent({
  name: 'Col',
  props: z.object({ children: z.any(), gap: z.number().optional(), align: alignment.optional() }),
  description: 'Vertical content layout. Spacing values use four-pixel steps.',
  component: ({ props, renderNode }) => (
    <div className="oui-col" style={{ display: 'flex', flexDirection: 'column', minWidth: 0, gap: spacing(props.gap, 4), alignItems: align(props.align, 'stretch') }}>
      {renderNode(props.children)}
    </div>
  ),
});

const Row = defineComponent({
  name: 'Row',
  props: z.object({
    children: z.any(), gap: z.number().optional(), align: alignment.optional(),
    justify: justification.optional(), wrap: z.boolean().optional(),
  }),
  description: 'Horizontal content layout with optional wrapping and alignment.',
  component: ({ props, renderNode }) => (
    <div className="oui-row" style={{ display: 'flex', flexDirection: 'row', minWidth: 0, gap: spacing(props.gap, 3), alignItems: align(props.align, 'center'), justifyContent: justify(props.justify), flexWrap: props.wrap ? 'wrap' : 'nowrap' }}>
      {renderNode(props.children)}
    </div>
  ),
});

const Box = defineComponent({
  name: 'Box',
  props: z.object({
    children: z.any().optional(), direction: z.enum(['row', 'col']).optional(),
    gap: z.number().optional(), padding: z.number().optional(), background: z.string().optional(),
    radius: radius.optional(), border: z.boolean().optional(), width: length.optional(),
    height: length.optional(), flex: z.number().optional(), align: alignment.optional(),
    justify: justification.optional(), wrap: z.boolean().optional(),
  }),
  description: 'Configurable flex box with safe lengths, color, border, and rounded clipping.',
  component: ({ props, renderNode }) => (
    <div className="oui-box" style={{
      display: 'flex', flexDirection: props.direction === 'row' ? 'row' : 'column', minWidth: 0,
      boxSizing: 'border-box', gap: spacing(props.gap, 0), padding: spacing(props.padding, 0),
      backgroundColor: safeColor(props.background, true), borderRadius: safeRadius(props.radius),
      overflow: props.radius == null ? undefined : 'hidden',
      border: props.border ? `1px solid ${borderColor}` : undefined,
      width: safeLength(props.width), height: safeLength(props.height),
      flex: props.flex == null ? undefined : bounded(props.flex, 0, 100),
      alignItems: align(props.align, 'stretch'), justifyContent: justify(props.justify),
      flexWrap: props.wrap ? 'wrap' : 'nowrap',
    }}>
      {renderNode(props.children)}
    </div>
  ),
});

const Grid = defineComponent({
  name: 'Grid',
  props: z.object({ children: z.any(), columns: z.number().optional(), minChildWidth: z.number().optional(), gap: z.number().optional() }),
  description: 'Equal-width grid, optionally responsive to a minimum child width in pixels.',
  component: ({ props, renderNode }) => {
    const columns = Math.max(1, Math.floor(bounded(props.columns, 2, 24)));
    const minWidth = Math.max(1, bounded(props.minChildWidth, 240));
    return (
      <div className="oui-grid" style={{ display: 'grid', minWidth: 0, gap: spacing(props.gap, 4), gridTemplateColumns: props.minChildWidth == null ? `repeat(${columns}, minmax(0, 1fr))` : `repeat(auto-fit, minmax(min(100%, ${minWidth}px), 1fr))` }}>
        {renderNode(props.children)}
      </div>
    );
  },
});

const Card = defineComponent({
  name: 'Card',
  props: z.object({ children: z.any(), variant: z.enum(['outline', 'soft']).optional(), padding: z.number().optional(), gap: z.number().optional() }),
  description: 'Rounded panel with an outline or a soft filled surface.',
  component: ({ props, renderNode }) => (
    <section className="oui-card" style={{ display: 'flex', flexDirection: 'column', minWidth: 0, gap: spacing(props.gap, 3), padding: spacing(props.padding, 4), borderRadius: 12, border: `1px solid ${props.variant === 'soft' ? 'transparent' : borderColor}`, backgroundColor: props.variant === 'soft' ? surfaceColor : 'var(--oui-panel, #fff)', boxSizing: 'border-box' }}>
      {renderNode(props.children)}
    </section>
  ),
});

const Carousel = defineComponent({
  name: 'Carousel',
  props: z.object({ children: z.any(), itemWidth: z.number().optional(), gap: z.number().optional() }),
  description: 'Keyboard-scrollable horizontal collection with snapping items.',
  component: ({ props, renderNode }) => {
    const children = Array.isArray(props.children) ? props.children : props.children == null ? [] : [props.children];
    const itemWidth = Math.max(1, bounded(props.itemWidth, 240));
    return (
      <div className="oui-carousel" role="region" aria-label="Scrollable items" tabIndex={0} style={{ display: 'flex', minWidth: 0, maxWidth: '100%', overflowX: 'auto', gap: spacing(props.gap, 3), paddingBottom: 8, scrollSnapType: 'x proximity', overscrollBehaviorX: 'contain' }}>
        {children.map((child, index) => (
          <div key={index} style={{ flex: `0 0 ${itemWidth}px`, maxWidth: '100%', minWidth: 0, scrollSnapAlign: 'start' }}>
            {renderNode(child)}
          </div>
        ))}
      </div>
    );
  },
});

const Divider = defineComponent({
  name: 'Divider', props: z.object({}), description: 'Horizontal section separator.',
  component: () => <hr className="oui-divider" style={{ width: '100%', margin: 0, border: 0, borderTop: `1px solid ${borderColor}` }} />,
});

const Spacer = defineComponent({
  name: 'Spacer', props: z.object({}), description: 'Flexible space between layout siblings.',
  component: () => <div className="oui-spacer" aria-hidden="true" style={{ flex: '1 1 0', minWidth: 0, minHeight: 0 }} />,
});

export const layoutComponents: DefinedComponent<any>[] = [Col, Row, Box, Grid, Card, Carousel, Divider, Spacer];
