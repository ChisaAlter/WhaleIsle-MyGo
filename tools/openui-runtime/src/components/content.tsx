// Original, sandbox-safe renderers for the published content component interfaces.
import React, { createContext, useContext, useEffect, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import { defineComponent, type DefinedComponent } from '@openuidev/react-lang';
import ReactMarkdown, { type Components } from 'react-markdown';
import remarkGfm from 'remark-gfm';
import katex from 'katex';
import { icons, type LucideIcon } from 'lucide-react';
import { z } from 'zod/v4';
import { HighlightedCodeBlock, MarkdownCodeBlock } from './highlight';
import { HostLink } from './native-action-ui';

const ink = 'var(--oui-ink, #172337)';
const muted = 'var(--oui-muted, #526379)';
const surface = 'var(--oui-surface, #f3f6fa)';
const border = 'var(--oui-border, #d9e0e8)';
const mono = 'ui-monospace, SFMono-Regular, Consolas, monospace';

function safeColor(value: unknown, fallback = ink): string {
  if (typeof value !== 'string') return fallback;
  const colors: Record<string, string> = {
    ink, muted, faint: 'var(--oui-faint, #7b899b)', surface,
    info: '#245fba', success: '#18734d', warning: '#936015', danger: '#b43245',
    purple: '#7945b7', pink: '#b5357a', chart1: '#386cc9', chart2: '#188773',
    chart3: '#ad7226', chart4: '#8451bc', chart5: '#c15371', chart6: '#2484a6',
  };
  const text = value.trim();
  if (Object.hasOwn(colors, text)) return colors[text];
  if (/^#[\da-f]{3,4}$|^#[\da-f]{6}$|^#[\da-f]{8}$/i.test(text)) return text;
  if (/^[a-z]+$/i.test(text)) return text;
  if (/^(?:rgb|rgba|hsl|hsla|hwb|lab|lch|oklab|oklch)\([\d\s.,%/+\-]+\)$/i.test(text)) return text;
  return fallback;
}

function BlockedLink({ children }: { children: ReactNode }) {
  return (
    <span role="link" aria-disabled="true" title="Opening links is unavailable in this isolated viewer." style={{ color: '#245fba', textDecoration: 'underline', textDecorationStyle: 'dotted' }}>
      {children}<small style={{ textDecoration: 'none', marginInlineStart: 4, color: muted }}>(link disabled)</small>
    </span>
  );
}

const markdownComponents: Components = {
  a: ({ children }) => <BlockedLink>{children}</BlockedLink>,
  img: ({ alt }) => <span role="note" style={{ color: muted }}>[Image unavailable: {alt || 'external image'}]</span>,
  p: ({ children }) => <p style={{ margin: '0 0 0.85em', overflowWrap: 'anywhere' }}>{children}</p>,
  h1: ({ children }) => <h1 style={{ fontSize: '1.8em', lineHeight: 1.25, margin: '1em 0 0.5em' }}>{children}</h1>,
  h2: ({ children }) => <h2 style={{ fontSize: '1.5em', lineHeight: 1.3, margin: '1em 0 0.5em' }}>{children}</h2>,
  h3: ({ children }) => <h3 style={{ fontSize: '1.25em', lineHeight: 1.35, margin: '1em 0 0.5em' }}>{children}</h3>,
  h4: ({ children }) => <h4 style={{ fontSize: '1.1em', margin: '1em 0 0.5em' }}>{children}</h4>,
  h5: ({ children }) => <h5 style={{ fontSize: '1em', margin: '1em 0 0.5em' }}>{children}</h5>,
  h6: ({ children }) => <h6 style={{ fontSize: '0.95em', margin: '1em 0 0.5em' }}>{children}</h6>,
  blockquote: ({ children }) => <blockquote style={{ margin: '1em 0', paddingInlineStart: 16, borderInlineStart: `3px solid ${border}`, color: muted }}>{children}</blockquote>,
  pre: ({ node, children }) => <MarkdownCodeBlock node={node}>{children}</MarkdownCodeBlock>,
  code: ({ children, className }) => <code className={className} style={{ fontFamily: mono, fontSize: '0.9em', backgroundColor: surface, borderRadius: 3, padding: '0.1em 0.2em' }}>{children}</code>,
  ul: ({ children }) => <ul style={{ margin: '0.7em 0', paddingInlineStart: 24 }}>{children}</ul>,
  ol: ({ children, start }) => <ol start={start} style={{ margin: '0.7em 0', paddingInlineStart: 24 }}>{children}</ol>,
  li: ({ children }) => <li style={{ marginBlock: 4 }}>{children}</li>,
  table: ({ children }) => <div style={{ overflowX: 'auto', maxWidth: '100%' }}><table style={{ width: '100%', borderCollapse: 'collapse', marginBlock: 12 }}>{children}</table></div>,
  th: ({ children, align }) => <th style={{ padding: '8px 12px', border: `1px solid ${border}`, backgroundColor: surface, textAlign: align === 'center' || align === 'right' ? align : 'left' }}>{children}</th>,
  td: ({ children, align }) => <td style={{ padding: '8px 12px', border: `1px solid ${border}`, textAlign: align === 'center' || align === 'right' ? align : 'left' }}>{children}</td>,
  hr: () => <hr style={{ border: 0, borderTop: `1px solid ${border}`, marginBlock: 16 }} />,
  input: ({ checked }) => <input type="checkbox" checked={Boolean(checked)} disabled aria-label="Markdown task status" />,
};

const inlineMarkdownComponents: Components = {
  ...markdownComponents,
  p: ({ children }) => <span>{children}</span>,
};

function SafeMarkdown({ text, inline = false }: { text: string; inline?: boolean }) {
  // Preserve spaces around separately-rendered strings in Text's mixed-content arrays.
  const leading = inline ? text.match(/^\s*/)?.[0] || '' : '';
  const trailing = inline && leading.length < text.length ? text.match(/\s*$/)?.[0] || '' : '';
  const content = inline ? text.slice(leading.length, trailing ? -trailing.length : undefined) : text;
  return <>{leading}<ReactMarkdown
    remarkPlugins={[remarkGfm]}
    skipHtml
    urlTransform={() => ''}
    components={inline ? inlineMarkdownComponents : markdownComponents}
    allowedElements={inline ? ['p', 'strong', 'em', 'code', 'del', 'a', 'img', 'br'] : undefined}
    unwrapDisallowed={inline}
  >{content}</ReactMarkdown>{trailing}</>;
}

function NamedIcon({ name, size = 16, color }: { name: string; size?: number; color?: string }) {
  const key = name.trim().split(/[-_\s]+/).map((part) => part.charAt(0).toUpperCase() + part.slice(1)).join('');
  const registry = icons as Record<string, LucideIcon>;
  const Component = Object.hasOwn(registry, key) ? registry[key] : null;
  return Component
    ? <Component size={size} color={color} strokeWidth={1.8} role="img" aria-label={name} style={{ flexShrink: 0, verticalAlign: 'middle' }} />
    : <small role="note" style={{ color: muted }}>[Unknown icon: {name}]</small>;
}

const Title = defineComponent({
  name: 'Title',
  props: z.object({ text: z.string(), size: z.enum(['sm', 'md', 'lg', 'xl', '2xl', '3xl']).optional(), color: z.string().optional() }),
  description: 'A semantic heading with a selectable size and safe text color.',
  component: ({ props }) => {
    const size = props.size || 'lg';
    const sizes = { sm: 16, md: 18, lg: 24, xl: 30, '2xl': 40, '3xl': 52 };
    const Heading = size === 'sm' || size === 'md' ? 'h3' : size === 'lg' ? 'h2' : 'h1';
    return <Heading className="oui-title" style={{ margin: 0, color: safeColor(props.color), fontSize: sizes[size], fontWeight: 650, lineHeight: 1.2, overflowWrap: 'anywhere', letterSpacing: size === '2xl' || size === '3xl' ? '-0.025em' : undefined }}>{props.text}</Heading>;
  },
});

const Text = defineComponent({
  name: 'Text',
  props: z.object({
    text: z.any(), size: z.enum(['xs', 'sm', 'md', 'lg']).optional(), color: z.string().optional(),
    weight: z.enum(['normal', 'medium', 'semibold']).optional(), italic: z.boolean().optional(),
    lineThrough: z.boolean().optional(), tabularNums: z.boolean().optional(), maxLines: z.number().optional(),
  }),
  description: 'Paragraph content with inline Markdown or embedded inline components.',
  component: ({ props, renderNode }) => {
    const sizes = { xs: 12, sm: 14, md: 16, lg: 18 };
    const weights = { normal: 400, medium: 500, semibold: 600 };
    const lines = typeof props.maxLines === 'number' && Number.isFinite(props.maxLines) && props.maxLines > 0 ? Math.max(1, Math.min(100, Math.floor(props.maxLines))) : undefined;
    const style: CSSProperties = {
      margin: 0, minWidth: 0, color: safeColor(props.color), fontSize: sizes[props.size || 'md'],
      fontWeight: weights[props.weight || 'normal'], fontStyle: props.italic ? 'italic' : undefined,
      textDecoration: props.lineThrough ? 'line-through' : undefined, fontVariantNumeric: props.tabularNums ? 'tabular-nums' : undefined,
      lineHeight: 1.6, overflowWrap: 'anywhere',
      ...(lines ? { display: '-webkit-box', WebkitBoxOrient: 'vertical', WebkitLineClamp: lines, overflow: 'hidden' } : {}),
    };
    const renderInline = (part: unknown) => typeof part === 'string' ? <SafeMarkdown text={part} inline /> : renderNode(part);
    return <div className="oui-text" style={style}>{Array.isArray(props.text) ? props.text.map((part, index) => <React.Fragment key={index}>{renderInline(part)}</React.Fragment>) : renderInline(props.text)}</div>;
  },
});

const Caption = defineComponent({
  name: 'Caption', props: z.object({ text: z.string(), color: z.string().optional() }),
  description: 'Small secondary explanatory text.',
  component: ({ props }) => <p className="oui-caption" style={{ margin: 0, fontSize: 12, lineHeight: 1.5, color: safeColor(props.color, muted), overflowWrap: 'anywhere' }}>{props.text}</p>,
});

const Label = defineComponent({
  name: 'Label', props: z.object({ text: z.string() }), description: 'A visible group or field label.',
  component: ({ props }) => <div className="oui-label" style={{ fontSize: 13, fontWeight: 600, lineHeight: 1.4, color: ink }}>{props.text}</div>,
});

const Badge = defineComponent({
  name: 'Badge', props: z.object({ text: z.string(), color: z.enum(['neutral', 'info', 'success', 'warning', 'danger', 'purple', 'pink']).optional(), icon: z.string().optional() }),
  description: 'A small colored tag with an optional local Lucide icon.',
  component: ({ props }) => {
    const tints = { neutral: surface, info: '#eaf2ff', success: '#e8f6ed', warning: '#fff4df', danger: '#ffedf0', purple: '#f2ebff', pink: '#ffedf6' };
    return <span className="oui-badge" style={{ display: 'inline-flex', alignItems: 'center', gap: 4, maxWidth: '100%', borderRadius: 999, padding: '3px 9px', fontSize: 12, fontWeight: 600, lineHeight: 1.3, color: safeColor(props.color === 'neutral' ? 'muted' : props.color, muted), backgroundColor: tints[props.color || 'neutral'], verticalAlign: 'middle', overflowWrap: 'anywhere' }}>{props.icon ? <NamedIcon name={props.icon} size={13} /> : null}{props.text}</span>;
  },
});

const Icon = defineComponent({
  name: 'Icon', props: z.object({ name: z.string(), size: z.enum(['sm', 'md', 'lg', 'xl']).optional(), color: z.string().optional() }),
  description: 'A locally bundled Lucide icon; unknown names have an explicit fallback.',
  component: ({ props }) => <NamedIcon name={props.name} size={({ sm: 16, md: 20, lg: 28, xl: 36 })[props.size || 'md']} color={safeColor(props.color)} />,
});

const Markdown = defineComponent({
  name: 'Markdown', props: z.object({ text: z.string() }), description: 'Markdown prose, tables, lists, and code. Raw HTML and external media are disabled.',
  component: ({ props }) => <div className="oui-markdown" style={{ color: ink, minWidth: 0, fontSize: 16, lineHeight: 1.6, overflowWrap: 'anywhere' }}><SafeMarkdown text={props.text} /></div>,
});

const Quote = defineComponent({
  name: 'Quote', props: z.object({ text: z.string(), cite: z.string().optional() }), description: 'A quotation with an optional plain-text source.',
  component: ({ props }) => <blockquote className="oui-quote" style={{ margin: 0, borderInlineStart: '3px solid #6686af', padding: '4px 0 4px 16px', color: ink }}><div style={{ fontSize: 17, lineHeight: 1.6, fontStyle: 'italic', overflowWrap: 'anywhere' }}>{props.text}</div>{props.cite ? <cite style={{ display: 'block', marginTop: 8, color: muted, fontSize: 13 }}>{props.cite}</cite> : null}</blockquote>,
});

const Code = defineComponent({
  name: 'Code', props: z.object({ code: z.string(), language: z.string().optional() }), description: 'Locally highlighted code with explicit plain-text fallback and user-initiated Host copy of the original source.',
  component: ({ props }) => <HighlightedCodeBlock code={props.code} language={props.language} copyButton />,
});

function SafeMath({ tex, inline }: { tex: string; inline?: boolean }) {
  const host = useRef<HTMLSpanElement>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (!host.current) return;
    host.current.replaceChildren();
    setError(null);
    if (tex.length > 20000) {
      setError('Formula is too long for the isolated renderer.');
      return;
    }
    try {
      katex.render(tex, host.current, {
        displayMode: !inline, output: 'mathml', trust: false, throwOnError: true,
        strict: 'error', maxSize: 30, maxExpand: 1000, macros: {}, globalGroup: false,
      });
    } catch {
      host.current.replaceChildren();
      setError('Formula could not be rendered safely.');
    }
    return () => { host.current?.replaceChildren(); };
  }, [tex, inline]);
  return <span className="oui-math" style={{ display: inline ? 'inline-block' : 'block', maxWidth: '100%', overflowX: inline ? undefined : 'auto', verticalAlign: 'middle', paddingBlock: inline ? 0 : 8 }}><span ref={host} />{error ? <span role="note" style={{ color: '#b43245', fontSize: 13 }}>{error} <code style={{ fontFamily: mono }}>{tex.slice(0, 2000)}</code></span> : null}</span>;
}

