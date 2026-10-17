// Local syntax highlighting: licensed grammar output becomes React text and spans only.
import React, { useMemo, type CSSProperties, type ReactNode } from 'react';
import { common, createLowlight } from 'lowlight';

// Deliberately fixed coverage. No language guessing or downloaded grammars.
export const highlightLanguages: Readonly<Record<string, readonly string[]>> = Object.freeze({
  arduino: ['ino'], bash: ['sh', 'zsh'], c: ['h'],
  cpp: ['cc', 'c++', 'h++', 'hpp', 'hh', 'hxx', 'cxx'], csharp: ['cs', 'c#'],
  css: [], diff: ['patch'], go: ['golang'], graphql: ['gql'], ini: ['toml'], java: ['jsp'],
  javascript: ['js', 'jsx', 'mjs', 'cjs'], json: ['jsonc'], kotlin: ['kt', 'kts'],
  less: [], lua: ['pluto'], makefile: ['mk', 'mak', 'make'], markdown: ['md', 'mkdown', 'mkd'],
  objectivec: ['mm', 'objc', 'obj-c', 'obj-c++', 'objective-c++'], perl: ['pl', 'pm'],
  php: [], 'php-template': [], plaintext: ['text', 'txt'], python: ['py', 'gyp', 'ipython'],
  'python-repl': ['pycon'], r: [], ruby: ['rb', 'gemspec', 'podspec', 'thor', 'irb'],
  rust: ['rs'], scss: [], shell: ['console', 'shellsession'], sql: [], swift: [],
  typescript: ['ts', 'tsx', 'mts', 'cts'], vbnet: ['vb'], wasm: [],
  xml: ['html', 'xhtml', 'rss', 'atom', 'xjb', 'xsd', 'xsl', 'plist', 'wsf', 'svg'], yaml: ['yml'],
});

export const highlightLimits = Object.freeze({ codeUnits: 20000, lines: 1000, lineCodeUnits: 4000, astDepth: 48, astNodes: 12000 });
const highlighter = createLowlight(common);
const canonicalLanguages = new Map<string, string>();
for (const [name, aliases] of Object.entries(highlightLanguages)) {
  canonicalLanguages.set(name, name);
  for (const alias of aliases) canonicalLanguages.set(alias, name);
}

type SafeToken = { type: 'text'; value: string } | { type: 'span'; className: string; children: SafeToken[] };
type HighlightResult = { tokens?: SafeToken[]; label: string; notice?: string; highlighted: boolean };
class HighlightFallback extends Error {}

function record(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : null;
}

function sourceBudget(code: string): string | undefined {
  if (code.length > highlightLimits.codeUnits) return 'Syntax highlighting skipped: source exceeds the 20,000-character highlighting budget. The complete source is shown.';
  let lines = 1;
  let column = 0;
  for (let index = 0; index < code.length; index += 1) {
    if (code[index] === '\n') { lines += 1; column = 0; } else { column += 1; }
    if (lines > highlightLimits.lines || column > highlightLimits.lineCodeUnits) {
      return 'Syntax highlighting skipped: source exceeds 1,000 lines or 4,000 characters on one line. The complete source is shown.';
    }
  }
  return undefined;
}

