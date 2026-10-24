#!/usr/bin/env node
/** Round116: compact installed run_code CSS/DOM measurement in one owned Chrome. */
import fs from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';
import {createHash, randomUUID} from 'node:crypto';
import net from 'node:net';
import vm from 'node:vm';
import {spawn} from 'node:child_process';

const ROOT = 'C:\\AI\\WhaleIsle-MyGo';
const SCRIPT = path.join(ROOT, 'scripts', 'tool-code-reference-chrome.mjs');
const INSTALL = 'C:\\软件\\Whale Isle\\resources\\vendor\\deepseek-harness';
const REF = path.join(ROOT, 'reference', 'electron', 'vendor', 'deepseek-harness', 'packages', 'client', 'ui-primitives', 'src');
const SOURCE_PATHS = {
  runner: SCRIPT,
  row: path.join(INSTALL, 'packages/client/ui-tool/lib/types/client/tool/components/ToolRow.js'),
  model: path.join(INSTALL, 'packages/client/ui-tool/lib/types/client/tool/models/tool-call-model.js'),
  labels: path.join(INSTALL, 'packages/client/ui-tool/lib/types/client/tool/models/primitive-labels.js'),
  tool: path.join(INSTALL, 'packages/client/ui-tool/lib/client.js'),
  theme: path.join(INSTALL, 'packages/client/ui-theme/lib/client.js'),
  webJS: path.join(INSTALL, 'apps/web/dist/assets/index-wKUmVWbt.js'),
  webCSS: path.join(INSTALL, 'apps/web/dist/assets/index-l5jToo6l.css'),
  locale: path.join(INSTALL, 'packages/client/ui-conversation/lib/types/client/locales.js'),
  common: path.join(INSTALL, 'packages/client/locale/lib/types/locales/en.js'),
  icons: path.join(REF, 'icons/index.tsx'),
  refBlock: path.join(REF, 'markdown/CodeBlock.tsx'),
  refBlockCSS: path.join(REF, 'markdown/CodeBlock.module.css'),
  refToolbar: path.join(REF, 'CodeToolbar.tsx'),
  refCardCSS: path.join(REF, 'CodeCard.module.css'),
  refHighlight: path.join(REF, 'markdown/highlight.ts'),
  priorTerminal: path.join(ROOT, 'scripts/terminal-reference-chrome.mjs'),
  priorRow: path.join(ROOT, 'scripts/tool-row-reference-chrome.mjs'),
};
const CHROME = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const MODE = process.argv[2];
const suppliedPrep = MODE === '--accept-prepared' ? process.argv[3] : null;
const suppliedRun = suppliedPrep && path.basename(suppliedPrep).match(/^r116-tool-code-([0-9a-fTZ-]+)-preflight\.json$/)?.[1];
const RUN_ID = suppliedRun ?? new Date().toISOString().replace(/[:.]/g, '-') + '-' + randomUUID().slice(0, 8);
const BASE = path.join(ROOT, '.devdata', 'r116-tool-code-' + RUN_ID);
const FIXTURE = BASE + '.html';
const PROFILE = BASE + '-profile';
const PREP = BASE + '-preflight.json';
const ATTEMPT = path.join(ROOT, '.devdata', 'r116-tool-code-acceptance-attempt.json');
const PREFIX = path.join(ROOT, 'docs/review-20261022/evidence/tool-code-pass260-reference-' + RUN_ID);
const OUTPUT = {
  source: PREFIX + '-source.json', json: PREFIX + '.json', report: PREFIX + '-report.md', text: PREFIX + '.txt',
  initial: PREFIX + '-initial.png', wrap: PREFIX + '-unwrapped.png', copy: PREFIX + '-copied.png', scroll: PREFIX + '-scroll.png',
};
const PROTECTED = new Set([20012, 16008, 12516]);
const norm = value => path.resolve(value).toLowerCase();
const hash = value => createHash('sha256').update(value).digest('hex');
const invariant = (value, message) => { if (!value) throw Error(message); };
const near = (a, b) => Math.abs(a - b) < 0.03;
const delay = ms => { invariant(ms <= 1000, 'Long blocking wait forbidden'); return new Promise(resolve => setTimeout(resolve, ms)); };
const allowed = new Set([FIXTURE, PREP, ATTEMPT, ...Object.values(OUTPUT)].map(norm));
const E = {
  schema: 'tool-code-pass260-reference/v1', round: 116, runId: RUN_ID, startedAt: new Date().toISOString(), status: 'running',
  attemptPolicy: 'Exactly one Chrome startup/acceptance after syntax and source/fixture preflight. Exclusive round-wide attempt file prevents another acceptance. Each case executes once; failed evidence is retained without an accepting rerun.',
  scope: {
    script: SCRIPT, fixture: FIXTURE, profile: PROFILE,
    description: 'Verbatim installed ToolRow, CodeBlock, CodeCard/CodeToolbar and DisclosureRow/TextShimmer CSS with current installed theme tokens. Manually assembled installed containment and class relationships, synthetic run_code arguments/results. Local fixture disclosure/wrap/copy handlers; no React, app/session/model/host, actual Shiki tokenizer, tooltip portal or motion execution.',
    protected: {electronPID: 20012, backendPID: 16008, lockAppPID: 12516, guiPort: 3080, mutated: false, requested: false},
    input: 'CDP events only in the target of the spawned, exact PID/profile/port-verified headless Chrome. No physical input or monitor claim. Keyboard activation exercises only the disclosure div explicit keydown handler; no native button Enter activation assertion.',
    network: 'No server. File CSP denies network; CDP blocks HTTP/HTTPS/WS/WSS page requests. Only this owned loopback CDP endpoint is contacted.',
  },
  compactPolicy: 'Unique CSS/maps/source hashes occur once in the source sidecar. Computed style records are interned once. Cases contain assertions only; measurements hold one final box snapshot per scenario per case, with scalar transition observations.',
  sourceEvidence: OUTPUT.source, cases: [], measurements: {}, screenshots: {}, errors: [], writes: [],
  limitations: [
    'Installed artifacts are the authority. Read-only TypeScript reference only locates and explains symbols; current installed sZ includes tokenizeMaxLineLength absent from the older reference.',
    'Manual DOM and local handlers do not execute React, original useDisclosure/usePresence, lazy remount, streaming/highlighting observers, application dispatch or session data.',
    'Clipboard writes are a successful/refused in-memory fixture adapter. The system clipboard and original application clipboard are never accessed. Copy selects pre.textContent after trimming one terminal LF, exactly as installed source; it does not preserve that removed terminator.',
    'The colored sample uses manually authored spans with exact installed --shiki token variables. This measures token colors, not actual grammar tokenization or Shiki output.',
    'Toolbar tooltip labels/side/portal are recorded from installed source. This fixture exposes label metadata and aria attributes without executing Tooltip portals, hover delay, positioning or animation.',
    'Baseline light installed tokens and system-resolved font stacks are measured. No persisted preference or remote font loading. DPR1 is a startup flag only; no viewport/scale Emulation, hardware input, protected app or Electron pixel claim.',
  ],
};
let source, browser, page, child, exitInfo, port, verified = false, spawnCount = 0;
const requests = [], exceptions = [], protocolErrors = [], methodCounts = {};