const MathComponent = defineComponent({
  name: 'Math', props: z.object({ tex: z.string(), inline: z.boolean().optional() }), description: 'Local KaTeX formula rendered as native MathML with trusted commands disabled.',
  component: ({ props }) => <SafeMath tex={props.tex} inline={props.inline} />,
});

const Link = defineComponent({
  name: 'Link', props: z.object({ text: z.string(), url: z.string() }), description: 'Requests the Host to open an absolute HTTP(S) URL after a user click; reports Host acceptance without claiming the page loaded.',
  component: ({ props }) => <HostLink text={props.text} url={props.url} />,
});

type ListContextValue = { marker: 'bullet' | 'number' | 'none'; index: number; last: boolean; timeline: boolean };
const ListContext = createContext<ListContextValue | null>(null);

function ListEntry({ children, marker }: { children: ReactNode; marker?: string }) {
  const context = useContext(ListContext);
  const resolvedMarker = marker ?? (context?.marker === 'number' ? `${context.index + 1}.` : context?.marker === 'bullet' ? '•' : '');
  const timeline = context?.timeline ?? false;
  const hasMarker = Boolean(resolvedMarker) || timeline;
  return <div className="oui-list-entry" style={{ display: 'grid', gridTemplateColumns: hasMarker ? 'minmax(20px, max-content) minmax(0, 1fr)' : 'minmax(0, 1fr)', alignItems: 'stretch', columnGap: 10 }}>
    {hasMarker ? <span aria-hidden={marker ? undefined : true} style={{ position: 'relative', color: muted, textAlign: 'center', fontSize: 13, lineHeight: '24px', minWidth: 20 }}>{resolvedMarker || '•'}{timeline && !context?.last ? <span style={{ position: 'absolute', top: 24, bottom: -12, left: '50%', width: 1, backgroundColor: border }} /> : null}</span> : null}
    <div style={{ minWidth: 0, overflowWrap: 'anywhere', lineHeight: 1.6 }}>{children}</div>
  </div>;
}