function safeTokens(tree: unknown, source: string): SafeToken[] {
  const root = record(tree);
  if (root?.type !== 'root' || !Array.isArray(root.children)) throw new HighlightFallback('Unexpected highlighting output. The complete source is shown.');
  let nodes = 0;
  let textLength = 0;
  const text: string[] = [];
  const visit = (value: unknown, depth: number): SafeToken => {
    nodes += 1;
    if (nodes > highlightLimits.astNodes || depth > highlightLimits.astDepth) throw new HighlightFallback('Syntax highlighting skipped: token count or nesting exceeds the rendering budget. The complete source is shown.');
    const node = record(value);
    if (node?.type === 'text' && typeof node.value === 'string') {
      textLength += node.value.length;
      if (textLength > source.length) throw new HighlightFallback('Highlighting did not preserve the original text. The complete source is shown.');
      text.push(node.value);
      return { type: 'text', value: node.value };
    }
    if (node?.type !== 'element' || node.tagName !== 'span' || !Array.isArray(node.children)) throw new HighlightFallback('Unsupported highlighting token. The complete source is shown.');
    const properties = record(node.properties);
    if (!properties || Object.keys(properties).some((key) => key !== 'className')) throw new HighlightFallback('Unsupported highlighting attributes. The complete source is shown.');
    const classes = properties.className ?? [];
    if (!Array.isArray(classes) || classes.some((name) => typeof name !== 'string' || !/^[a-zA-Z_][a-zA-Z0-9_+-]*$/.test(name))) throw new HighlightFallback('Unsupported highlighting class. The complete source is shown.');
    return { type: 'span', className: classes.join(' '), children: node.children.map((child) => visit(child, depth + 1)) };
  };
  const tokens = root.children.map((child) => visit(child, 1));
  if (text.join('') !== source) throw new HighlightFallback('Highlighting did not preserve the original text. The complete source is shown.');
  return tokens;
}

function highlight(code: string, language?: string): HighlightResult {
  const requested = language?.trim().toLowerCase() || '';
  const canonical = canonicalLanguages.get(requested);
  if (!requested) return { label: 'Plain text', notice: 'No language specified; source is shown as plain text.', highlighted: false };
  if (!canonical) return { label: language || 'Unknown language', notice: 'This language is not supported by the local highlighter. The complete source is shown as plain text.', highlighted: false };
  if (canonical === 'plaintext') return { label: 'Plain text', highlighted: false };
  const budgetNotice = sourceBudget(code);
  if (budgetNotice) return { label: canonical, notice: budgetNotice, highlighted: false };
  try {
    if (!highlighter.registered(canonical)) throw new HighlightFallback('This grammar is unavailable locally. The complete source is shown as plain text.');
    const tokens = safeTokens(highlighter.highlight(canonical, code), code);
    return { tokens, label: canonical, highlighted: true };
  } catch (error: unknown) {
    return { label: canonical, notice: error instanceof HighlightFallback ? error.message : 'Syntax highlighting failed. The complete source is shown as plain text.', highlighted: false };
  }
}

function tokenNodes(tokens: SafeToken[], prefix = 't'): ReactNode[] {
  return tokens.map((token, index) => token.type === 'text'
    ? token.value
    : <span key={`${prefix}-${index}`} className={token.className}>{tokenNodes(token.children, `${prefix}-${index}`)}</span>);
}