async function noLinks(target) {
  const absolute = path.resolve(target);
  let current = path.parse(absolute).root;
  for (const part of absolute.slice(current.length).split(path.sep)) {
    if (!part) continue;
    current = path.join(current, part);
    try { invariant(!(await fs.lstat(current)).isSymbolicLink(), 'Refuse symlink ' + current); }
    catch (error) { if (error.code !== 'ENOENT') throw error; }
  }
}
async function absent(target) {
  try { await fs.lstat(target); throw Error('Refuse existing ' + target); }
  catch (error) { if (error.code !== 'ENOENT') throw error; }
}
async function writeOwned(target, data) {
  invariant(allowed.has(norm(target)), 'Out of scope write ' + target);
  await noLinks(target); await absent(target);
  await fs.writeFile(target, data, {flag: 'wx'}); E.writes.push(target);
}
async function scope() {
  invariant(process.platform === 'win32' && Number(process.versions.node.split('.')[0]) >= 24, 'Windows Node24 required');
  invariant((MODE === '--prepare-only' && process.argv.length === 3) ||
    (MODE === '--accept-prepared' && process.argv.length === 4 && suppliedRun && norm(suppliedPrep) === norm(PREP)), 'Only prepare/own-prepared modes allowed');
  invariant(norm(fileURLToPath(import.meta.url)) === norm(SCRIPT) && norm(process.cwd()) === norm(ROOT), 'Wrong runner or workspace');
  for (const target of [...Object.values(SOURCE_PATHS), CHROME, PROFILE, ...allowed]) await noLinks(target);
  for (const target of [ROOT, path.dirname(FIXTURE), path.dirname(PREFIX)]) invariant(norm(await fs.realpath(target)) === norm(target), 'Redirected root ' + target);
  for (const target of [PROFILE, ATTEMPT, ...Object.values(OUTPUT).filter(p => MODE === '--prepare-only' || p !== OUTPUT.source), ...(MODE === '--prepare-only' ? [PREP, FIXTURE] : [])]) await absent(target);
}
function literal(text, name) {
  const prefix = 'var ' + name + '_css_default = ';
  const line = text.split('\n').find(value => value.trimStart().startsWith(prefix));
  invariant(line, 'Missing installed theme literal ' + name);
  return {text: JSON.parse(line.trim().slice(prefix.length).replace(/;$/, '')), source: 'theme', line: text.slice(0, text.indexOf(line)).split('\n').length};
}
function fragment(text, startText, endText, sourceID) {
  const start = text.indexOf(startText), end = text.indexOf(endText, start);
  invariant(start >= 0 && end > start, 'Missing installed fragment ' + startText);
  return {source: sourceID, startUtf16Offset: start, endUtf16Offset: end, text: text.slice(start, end)};
}
function cssMap(css, suffix, names) {
  return Object.fromEntries(names.map(name => {
    const found = css.match(new RegExp('\\._' + name + '_' + suffix + '_\\d+'));
    invariant(found, 'Missing CSS class ' + suffix + '/' + name);
    return [name, found[0].slice(1)];
  }));
}
function pureFunction(text, name) {
  const match = text.match(new RegExp('(?:export )?function ' + name + '\\([^]*?\\n}'));
  invariant(match, 'Missing installed pure function ' + name); return match[0].replace(/^export /, '');
}
function label(text, key) {
  const found = text.match(new RegExp("'" + key.replaceAll('.', '\\.') + "': '([^']*)'"));
  invariant(found, 'Missing installed locale ' + key); return found[1];
}
async function extract() {
  const entries = await Promise.all(Object.entries(SOURCE_PATHS).map(async ([id, sourcePath]) => [id, await fs.readFile(sourcePath, 'utf8')]));
  const texts = Object.fromEntries(entries), {tool, theme, webCSS, webJS, model, row, icons} = texts;
  const rowLiteral = tool.match(/^\t\tconst css\$3 = ("(?:[^"\\]|\\.)*");\r?$/m);
  invariant(rowLiteral, 'Missing current ToolRow CSS');
  const rowCSS = JSON.parse(rowLiteral[1]);
  const mapStart = tool.indexOf('var ToolRow_module_css_default = '), mapEnd = tool.indexOf('};', mapStart) + 2;
  invariant(mapStart >= 0 && mapEnd > mapStart, 'Missing installed row map');
  const rowMap = Object.fromEntries([...tool.slice(mapStart, mapEnd).matchAll(/"([^"]+)": "([^"]+)"/g)].map(value => [value[1], value[2]]));
  const allRules = [...webCSS.matchAll(/([^{}]+)\{([^{}]*)\}/g)];
  const classCSS = suffix => allRules.filter(value => value[1].includes('_' + suffix + '_')).map(value => value[0]).join('');
  const codeCSS = classCSS('7gxqk'), cardCSS = classCSS('1pq26');
  const disclosureCSS = fragment(webCSS, '._root_1rdzk_1{', '._button_ll2mm_2{', 'webCSS');
  const codeJS = fragment(webJS, 'function rx({code:', 'function ix(', 'webJS');
  const toolbarJS = fragment(webJS, 'function r4({lang:', 'const VZ=', 'webJS');
  const highlightJS = fragment(webJS, 'function sZ(', 'var Mw=class', 'webJS');
  const disclosureJS = fragment(webJS, 'const hU=$.memo', ',mU=400;', 'webJS');
  invariant(row.includes('run_code') === false && model.includes("run_code: 'code'"), 'Installed classification changed');
  invariant(row.includes('code: bodyText, lang: "typescript"') && row.includes('className: css.bodyScroll') && row.includes("variant === 'code' ? null : bodyText"), 'Installed code callsite changed');
  invariant(rowCSS.includes('max-height:260px;overflow-y:auto') && !rowCSS.includes('.KZEDha_codeBody{'), 'Current codeBody override changed');
  invariant(codeJS.text.includes('[V,te]=$.useState(!0)') && codeJS.text.includes('querySelector("pre")?.textContent??v') && codeJS.text.includes('window.setTimeout(()=>{U(!1)},1e3)'), 'Installed wrap/copy semantics changed');
  invariant(toolbarJS.text.includes('"aria-label":i.wrapLabel,"aria-pressed":d') && toolbarJS.text.includes('side:"top",portal:!0'), 'Installed toolbar semantics changed');
  const maps = {
    row: rowMap,
    code: cssMap(codeCSS, '7gxqk', ['block', 'bannerWrap', 'banner', 'infostring', 'action', 'copyButton', 'content', 'plain', 'card', 'numbered']),
    toolbar: cssMap(cardCSS, '1pq26', ['card', 'header', 'heading', 'language', 'title', 'actions', 'status', 'action', 'body']),
    disclosure: cssMap(disclosureCSS.text, '1j0v6', ['root', 'header', 'row', 'leading', 'iconIdle', 'chevronHover', 'title']),
    shimmer: cssMap(disclosureCSS.text, '1rdzk', ['root', 'content', 'text']),
  };
  for (const value of Object.values(maps).flatMap(Object.values)) invariant(webJS.includes('"' + value + '"') || tool.includes('"' + value + '"'), 'Mapped class absent from installed JS ' + value);
  const css = {
    row: {source: 'tool', line: tool.slice(0, rowLiteral.index).split('\n').length, text: rowCSS},
    code: {source: 'webCSS', text: codeCSS}, toolbar: {source: 'webCSS', text: cardCSS}, disclosure: disclosureCSS,
    ...Object.fromEntries(['base', 'motion', 'corner_shape', 'design_platform', 'focus', 'scrollbar', 'gradient_shadow_text', 'shiki'].map(name => ['theme_' + name, literal(theme, name)])),
  };
  const en = texts.locale.slice(texts.locale.indexOf('export const en = {'));
  const labels = Object.fromEntries(['row.input', 'row.output', 'tool.title.code', 'row.failed'].map(key => [key, label(en, key)]));
  for (const key of ['copy', 'copied', 'codeBlock.title', 'codeBlock.wrap', 'codeBlock.unwrap']) labels[key] = label(texts.common, key);
  const artwork = {};
  for (const [key, name] of [['up', 'IconChevronUpOutlineArtwork'], ['down', 'IconChevronDownOutlineArtwork'], ['wrap', 'IconWrapFillArtwork'], ['nowrap', 'IconNowrapFillArtwork'], ['copy', 'IconCopyOutlineArtwork'], ['check', 'IconCheckOutlineArtwork']]) {
    const start = icons.indexOf(name + ' = '), end = icons.indexOf('\n)', start) + 2;
    invariant(start >= 0 && end > start, 'Missing reference icon ' + name);
    const block = icons.slice(start, end);
    const paths = [...block.matchAll(/<path\s+d="([^"]+)"([^>]*)\/>/g)].map(value => ({d: value[1], ...(value[2].includes('fill="currentColor"') ? {fill: 'currentColor'} : {}), ...(value[2].includes('stroke="currentColor"') ? {stroke: 'currentColor'} : {})}));
    invariant(paths.length > 0 && paths.every(value => webJS.includes(value.d)), 'Icon path not verified in installed output ' + name);
    const rects = [...block.matchAll(/<rect\s+([^>]+)\/>/g)].map(value => Object.fromEntries([...value[1].matchAll(/([\w-]+)="([^"]+)"/g)].map(attribute => [attribute[1], attribute[2]])));
    for (const rect of rects) {
      const installedObject = Object.entries(rect).map(([attribute, value]) => attribute + ':' + JSON.stringify(value)).join(',');
      invariant(webJS.includes(installedObject), 'Icon rect not verified in installed output ' + name);
    }
    const shapes = [...block.matchAll(/<(path|rect)\s+([^>]+)\/>/g)].map(value => ({tag: value[1], attributes: Object.fromEntries([...value[2].matchAll(/([\w-]+)="([^"]+)"/g)].map(attribute => [attribute[1], attribute[2]]))}));
    artwork[key] = {source: 'icons', line: icons.slice(0, start).split('\n').length, name, viewBox: '0 0 16 16', strokeWidth: 1, shapes, pathsAndRectsVerifiedInInstalledBundle: true};
  }
  const parseArgsSource = pureFunction(model, 'parseArgs'), formatSource = pureFunction(model, 'formatToolBody');
  const pure = vm.runInNewContext(parseArgsSource + '\n' + formatSource + '\n({formatToolBody})', Object.create(null), {timeout: 1000});
  const formatInputs = [
    ['empty-raw', 'code', '', null], ['invalid-json', 'code', '{"code":"partial', '{"code":"partial'],
    ['program', 'code', '{"description":"ignored envelope","code":"const x = 1;\\n"}', 'const x = 1;\n'],
    ['empty-code-field', 'code', '{"code":""}', '{\n  "code": ""\n}'],
    ['whitespace-code', 'code', '{"code":"  \\n"}', '  \n'],
    ['missing-code', 'code', '{"description":"only description"}', '{\n  "description": "only description"\n}'],
    ['nonstring-code', 'code', '{"code":7}', '{\n  "code": 7\n}'],
    ['array-object-code', 'code', '[{"code":"nested"}]', '[\n  {\n    "code": "nested"\n  }\n]'],
    ['json-string', 'code', '"program text"', '"program text"'], ['json-null', 'code', 'null', 'null'],
    ['json-false', 'code', 'false', 'false'], ['json-number', 'code', '42', '42'],
    ['other-variant', 'others', '{"code":"kept envelope"}', '{\n  "code": "kept envelope"\n}'],
    ['crlf-program', 'code', '{"code":"a\\r\\nb\\r\\n"}', 'a\r\nb\r\n'],
  ];
  const formatTable = formatInputs.map(([id, variant, raw, expected]) => {
    const actual = pure.formatToolBody(variant, raw);
    return {id, variant, raw, result: actual, pass: actual === expected};
  });
  invariant(formatTable.every(value => value.pass), 'Installed source-only format table differs');
  const sourceFiles = await Promise.all(Object.entries(SOURCE_PATHS).map(async ([id, sourcePath]) => ({id, path: sourcePath, bytes: (await fs.stat(sourcePath)).size, sha256: hash(await fs.readFile(sourcePath))})));
  source = {
    schema: 'tool-code-pass260-reference-source/v1', runId: RUN_ID, authority: INSTALL, files: sourceFiles, css, maps, labels, artwork,
    installedJS: {codeBlock: codeJS, toolbar: toolbarJS, highlighting: highlightJS, disclosure: disclosureJS},
    sourceOnly: {formatToolBody: {source: 'model', line: 225, dependencies: parseArgsSource, function: formatSource, table: formatTable},
      callsite: {source: 'row', lines: [54, 57, 59, 93, 95, 124], code: 'bodyText', lang: 'typescript', toolbarLabels: true, className: 'codeBody', parentClassName: 'bodyScroll', cardBody: null, outputText: 'output ?? null', defaultProps: {lineNumbers: false, showHeader: true, wrap: 'omitted => local true', streaming: 'omitted'}, outputCard: 'One OUT section iff outputText !== null; even output empty string. No IN, divider, or output copy control for code variant.'},
      behavior: {trim: 'Remove exactly one terminal LF, preserving any preceding CR or another LF.', copy: 'pre.textContent ?? trimmed, guarded while copied; successful write => copied=true; setTimeout(...,1000). Refused write has no feedback.', wrap: 'local useState(true), owner wrap omitted; toggles data-code-wrap and aria-pressed. aria-label always wrapLabel; tooltip changes unwrapLabel/wrapLabel.', toolbar: 'language typescript; title/status omitted; 2 native buttons, 14px glyphs; no visible Copy text.', highlighting: 'Installed sZ uses codeToHtml(..., theme css-variables, tokenizeMaxLineLength:e4); viewport/grammar/streaming behavior not executed.'}},
    referenceDifference: 'Older read-only highlight.ts omits tokenizeMaxLineLength present in installed sZ; copied CodeBlock/CodeToolbar structural behavior and CSS declarations are verified against installed artifacts. No reference source is authoritative over the current bundle.',
  };
  return {maps, css, labels, artwork, format: (variant, raw) => pure.formatToolBody(variant, raw)};
}

const esc = value => String(value).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;');
function makeModels(x) {
  const program = 'const message: string = "fixture";\nconsole.log(message);\n';
  const longCode = Array.from({length: 50}, (_, index) => 'const line' + String(index + 1).padStart(2, '0') + ': number = ' + (index + 1) + ';').join('\n') + '\n';
  const longOutput = Array.from({length: 45}, (_, index) => 'result ' + String(index + 1).padStart(2, '0') + ': fixture output').join('\n') + '\n';
  return [
    {id: 'baseline', raw: JSON.stringify({description: 'Fixture code and result', code: program}), output: 'fixture\ncompleted\n', x: 40, y: 105, w: 600},
    {id: 'invalid', raw: '{"code":"const unfinished =', output: null, x: 40, y: 335, w: 600},
    {id: 'empty-field', raw: JSON.stringify({description: 'Fixture empty code field', code: ''}), output: null, x: 40, y: 525, w: 600},
    {id: 'empty-raw', raw: '', output: 'output only', x: 40, y: 760, w: 600},
    {id: 'empty-raw-no-output', raw: '', output: null, x: 40, y: 910, w: 600},
    {id: 'empty-output', raw: JSON.stringify({code: 'return 0;\n'}), output: '', state: 'error', x: 700, y: 105, w: 600},
    {id: 'null-output', raw: JSON.stringify({code: 'const onlyCode = true;\n'}), output: null, x: 700, y: 335, w: 600},
    {id: 'long-caps', raw: JSON.stringify({description: 'Independent code and output', code: longCode}), output: longOutput, x: 700, y: 525, w: 600},
    {id: 'wrap', raw: JSON.stringify({code: 'const longValue = "' + 'long-fixture-'.repeat(12) + '";\n'}), output: null, x: 40, y: 1030, w: 320},
    {id: 'manual-highlight', raw: JSON.stringify({code: 'const answer = 42;\n'}), output: null, manualHighlight: true, x: 400, y: 1030, w: 240},
  ].map(value => ({...value, state: value.state ?? 'ok', bodyText: x.format('code', value.raw)}));
}
function fixture(x) {
  const m = x.maps, L = x.labels, models = makeModels(x);
  const svg = key => '<svg data-icon="' + key + '" width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true" stroke-width="1">' + x.artwork[key].shapes.map(shape => '<' + shape.tag + ' ' + Object.entries(shape.attributes).map(([name, value]) => name + '="' + esc(value) + '"').join(' ') + '/>').join('') + '</svg>';
  const code = value => {
    if (value.bodyText === null) return '';
    const trimmed = value.bodyText.endsWith('\n') ? value.bodyText.slice(0, -1) : value.bodyText;
    const pre = value.manualHighlight
      ? '<div data-manual-shiki-container><pre class="shiki css-variables" tabindex="0" style="background-color:var(--shiki-background);color:var(--shiki-foreground)"><code><span class="line"><span data-token="keyword" style="color:var(--shiki-token-keyword)">const</span><span> answer </span><span data-token="punctuation" style="color:var(--shiki-token-punctuation)">=</span><span> </span><span data-token="constant" style="color:var(--shiki-token-constant)">42</span><span data-token="punctuation" style="color:var(--shiki-token-punctuation)">;</span></span></code></pre></div>'
      : '<pre class="' + m.code.plain + '"><code>' + esc(trimmed) + '</code></pre>';
    return '<div class="' + m.row.bodyScroll + '" data-body-scroll><div class="' + m.code.block + ' md-code-block ' + m.code.card + ' ' + m.row.codeBody + '" data-code-block data-code-wrap="true"><div class="' + m.code.bannerWrap + '" data-banner-wrap><div class="' + m.toolbar.header + '" data-code-block-banner><div class="' + m.toolbar.heading + '"><span class="' + m.toolbar.language + '" data-language>typescript</span></div><div class="' + m.toolbar.actions + '" data-toolbar-actions><button type="button" class="' + m.toolbar.action + '" aria-label="' + esc(L['codeBlock.wrap']) + '" aria-pressed="true" data-wrap data-fixture-tooltip-label="' + esc(L['codeBlock.unwrap']) + '">' + svg('nowrap') + '</button><button type="button" class="' + m.toolbar.action + '" aria-label="' + esc(L.copy) + '" data-copy data-fixture-tooltip-label="' + esc(L.copy) + '">' + svg('copy') + '</button></div></div></div><div class="' + m.code.content + '" data-code-block-content>' + pre + '</div></div></div>';
  };
  const io = value => value.output === null ? '' : '<div class="' + m.row.ioCard + '" data-io-card><div class="' + m.row.ioSection + '" data-io-section="output"><span class="' + m.row.ioLabel + '" data-io-label>' + esc(L['row.output']) + '</span><span class="' + m.row.ioText + '" data-io-text' + (value.state === 'error' ? ' data-error="true"' : '') + '>' + esc(value.output) + '</span></div></div>';
  const scenario = value => '<section class="scenario" style="left:' + value.x + 'px;top:' + value.y + 'px;width:' + value.w + 'px"><p class="caption">' + esc(value.id) + (value.manualHighlight ? ' · manual spans, token color only' : '') + '</p><div class="' + m.row.root + '" data-own-card="' + value.id + '" data-tool="run_code" data-variant="code" data-state="' + value.state + '"><div class="' + m.disclosure.root + '" data-disclosure-root data-open="true"><div class="' + m.disclosure.header + '" data-disclosure-header><div class="' + m.disclosure.row + ' ' + m.row.row + '" data-disclosure-row data-expandable role="button" tabindex="0" aria-expanded="true"><span class="' + m.disclosure.leading + ' ' + m.row.leading + '" data-leading>' + svg('up') + '</span><span class="' + m.shimmer.root + '"><span class="' + m.shimmer.content + '"><span class="' + m.shimmer.text + ' ' + m.disclosure.title + ' ' + m.row.title + '">' + esc(L['tool.title.code']) + '</span><span class="' + m.row.sep + '" aria-hidden="true"></span><span class="' + m.row.summary + (value.state === 'error' ? ' ' + m.row.errorSummary : '') + '"><span class="' + m.shimmer.text + '">Fixture ' + value.id + '</span></span></span></span></div></div><div data-fixture-motion data-dsh-motion="fade" data-state="open"><div class="' + m.row.bodyWrap + '" data-body-wrap>' + code(value) + io(value) + '</div></div></div></div></section>';
  const tokenOrder = ['base', 'motion', 'corner_shape', 'design_platform', 'focus', 'scrollbar', 'gradient_shadow_text', 'shiki'];
  const css = [...tokenOrder.map(name => x.css['theme_' + name].text), x.css.disclosure.text, x.css.toolbar.text, x.css.code.text, x.css.row.text];
  const data = {models, maps: m, labels: L, artwork: x.artwork};
  const html = '<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src \'none\';style-src \'unsafe-inline\';script-src \'unsafe-inline\';connect-src \'none\';img-src data:"><title>Round116 installed run_code reference</title>' + css.map((text, index) => '<style data-installed-css="' + index + '">' + text + '</style>').join('') + '<style>html,body{margin:0;min-height:100%}body{font-family:var(--dsw-font-family);font-size:14px;color:var(--dsw-alias-label-primary);background:var(--dsw-alias-bg-base)}#intro{position:absolute;left:40px;top:20px;font-size:12px;color:var(--dsw-alias-label-secondary)}h1{font-size:18px;font-weight:500;margin:0 0 5px}#intro p{margin:0}.scenario{position:absolute}.caption{font-size:12px;line-height:18px;color:var(--dsw-alias-label-secondary);margin:0 0 8px}</style></head><body data-platform="windows"><header id="intro"><h1>Installed run_code · isolated Chrome DPR1</h1><p>Exact installed CSS/containment. Synthetic arguments/output; local handlers and memory clipboard. No React, actual grammar, host/app or physical-input claim.</p></header>' + models.map(scenario).join('') + '<script>\n' + inlineFixture.toString() + '\ninlineFixture(' + JSON.stringify(data).replaceAll('<', '\\u003c') + ');\n</script></body></html>';
  return {html, models};
}

function inlineFixture(data) {
  'use strict';
  const {models, maps: m, labels, artwork} = data;
  const events = [], clipboard = [], feedback = [], styles = [], styleIDs = new Map();
  let clipboardOK = true;
  const rect = element => {
    if (!element) return null;
    const box = element.getBoundingClientRect(); return {x: box.x, y: box.y, w: box.width, h: box.height};
  };
  const keys = ['display', 'position', 'boxSizing', 'fontSize', 'lineHeight', 'fontFamily', 'fontWeight', 'color', 'backgroundColor', 'whiteSpace', 'wordBreak', 'overflowWrap', 'overflowX', 'overflowY', 'maxHeight', 'minWidth', 'paddingTop', 'paddingRight', 'paddingBottom', 'paddingLeft', 'marginTop', 'marginRight', 'marginBottom', 'marginLeft', 'borderTopWidth', 'borderTopColor', 'borderRadius', 'borderTopLeftRadius', 'borderBottomLeftRadius', 'columnGap', 'gridTemplateColumns', 'alignItems', 'top', 'zIndex', 'outlineStyle', 'outlineWidth', 'outlineColor', 'outlineOffset'];
  const style = element => element ? Object.fromEntries(keys.map(key => [key, getComputedStyle(element)[key]])) : null;
  const intern = element => {
    if (!element) return null;
    const value = style(element), serialized = JSON.stringify(value);
    if (!styleIDs.has(serialized)) { styleIDs.set(serialized, 'S' + (styles.length + 1)); styles.push({id: styleIDs.get(serialized), value}); }
    return styleIDs.get(serialized);
  };
  const scroll = element => element ? {x: element.scrollLeft, y: element.scrollTop, sw: element.scrollWidth, sh: element.scrollHeight, cw: element.clientWidth, ch: element.clientHeight} : null;
  const box = (element, withScroll = false) => element ? {box: rect(element), style: intern(element), ...(withScroll ? {scroll: scroll(element)} : {})} : null;
  const own = id => document.querySelector('[data-own-card="' + id + '"]');
  const svg = key => {
    const element = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    for (const [name, value] of Object.entries({'data-icon': key, width: '14', height: '14', viewBox: '0 0 16 16', fill: 'none', 'aria-hidden': 'true', 'stroke-width': '1'})) element.setAttribute(name, value);
    for (const shape of artwork[key].shapes) { const p = document.createElementNS(element.namespaceURI, shape.tag); for (const [name, text] of Object.entries(shape.attributes)) p.setAttribute(name, text); element.append(p); }
    return element;
  };
  for (const type of ['mousemove', 'pointerdown', 'pointerup', 'mousedown', 'mouseup', 'click', 'keydown', 'keyup', 'wheel']) document.addEventListener(type, event => {
    events.push({type, trusted: event.isTrusted, id: event.target.closest('[data-own-card]')?.dataset.ownCard ?? null, key: event.key ?? null, target: event.target.closest('[data-copy]') ? 'copy' : event.target.closest('[data-wrap]') ? 'wrap' : event.target.closest('[data-disclosure-row]') ? 'row' : event.target.closest('[data-body-scroll]') ? 'code' : event.target.closest('[data-io-section]') ? 'output' : null});
  }, true);
  for (const value of models) {
    const root = own(value.id), row = root.querySelector('[data-disclosure-row]'), motion = root.querySelector('[data-fixture-motion]');
    const toggle = () => {
      const open = row.getAttribute('aria-expanded') !== 'true'; row.setAttribute('aria-expanded', String(open)); motion.hidden = !open;
      const disclosure = root.querySelector('[data-disclosure-root]'); if (open) disclosure.dataset.open = 'true'; else delete disclosure.dataset.open;
      root.querySelector('[data-leading]').replaceChildren(svg(open ? 'up' : 'down'));
    };
    row.addEventListener('click', toggle);
    // Only these explicit keydown handlers are exercised by keyboard activation.
    row.addEventListener('keydown', event => { if (event.key !== 'Enter' && event.key !== ' ') return; event.preventDefault(); toggle(); });
    const block = root.querySelector('[data-code-block]'), wrap = root.querySelector('[data-wrap]'), copy = root.querySelector('[data-copy]');
    wrap?.addEventListener('click', () => {
      const wrapped = block.dataset.codeWrap !== 'true'; block.dataset.codeWrap = String(wrapped); wrap.setAttribute('aria-pressed', String(wrapped));
      wrap.dataset.fixtureTooltipLabel = labels[wrapped ? 'codeBlock.unwrap' : 'codeBlock.wrap']; wrap.replaceChildren(svg(wrapped ? 'nowrap' : 'wrap'));
    });
    if (copy) {
      let copied = false;
      copy.addEventListener('click', () => {
        if (copied) return;
        const text = root.querySelector('pre')?.textContent ?? (value.bodyText.endsWith('\n') ? value.bodyText.slice(0, -1) : value.bodyText);
        const ok = clipboardOK;
        clipboard.push({id: value.id, text, ok, at: performance.now(), adapter: 'in-memory fixture only'});
        void Promise.resolve(ok).then(success => {
          if (!success) return;
          copied = true; copy.setAttribute('aria-label', labels.copied); copy.dataset.fixtureTooltipLabel = labels.copied; copy.replaceChildren(svg('check'));
          const start = performance.now(); feedback.push({id: value.id, event: 'copied', at: start});
          setTimeout(() => { copied = false; copy.setAttribute('aria-label', labels.copy); copy.dataset.fixtureTooltipLabel = labels.copy; copy.replaceChildren(svg('copy')); feedback.push({id: value.id, event: 'reset', at: performance.now()}); }, 1000);
        });
      });
    }
  }
  function quick(id) {
    const root = own(id), row = root.querySelector('[data-disclosure-row]'), wrap = root.querySelector('[data-wrap]'), copy = root.querySelector('[data-copy]'), pre = root.querySelector('pre');
    return {expanded: row.getAttribute('aria-expanded'), visible: !root.querySelector('[data-fixture-motion]').hidden, wrapped: root.querySelector('[data-code-block]')?.dataset.codeWrap ?? null, wrapLabel: wrap?.getAttribute('aria-label') ?? null, wrapPressed: wrap?.getAttribute('aria-pressed') ?? null, wrapTooltip: wrap?.dataset.fixtureTooltipLabel ?? null, wrapIcon: wrap?.querySelector('svg')?.dataset.icon ?? null, copyLabel: copy?.getAttribute('aria-label') ?? null, copyIcon: copy?.querySelector('svg')?.dataset.icon ?? null, preText: pre?.textContent ?? null, preHeight: pre?.getBoundingClientRect().height ?? null, preScroll: scroll(pre), codeScroll: scroll(root.querySelector('[data-body-scroll]')), outputScroll: scroll(root.querySelector('[data-io-section]')), focus: document.activeElement === wrap ? 'wrap' : document.activeElement === copy ? 'copy' : document.activeElement === row ? 'row' : document.activeElement?.tagName, focusVisible: wrap?.matches(':focus-visible') ?? false};
  }
  function one(id) {
    const root = own(id), row = root.querySelector('[data-disclosure-row]'), body = root.querySelector('[data-body-wrap]'), viewport = root.querySelector('[data-body-scroll]'), block = root.querySelector('[data-code-block]'), banner = root.querySelector('[data-banner-wrap]'), toolbar = root.querySelector('[data-code-block-banner]'), language = root.querySelector('[data-language]'), actions = root.querySelector('[data-toolbar-actions]'), pre = root.querySelector('pre'), content = root.querySelector('[data-code-block-content]'), io = root.querySelector('[data-io-card]'), section = root.querySelector('[data-io-section]'), text = root.querySelector('[data-io-text]');
    return {id, own: box(root), row: {...box(row), expanded: row.getAttribute('aria-expanded'), role: row.getAttribute('role'), tabIndex: row.tabIndex}, body: box(body), viewport: box(viewport, true), block: block ? {...box(block, true), wrapped: block.dataset.codeWrap, lineNumbers: block.hasAttribute('data-line-numbers'), mdCodeBlock: block.classList.contains('md-code-block')} : null, banner: box(banner), toolbar: box(toolbar), language: language ? {...box(language), text: language.textContent} : null, actions: box(actions), buttons: [...root.querySelectorAll('[data-code-block-banner] button')].map(element => ({kind: element.hasAttribute('data-wrap') ? 'wrap' : 'copy', ...box(element), tag: element.tagName, type: element.type, label: element.getAttribute('aria-label'), pressed: element.getAttribute('aria-pressed'), tooltipMetadata: element.dataset.fixtureTooltipLabel, focused: document.activeElement === element, focusVisible: element.matches(':focus-visible'), text: element.textContent, icon: {key: element.querySelector('svg')?.dataset.icon, box: rect(element.querySelector('svg'))}})), content: box(content), pre: pre ? {...box(pre, true), chars: pre.textContent.length, lines: pre.textContent.split('\n').length, tabIndex: pre.getAttribute('tabindex'), className: pre.className} : null, code: box(pre?.querySelector('code')), io: io ? {...box(io), buttonCount: io.querySelectorAll('button').length, sectionCount: io.querySelectorAll('[data-io-section]').length, dividerCount: io.querySelectorAll('.' + m.row.ioDivider).length, section: {...box(section, true), label: {...box(section.querySelector('[data-io-label]')), text: section.querySelector('[data-io-label]').textContent}, text: {...box(text), chars: text.textContent.length, lines: text.textContent.split('\n').length, error: text.hasAttribute('data-error')}}} : null, manualTokens: [...root.querySelectorAll('[data-token]')].map(element => ({token: element.dataset.token, text: element.textContent, ...box(element)}))};
  }
  function metrics(id) {
    const root = own(id); return Object.fromEntries(['[data-disclosure-row]', '[data-body-wrap]', '[data-body-scroll]', '[data-code-block]', '[data-banner-wrap]', '[data-code-block-banner]', '[data-language]', '[data-toolbar-actions]', '[data-wrap]', '[data-copy]', '[data-code-block-content]', 'pre', '[data-io-card]', '[data-io-section]', '[data-io-label]', '[data-io-text]'].map(selector => [selector, style(root.querySelector(selector))]));
  }
  const tokens = () => Object.fromEntries(['--dsw-radius-lg', '--dsw-radius-sm', '--dsw-font-markdown-code-block', '--dsw-font-markdown-code-block-small', '--dsh-code-font-size', '--dsw-alias-markdown-code-block', '--dsw-alias-label-primary', '--dsw-alias-label-secondary', '--dsw-alias-label-tertiary', '--dsw-alias-label-caption', '--dsw-alias-state-error-primary', '--shiki-token-keyword', '--shiki-token-constant', '--shiki-token-punctuation'].map(name => [name, getComputedStyle(document.body).getPropertyValue(name).trim()]));
  window.harness = {ready: true, quick, one, metrics, tokens, styles: () => styles, events: () => events, clipboard: () => clipboard, feedback: () => feedback, setClipboardOK: value => { clipboardOK = value; }, box: (id, selector) => rect(own(id).querySelector(selector))};
}

class CDP {
  constructor(socket) {
    this.socket = socket; this.id = 0; this.pending = new Map(); this.listeners = [];
    socket.addEventListener('message', event => {
      const message = JSON.parse(String(event.data));
      if (message.id) {
        const pending = this.pending.get(message.id); if (!pending) return;
        clearTimeout(pending.timer); this.pending.delete(message.id);
        if (message.error) { protocolErrors.push({method: pending.method, error: message.error}); pending.reject(Error(pending.method + ' ' + JSON.stringify(message.error))); }
        else pending.resolve(message.result);
      } else for (const listener of this.listeners) listener(message);
    });
    socket.addEventListener('close', () => { for (const pending of this.pending.values()) { clearTimeout(pending.timer); pending.reject(Error('Owned CDP closed ' + pending.method)); } this.pending.clear(); });
  }
  static async connect(url) {
    const parsed = new URL(url); invariant(parsed.protocol === 'ws:' && parsed.hostname === '127.0.0.1' && +parsed.port === port, 'Wrong CDP URL');
    const socket = new WebSocket(url);
    await new Promise((resolve, reject) => { const timer = setTimeout(() => reject(Error('CDP connect timeout')), 10000); socket.addEventListener('open', () => { clearTimeout(timer); resolve(); }, {once: true}); socket.addEventListener('error', () => { clearTimeout(timer); reject(Error('CDP connect error')); }, {once: true}); });
    return new CDP(socket);
  }
  send(method, params = {}, sessionId) {
    invariant(!method.startsWith('Emulation.'), 'All scale/viewport Emulation forbidden');
    if (method.startsWith('Input.')) invariant(verified, 'Input before PID/profile/port verification');
    methodCounts[method] = (methodCounts[method] ?? 0) + 1;
    const id = ++this.id;
    return new Promise((resolve, reject) => { const timer = setTimeout(() => { this.pending.delete(id); reject(Error(method + ' timeout')); }, 10000); this.pending.set(id, {resolve, reject, timer, method}); this.socket.send(JSON.stringify({id, method, params, ...(sessionId ? {sessionId} : {})})); });
  }
}
async function waitFor(fn, message, ms = 10000) { const end = Date.now() + ms; do { if (await fn()) return; await delay(40); } while (Date.now() < end); throw Error('Timeout ' + message); }
async function freePort() {
  const server = net.createServer(); await new Promise((resolve, reject) => { server.once('error', reject); server.listen({host: '127.0.0.1', port: 0, exclusive: true}, resolve); });
  const value = server.address().port; invariant(value >= 1024 && ![3080, 20012, 16008, 12516].includes(value), 'Protected port');
  await new Promise((resolve, reject) => server.close(error => error ? reject(error) : resolve())); return value;
}
async function portFree() { return new Promise(resolve => { const server = net.createServer(); server.once('error', () => resolve(false)); server.listen({host: '127.0.0.1', port, exclusive: true}, () => server.close(() => resolve(true))); }); }
async function evaluate(expression) { const result = await page.send('Runtime.evaluate', {expression, returnByValue: true, awaitPromise: true}); invariant(!result.exceptionDetails, 'Fixture evaluation ' + JSON.stringify(result.exceptionDetails)); return result.result.value; }
const frames = () => evaluate('new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve)))');
const quick = id => evaluate('window.harness.quick(' + JSON.stringify(id) + ')');
const one = id => evaluate('window.harness.one(' + JSON.stringify(id) + ')');
const metrics = id => evaluate('window.harness.metrics(' + JSON.stringify(id) + ')');
const targetBox = (id, selector) => evaluate('window.harness.box(' + JSON.stringify(id) + ',' + JSON.stringify(selector) + ')');
async function start() {
  invariant(++spawnCount === 1, 'Second Chrome startup forbidden'); port = await freePort();
  await noLinks(PROFILE); await absent(PROFILE); await fs.mkdir(PROFILE); E.writes.push(PROFILE + path.sep + ' (owned temporary Chrome profile)');
  const args = ['--headless=new', '--user-data-dir=' + PROFILE, '--remote-debugging-address=127.0.0.1', '--remote-debugging-port=' + port, '--force-device-scale-factor=1', '--enable-automation', '--no-first-run', '--no-default-browser-check', '--disable-background-networking', '--disable-background-timer-throttling', '--disable-backgrounding-occluded-windows', '--disable-renderer-backgrounding', '--disable-component-update', '--disable-sync', '--disable-extensions', '--disable-default-apps', '--disable-domain-reliability', '--disable-client-side-phishing-detection', '--safebrowsing-disable-auto-update', '--disable-features=MediaRouter,OptimizationHints,Translate,AutofillServerCommunication', '--metrics-recording-only', '--password-store=basic', '--no-proxy-server', '--host-resolver-rules=MAP * ~NOTFOUND', '--disable-breakpad', '--disable-gpu', '--window-size=1600,1500', 'about:blank'];
  child = spawn(CHROME, args, {cwd: ROOT, stdio: 'ignore', windowsHide: true, shell: false});
  invariant(Number.isSafeInteger(child.pid) && !PROTECTED.has(child.pid), 'Wrong spawned PID');
  E.browser = {startupCount: spawnCount, port, launch: {pid: child.pid, executable: CHROME, args, stdio: 'ignore', shell: false}};
  child.once('error', error => { exitInfo = {error: String(error)}; }); child.once('exit', (code, signal) => { exitInfo = {code, signal}; });
  let version;
  await waitFor(async () => { invariant(!exitInfo, 'Own Chrome exited early'); try { const response = await fetch('http://127.0.0.1:' + port + '/json/version', {signal: AbortSignal.timeout(750)}); if (response.ok) { version = await response.json(); return !!version.webSocketDebuggerUrl; } } catch (error) { if (!['TimeoutError', 'TypeError'].includes(error.name)) throw error; } return false; }, 'owned Chrome endpoint', 15000);
  browser = await CDP.connect(version.webSocketDebuggerUrl);
  const processInfo = (await browser.send('SystemInfo.getProcessInfo')).processInfo, command = (await browser.send('Browser.getBrowserCommandLine')).arguments;
  E.browser.initialProcesses = processInfo.map(({type, id}) => ({type, id})); E.browser.version = await browser.send('Browser.getVersion');
  invariant(processInfo.some(value => value.type === 'browser' && value.id === child.pid), 'CDP does not own spawned browser PID');
  invariant(processInfo.every(value => !PROTECTED.has(value.id)), 'Protected PID in owned list');
  for (const arg of ['--user-data-dir=' + PROFILE, '--remote-debugging-port=' + port, '--force-device-scale-factor=1']) invariant(command.includes(arg), 'Owned Chrome argument mismatch ' + arg);
  E.browser.ownership = {pass: true, verifiedBeforeInputAt: new Date().toISOString(), method: 'Spawn PID equals CDP browser PID; exact profile/port/DPR startup arguments match Browser.getBrowserCommandLine; exclusive loopback port.', verifiedArgumentCount: 3}; verified = true;
  const target = await browser.send('Target.createTarget', {url: 'about:blank'}), attached = await browser.send('Target.attachToTarget', {targetId: target.targetId, flatten: true});
  E.browser.targetId = target.targetId; page = {send: (method, params = {}) => browser.send(method, params, attached.sessionId)};
  browser.listeners.push(event => { if (event.sessionId !== attached.sessionId) return; if (event.method === 'Runtime.exceptionThrown') exceptions.push(event.params.exceptionDetails); if (event.method === 'Network.requestWillBeSent') requests.push({url: event.params.request.url, method: event.params.request.method}); });
  await page.send('Runtime.enable'); await page.send('Page.enable'); await page.send('Network.enable'); await page.send('Network.setBlockedURLs', {urls: ['http://*', 'https://*', 'ws://*', 'wss://*']});
  const navigation = await page.send('Page.navigate', {url: pathToFileURL(FIXTURE).href}); invariant(!navigation.errorText, 'Fixture navigation ' + navigation.errorText);
  await waitFor(async () => (await evaluate('Boolean(window.harness?.ready)')) === true, 'fixture ready'); await frames();
}
async function move(x, y) { await page.send('Input.dispatchMouseEvent', {type: 'mouseMoved', x, y, button: 'none', buttons: 0}); await delay(100); await frames(); }
async function click(box) { invariant(box?.w > 0 && box.h > 0, 'Missing clickable fixture box'); const x = box.x + box.w / 2, y = box.y + box.h / 2; await move(x, y); await page.send('Input.dispatchMouseEvent', {type: 'mousePressed', x, y, button: 'left', buttons: 1, clickCount: 1}); await page.send('Input.dispatchMouseEvent', {type: 'mouseReleased', x, y, button: 'left', buttons: 0, clickCount: 1}); await frames(); }
async function key(keyValue, code, vk) { await page.send('Input.dispatchKeyEvent', {type: 'keyDown', key: keyValue, code, windowsVirtualKeyCode: vk, nativeVirtualKeyCode: vk}); await page.send('Input.dispatchKeyEvent', {type: 'keyUp', key: keyValue, code, windowsVirtualKeyCode: vk, nativeVirtualKeyCode: vk}); await frames(); }
async function wheel(box, deltaY, deltaX = 0) { const x = box.x + Math.min(120, box.w / 2), y = box.y + Math.min(60, box.h / 2); await move(x, y); await page.send('Input.dispatchMouseEvent', {type: 'mouseWheel', x, y, deltaY, deltaX}); await delay(220); await frames(); }
async function capture(name) { const result = await page.send('Page.captureScreenshot', {format: 'png', fromSurface: true, captureBeyondViewport: false}), bytes = Buffer.from(result.data, 'base64'); await writeOwned(OUTPUT[name], bytes); E.screenshots[name] = {path: OUTPUT[name], sha256: hash(bytes)}; }
async function accessibility(selector) { const doc = await page.send('DOM.getDocument', {depth: 0}), node = await page.send('DOM.querySelector', {nodeId: doc.root.nodeId, selector}); invariant(node.nodeId, 'Missing AX selector'); return (await page.send('Accessibility.getPartialAXTree', {nodeId: node.nodeId, fetchRelatives: false})).nodes.map(value => ({role: value.role?.value, name: value.name?.value, properties: value.properties?.map(p => ({name: p.name, value: p.value?.value}))})); }
async function runCase(id, fn) {
  const current = {id, status: 'running', assertions: []}; E.cases.push(current);
  const check = (name, pass, failureDetail) => current.assertions.push({name, pass: !!pass, ...(!pass && failureDetail !== undefined ? {detail: failureDetail} : {})});
  try { await fn(check); current.status = current.assertions.every(value => value.pass) ? 'passed' : 'failed'; }
  catch (error) { current.status = 'failed'; current.error = error.stack ?? String(error); }
  console.log(current.status.toUpperCase() + ' ' + id + ' ' + current.assertions.filter(value => value.pass).length + '/' + current.assertions.length);
}
async function acceptance(models) {
  await move(5, 5); E.tokens = await evaluate('window.harness.tokens()'); E.browser.viewport = await evaluate('({dpr:devicePixelRatio,innerWidth,innerHeight})');
  E.fixtureData = models.map(value => ({id: value.id, rawChars: value.raw.length, rawSHA256: hash(value.raw), programChars: value.bodyText?.length ?? null, programSHA256: value.bodyText === null ? null : hash(value.bodyText), outputChars: value.output?.length ?? null, outputSHA256: value.output === null ? null : hash(value.output), state: value.state}));
  await runCase('source-only-formatToolBody-table', async check => {
    const table = source.sourceOnly.formatToolBody.table;
    check('all14 cases run exact installed parseArgs/formatToolBody without imports', table.length === 14 && table.every(value => value.pass));
    check('empty raw is null; empty code field retains JSON envelope', table.find(value => value.id === 'empty-raw').result === null && table.find(value => value.id === 'empty-code-field').result === '{\n  "code": ""\n}');
    check('invalid JSON raw verbatim, whitespace and CRLF programs preserved', table.find(value => value.id === 'invalid-json').result === '{"code":"partial' && table.find(value => value.id === 'whitespace-code').result === '  \n' && table.find(value => value.id === 'crlf-program').result === 'a\r\nb\r\n');
  });
  await runCase('open-program-output-exact-metrics', async check => {
    const s = await one('baseline'), k = await metrics('baseline'); E.measurements.geometry = {final: [s]};
    check('single startup DPR1 with600px fixture row24', spawnCount === 1 && E.browser.viewport.dpr === 1 && near(s.own.box.w, 600) && near(s.row.box.h, 24), E.browser.viewport);
    check('body flex-column and code parent max260px auto scroll', k['[data-body-wrap]'].display === 'flex' && k['[data-body-scroll]'].maxHeight === '260px' && k['[data-body-scroll]'].overflowY === 'auto');
    check('codeBody margin4 0 4 4, radius16, no border', ['marginTop', 'marginBottom', 'marginLeft'].every(name => k['[data-code-block]'][name] === '4px') && k['[data-code-block]'].marginRight === '0px' && k['[data-code-block]'].borderRadius === '16px' && k['[data-code-block]'].borderTopWidth === '0px');
    check('code x4, width596 and y4 from row bottom', near(s.block.box.x - s.row.box.x, 4) && near(s.block.box.w, 596) && near(s.block.box.y - s.row.box.y - s.row.box.h, 4));
    check('toolbar height42 padding10 18 8 22; gap12; font11/18', near(s.toolbar.box.h, 42) && k['[data-code-block-banner]'].paddingTop === '10px' && k['[data-code-block-banner]'].paddingRight === '18px' && k['[data-code-block-banner]'].paddingBottom === '8px' && k['[data-code-block-banner]'].paddingLeft === '22px' && k['[data-code-block-banner]'].columnGap === '12px' && k['[data-code-block-banner]'].fontSize === '11px' && k['[data-code-block-banner]'].lineHeight === '18px');
    check('toolbar2 icon buttons24x24 glyph14 gap4 without visible copy text', s.buttons.length === 2 && s.buttons.every(value => near(value.box.w, 24) && near(value.box.h, 24) && near(value.icon.box.w, 14) && near(value.icon.box.h, 14) && value.tag === 'BUTTON' && value.type === 'button' && value.text === '') && k['[data-toolbar-actions]'].columnGap === '4px');
    check('typescript language11/18; sticky banner top0 z6; content display contents', s.language.text === 'typescript' && k['[data-language]'].fontSize === '11px' && k['[data-language]'].lineHeight === '18px' && k['[data-banner-wrap]'].position === 'sticky' && k['[data-banner-wrap]'].top === '0px' && k['[data-banner-wrap]'].zIndex === '6' && k['[data-code-block-content]'].display === 'contents');
    check('pre font11/19 padding6 22 20 and default wrap anywhere normal word-break', k.pre.fontSize === '11px' && k.pre.lineHeight === '19px' && k.pre.paddingTop === '6px' && k.pre.paddingRight === '22px' && k.pre.paddingBottom === '20px' && k.pre.paddingLeft === '22px' && k.pre.whiteSpace === 'pre-wrap' && k.pre.wordBreak === 'normal' && k.pre.overflowWrap === 'anywhere');
    check('short2line pre64; block106; viewport114 content-box', s.pre.lines === 2 && near(s.pre.box.h, 64) && near(s.block.box.h, 106) && near(s.viewport.box.h, 114) && k['[data-body-scroll]'].boxSizing === 'content-box');
    check('OUT only1section noIN no divider no output copy', s.io.sectionCount === 1 && s.io.section.label.text === 'OUT' && s.io.dividerCount === 0 && s.io.buttonCount === 0);
    check('OUT margin4 0 4 4, declared halfpixel used1 border, radius16', k['[data-io-card]'].marginTop === '4px' && k['[data-io-card]'].marginLeft === '4px' && k['[data-io-card]'].marginBottom === '4px' && k['[data-io-card]'].marginRight === '0px' && k['[data-io-card]'].borderTopWidth === '1px' && k['[data-io-card]'].borderRadius === '16px');
    check('OUT11/16 max150 content-box padding12x16 grid gap14 sticky label', k['[data-io-card]'].fontSize === '11px' && k['[data-io-card]'].lineHeight === '16px' && k['[data-io-section]'].maxHeight === '150px' && k['[data-io-section]'].boxSizing === 'content-box' && k['[data-io-section]'].paddingTop === '12px' && k['[data-io-section]'].paddingBottom === '12px' && k['[data-io-section]'].paddingLeft === '16px' && k['[data-io-section]'].paddingRight === '16px' && k['[data-io-section]'].display === 'grid' && k['[data-io-section]'].columnGap === '14px' && k['[data-io-label]'].position === 'sticky' && k['[data-io-label]'].top === '0px');
    check('OUT text pre-wrap break-word secondary and nonerror', k['[data-io-text]'].whiteSpace === 'pre-wrap' && k['[data-io-text]'].wordBreak === 'break-word' && k['[data-io-text]'].color === k['[data-code-block-banner]'].color && !s.io.section.text.error);
    await capture('initial');
  });
  await runCase('invalid-empty-code-and-raw-branches', async check => {
    const invalid = await quick('invalid'), emptyField = await quick('empty-field');
    const snapshots = await Promise.all(['invalid', 'empty-field', 'empty-raw', 'empty-raw-no-output'].map(one)); E.measurements.inputBranches = {final: snapshots};
    check('invalid arguments render raw in typescript CodeBlock without fabricated IN', invalid.preText === models.find(value => value.id === 'invalid').raw && snapshots[0].language.text === 'typescript' && snapshots[0].io === null);
    check('empty code field renders pretty JSON fallback rather than no-code placeholder', emptyField.preText === JSON.stringify(JSON.parse(models.find(value => value.id === 'empty-field').raw), null, 2) && snapshots[1].block !== null && snapshots[1].pre.lines === 4);
    check('empty raw suppresses CodeBlock while output still has OUT card', snapshots[2].block === null && snapshots[2].viewport === null && snapshots[2].io?.section.label.text === 'OUT');
    check('empty raw and null output retain expandable24px header with empty body', snapshots[3].block === null && snapshots[3].io === null && snapshots[3].row.expanded === 'true' && near(snapshots[3].own.box.h, 24));
  });
  await runCase('empty-output-null-output-and-error-color', async check => {
    const empty = await one('empty-output'), missing = await one('null-output'), k = await metrics('empty-output'); E.measurements.outputBranches = {final: [empty, missing]};
    check('empty string output produces OUT1section plus code block', empty.block !== null && empty.io?.sectionCount === 1 && empty.io.section.text.chars === 0);
    check('empty OUT still label16 plus padding24 => section40 card42', near(empty.io.section.box.h, 40) && near(empty.io.box.h, 42));
    check('null output omits entire IO card and keeps program', missing.io === null && missing.block !== null);
    check('error OUT marks data-error and uses installed error token', empty.io.section.text.error && k['[data-io-text]'].color !== k['[data-code-block-banner]'].color && k['[data-io-text]'].color === await evaluate('getComputedStyle(document.querySelector(\'[data-own-card="empty-output"] .' + source.maps.row.errorSummary + '\')).color'), {actual: k['[data-io-text]'].color, token: E.tokens['--dsw-alias-state-error-primary']});
  });
  await runCase('default-wrapped-and-click-unwrapped-controls', async check => {
    const before = await quick('wrap'), styleBefore = (await metrics('wrap')).pre;
    await click(await targetBox('wrap', '[data-wrap]'));
    const unwrapped = await quick('wrap'), k = await metrics('wrap');
    E.measurements.wrap = {transition: {wrapped: {pressed: before.wrapPressed, tooltip: before.wrapTooltip, icon: before.wrapIcon, height: before.preHeight, horizontalOverflow: before.preScroll.sw > before.preScroll.cw}, unwrapped: {pressed: unwrapped.wrapPressed, tooltip: unwrapped.wrapTooltip, icon: unwrapped.wrapIcon, height: unwrapped.preHeight, horizontalOverflow: unwrapped.preScroll.sw > unwrapped.preScroll.cw}}, final: [await one('wrap')]};
    check('default wrappedtrue pressedtrue nowrapglyph and unwrap tooltip', before.wrapped === 'true' && before.wrapPressed === 'true' && before.wrapIcon === 'nowrap' && before.wrapTooltip === 'Do not wrap lines' && styleBefore.whiteSpace === 'pre-wrap');
    check('trusted click false swaps wrapglyph, pressedfalse and wrap tooltip', unwrapped.wrapped === 'false' && unwrapped.wrapPressed === 'false' && unwrapped.wrapIcon === 'wrap' && unwrapped.wrapTooltip === 'Wrap lines');
    check('aria-label remains Wrap lines in both states', before.wrapLabel === 'Wrap lines' && unwrapped.wrapLabel === 'Wrap lines');
    check('wrapped long source fits horizontally; unwrapped overflows and shrinks to45px pre', before.preScroll.sw === before.preScroll.cw && unwrapped.preScroll.sw > unwrapped.preScroll.cw && before.preHeight > unwrapped.preHeight && near(unwrapped.preHeight, 45));
    check('unwrapped pre/normal retains overflow-x-auto raw text', k.pre.whiteSpace === 'pre' && k.pre.overflowWrap === 'normal' && k.pre.overflowX === 'auto' && unwrapped.preText === before.preText);
    check('wrap button click leaves disclosure open', unwrapped.expanded === 'true'); await capture('wrap');
  });
  await runCase('copy-displayed-source-and-one-second-feedback', async check => {
    const expectedRaw = models.find(value => value.id === 'baseline').bodyText, beforeAX = await accessibility('[data-own-card="baseline"] [data-copy]');
    await click(await targetBox('baseline', '[data-copy]')); const copied = await quick('baseline'), afterAX = await accessibility('[data-own-card="baseline"] [data-copy]');
    await capture('copy'); await click(await targetBox('baseline', '[data-copy]'));
    const blockedCount = (await evaluate('window.harness.clipboard()')).length;
    await waitFor(async () => (await quick('baseline')).copyLabel === 'Copy', 'copy feedback reset', 2500);
    const feedback = await evaluate('window.harness.feedback()'), first = feedback.find(value => value.id === 'baseline' && value.event === 'copied'), reset = feedback.find(value => value.id === 'baseline' && value.event === 'reset');
    await evaluate('window.harness.setClipboardOK(false)'); await click(await targetBox('baseline', '[data-copy]')); const refused = await quick('baseline'); await evaluate('window.harness.setClipboardOK(true)');
    const records = await evaluate('window.harness.clipboard()');
    E.measurements.copy = {behavior: {beforeAX, afterAX, copiedLabel: copied.copyLabel, copiedIcon: copied.copyIcon, guardedRepeatedClickCount: blockedCount, feedbackElapsedMS: reset.at - first.at, refusedLabel: refused.copyLabel, records: records.map(value => ({id: value.id, ok: value.ok, chars: value.text.length, sha256: hash(value.text), adapter: value.adapter}))}, final: [await one('baseline')]};
    check('copies displayed raw program without JSON envelope, trims exactly1 terminalLF', records[0].text === expectedRaw.slice(0, -1) && records[0].text !== models.find(value => value.id === 'baseline').raw && !records[0].text.endsWith('\n'));
    check('native button AX Copy changes Copied and glyphcheck', beforeAX.some(value => value.role === 'button' && value.name === 'Copy') && afterAX.some(value => value.role === 'button' && value.name === 'Copied') && copied.copyLabel === 'Copied' && copied.copyIcon === 'check');
    check('click during copied flag does not write again', blockedCount === 1);
    check('feedback lasts installed1000ms and resets label/glyph', reset.at - first.at >= 995 && reset.at - first.at < 1500 && refused.copyLabel === 'Copy' && refused.copyIcon === 'copy', {elapsedMS: reset.at - first.at});
    check('refused in-memory write has no copied feedback', records.length === 2 && !records[1].ok && feedback.filter(value => value.event === 'copied').length === 1);
    check('copy never collapses disclosure; system clipboard adapter absent', copied.expanded === 'true' && refused.expanded === 'true' && records.every(value => value.adapter === 'in-memory fixture only'));
  });
  await runCase('explicit-disclosure-keydown-and-toolbar-tab-focus', async check => {
    await click(await targetBox('baseline', '[data-disclosure-row]')); const closedClick = await quick('baseline');
    await key('Enter', 'Enter', 13); const openEnter = await quick('baseline'); await key(' ', 'Space', 32); const closeSpace = await quick('baseline'); await key('Enter', 'Enter', 13);
    await key('Tab', 'Tab', 9); const focused = await quick('baseline'), k = await metrics('baseline');
    E.measurements.keyboard = {transition: {clickExpanded: closedClick.expanded, enterExpanded: openEnter.expanded, spaceExpanded: closeSpace.expanded, focusAfterTab: focused.focus, focusVisible: focused.focusVisible}, final: [await one('baseline')]};
    check('trusted row click closes only fixture disclosure', closedClick.expanded === 'false' && !closedClick.visible);
    check('explicit row keydown Enter reopens, Space closes, Enter reopens', openEnter.expanded === 'true' && closeSpace.expanded === 'false' && focused.expanded === 'true');
    check('Tab reaches first toolbar button with native focus-visible', focused.focus === 'wrap' && focused.focusVisible);
    check('installed toolbar focus ring1px and offset2', k['[data-wrap]'].outlineStyle === 'solid' && k['[data-wrap]'].outlineWidth === '1px' && k['[data-wrap]'].outlineOffset === '2px');
    check('no native button keyboard activation is claimed', E.scope.input.includes('no native button Enter'));
  });
  await runCase('long-code-cap-output-cap-independent-wheels', async check => {
    const before = await quick('long-caps'), k = await metrics('long-caps');
    const codeBox = await targetBox('long-caps', '[data-body-scroll]'), outputBox = await targetBox('long-caps', '[data-io-section]');
    await wheel(codeBox, 220); const codeScrolled = await quick('long-caps'); await wheel(outputBox, 220); const both = await quick('long-caps');
    const final = await one('long-caps'); E.measurements.scroll = {transition: {initialCodeY: before.codeScroll.y, initialOutputY: before.outputScroll.y, afterCodeWheel: {codeY: codeScrolled.codeScroll.y, outputY: codeScrolled.outputScroll.y}, afterOutputWheel: {codeY: both.codeScroll.y, outputY: both.outputScroll.y}}, final: [final]};
    check('code parent260 outer with50 lines and no expansion control', near(final.viewport.box.h, 260) && before.codeScroll.sh > before.codeScroll.ch && final.pre.lines === 50 && final.buttons.length === 2);
    check('output max150 content-box creates174 section and176 card', k['[data-io-section]'].maxHeight === '150px' && k['[data-io-section]'].boxSizing === 'content-box' && near(final.io.section.box.h, 174) && near(final.io.box.h, 176) && before.outputScroll.sh > before.outputScroll.ch);
    check('trusted code wheel scrolls code while output stays0', codeScrolled.codeScroll.y > 0 && codeScrolled.outputScroll.y === 0);
    check('trusted output wheel scrolls output while code position retained', both.outputScroll.y > 0 && both.codeScroll.y === codeScrolled.codeScroll.y);
    check('sticky toolbar remains code-scrollport top after codewheel', near(final.banner.box.y, final.viewport.box.y) && near(final.toolbar.box.y, final.viewport.box.y));
    check('root and code block do not acquire independent scroll offsets', final.block.scroll.y === 0 && final.io.section.scroll.y > 0); await capture('scroll');
  });
  await runCase('manual-spans-exact-installed-token-colors', async check => {
    const s = await one('manual-highlight'), registry = await evaluate('window.harness.styles()'), style = id => registry.find(value => value.id === id).value;
    E.measurements.highlight = {mode: 'manual authored spans, no grammar/tokenizer execution', final: [s]};
    check('manual span program equals displayed source exactly', (await quick('manual-highlight')).preText === 'const answer = 42;');
    check('keyword exact installed #d6336c and constant #1c7ed6', s.manualTokens.some(value => value.token === 'keyword' && style(value.style).color === 'rgb(214, 51, 108)') && s.manualTokens.some(value => value.token === 'constant' && style(value.style).color === 'rgb(28, 126, 214)'));
    check('punctuation exact installed #495057', s.manualTokens.filter(value => value.token === 'punctuation').every(value => style(value.style).color === 'rgb(73, 80, 87)'));
    check('shiki-like manual wrapper keeps font11/19 padding and tabindex0', style(s.pre.style).fontSize === '11px' && style(s.pre.style).lineHeight === '19px' && style(s.pre.style).paddingTop === '6px' && s.pre.tabIndex === '0');
  });
  await runCase('source-input-network-custody', async check => {
    const changed = [];
    for (const item of source.files) if (hash(await fs.readFile(item.path)) !== item.sha256) changed.push(item.id);
    E.sourceIntegrity = {allUnchanged: changed.length === 0, changedSourceIDs: changed};
    const events = await evaluate('window.harness.events()'); E.input = {count: events.length, trusted: events.every(value => value.trusted), byType: Object.fromEntries([...new Set(events.map(value => value.type))].map(type => [type, events.filter(value => value.type === type).length])), keyboard: events.filter(value => value.type === 'keydown').map(({id, key, target, trusted}) => ({id, key, target, trusted})), clipboardNativeCalls: 0};
    E.network = {requests, exceptions, protocolErrors};
    check('all installed/reference/prior/new runner source bytes unchanged', changed.length === 0, changed);
    check('page file only, no runtime or protocol errors', requests.every(value => !/^https?:|^wss?:/i.test(value.url)) && exceptions.length === 0 && protocolErrors.length === 0, E.network);
    check('observed CDP hover click key wheel all trusted after ownership gate', E.input.trusted && ['mousemove', 'click', 'keydown', 'wheel'].every(type => E.input.byType[type] > 0) && E.browser.ownership.pass);
    check('activation keydowns target explicit row only; Tab used for focus', E.input.keyboard.every(value => value.key === 'Tab' || value.target === 'row'), E.input.keyboard);
    check('one startup, no Emulation or native system clipboard methods', spawnCount === 1 && Object.keys(methodCounts).every(value => !value.startsWith('Emulation.')) && E.input.clipboardNativeCalls === 0);
  });
  E.computedStyles = await evaluate('window.harness.styles()');
}
async function cleanup() {
  const current = {target: 'Only spawned exact Chrome PID/profile/port; no protected process queries/signals or original GUI requests', protectedProcessesSignalled: false, protectedGuiRequested: false}; E.cleanup = current;
  if (!child) { current.pass = true; current.browserNeverStarted = true; return; }
  invariant(!PROTECTED.has(child.pid), 'Protected cleanup PID');
  if (verified && browser?.socket.readyState === WebSocket.OPEN) {
    try { current.lastProcesses = (await browser.send('SystemInfo.getProcessInfo')).processInfo.map(({type, id}) => ({type, id})); invariant(current.lastProcesses.every(value => !PROTECTED.has(value.id)), 'Protected PID in own cleanup process list'); await browser.send('Browser.close'); current.closedBy = 'owned Browser.close'; }
    catch (error) { current.closeError = String(error); }
  }
  const end = Date.now() + 7000; while (!exitInfo && Date.now() < end) await delay(50);
  if (!exitInfo) { current.killedSpawnedChild = child.kill(); await delay(500); }
  current.exit = exitInfo ?? {pending: true}; if (browser?.socket.readyState === WebSocket.OPEN) browser.socket.close();
  await waitFor(async () => { current.portReleased = await portFree(); return current.portReleased; }, 'owned port release', 5000);
  const ids = [...new Set([...(E.browser?.initialProcesses ?? []), ...(current.lastProcesses ?? []), {id: child.pid}].map(value => value.id))];
  const alive = id => { invariant(!PROTECTED.has(id), 'Protected PID liveness probe'); try { process.kill(id, 0); return true; } catch (error) { if (error.code === 'ESRCH') return false; throw error; } };
  await waitFor(async () => { current.processes = ids.map(pid => ({pid, alive: alive(pid)})); return current.processes.every(value => !value.alive); }, 'owned process exits', 5000);
  current.allObservedProcessesExited = ids.length > 0 && current.processes.every(value => !value.alive);
  if (exitInfo && current.portReleased && current.allObservedProcessesExited) {
    invariant(norm(PROFILE) === norm(BASE + '-profile') && norm(PROFILE).startsWith(norm(path.join(ROOT, '.devdata/r116-tool-code-'))), 'Wrong profile removal target');
    await noLinks(PROFILE); invariant(norm(await fs.realpath(PROFILE)) === norm(PROFILE), 'Redirected cleanup profile');
    current.profileRemovalTarget = PROFILE; await fs.rm(PROFILE, {recursive: true, force: false}); current.profileRemoved = true;
  }
  current.pass = exitInfo?.code === 0 && current.portReleased && current.allObservedProcessesExited && current.profileRemoved;
}
function textReport() {
  return ['Round116 installed run_code: ' + E.status.toUpperCase(), 'Run ' + RUN_ID,
    'Cases ' + E.summary.passedCases + '/' + E.cases.length + '; assertions ' + E.summary.passedAssertions + '/' + (E.summary.passedAssertions + E.summary.failedAssertions),
    ...E.cases.map(value => value.status.toUpperCase() + ' ' + value.id + ' ' + value.assertions.filter(item => item.pass).length + '/' + value.assertions.length),
    ...E.cases.flatMap(value => value.assertions.filter(item => !item.pass).map(item => 'FAIL ' + value.id + ': ' + item.name + ' ' + JSON.stringify(item.detail ?? null))),
    ...E.cases.filter(value => value.error).map(value => value.id + ': ' + value.error),
    'Exact installed baseline: code viewport260; codeBody margin4 0 4 4; border0/radius16; toolbar42 padding10 18 8 22; buttons24 glyph14; pre11/19 padding6 22 20; OUT11/16 max150 content/174 outer.',
    'Copy uses displayed program trimmed by exactly one terminal LF, memory adapter only. Wrapping defaults true. No native-button Enter assertion. Manual token color sample is not tokenizer execution.',
    'Source evidence ' + OUTPUT.source, 'Cleanup ' + JSON.stringify(E.cleanup), 'Errors ' + JSON.stringify(E.errors), ''].join('\n');
}
function report() {
  const s = E.measurements.geometry?.final[0], l = E.measurements.scroll?.final[0];
  const mdLink = (name, target) => '[' + name + '](<' + target.replaceAll('\\', '/') + '>)';
  return '# Round116 installed run_code measurement\n\n' + E.status.toUpperCase() + ': ' + E.summary.passedCases + '/' + E.cases.length + ' cases; ' + E.summary.passedAssertions + '/' + (E.summary.passedAssertions + E.summary.failedAssertions) + ' assertions. ' + mdLink('Compact measurements', OUTPUT.json) + ' · ' + mdLink('Unique CSS/maps/source hashes and source-only table', OUTPUT.source) + ' · ' + mdLink('Runner', SCRIPT) + '.\n\n' +
    E.scope.description + '\n\n' +
    '| Installed baseline | Exact declaration / measured use |\n|---|---|\n' +
    '| ToolRow header | 24px high; code block begins +4px horizontally and +4px below header. |\n' +
    '| Code scroll owner | bodyScroll max-height:260px; overflow-y:auto; content-box. The codeBody override only supplies margin:4px 0 4px 4px. |\n' +
    '| CodeBlock | border:0; radius-lg=16px; width596 at row600. Default wrapped=true; no line numbers. |\n' +
    '| Toolbar | height42; padding10px 18px 8px 22px; gap12; text11px/18px; actions gap4; native buttons24×24, glyph14. |\n' +
    '| Header containment | bannerWrap sticky top0/z6; CodeToolbar inside; language typescript; title/status omitted. content display:contents. |\n' +
    '| Program pre | 11px/19px; padding6px 22px 20px; pre-wrap/word-break:normal/overflow-wrap:anywhere. Unwrapped uses pre/overflow-wrap:normal and horizontal auto overflow. |\n' +
    '| OUT card | Only OUT, no IN or divider/copy. Margin4px 0 4px 4px; declared .5px border uses1px at DPR1; radius16. Font11px/16px. |\n' +
    '| OUT section | grid max-content 1fr; gap14; padding12px 16px; max-height150px content-box gives174px outer section,176px bordered card. Sticky label top0. |\n\n' +
    (s ? 'Measured short program: pre ' + s.pre.box.w + '×' + s.pre.box.h + ', CodeBlock ' + s.block.box.w + '×' + s.block.box.h + ', code viewport ' + s.viewport.box.w + '×' + s.viewport.box.h + '. ' : '') +
    (l ? 'Long program viewport ' + l.viewport.box.h + 'px and OUT section ' + l.io.section.box.h + 'px scroll independently; trusted wheel retained the other scroll offset and the toolbar remained at the code-scrollport top.\n\n' : '\n\n') +
    'The source-only formatToolBody table executes the two exact installed pure functions in a VM without model imports: empty raw is null, invalid JSON remains raw, a nonempty code string is returned verbatim, and empty/missing/nonstring code falls back to pretty JSON. Thus code:"" still draws a CodeBlock; raw empty suppresses it. Output empty string still draws a label and40px OUT section; output null omits IO.\n\n' +
    'The wrap control keeps aria-label="Wrap lines" while aria-pressed and nowrap/wrap glyph change. Tooltip label metadata changes between "Do not wrap lines" and "Wrap lines". Copy reads pre.textContent after removal of exactly one terminal LF; it copies the program without the JSON envelope or line-number text, then shows Copied/check for1000ms after a successful local adapter result. Repeated clicks during feedback are ignored; a refused adapter result keeps Copy. No system clipboard access occurred.\n\n' +
    'Disclosure keyboard assertions cover only the explicit fixture div keydown handlers for Enter/Space. Tab focuses the first native toolbar button and exposes the1px ring with2px offset. Native-button Enter activation was deliberately not asserted because keydown/keyup without a char event can miss that Chrome default action. Tooltip portals, React, usePresence/motion, Shiki tokenization/grammar loading/streaming, and the original GUI/model/host remain unexecuted. The colored sample uses honest manual spans with exact installed light token colors (#d6336c keyword, #1c7ed6 constant, #495057 punctuation).\n\n' +
    'Source files and prior runners unchanged: ' + String(E.sourceIntegrity?.allUnchanged) + '. Startup count: ' + spawnCount + '; measured DPR: ' + E.browser?.viewport?.dpr + '. Cleanup: ' + String(E.cleanup?.pass) + ' (all observed owned PIDs exited, loopback port released, owned profile removed). Protected Electron20012/backend16008/LockApp12516 and port3080 were not used or mutated.\n\n' +
    Object.entries(E.screenshots).map(([name, value]) => mdLink(name, value.path)).join(' · ') + '\n\n' +
    E.cases.filter(value => value.status !== 'passed').map(value => 'Failure retained: ' + value.id + ' ' + (value.error ?? value.assertions.filter(item => !item.pass).map(item => item.name).join('; '))).join('\n\n') + (E.errors.length ? '\n\nErrors: ' + E.errors.join('\n\n') : '') + '\n';
}
async function main() {
  await scope();
  const extracted = await extract(), built = fixture(extracted), inline = built.html.match(/<script>\n([\s\S]*)<\/script>/);
  invariant(inline, 'Missing inline fixture'); new vm.Script(inline[1], {filename: 'r116-tool-code-inline.js'});
  if (MODE === '--prepare-only') {
    await writeOwned(OUTPUT.source, JSON.stringify(source) + '\n'); await writeOwned(FIXTURE, built.html);
    const preflight = {schema: 'r116-tool-code-preflight/v1', status: 'prepared-no-browser', runId: RUN_ID, sourceEvidence: OUTPUT.source, sourceEvidenceSHA256: hash(await fs.readFile(OUTPUT.source)), fixture: FIXTURE, fixtureSHA256: hash(built.html), inlineParsed: true, sourceTableCases: source.sourceOnly.formatToolBody.table.length, chromeNotStarted: true, scopeWritePathsValidated: true};
    await writeOwned(PREP, JSON.stringify(preflight, null, 2) + '\n');
    console.log(JSON.stringify({...preflight, path: PREP, sourceEvidenceBytes: (await fs.stat(OUTPUT.source)).size})); return;
  }
  const prep = JSON.parse(await fs.readFile(PREP, 'utf8'));
  invariant(prep.status === 'prepared-no-browser' && prep.runId === RUN_ID && norm(prep.sourceEvidence) === norm(OUTPUT.source), 'Unowned preparation');
  invariant(hash(await fs.readFile(OUTPUT.source)) === prep.sourceEvidenceSHA256, 'Source sidecar changed after preflight');
  const preparedSource = JSON.parse(await fs.readFile(OUTPUT.source, 'utf8'));
  invariant(JSON.stringify(source) === JSON.stringify(preparedSource), 'Installed source or runner changed after preflight');
  invariant(prep.fixtureSHA256 === hash(built.html) && hash(await fs.readFile(FIXTURE)) === prep.fixtureSHA256, 'Prepared fixture changed');
  E.preflight = {path: PREP, inlineParsed: true, sourceEvidenceVerified: true, fixtureVerified: true};
  await writeOwned(ATTEMPT, JSON.stringify({round: 116, runId: RUN_ID, startedAt: E.startedAt, prepared: PREP, chromeStartupsAllowed: 1, rerunForbidden: true}, null, 2) + '\n');
  try { await start(); await acceptance(built.models); }
  catch (error) { E.errors.push(error.stack ?? String(error)); console.error(error.stack ?? String(error)); }
  finally { try { await cleanup(); } catch (error) { E.errors.push('Cleanup ' + (error.stack ?? String(error))); } }
  E.finishedAt = new Date().toISOString(); E.cdpMethods = methodCounts;
  E.summary = {passedCases: E.cases.filter(value => value.status === 'passed').length, failedCases: E.cases.filter(value => value.status === 'failed').length, passedAssertions: E.cases.flatMap(value => value.assertions).filter(value => value.pass).length, failedAssertions: E.cases.flatMap(value => value.assertions).filter(value => !value.pass).length};
  E.status = E.cases.length === 10 && E.summary.failedCases === 0 && E.errors.length === 0 && E.cleanup?.pass ? 'passed' : 'failed';
  E.compact = {styleRecords: E.computedStyles?.length ?? 0, sourceJSONBytes: (await fs.stat(OUTPUT.source)).size, measurementJSONBytes: 0, targetCombinedJSONBytes: 300000};
  let encoded = JSON.stringify(E) + '\n'; E.compact.measurementJSONBytes = Buffer.byteLength(encoded); encoded = JSON.stringify(E) + '\n';
  E.compact.measurementJSONBytes = Buffer.byteLength(encoded); encoded = JSON.stringify(E) + '\n';
  if (Buffer.byteLength(encoded) + E.compact.sourceJSONBytes >= 300000) { E.errors.push('Combined JSON exceeds300KB target'); E.status = 'failed'; encoded = JSON.stringify(E) + '\n'; }
  await writeOwned(OUTPUT.text, textReport()); await writeOwned(OUTPUT.report, report()); await writeOwned(OUTPUT.json, encoded);
  console.log(JSON.stringify({status: E.status, ...E.summary, evidence: OUTPUT, combinedJSONBytes: Buffer.byteLength(encoded) + E.compact.sourceJSONBytes, cleanup: E.cleanup?.pass}));
  if (E.status !== 'passed') process.exitCode = 1;
}
await main();