const List = defineComponent({
  name: 'List', props: z.object({ items: z.any(), marker: z.enum(['bullet', 'number', 'none']).optional(), timeline: z.boolean().optional() }),
  description: 'Bulleted, numbered, or unmarked items with optional timeline connectors.',
  component: ({ props, renderNode }) => {
    const items = Array.isArray(props.items) ? props.items : props.items == null ? [] : [props.items];
    const marker = props.marker || 'bullet';
    const ListTag = marker === 'number' ? 'ol' : 'ul';
    return <ListTag className="oui-list" role="list" style={{ listStyle: 'none', padding: 0, margin: 0, display: 'flex', flexDirection: 'column', gap: 12 }}>{items.map((item, index) => <li key={index} style={{ minWidth: 0 }}><ListContext.Provider value={{ marker, index, last: index === items.length - 1, timeline: Boolean(props.timeline) }}>{typeof item === 'string' ? <ListEntry><SafeMarkdown text={item} inline /></ListEntry> : renderNode(item)}</ListContext.Provider></li>)}</ListTag>;
  },
});

const ListItem = defineComponent({
  name: 'ListItem', props: z.object({ title: z.string(), description: z.string().optional(), marker: z.string().optional(), children: z.any().optional() }),
  description: 'List content with title, description, a custom marker, and nested content.',
  component: ({ props, renderNode }) => <ListEntry marker={props.marker}><div style={{ fontWeight: 600, color: ink }}><SafeMarkdown text={props.title} inline /></div>{props.description ? <div style={{ color: muted, fontSize: 14, marginTop: 2 }}><SafeMarkdown text={props.description} inline /></div> : null}{props.children == null ? null : <div style={{ marginTop: 8, minWidth: 0 }}>{renderNode(props.children)}</div>}</ListEntry>,
});

export const contentComponents: DefinedComponent<any>[] = [Title, Text, Caption, Label, Badge, Markdown, Quote, Code, MathComponent, Link, List, ListItem, Icon];