// Original palette, scoped to these blocks; no third-party theme or remote styles.
const palette = `
.oui-syntax { --syntax-comment:#52657b; --syntax-keyword:#7935a6; --syntax-string:#226b3e; --syntax-number:#96500b; --syntax-title:#195eae; --syntax-meta:#8f355c; --syntax-built:#216a80; --syntax-attr:#6c50a0; --syntax-add:#166344; --syntax-delete:#9a2540; }
:root[data-appearance="dark"] .oui-syntax { --syntax-comment:#b3bdc9; --syntax-keyword:#d1a1ff; --syntax-string:#a6dfb6; --syntax-number:#ffc389; --syntax-title:#9bcaff; --syntax-meta:#f7a7ca; --syntax-built:#91dbe9; --syntax-attr:#d3b6ff; --syntax-add:#9de3b5; --syntax-delete:#ffa6ba; }
.oui-syntax .hljs-comment,.oui-syntax .hljs-quote { color:var(--syntax-comment); }
.oui-syntax .hljs-keyword,.oui-syntax .hljs-selector-tag,.oui-syntax .hljs-literal,.oui-syntax .hljs-type { color:var(--syntax-keyword); }
.oui-syntax .hljs-string,.oui-syntax .hljs-regexp,.oui-syntax .hljs-symbol,.oui-syntax .hljs-template-tag { color:var(--syntax-string); }
.oui-syntax .hljs-number { color:var(--syntax-number); }
.oui-syntax .hljs-title,.oui-syntax .hljs-section,.oui-syntax .hljs-name { color:var(--syntax-title); }
.oui-syntax .hljs-meta,.oui-syntax .hljs-doctag { color:var(--syntax-meta); }
.oui-syntax .hljs-built_in,.oui-syntax .hljs-variable,.oui-syntax .hljs-template-variable { color:var(--syntax-built); }
.oui-syntax .hljs-attr,.oui-syntax .hljs-attribute,.oui-syntax .hljs-selector-attr,.oui-syntax .hljs-selector-class,.oui-syntax .hljs-selector-id { color:var(--syntax-attr); }
.oui-syntax .hljs-addition { color:var(--syntax-add); }
.oui-syntax .hljs-deletion { color:var(--syntax-delete); }
.oui-syntax .hljs-emphasis { font-style:italic; }
.oui-syntax .hljs-strong { font-weight:700; }
`;
const preStyle: CSSProperties = { margin: 0, padding: 14, maxWidth: '100%', overflowX: 'auto', fontSize: 13, lineHeight: 1.6, fontFamily: 'ui-monospace, SFMono-Regular, Consolas, monospace', whiteSpace: 'pre', overflowWrap: 'normal', wordBreak: 'normal', background: 'var(--oui-surface, #f3f6fa)', color: 'var(--oui-ink, #172337)' };

export function HighlightedCodeBlock({ code, language, clipboardNotice = false }: { code: string; language?: string; clipboardNotice?: boolean }) {
  const result = useMemo(() => highlight(code, language), [code, language]);
  return <><style>{palette}</style><figure className="oui-code oui-syntax" data-highlighted={result.highlighted ? 'true' : 'false'} style={{ margin: 0, minWidth: 0, border: '1px solid var(--oui-border, #d9e0e8)', borderRadius: 8, overflow: 'hidden' }}>
    <figcaption style={{ display: 'flex', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12, padding: '8px 12px', fontSize: 12, color: 'var(--oui-muted, #526379)', background: 'var(--oui-surface, #f3f6fa)' }}><span>{result.label}</span>{clipboardNotice ? <span>Copy unavailable in isolated viewer</span> : null}</figcaption>
    {result.notice ? <div role="note" style={{ padding: '0 12px 8px', color: 'var(--oui-muted, #526379)', background: 'var(--oui-surface, #f3f6fa)', fontSize: 12 }}>{result.notice}</div> : null}
    <pre style={preStyle}><code>{result.tokens ? tokenNodes(result.tokens) : code}</code></pre>
  </figure></>;
}

export function MarkdownCodeBlock({ node, children }: { node?: unknown; children?: ReactNode }) {
  const pre = record(node);
  const candidates = Array.isArray(pre?.children) ? pre.children : [];
  const element = candidates.length === 1 ? record(candidates[0]) : null;
  if (element?.type === 'element' && element.tagName === 'code' && Array.isArray(element.children)) {
    const parts = element.children.map(record);
    if (parts.every((part) => part?.type === 'text' && typeof part.value === 'string')) {
      // Use the exact Markdown parser output, including all whitespace and final newlines.
      const code = parts.map((part) => part!.value as string).join('');
      const classes = record(element.properties)?.className;
      const languageClass = Array.isArray(classes) ? classes.find((name) => typeof name === 'string' && name.startsWith('language-')) : undefined;
      const language = typeof languageClass === 'string' ? languageClass.slice('language-'.length) : undefined;
      return <HighlightedCodeBlock code={code} language={language} />;
    }
  }
  return <div><div role="note" style={{ color: 'var(--oui-muted, #526379)', fontSize: 12 }}>Syntax highlighting unavailable for this code block. Original rendered content is shown.</div><pre style={preStyle}>{children}</pre></div>;
}
