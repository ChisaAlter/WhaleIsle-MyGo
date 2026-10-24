#!/usr/bin/env node
/** Round118: current installed ReadBlock chrome and read metadata in one owned Chrome. */
import fs from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';
import {createHash, randomUUID} from 'node:crypto';
import net from 'node:net';
import vm from 'node:vm';
import {spawn} from 'node:child_process';

const ROOT = 'C:\\AI\\WhaleIsle-MyGo';
const SCRIPT = path.join(ROOT, 'scripts/tool-read-reference-chrome.mjs');
const INSTALL = 'C:\\软件\\Whale Isle\\resources\\vendor\\deepseek-harness';
const TOOL = path.join(INSTALL, 'packages/client/ui-tool/lib');
const REF = path.join(ROOT, 'reference/electron/vendor/deepseek-harness/packages/client/ui-primitives/src');
const SOURCE_PATHS = {
  runner: SCRIPT, webJS: path.join(INSTALL, 'apps/web/dist/assets/index-wKUmVWbt.js'), webCSS: path.join(INSTALL, 'apps/web/dist/assets/index-l5jToo6l.css'),
  tool: path.join(TOOL, 'client.js'), row: path.join(TOOL, 'types/client/tool/components/ToolRow.js'),
  read: path.join(TOOL, 'types/client/tool/models/read-card-model.js'), raw: path.join(TOOL, 'types/client/tool/models/raw-tool-call.js'),
  caller: path.join(TOOL, 'types/client/tool/toolviews/read-row.js'), family: path.join(TOOL, 'types/client/tool/toolviews/read-family-row.js'),
  labels: path.join(TOOL, 'types/client/tool/models/primitive-labels.js'), paths: path.join(INSTALL, 'packages/util/workspace-path/lib/types/index.js'),
  theme: path.join(INSTALL, 'packages/client/ui-theme/lib/client.js'), locale: path.join(INSTALL, 'packages/client/ui-conversation/lib/types/client/locales.js'),
  common: path.join(INSTALL, 'packages/client/locale/lib/types/locales/en.js'),
  refRead: path.join(REF, 'ReadBlock.tsx'), refReadCSS: path.join(REF, 'ReadBlock.module.css'), refCardCSS: path.join(REF, 'CodeCard.module.css'),
  refToolbar: path.join(REF, 'CodeToolbar.tsx'), refFold: path.join(REF, 'FoldToggle.tsx'), icons: path.join(REF, 'icons/index.tsx'),
  priorCode: path.join(ROOT, 'scripts/tool-code-reference-chrome.mjs'), priorSummary: path.join(ROOT, 'scripts/tool-summary-reference-chrome.mjs'),
};
const CHROME = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const MODE = process.argv[2];
const suppliedPrep = MODE === '--accept-prepared' ? process.argv[3] : null;
const suppliedRun = suppliedPrep && path.basename(suppliedPrep).match(/^r118-tool-read-([0-9a-fTZ-]+)-preflight\.json$/)?.[1];
const RUN_ID = suppliedRun ?? new Date().toISOString().replace(/[:.]/g, '-') + '-' + randomUUID().slice(0, 8);
const BASE = path.join(ROOT, '.devdata/r118-tool-read-' + RUN_ID);
const FIXTURE = BASE + '.html', PROFILE = BASE + '-profile', PREP = BASE + '-preflight.json';
const ATTEMPT = path.join(ROOT, '.devdata/r118-tool-read-acceptance-attempt.json');
const PREFIX = path.join(ROOT, 'docs/review-20261022/evidence/tool-read-pass262-reference-' + RUN_ID);
const OUTPUT = {source: PREFIX + '-source.json', json: PREFIX + '.json', report: PREFIX + '-report.md', text: PREFIX + '.txt', initial: PREFIX + '-initial.png', copied: PREFIX + '-copied.png', expanded: PREFIX + '-expanded.png', wrapped: PREFIX + '-wrapped.png'};
const PROTECTED = new Set([20012, 16008, 12516]);
const norm = value => path.resolve(value).toLowerCase();
const hash = value => createHash('sha256').update(value).digest('hex');
const invariant = (value, message) => { if (!value) throw Error(message); };
const near = (a, b) => Math.abs(a - b) < 0.04;
const delay = ms => { invariant(ms <= 1000, 'Long blocking wait forbidden'); return new Promise(resolve => setTimeout(resolve, ms)); };
const allowed = new Set([FIXTURE, PREP, ATTEMPT, ...Object.values(OUTPUT)].map(norm));
const E = {
  schema: 'tool-read-pass262-reference/v1', round: 118, runId: RUN_ID, startedAt: new Date().toISOString(), status: 'running',
  attemptPolicy: 'Exactly one exclusive round-wide Chrome acceptance after syntax/source/fixture preflight. Every case once; retain all failures without an accepting rerun.',
  scope: {
    script: SCRIPT, fixture: FIXTURE, profile: PROFILE,
    description: 'Manual installed ReadBlock/CodeCard/CodeToolbar/FoldToggle and ToolRow containment, verbatim current installed CSS and baseline light tokens, synthetic read windows. Explicit local fold/wrap/copy handlers and an in-memory clipboard adapter. Exact installed pure read-card functions execute source-only in VM, without importing the application.',
    input: 'CDP events only inside the target of the spawned exact PID/profile/port-verified headless Chrome; no hardware input, monitor or original Electron assertion. Tab focus only; no native-button Enter/Space activation assertion.',
    network: 'No server. File CSP denies network; CDP blocks HTTP/HTTPS/WS/WSS. Only own loopback debugging endpoint contacted.',
    protected: {electronPID: 20012, backendPID: 16008, lockAppPID: 12516, guiPort: 3080, touched: false},
  },
  compactPolicy: 'Unique installed CSS/maps/extracted functions/source hashes only in sidecar; computed styles interned once. One final geometry snapshot per fixture and scalar transitions. Combined source and metrics JSON strictly below250000 bytes.',
  sourceEvidence: OUTPUT.source, cases: [], measurements: {}, screenshots: {}, errors: [], writes: [],
  limitations: [
    'Installed JS/CSS and ui-tool artifacts are authority. Read-only older TypeScript locates symbols and icon artwork, verified against the actual installed bundle.',
    'Manual DOM/local handlers do not run React, useMemo/useSyncExternalStore/useState, viewport observation, lazy grammar imports, Shiki/highlightLines, React reconciliation or remount behavior.',
    'Fold/wrap/copy behavior is explicit local fixture code derived from installed source. Actual React callbacks are recorded, not executed. No motion/useDisclosure/usePresence, tooltip portal/position/timer or host behavior is executed.',
    'Successful and refused clipboard writes go only to an in-memory fixture adapter; no OS/application clipboard call, read or write. The installed copied flag guards clicks after success, not pending writes; this fixture does not certify a pending-write race.',
    'Source-only VM metadata tables execute pure installed parser/path/read functions with synthetic data; no real file, model dispatch, backend, session or Host action executes.',
    'Baseline installed light tokens and system-resolved font stacks only. DPR1 comes from startup flags and is measured; no Emulation or physical display claim.',
  ],
};
let source, browser, page, child, exitInfo, port, verified = false, spawnCount = 0;
const requests = [], exceptions = [], protocolErrors = [], methodCounts = {};
async function noLinks(target) {
  const absolute = path.resolve(target); let current = path.parse(absolute).root;
  for (const part of absolute.slice(current.length).split(path.sep)) {
    if (!part) continue; current = path.join(current, part);
    try { invariant(!(await fs.lstat(current)).isSymbolicLink(), 'Refuse symlink ' + current); }
    catch (error) { if (error.code !== 'ENOENT') throw error; }
  }
}
async function absent(target) { try { await fs.lstat(target); throw Error('Refuse existing ' + target); } catch (error) { if (error.code !== 'ENOENT') throw error; } }
async function writeOwned(target, data) {
  invariant(allowed.has(norm(target)), 'Out of scope write ' + target); await noLinks(target); await absent(target);
  await fs.writeFile(target, data, {flag: 'wx'}); E.writes.push(target);
}
async function scope() {
  invariant(process.platform === 'win32' && Number(process.versions.node.split('.')[0]) >= 24, 'Windows Node24 required');
  invariant((MODE === '--prepare-only' && process.argv.length === 3) || (MODE === '--accept-prepared' && process.argv.length === 4 && suppliedRun && norm(suppliedPrep) === norm(PREP)), 'Only prepare/own-prepared modes');
  invariant(norm(fileURLToPath(import.meta.url)) === norm(SCRIPT) && norm(process.cwd()) === norm(ROOT), 'Wrong runner/workspace');
  for (const target of [...Object.values(SOURCE_PATHS), CHROME, PROFILE, ...allowed]) await noLinks(target);
  for (const target of [ROOT, path.dirname(BASE), path.dirname(PREFIX)]) invariant(norm(await fs.realpath(target)) === norm(target), 'Redirected root ' + target);
  for (const target of [PROFILE, ATTEMPT, ...Object.values(OUTPUT).filter(p => MODE === '--prepare-only' || p !== OUTPUT.source), ...(MODE === '--prepare-only' ? [PREP, FIXTURE] : [])]) await absent(target);
}
function fragment(text, from, until, id) {
  const start = text.indexOf(from), end = text.indexOf(until, start);
  invariant(start >= 0 && end > start, 'Missing installed fragment ' + from);
  return {source: id, startUtf16Offset: start, endUtf16Offset: end, text: text.slice(start, end)};
}
function pureFunction(text, name) {
  const found = text.match(new RegExp('(?:export )?function ' + name + '\\([^]*?\\n}'));
  invariant(found, 'Missing installed pure function ' + name); return found[0].replace(/^export /, '');
}
function literal(text, name) {
  const prefix = 'var ' + name + '_css_default = ', line = text.split('\n').find(v => v.trimStart().startsWith(prefix));
  invariant(line, 'Missing installed theme literal ' + name);
  return {source: 'theme', line: text.slice(0, text.indexOf(line)).split('\n').length, text: JSON.parse(line.trim().slice(prefix.length).replace(/;$/, ''))};
}
function cssMap(css, suffix, names) {
  return Object.fromEntries(names.map(name => { const found = css.match(new RegExp('\\._' + name + '_' + suffix + '_\\d+')); invariant(found, 'Missing CSS class ' + name); return [name, found[0].slice(1)]; }));
}
function localeLabel(text, key) {
  const found = text.match(new RegExp("'" + key.replaceAll('.', '\\.') + "': '([^']*)'")); invariant(found, 'Missing installed locale ' + key); return found[1];
}
function readModelTable(pure, bundled) {
  const baseArgs = {file_path: 'C:\\fixture\\args.ts', offset: 37, limit: 8};
  const baseMeta = {path: 'C:\\fixture\\src\\reported.ts', offset: 37, lines: [{number: 37, text: 'metadata text'}, {number: 40, text: ''}], totalLines: 50, lang: 'typescript'};
  const envelope = '<path>C:\\fixture\\envelope.ts</path>\n<type>file</type>\n<content>\nBODY DIFFERENT FROM METADATA\n</content>';
  const base = {kind: 'tool_result', call: {name: 'read', argsRaw: JSON.stringify(baseArgs)}, content: [{type: 'text', text: envelope}], isError: false, meta: baseMeta};
  const samples = [
    ['valid-window-noncontiguous', {}, true], ['running-no-kind', {running: true}, false], ['child', {parentCallId: 'parent'}, false], ['parent-null-still-child', {parentCallId: null}, false], ['error', {isError: true}, false],
    ['other-tool', {name: 'grep'}, false], ['args-invalid-json', {argsRaw: '{'}, false], ['args-array', {argsRaw: '[]'}, false], ['args-null', {argsRaw: 'null'}, false],
    ['empty-arg-path', {args: {file_path: ''}}, false], ['whitespace-arg-path', {args: {file_path: '  '}}, false], ['nonstring-arg-path', {args: {file_path: 7}}, false],
    ['zero-arg-offset', {args: {offset: 0}}, false], ['fraction-arg-offset', {args: {offset: 1.5}}, false], ['string-arg-offset', {args: {offset: '37'}}, false],
    ['negative-arg-limit', {args: {limit: -1}}, false], ['string-arg-limit', {args: {limit: '8'}}, false], ['null-arg-limit', {args: {limit: null}}, false],
    ['relative-arg-path', {args: {file_path: 'src/relative.ts'}}, true], ['args-meta-offset-mismatch', {args: {offset: 1}}, true],
    ['meta-null', {metaReplace: null}, false], ['meta-array', {metaReplace: []}, false], ['meta-path-nonstring', {meta: {path: 3}}, false],
    ['meta-path-empty-accepted', {meta: {path: ''}}, true, ''], ['meta-offset-zero', {meta: {offset: 0}}, false], ['meta-offset-fraction', {meta: {offset: 1.5}}, false], ['meta-offset-string', {meta: {offset: '37'}}, false],
    ['meta-total-negative', {meta: {totalLines: -1}}, false], ['meta-total-fraction', {meta: {totalLines: 50.5}}, false], ['meta-total-string', {meta: {totalLines: '50'}}, false],
    ['meta-lines-not-array', {meta: {lines: {}}}, false], ['meta-line-null', {meta: {lines: [null]}}, false], ['meta-line-array', {meta: {lines: [[]]}}, false],
    ['meta-line-zero', {meta: {lines: [{number: 0, text: ''}]}}, false], ['meta-line-before-offset', {meta: {lines: [{number: 36, text: ''}]}}, false], ['meta-line-above-total', {meta: {lines: [{number: 51, text: ''}]}}, false],
    ['meta-line-duplicate', {meta: {lines: [{number: 37, text: 'a'}, {number: 37, text: 'b'}]}}, false], ['meta-line-descending', {meta: {lines: [{number: 40, text: 'a'}, {number: 39, text: 'b'}]}}, false],
    ['meta-line-fraction', {meta: {lines: [{number: 37.5, text: ''}]}}, false], ['meta-line-string', {meta: {lines: [{number: '37', text: ''}]}}, false], ['meta-text-nonstring', {meta: {lines: [{number: 37, text: 7}]}}, false],
    ['meta-lang-null', {meta: {lang: null}}, false], ['meta-lang-unknown-accepted', {meta: {lang: 'synthetic-unknown'}}, true],
    ['empty-file-valid', {meta: {offset: 1, lines: [], totalLines: 0}}, true], ['empty-window-offset-past-total-accepted', {meta: {offset: 99, lines: [], totalLines: 0}}, true],
    ['zero-content-blocks', {content: []}, false], ['two-content-blocks', {content: [{type: 'text', text: envelope}, {type: 'text', text: 'extra'}]}, false], ['nontext-content', {content: [{type: 'image'}]}, false],
    ['malformed-envelope', {text: 'plain raw text'}, false], ['wrong-envelope-type', {text: envelope.replace('<type>file', '<type>directory')}, false], ['envelope-CRLF', {text: envelope.replaceAll('\n', '\r\n')}, false], ['envelope-trailing-LF', {text: envelope + '\n'}, false],
    ['home-abbreviation', {meta: {path: '/home/fixture/outside.ts'}, cwd: '/workspace', home: '/home/fixture'}, true, '~/outside.ts'],
  ];
  const table = samples.map(([id, patch, accepted, expectedLabel = 'src\\reported.ts']) => {
    const args = {...baseArgs, ...patch.args};
    const call = {name: patch.name ?? 'read', argsRaw: patch.argsRaw ?? JSON.stringify(args)};
    let block = {...base, call, meta: Object.hasOwn(patch, 'metaReplace') ? patch.metaReplace : {...baseMeta, ...patch.meta}, content: patch.content ?? [{type: 'text', text: patch.text ?? envelope}], isError: patch.isError ?? false};
    if (Object.hasOwn(patch, 'parentCallId')) block.parentCallId = patch.parentCallId;
    if (patch.running) block = {phase: 'start', ...call};
    const actual = pure.readCardModel(block, patch.cwd ?? 'C:\\fixture', patch.home), other = bundled.readCardModel(block, patch.cwd ?? 'C:\\fixture', patch.home);
    const line = pure.readCallLine(block), otherLine = bundled.readCallLine(block);
    const pass = (actual !== null) === accepted && (!accepted || actual.label === expectedLabel) && JSON.stringify(actual) === JSON.stringify(other) && line === otherLine;
    return {id, patch, expectedAccepted: accepted, result: actual === null ? null : {label: actual.label, numbers: actual.lines.map(v => v.number), totalLines: actual.totalLines, lang: actual.lang ?? null}, callLine: line ?? null, emittedMatchesBundled: JSON.stringify(actual) === JSON.stringify(other) && line === otherLine, pass};
  });
  invariant(table.every(v => v.pass), 'Exact installed read model table differs: ' + JSON.stringify(table.filter(v => !v.pass)));
  return {baseline: {args: baseArgs, meta: baseMeta, envelope, cwd: 'C:\\fixture'}, table};
}
async function extract() {
  const texts = Object.fromEntries(await Promise.all(Object.entries(SOURCE_PATHS).map(async ([id, file]) => [id, await fs.readFile(file, 'utf8')])));
  const {webJS, webCSS, tool, theme, row} = texts;
  const rowLiteral = tool.match(/^\t\tconst css\$3 = ("(?:[^"\\]|\\.)*");\r?$/m); invariant(rowLiteral, 'Missing installed ToolRow CSS');
  const rowMap = fragment(tool, 'var ToolRow_module_css_default = ', '\n\t\t//#endregion', 'tool');
  const allRules = [...webCSS.matchAll(/([^{}]+)\{([^{}]*)\}/g)];
  const classCSS = suffix => allRules.filter(v => v[1].includes('_' + suffix + '_')).map(v => v[0]).join('');
  const readCSS = classCSS('1ay0u'), cardCSS = classCSS('1pq26');
  const disclosureCSS = fragment(webCSS, '._root_1rdzk_1{', '._button_ll2mm_2{', 'webCSS');
  const maps = {row: Object.fromEntries([...rowMap.text.matchAll(/"([^"]+)": "([^"]+)"/g)].map(v => [v[1], v[2]])), read: cssMap(readCSS, '1ay0u', ['block', 'line', 'gutter', 'content', 'expand']), card: cssMap(cardCSS, '1pq26', ['card', 'header', 'heading', 'language', 'title', 'actions', 'status', 'action', 'body']), disclosure: cssMap(disclosureCSS.text, '1j0v6', ['root', 'header', 'row', 'leading', 'title']), shimmer: cssMap(disclosureCSS.text, '1rdzk', ['root', 'content', 'text'])};
  const css = {read: {source: 'webCSS', text: readCSS}, card: {source: 'webCSS', text: cardCSS}, row: {source: 'tool', line: tool.slice(0, rowLiteral.index).split('\n').length, text: JSON.parse(rowLiteral[1])}, disclosure: disclosureCSS, ...Object.fromEntries(['base', 'motion', 'corner_shape', 'design_platform', 'focus', 'scrollbar', 'gradient_shadow_text'].map(name => ['theme_' + name, literal(theme, name)]))};
  const installedJS = {readBlock: fragment(webJS, 'function JZ({label:', 'const KZ=16;', 'webJS'), toolbar: fragment(webJS, 'function r4({lang:', 'const VZ=', 'webJS'), fold: fragment(webJS, 'function Vw({className:', 'function r4(', 'webJS')};
  invariant(installedJS.readBlock.text.includes('[j,C]=$.useState(!1)') && installedJS.readBlock.text.includes('"data-code-wrap":j') && installedJS.readBlock.text.includes('maxLines:s=16'), 'Installed ReadBlock defaults changed');
  invariant(installedJS.readBlock.text.includes('N=r.length-s,L=N>0&&!v,O=Math.ceil(s/2),U=s-O,V=r.length<i') && installedJS.readBlock.text.includes('r.length>0?I:void 0') && installedJS.readBlock.text.includes('1e3'), 'Installed ReadBlock cap/copy/window semantics changed');
  invariant(installedJS.toolbar.text.includes('side:"top",portal:!0') && installedJS.toolbar.text.includes('"aria-label":i.wrapLabel,"aria-pressed":d'), 'Installed toolbar semantics changed');
  invariant(readCSS.includes('min-height:1lh') && readCSS.includes('margin-right:12px') && readCSS.includes('calc(var(--dsl-read-gutter) + 12px)'), 'Installed read CSS changed');
  invariant([...css.row.text.matchAll(/([^{}]+)\{([^{}]*)\}/g)].some(v => v[1].split(',').includes('.KZEDha_readBody') && v[2] === 'margin:4px 0 4px 4px'), 'Installed readBody margin changed');
  invariant(row.includes('maxLines: CHAT_READ_MAX_LINES, className: css.readBody') && !row.slice(row.indexOf(': readBody !== null'), row.indexOf(': imageBody !== null')).includes('bodyScroll'), 'ReadBlock callsite changed');
  invariant(texts.read.includes('CHAT_READ_MAX_LINES = 8;') && texts.caller.includes('read: readCardModel(block, cwd, home)') && texts.family.includes('...card, state: model.state'), 'Installed read caller changed');
  for (const value of Object.values(maps).flatMap(Object.values)) invariant(webJS.includes('"' + value + '"') || tool.includes('"' + value + '"'), 'Map class absent in installed JS ' + value);
  const en = texts.locale.slice(texts.locale.indexOf('export const en = {'));
  const dictionary = Object.fromEntries(['read.window', 'read.collapseAria', 'read.expandAria', 'read.expandRest', 'tool.title.read'].map(key => [key, localeLabel(en, key)]));
  for (const key of ['copy', 'copied', 'collapse', 'codeBlock.title', 'codeBlock.wrap', 'codeBlock.unwrap']) dictionary[key] = localeLabel(texts.common, key);
  const labelsCode = ['codeToolbarLabels', 'readBlockLabels'].map(name => pureFunction(texts.labels, name)).join('\n');
  const localize = (key, args) => { invariant(Object.hasOwn(dictionary, key), 'Unknown locale ' + key); return Object.entries(args ?? {}).reduce((out, [k, v]) => out.replaceAll('{' + k + '}', String(v)), dictionary[key]); };
  const labels = vm.runInNewContext(labelsCode + '\n({readBlockLabels})', Object.create(null), {timeout: 1000}).readBlockLabels(localize);
  const rawCode = ['parsedToolCall', 'singleResultText'].map(name => pureFunction(texts.raw, name)).join('\n');
  const readCode = ['positiveInteger', 'validReadCall', 'readMeta', 'readCallLine', 'readCardModel'].map(name => pureFunction(texts.read, name)).join('\n');
  const pathCode = ['isWindowsStylePath', 'abbreviateHomePath', 'relativizeToCwd'].map(name => pureFunction(texts.paths, name)).join('\n');
  const bundledRead = fragment(tool, '\t\tfunction positiveInteger$1(', '\n\t\t//#endregion', 'tool');
  const runVM = code => vm.runInNewContext('const parsedCalls=new WeakMap();\n' + rawCode + '\n' + pathCode + '\n' + code + '\n({readCardModel,readCallLine})', Object.create(null), {timeout: 1000});
  const pure = runVM(readCode), bundled = runVM(bundledRead.text), model = readModelTable(pure, bundled);
  const artwork = {};
  for (const [key, name] of [['up', 'IconChevronUpOutlineArtwork'], ['down', 'IconChevronDownOutlineArtwork'], ['wrap', 'IconWrapFillArtwork'], ['nowrap', 'IconNowrapFillArtwork'], ['copy', 'IconCopyOutlineArtwork'], ['check', 'IconCheckOutlineArtwork']]) {
    const start = texts.icons.indexOf(name + ' = '), end = texts.icons.indexOf('\n)', start) + 2; invariant(start >= 0 && end > start, 'Missing reference icon ' + name);
    const shapes = [...texts.icons.slice(start, end).matchAll(/<(path|rect)\s+([^>]+)\/>/g)].map(v => ({tag: v[1], attributes: Object.fromEntries([...v[2].matchAll(/([\w-]+)="([^"]+)"/g)].map(a => [a[1], a[2]]))}));
    invariant(shapes.length && shapes.every(s => s.tag !== 'path' || webJS.includes(s.attributes.d)), 'Artwork path not in installed bundle');
    for (const s of shapes.filter(v => v.tag === 'rect')) invariant(webJS.includes(Object.entries(s.attributes).map(([k, v]) => k + ':' + JSON.stringify(v)).join(',')), 'Artwork rect not in installed bundle');
    artwork[key] = {source: 'icons', name, line: texts.icons.slice(0, start).split('\n').length, shapes, installedVerified: true};
  }
  source = {schema: 'tool-read-pass262-reference-source/v1', runId: RUN_ID, authority: INSTALL,
    files: await Promise.all(Object.entries(SOURCE_PATHS).map(async ([id, file]) => ({id, path: file, bytes: (await fs.stat(file)).size, sha256: hash(await fs.readFile(file))}))),
    css, maps, dictionary, artwork, installedJS,
    sourceOnly: {readModel: {source: 'read', lines: [13, 15, 18, 31, 73, 88], functions: readCode, rawDependencies: {source: 'raw', functions: rawCode}, pathDependencies: {source: 'paths', functions: pathCode}, bundledVerification: {source: 'tool', startUtf16Offset: bundledRead.startUtf16Offset, endUtf16Offset: bundledRead.endUtf16Offset, sha256: hash(bundledRead.text), method: 'Also execute current client.js read model fragment against every emitted-function table row; no application imports.'}, ...model}, labels: {source: 'labels', lines: [7, 42], functions: labelsCode},
      callsite: {source: 'row', lines: [39, 46, 56, 97, 103, 104], maxLines: 8, className: maps.row.readBody, containment: 'ToolRow.bodyWrap flex column -> ReadBlock (no bodyScroll); read wins generic IO; primitive own default16; installed read-row derives readCardModel and readCallLine, family spreads card.'},
      behavior: {rawCopy: 'All returned line.text joined with one LF between lines; no banner/gutter/fold strings and no trim. Empty files omit copy; a single empty-text row still offers Copy.', fold: 'hidden=returned-maxLines; head ceil(max/2), tail=max-head. Expanded shows every line followed by collapse control. Gaps in original numbers survive.', window: 'returned count<totalLines only; count is not numeric last-first range.', gutter: 'Maximum String(line.number).length across the whole window, minimum3, unit ch. Each gutter has margin-right12px; fold content aligned at gutter+12px.', copy: 'Guard copied flag, call Jo(raw).then(ok=>...); success copied/check then1000ms timeout; refusal leaves Copy. Pending write race not certified.', wrap: 'Local false initially; toggles data-code-wrap and aria-pressed. Stable aria-label Wrap lines; tooltip text changes and glyph swaps.', highlight: 'Ww viewport activation; Lw whole-window highlight with grammar-load external store; all recorded, none executed.'}},
    referenceVerification: 'Read-only ReadBlock/CodeToolbar/FoldToggle reference locates current JZ/r4/Vw; installed source invariants and CSS maps govern fixture. Icons verified against installed path/rect strings. No original/dependency/build/native/model/test changes.',
  };
  return {css, maps, labels, dictionary, artwork};
}
const esc = v => String(v).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;');
function makeModels() {
  const lines = (count, start = 1) => Array.from({length: count}, (_, i) => ({number: start + i, text: 'const fixture' + (start + i) + ' = ' + (i + 1) + ';'}));
  const windowLines = lines(14, 37); windowLines[5].text = '    middle hidden text with spaces    '; windowLines[13].text = '';
  return [
    {id: 'short', x: 40, y: 110, w: 620, label: 'src/short.ts', lang: 'typescript', lines: [{number: 1, text: 'const greeting = "fixture";'}, {number: 2, text: 'console.log(greeting);'}], totalLines: 2},
    {id: 'empty', x: 40, y: 305, w: 620, label: 'empty.txt', lines: [], totalLines: 0},
    {id: 'empty-text', x: 40, y: 460, w: 620, label: 'blank.txt', lines: [{number: 7, text: ''}], totalLines: 7},
    {id: 'eight', x: 40, y: 625, w: 620, label: 'src/eight.ts', lang: 'typescript', lines: lines(8), totalLines: 8},
    {id: 'nine', x: 40, y: 995, w: 620, label: 'src/nine.ts', lang: 'typescript', lines: lines(9), totalLines: 9},
    {id: 'window14', x: 720, y: 110, w: 720, label: 'src/window.ts', lang: 'typescript', lines: windowLines, totalLines: 100},
    {id: 'noncontiguous', x: 720, y: 525, w: 720, label: 'src/noncontiguous.ts', lang: 'typescript', lines: [996, 1000, 1010, 1042].map((number, i) => ({number, text: 'actual file line ' + number + ' // gap ' + i})), totalLines: 2000},
    {id: 'wrap', x: 720, y: 800, w: 340, label: 'long.ts', lang: 'typescript', lines: [{number: 1, text: 'const longText = "' + 'long-fixture-'.repeat(24) + '";'}], totalLines: 1},
    {id: 'fallback', x: 1100, y: 800, w: 340, lang: 'synthetic-unknown', lines: [{number: 1, text: 'plain fallback text'}], totalLines: 1},
  ].map(v => ({...v, maxLines: 8, raw: v.lines.map(line => line.text).join('\n')}));
}
function fixture(x) {
  const models = makeModels(), m = x.maps, L = x.labels;
  const svg = key => '<svg data-icon="' + key + '" width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true" stroke-width="1">' + x.artwork[key].shapes.map(s => '<' + s.tag + ' ' + Object.entries(s.attributes).map(([k, v]) => k + '="' + esc(v) + '"').join(' ') + '/>').join('') + '</svg>';
  const rows = values => values.map(v => '<div class="' + m.read.line + '" data-read-line><span class="' + m.read.gutter + '" aria-hidden="true" data-gutter>' + v.number + '</span><span class="' + m.read.content + '" data-line-content>' + esc(v.text) + '</span></div>').join('');
  const read = v => {
    const hidden = v.lines.length - v.maxLines, capped = hidden > 0, head = Math.ceil(v.maxLines / 2), tail = v.maxLines - head;
    const gutter = Math.max(3, ...v.lines.map(line => String(line.number).length));
    return '<div class="' + m.card.card + ' ' + m.read.block + ' ' + m.row.readBody + '" data-read="" data-code-wrap="false" style="--dsl-read-gutter:' + gutter + 'ch"><div class="' + m.card.header + '" data-code-block-banner><div class="' + m.card.heading + '" data-heading><span class="' + m.card.language + '" data-language>' + esc(v.lang === 'typescript' ? v.lang : L.codeLabel) + '</span>' + (v.label === undefined ? '' : '<span class="' + m.card.title + '" title="' + esc(v.label) + '" data-read-title>' + esc(v.label) + '</span>') + '</div><div class="' + m.card.actions + '" data-actions>' + (v.lines.length < v.totalLines ? '<span class="' + m.card.status + '" data-window-status>' + esc(L.window(v.lines.length, v.totalLines)) + '</span>' : '') + '<button type="button" class="' + m.card.action + '" data-wrap aria-label="' + esc(L.wrapLabel) + '" aria-pressed="false" data-fixture-tooltip-label="' + esc(L.wrapLabel) + '">' + svg('wrap') + '</button>' + (v.lines.length ? '<button type="button" class="' + m.card.action + '" data-copy aria-label="' + esc(L.copy) + '" data-fixture-tooltip-label="' + esc(L.copy) + '">' + svg('copy') + '</button>' : '') + '</div></div><div class="' + m.card.body + '" data-read-body>' + rows(capped ? v.lines.slice(0, head) : v.lines) + (hidden > 0 ? '<button type="button" class="' + m.read.expand + '" data-fold aria-expanded="false" aria-label="' + esc(L.expandAria(hidden)) + '">' + esc(L.expand(hidden)) + '</button>' : '') + (capped ? rows(v.lines.slice(v.lines.length - tail)) : '') + '</div></div>';
  };
  const scenario = v => '<section class="scenario" style="left:' + v.x + 'px;top:' + v.y + 'px;width:' + v.w + 'px"><p class="caption">' + esc(v.id) + ' · ' + v.lines.length + '/' + v.totalLines + ' returned lines</p><div class="' + m.row.root + '" data-own-card="' + v.id + '" data-tool="read" data-variant="read" data-state="ok"><div class="' + m.disclosure.root + '" data-disclosure-root data-open="true"><div class="' + m.disclosure.header + '"><div class="' + m.disclosure.row + ' ' + m.row.row + '" data-disclosure-row data-expandable role="button" tabindex="0" aria-expanded="true"><span class="' + m.disclosure.leading + ' ' + m.row.leading + '">' + svg('up') + '</span><span class="' + m.shimmer.root + '"><span class="' + m.shimmer.content + '"><span class="' + m.shimmer.text + ' ' + m.disclosure.title + ' ' + m.row.title + '">' + esc(x.dictionary['tool.title.read']) + '</span><span class="' + m.row.sep + '" aria-hidden="true"></span><span class="' + m.row.summary + '"><span class="' + m.shimmer.text + '">Fixture ' + esc(v.id) + '</span></span></span></span></div></div><div data-fixture-motion data-dsh-motion="fade" data-state="open"><div class="' + m.row.bodyWrap + '" data-body-wrap>' + read(v) + '</div></div></div></div></section>';
  const order = ['base', 'motion', 'corner_shape', 'design_platform', 'focus', 'scrollbar', 'gradient_shadow_text'];
  const sheets = [...order.map(name => x.css['theme_' + name].text), x.css.disclosure.text, x.css.card.text, x.css.read.text, x.css.row.text];
  const labelData = Object.fromEntries(Object.entries(L).filter(([, v]) => typeof v === 'string'));
  const data = {models, maps: m, labels: labelData, dictionary: x.dictionary, artwork: x.artwork};
  const html = '<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src \'none\';style-src \'unsafe-inline\';script-src \'unsafe-inline\';connect-src \'none\';img-src data:"><title>Round118 installed ReadBlock</title>' + sheets.map((text, i) => '<style data-installed-css="' + i + '">' + text + '</style>').join('') + '<style>html,body{margin:0;min-height:100%}body{font-family:var(--dsw-font-family);font-size:14px;color:var(--dsw-alias-label-primary);background:var(--dsw-alias-bg-base)}#intro{position:absolute;left:40px;top:20px;font-size:12px;color:var(--dsw-alias-label-secondary)}h1{font-size:18px;font-weight:500;margin:0 0 5px}#intro p{margin:0}.scenario{position:absolute}.caption{font-size:12px;line-height:18px;color:var(--dsw-alias-label-secondary);margin:0 0 8px}</style></head><body data-platform="windows"><header id="intro"><h1>Installed ReadBlock · isolated Chrome DPR1</h1><p>Verbatim installed CSS and manual containment. Synthetic read windows; local handlers and memory clipboard. No React, Shiki, motion, Host or physical input.</p></header>' + models.map(scenario).join('') + '<script>\n' + inlineFixture.toString() + '\ninlineFixture(' + JSON.stringify(data).replaceAll('<', '\\u003c') + ');\n</script></body></html>';
  return {html, models};
}
function inlineFixture(data) {
  'use strict';
  const {models, maps: m, labels: L, dictionary, artwork} = data;
  const events = [], clipboard = [], feedback = [], folds = [], styles = [], styleIDs = new Map(); let clipboardOK = true;
  const own = id => document.querySelector('[data-own-card="' + id + '"]');
  const rect = element => { if (!element) return null; const b = element.getBoundingClientRect(); return {x: b.x, y: b.y, w: b.width, h: b.height}; };
  const keys = ['display', 'position', 'boxSizing', 'fontFamily', 'fontSize', 'lineHeight', 'fontWeight', 'color', 'backgroundColor', 'whiteSpace', 'overflowWrap', 'wordBreak', 'overflowX', 'overflowY', 'minHeight', 'minWidth', 'width', 'flexShrink', 'textAlign', 'userSelect', 'paddingTop', 'paddingRight', 'paddingBottom', 'paddingLeft', 'marginTop', 'marginRight', 'marginBottom', 'marginLeft', 'borderTopWidth', 'borderRadius', 'columnGap', 'outlineStyle', 'outlineWidth', 'outlineColor', 'outlineOffset'];
  const style = element => element ? Object.fromEntries(keys.map(key => [key, getComputedStyle(element)[key]])) : null;
  const intern = element => { if (!element) return null; const value = style(element), serialized = JSON.stringify(value); if (!styleIDs.has(serialized)) { styleIDs.set(serialized, 'S' + (styles.length + 1)); styles.push({id: styleIDs.get(serialized), value}); } return styleIDs.get(serialized); };
  const scroll = element => element ? {x: element.scrollLeft, y: element.scrollTop, sw: element.scrollWidth, sh: element.scrollHeight, cw: element.clientWidth, ch: element.clientHeight} : null;
  const box = (element, withScroll = false) => element ? {box: rect(element), style: intern(element), ...(withScroll ? {scroll: scroll(element)} : {})} : null;
  const textLabel = (key, values) => Object.entries(values ?? {}).reduce((out, [name, value]) => out.replaceAll('{' + name + '}', String(value)), dictionary[key]);
  const svg = key => {
    const element = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    for (const [name, value] of Object.entries({'data-icon': key, width: '14', height: '14', viewBox: '0 0 16 16', fill: 'none', 'aria-hidden': 'true', 'stroke-width': '1'})) element.setAttribute(name, value);
    for (const s of artwork[key].shapes) { const shape = document.createElementNS(element.namespaceURI, s.tag); for (const [name, value] of Object.entries(s.attributes)) shape.setAttribute(name, value); element.append(shape); } return element;
  };
  const rowElement = line => { const element = document.createElement('div'); element.className = m.read.line; element.dataset.readLine = ''; const gutter = document.createElement('span'); gutter.className = m.read.gutter; gutter.dataset.gutter = ''; gutter.setAttribute('aria-hidden', 'true'); gutter.textContent = String(line.number); const content = document.createElement('span'); content.className = m.read.content; content.dataset.lineContent = ''; content.textContent = line.text; element.append(gutter, content); return element; };
  for (const type of ['mousemove', 'pointerdown', 'pointerup', 'mousedown', 'mouseup', 'click', 'keydown', 'keyup', 'wheel']) document.addEventListener(type, event => events.push({type, trusted: event.isTrusted, id: event.target.closest('[data-own-card]')?.dataset.ownCard ?? null, key: event.key ?? null, target: event.target.closest('[data-wrap]') ? 'wrap' : event.target.closest('[data-copy]') ? 'copy' : event.target.closest('[data-fold]') ? 'fold' : event.target.closest('[data-read-body]') ? 'body' : event.target.closest('[data-disclosure-row]') ? 'row' : null}), true);
  for (const value of models) {
    const root = own(value.id), block = root.querySelector('[data-read]'), body = root.querySelector('[data-read-body]'), wrap = root.querySelector('[data-wrap]'), copy = root.querySelector('[data-copy]'), fold = root.querySelector('[data-fold]');
    // LOCAL adapter of installed state transitions. React and native clipboard are absent.
    wrap.addEventListener('click', () => { const wrapped = block.dataset.codeWrap !== 'true'; block.dataset.codeWrap = String(wrapped); wrap.setAttribute('aria-pressed', String(wrapped)); wrap.dataset.fixtureTooltipLabel = L[wrapped ? 'unwrapLabel' : 'wrapLabel']; wrap.replaceChildren(svg(wrapped ? 'nowrap' : 'wrap')); });
    if (fold) {
      let expanded = false;
      fold.addEventListener('click', () => {
        expanded = !expanded; const hidden = value.lines.length - value.maxLines, head = Math.ceil(value.maxLines / 2), tail = value.maxLines - head;
        fold.setAttribute('aria-expanded', String(expanded)); fold.setAttribute('aria-label', expanded ? L.collapseAria : textLabel('read.expandAria', {count: hidden})); fold.textContent = expanded ? L.collapse : textLabel('read.expandRest', {count: hidden});
        body.replaceChildren(...(expanded ? value.lines : value.lines.slice(0, head)).map(rowElement), fold, ...(expanded ? [] : value.lines.slice(value.lines.length - tail).map(rowElement)));
        folds.push({id: value.id, expanded, numbers: [...body.querySelectorAll('[data-gutter]')].map(e => Number(e.textContent)), at: performance.now()});
      });
    }
    if (copy) {
      let copied = false;
      copy.addEventListener('click', () => {
        if (copied) return; const ok = clipboardOK, raw = value.lines.map(line => line.text).join('\n');
        clipboard.push({id: value.id, text: raw, ok, at: performance.now(), adapter: 'in-memory fixture only'});
        void Promise.resolve(ok).then(success => {
          if (!success) return; copied = true; copy.setAttribute('aria-label', L.copied); copy.dataset.fixtureTooltipLabel = L.copied; copy.replaceChildren(svg('check')); feedback.push({id: value.id, event: 'copied', at: performance.now()});
          setTimeout(() => { copied = false; copy.setAttribute('aria-label', L.copy); copy.dataset.fixtureTooltipLabel = L.copy; copy.replaceChildren(svg('copy')); feedback.push({id: value.id, event: 'reset', at: performance.now()}); }, 1000);
        });
      });
    }
  }
  function quick(id) {
    const root = own(id), block = root.querySelector('[data-read]'), body = root.querySelector('[data-read-body]'), wrap = root.querySelector('[data-wrap]'), copy = root.querySelector('[data-copy]'), fold = root.querySelector('[data-fold]');
    return {wrapped: block.dataset.codeWrap, gutter: block.style.getPropertyValue('--dsl-read-gutter'), rowExpanded: root.querySelector('[data-disclosure-row]').getAttribute('aria-expanded'), numbers: [...body.querySelectorAll('[data-gutter]')].map(e => Number(e.textContent)), lineHeights: [...body.querySelectorAll('[data-read-line]')].map(e => rect(e).h), bodyHeight: rect(body).h, bodyScroll: scroll(body), wrapLabel: wrap.getAttribute('aria-label'), wrapPressed: wrap.getAttribute('aria-pressed'), wrapTooltip: wrap.dataset.fixtureTooltipLabel, wrapIcon: wrap.querySelector('svg')?.dataset.icon, copyLabel: copy?.getAttribute('aria-label') ?? null, copyIcon: copy?.querySelector('svg')?.dataset.icon ?? null, fold: fold ? {expanded: fold.getAttribute('aria-expanded'), label: fold.getAttribute('aria-label'), text: fold.textContent, height: rect(fold).h, position: [...body.children].indexOf(fold)} : null, status: root.querySelector('[data-window-status]')?.textContent ?? null, focused: document.activeElement === wrap ? 'wrap' : document.activeElement === copy ? 'copy' : document.activeElement === fold ? 'fold' : document.activeElement?.tagName, focusVisible: document.activeElement?.matches(':focus-visible') ?? false};
  }
  function one(id) {
    const root = own(id), block = root.querySelector('[data-read]'), header = root.querySelector('[data-code-block-banner]'), body = root.querySelector('[data-read-body]'), fold = root.querySelector('[data-fold]');
    return {id, root: box(root), row: box(root.querySelector('[data-disclosure-row]')), bodyWrap: box(root.querySelector('[data-body-wrap]')), block: {...box(block), wrapped: block.dataset.codeWrap, gutter: block.style.getPropertyValue('--dsl-read-gutter')}, header: box(header), heading: box(root.querySelector('[data-heading]')), language: {...box(root.querySelector('[data-language]')), text: root.querySelector('[data-language]').textContent}, title: root.querySelector('[data-read-title]') ? {...box(root.querySelector('[data-read-title]')), text: root.querySelector('[data-read-title]').textContent, title: root.querySelector('[data-read-title]').title} : null, actions: box(root.querySelector('[data-actions]')), status: root.querySelector('[data-window-status]') ? {...box(root.querySelector('[data-window-status]')), text: root.querySelector('[data-window-status]').textContent} : null, buttons: [...header.querySelectorAll('button')].map(e => ({kind: e.hasAttribute('data-wrap') ? 'wrap' : 'copy', ...box(e), tag: e.tagName, type: e.type, label: e.getAttribute('aria-label'), pressed: e.getAttribute('aria-pressed'), tooltipMetadata: e.dataset.fixtureTooltipLabel, text: e.textContent, icon: {key: e.querySelector('svg')?.dataset.icon, box: rect(e.querySelector('svg'))}})), body: box(body, true), rows: [...body.querySelectorAll('[data-read-line]')].map(e => ({number: Number(e.querySelector('[data-gutter]').textContent), ...box(e), gutter: {...box(e.querySelector('[data-gutter]')), ariaHidden: e.querySelector('[data-gutter]').getAttribute('aria-hidden')}, content: {...box(e.querySelector('[data-line-content]')), chars: e.querySelector('[data-line-content]').textContent.length}})), fold: fold ? {...box(fold), text: fold.textContent, label: fold.getAttribute('aria-label'), expanded: fold.getAttribute('aria-expanded'), position: [...body.children].indexOf(fold)} : null, hasBodyScrollAncestor: !!block.closest('.' + m.row.bodyScroll)};
  }
  function metrics(id) { const root = own(id); return Object.fromEntries(['[data-read]', '[data-disclosure-row]', '[data-body-wrap]', '[data-code-block-banner]', '[data-heading]', '[data-language]', '[data-read-title]', '[data-actions]', '[data-window-status]', '[data-wrap]', '[data-copy]', '[data-read-body]', '[data-read-line]', '[data-gutter]', '[data-line-content]', '[data-fold]'].map(selector => [selector, style(root.querySelector(selector))])); }
  const tokens = () => Object.fromEntries(['--dsw-font-markdown-code-block', '--dsw-font-family', '--ds-font-family-code', '--dsh-code-font-delta', '--dsw-radius-lg', '--dsw-radius-sm', '--dsw-alias-markdown-code-block', '--dsw-alias-label-primary', '--dsw-alias-label-secondary', '--dsw-alias-label-tertiary'].map(name => [name, getComputedStyle(document.body).getPropertyValue(name).trim()]));
  window.harness = {ready: true, quick, one, metrics, tokens, styles: () => styles, events: () => events, clipboard: () => clipboard, feedback: () => feedback, folds: () => folds, setClipboardOK: value => { clipboardOK = value; }, focus: (id, selector) => { own(id).querySelector(selector).focus({preventScroll: true}); }, box: (id, selector) => rect(own(id).querySelector(selector))};
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
async function focus(id, selector) { await evaluate('window.harness.focus(' + JSON.stringify(id) + ',' + JSON.stringify(selector) + ')'); await frames(); }
async function acceptance(models) {
  await move(5, 5); E.tokens = await evaluate('window.harness.tokens()'); E.browser.viewport = await evaluate('({dpr:devicePixelRatio,innerWidth,innerHeight})');
  E.fixtureData = models.map(v => ({id: v.id, maxLines: v.maxLines, count: v.lines.length, totalLines: v.totalLines, numbers: v.lines.map(line => line.number), rawChars: v.raw.length, rawSHA256: hash(v.raw)}));
  const final = {};
  await runCase('exact-source-only-read-model-and-labels', async check => {
    const table = source.sourceOnly.readModel.table;
    check('all metadata cases execute emitted and bundled installed functions identically', table.length >= 50 && table.every(v => v.pass && v.emittedMatchesBundled), {cases: table.length, failures: table.filter(v => !v.pass)});
    check('root settled success and valid envelope required', ['running-no-kind', 'child', 'parent-null-still-child', 'error', 'malformed-envelope', 'envelope-CRLF', 'envelope-trailing-LF'].every(id => table.find(v => v.id === id).result === null));
    check('monotonically increasing gaps accepted and metadata text controls card', table.find(v => v.id === 'valid-window-noncontiguous').result.numbers.join(',') === '37,40' && source.sourceOnly.readModel.baseline.envelope.includes('BODY DIFFERENT FROM METADATA'));
    check('metadata path/offset not required to equal args or envelope path', table.find(v => v.id === 'args-meta-offset-mismatch').result.label === 'src\\reported.ts' && table.find(v => v.id === 'args-meta-offset-mismatch').callLine === 1);
    check('empty path and empty past-total metadata accepted by exact installed validation', table.find(v => v.id === 'meta-path-empty-accepted').result.label === '' && table.find(v => v.id === 'empty-window-offset-past-total-accepted').result.totalLines === 0);
    check('caller max8 vs primitive16 and no code bodyScroll containment', source.sourceOnly.callsite.maxLines === 8 && source.installedJS.readBlock.text.includes('maxLines:s=16') && source.sourceOnly.callsite.containment.includes('no bodyScroll'));
    check('exact installed English status/fold/copy labels', source.dictionary['read.window'] === 'Showing {shown} of {total} lines' && source.dictionary['read.expandRest'] === '… {count} more lines' && source.dictionary['read.expandAria'] === 'Expand {count} more lines' && source.dictionary['read.collapseAria'] === 'Collapse content' && source.dictionary.collapse === 'Collapse');
  });
  await runCase('short-read-card-toolbar-body-exact-metrics', async check => {
    const s = await one('short'), k = await metrics('short'); final.short = s;
    check('one owned startup DPR1 and620px root24px row', spawnCount === 1 && E.browser.viewport.dpr === 1 && near(s.root.box.w, 620) && near(s.row.box.h, 24), E.browser.viewport);
    check('readBody margin4 0 4 4 width616 begins4 below and right of row', k['[data-read]'].marginTop === '4px' && k['[data-read]'].marginRight === '0px' && k['[data-read]'].marginBottom === '4px' && k['[data-read]'].marginLeft === '4px' && near(s.block.box.w, 616) && near(s.block.box.x - s.row.box.x, 4) && near(s.block.box.y - s.row.box.y - s.row.box.h, 4));
    check('read direct flex bodyWrap child no260px bodyScroll owner', k['[data-body-wrap]'].display === 'flex' && !s.hasBodyScrollAncestor && s.block.box.h < 260);
    check('card baseline11/19 radius16 border0', k['[data-read]'].fontSize === '11px' && k['[data-read]'].lineHeight === '19px' && k['[data-read]'].borderRadius === '16px' && k['[data-read]'].borderTopWidth === '0px');
    check('header height42 padding10 18 8 22 gap12 font11/18', near(s.header.box.h, 42) && k['[data-code-block-banner]'].paddingTop === '10px' && k['[data-code-block-banner]'].paddingRight === '18px' && k['[data-code-block-banner]'].paddingBottom === '8px' && k['[data-code-block-banner]'].paddingLeft === '22px' && k['[data-code-block-banner]'].columnGap === '12px' && k['[data-code-block-banner]'].fontSize === '11px' && k['[data-code-block-banner]'].lineHeight === '18px');
    check('language and title11/18 title attr exact status absent', s.language.text === 'typescript' && s.title.text === 'src/short.ts' && s.title.title === 'src/short.ts' && s.status === null && k['[data-language]'].fontSize === '11px' && k['[data-language]'].lineHeight === '18px' && k['[data-read-title]'].fontSize === '11px' && k['[data-read-title]'].lineHeight === '18px');
    check('actions gap4 two native24x24 icon14 controls no visible labels', k['[data-actions]'].columnGap === '4px' && s.buttons.length === 2 && s.buttons.every(v => v.tag === 'BUTTON' && v.type === 'button' && near(v.box.w, 24) && near(v.box.h, 24) && near(v.icon.box.w, 14) && near(v.icon.box.h, 14) && v.text === ''));
    check('body padding6 22 20 font11/19 overflow-x-auto and64px', k['[data-read-body]'].paddingTop === '6px' && k['[data-read-body]'].paddingRight === '22px' && k['[data-read-body]'].paddingBottom === '20px' && k['[data-read-body]'].paddingLeft === '22px' && k['[data-read-body]'].fontSize === '11px' && k['[data-read-body]'].lineHeight === '19px' && k['[data-read-body]'].overflowX === 'auto' && near(s.body.box.h, 64));
    check('two rows min-height1lh resolve19 white-space-pre line content primary', s.rows.length === 2 && s.rows.every(v => near(v.box.h, 19)) && k['[data-read-line]'].display === 'flex' && k['[data-read-line]'].minHeight === '19px' && k['[data-read-line]'].whiteSpace === 'pre' && k['[data-line-content]'].minWidth === '0px' && k['[data-line-content]'].color === k['[data-read]'].color);
    check('gutter3ch right aligned nonselectable aria-hidden plus12px separation', s.block.gutter === '3ch' && k['[data-gutter]'].marginRight === '12px' && k['[data-gutter]'].textAlign === 'right' && k['[data-gutter]'].userSelect === 'none' && s.rows.every(v => v.gutter.ariaHidden === 'true' && near(v.content.box.x - v.gutter.box.x - v.gutter.box.w, 12)));
    check('short card106 and root138 default data-code-wrap false', near(s.block.box.h, 106) && near(s.root.box.h, 138) && s.block.wrapped === 'false' && s.buttons[0].pressed === 'false');
    await capture('initial');
  });
  await runCase('empty-files-empty-text-minheight-and-language-fallback', async check => {
    const empty = await one('empty'), text = await one('empty-text'), fallback = await one('fallback'); final.empty = empty; final['empty-text'] = text; final.fallback = fallback;
    const k = await metrics('empty-text');
    check('empty file omits copy but keeps wrap toolbar and body padding26', empty.rows.length === 0 && empty.buttons.length === 1 && empty.buttons[0].kind === 'wrap' && near(empty.body.box.h, 26) && near(empty.block.box.h, 68));
    check('empty file no status and root100 even without content', empty.status === null && near(empty.root.box.h, 100));
    check('one empty text is a19px row with45px body and copy available', text.rows.length === 1 && text.rows[0].content.chars === 0 && near(text.rows[0].box.h, 19) && near(text.body.box.h, 45) && text.buttons.some(v => v.kind === 'copy') && k['[data-read-line]'].minHeight === '19px');
    check('one returned blank versus7 total is window status11/18', text.status.text === 'Showing 1 of 7 lines' && (await metrics('empty-text'))['[data-window-status]'].fontSize === '11px' && (await metrics('empty-text'))['[data-window-status]'].lineHeight === '18px');
    check('missing or unknown grammar uses Code block; omitted label has no title', empty.language.text === 'Code block' && fallback.language.text === 'Code block' && fallback.title === null);
  });
  await runCase('threshold8-and9-head4-fold-tail4', async check => {
    const eight = await one('eight'), nine = await one('nine'); final.eight = eight; final.nine = nine;
    const q = await quick('nine'), k = await metrics('nine');
    check('exact8 rows do not fold body178 card220', eight.rows.length === 8 && eight.fold === null && near(eight.body.box.h, 178) && near(eight.block.box.h, 220));
    check('9 shows original1..4 fold then6..9 hidden1', q.numbers.join(',') === '1,2,3,4,6,7,8,9' && q.fold.position === 4 && q.fold.text === '… 1 more lines' && q.fold.label === 'Expand 1 more lines' && q.fold.expanded === 'false');
    check('fold font11/19 height19 body197 card239', k['[data-fold]'].fontSize === '11px' && k['[data-fold]'].lineHeight === '19px' && near(nine.fold.box.h, 19) && near(nine.body.box.h, 197) && near(nine.block.box.h, 239));
    check('fold padding aligns with content at3ch+12 width100%', near(Number.parseFloat(k['[data-fold]'].paddingLeft), nine.rows[0].gutter.box.w + 12) && near(nine.fold.box.x, nine.rows[0].box.x) && near(nine.fold.box.w, nine.rows[0].box.w));
    check('fold is native button, no border and no window status for9of9', k['[data-fold]'].borderTopWidth === '0px' && nine.status === null && q.rowExpanded === 'true');
  });
  await runCase('window14-full-raw-copy-one-second-guard-refusal', async check => {
    const value = models.find(v => v.id === 'window14'), before = await quick('window14'), beforeAX = await accessibility('[data-own-card="window14"] [data-copy]');
    await click(await targetBox('window14', '[data-copy]')); const copied = await quick('window14'), afterAX = await accessibility('[data-own-card="window14"] [data-copy]');
    await click(await targetBox('window14', '[data-copy]')); const guarded = (await evaluate('window.harness.clipboard()')).length; await capture('copied');
    await waitFor(async () => (await quick('window14')).copyLabel === 'Copy', 'copy feedback reset', 2500);
    const feedback = await evaluate('window.harness.feedback()'), start = feedback.find(v => v.id === 'window14' && v.event === 'copied'), reset = feedback.find(v => v.id === 'window14' && v.event === 'reset');
    await evaluate('window.harness.setClipboardOK(false)'); await click(await targetBox('window14', '[data-copy]')); const refused = await quick('window14'); await evaluate('window.harness.setClipboardOK(true)');
    const records = await evaluate('window.harness.clipboard()');
    E.measurements.copy = {copiedLabel: copied.copyLabel, copiedIcon: copied.copyIcon, guardedWriteCount: guarded, feedbackElapsedMS: reset.at - start.at, refusedLabel: refused.copyLabel, beforeAX, afterAX, records: records.map(v => ({id: v.id, ok: v.ok, chars: v.text.length, sha256: hash(v.text), adapter: v.adapter}))};
    check('14 window initially37..40 fold6 then47..50', before.numbers.join(',') === '37,38,39,40,47,48,49,50' && before.fold.position === 4 && before.fold.text === '… 6 more lines' && before.fold.label === 'Expand 6 more lines');
    check('status counts full14of100 even though8 rows visible', before.status === 'Showing 14 of 100 lines');
    check('copy full14 raw text including hidden middle whitespace and final blank terminator', records[0].text === value.raw && records[0].text.includes('    middle hidden text with spaces    ') && records[0].text.endsWith('\n') && records[0].text !== value.lines.filter(line => before.numbers.includes(line.number)).map(line => line.text).join('\n'));
    check('copy has no banner gutter fold strings and AX changes Copy to Copied', !records[0].text.includes('Showing ') && !records[0].text.includes('… ') && beforeAX.some(v => v.role === 'button' && v.name === 'Copy') && afterAX.some(v => v.role === 'button' && v.name === 'Copied') && copied.copyIcon === 'check');
    check('copied flag ignores subsequent successful feedback click', guarded === 1);
    check('feedback elapsed1000ms resets original copy glyph/label', reset.at - start.at >= 995 && reset.at - start.at < 1600 && refused.copyLabel === 'Copy' && refused.copyIcon === 'copy', {elapsed: reset.at - start.at});
    check('refused memory adapter adds write record with no copied feedback', records.length === 2 && !records[1].ok && feedback.filter(v => v.event === 'copied').length === 1 && records.every(v => v.adapter === 'in-memory fixture only'));
    check('toolbar copy leaves row and fold state unchanged', copied.rowExpanded === 'true' && copied.fold.expanded === 'false' && refused.fold.expanded === 'false');
  });
  await runCase('fold-expand-collapse-preserves-file-numbering', async check => {
    const before = await quick('window14'); await click(await targetBox('window14', '[data-fold]')); const expanded = await quick('window14'); await capture('expanded');
    await click(await targetBox('window14', '[data-fold]')); const collapsed = await quick('window14');
    const hover = await targetBox('window14', '[data-fold]'); await move(hover.x + hover.w / 2, hover.y + hover.h / 2); final.window14 = await one('window14');
    const k = await metrics('window14'); E.measurements.fold = {initialCount: before.numbers.length, expandedNumbers: expanded.numbers, expandedHeight: expanded.bodyHeight, expandedControlPosition: expanded.fold.position, expandedLabel: expanded.fold.label, expandedText: expanded.fold.text, collapsedNumbers: collapsed.numbers, collapsedHeight: collapsed.bodyHeight, transitions: await evaluate('window.harness.folds()')};
    check('expand all14 rows keeps37..50 and collapse after rows', expanded.numbers.join(',') === models.find(v => v.id === 'window14').lines.map(v => v.number).join(',') && expanded.fold.position === 14 && expanded.fold.expanded === 'true' && expanded.fold.label === 'Collapse content' && expanded.fold.text === 'Collapse');
    check('expanded body311 vs collapsed197 adds6*19, no parent260 cap', near(expanded.bodyHeight, 311) && near(collapsed.bodyHeight, 197) && near(expanded.bodyHeight - collapsed.bodyHeight, 6 * 19) && !final.window14.hasBodyScrollAncestor);
    check('collapse restores head4 and tail4 original numbers/status', collapsed.numbers.join(',') === before.numbers.join(',') && collapsed.fold.expanded === 'false' && collapsed.status === 'Showing 14 of 100 lines');
    check('hovered fold secondary color and padding matches full-window gutter', k['[data-fold]'].color === k['[data-code-block-banner]'].color && near(Number.parseFloat(k['[data-fold]'].paddingLeft), final.window14.rows[0].gutter.box.w + 12));
    check('fold click does not collapse ToolRow', expanded.rowExpanded === 'true' && collapsed.rowExpanded === 'true');
  });
  await runCase('noncontiguous-four-digit-gutter-and-status', async check => {
    const s = await one('noncontiguous'), k = await metrics('noncontiguous'); final.noncontiguous = s;
    const q = await quick('noncontiguous'), shortGutter = final.short.rows[0].gutter.box.w;
    check('original noncontiguous996100010101042 numbers retained', q.numbers.join(',') === '996,1000,1010,1042');
    check('4ch global gutter includes three-digit row and grows4/3', s.block.gutter === '4ch' && s.rows.every(v => near(v.gutter.box.w, s.rows[0].gutter.box.w)) && near(s.rows[0].gutter.box.w, shortGutter * 4 / 3), {threeCh: shortGutter, fourCh: s.rows[0].gutter.box.w});
    check('gutter12px separation and body22px horizontal padding retained', s.rows.every(v => near(v.content.box.x - v.gutter.box.x - v.gutter.box.w, 12)) && k['[data-read-body]'].paddingLeft === '22px' && k['[data-read-body]'].paddingRight === '22px');
    check('status counts4 not numeric span47, font11/18 margin-right8', s.status.text === 'Showing 4 of 2000 lines' && k['[data-window-status]'].fontSize === '11px' && k['[data-window-status]'].lineHeight === '18px' && k['[data-window-status]'].marginRight === '8px' && k['[data-window-status]'].whiteSpace === 'nowrap');
    check('language code font while title and status use text font', k['[data-language]'].fontFamily === k['[data-read-body]'].fontFamily && k['[data-read-title]'].fontFamily === k['[data-window-status]'].fontFamily && k['[data-window-status]'].fontFamily === k['[data-code-block-banner]'].fontFamily);
  });
  await runCase('default-unwrapped-horizontal-scroll-versus-wrapped-multiline', async check => {
    const before = await quick('wrap'), beforeK = await metrics('wrap'); await wheel(await targetBox('wrap', '[data-read-body]'), 0, 190); const scrolled = await quick('wrap');
    await click(await targetBox('wrap', '[data-wrap]')); const wrapped = await quick('wrap'), k = await metrics('wrap'); await capture('wrapped'); final.wrap = await one('wrap');
    E.measurements.wrap = {unwrapped: {pressed: before.wrapPressed, icon: before.wrapIcon, tooltip: before.wrapTooltip, lineHeight: before.lineHeights[0], bodyHeight: before.bodyHeight, horizontalOverflow: before.bodyScroll.sw > before.bodyScroll.cw, afterWheelX: scrolled.bodyScroll.x}, wrapped: {pressed: wrapped.wrapPressed, icon: wrapped.wrapIcon, tooltip: wrapped.wrapTooltip, lineHeight: wrapped.lineHeights[0], bodyHeight: wrapped.bodyHeight, horizontalOverflow: wrapped.bodyScroll.sw > wrapped.bodyScroll.cw, scrollX: wrapped.bodyScroll.x}};
    check('defaultfalse pressedfalse wrapglyph tooltip Wrap lines', before.wrapped === 'false' && before.wrapPressed === 'false' && before.wrapIcon === 'wrap' && before.wrapTooltip === 'Wrap lines' && beforeK['[data-read-line]'].whiteSpace === 'pre');
    check('unwrapped one19px row overflows body and trusted wheel scrolls horizontally', near(before.lineHeights[0], 19) && before.bodyScroll.sw > before.bodyScroll.cw && scrolled.bodyScroll.x > 0 && beforeK['[data-read-body]'].overflowX === 'auto');
    check('trusted wrap click changes true pressedtrue nowrapglyph tooltip', wrapped.wrapped === 'true' && wrapped.wrapPressed === 'true' && wrapped.wrapIcon === 'nowrap' && wrapped.wrapTooltip === 'Do not wrap lines');
    check('aria-label Wrap lines stays constant in both states', before.wrapLabel === 'Wrap lines' && wrapped.wrapLabel === 'Wrap lines');
    check('wrapped pre-wrap anywhere creates multiple19px lines and no horizontal overflow', k['[data-read-line]'].whiteSpace === 'pre-wrap' && k['[data-line-content]'].overflowWrap === 'anywhere' && wrapped.lineHeights[0] > 19 && near(wrapped.lineHeights[0] % 19, 0) && wrapped.bodyScroll.sw === wrapped.bodyScroll.cw && wrapped.bodyScroll.x === 0, {height: wrapped.lineHeights[0], scroll: wrapped.bodyScroll});
    check('wrapping leaves row and displayed file number intact', wrapped.rowExpanded === 'true' && wrapped.numbers.join(',') === '1');
  });
  await runCase('tab-focus-and-gutter-accessibility', async check => {
    await focus('short', '[data-disclosure-row]'); await key('Tab', 'Tab', 9); const wrapFocus = await quick('short'), k = await metrics('short'); await key('Tab', 'Tab', 9); const copyFocus = await quick('short');
    const foldAX = await accessibility('[data-own-card="nine"] [data-fold]'); E.measurements.focus = {firstTab: wrapFocus.focused, firstFocusVisible: wrapFocus.focusVisible, secondTab: copyFocus.focused, secondFocusVisible: copyFocus.focusVisible, foldAX};
    check('Tab sequence row to wrap then copy with focus-visible', wrapFocus.focused === 'wrap' && copyFocus.focused === 'copy' && wrapFocus.focusVisible && copyFocus.focusVisible);
    check('installed native control ring1px offset2', k['[data-wrap]'].outlineStyle === 'solid' && k['[data-wrap]'].outlineWidth === '1px' && k['[data-wrap]'].outlineOffset === '2px');
    check('fold AX is named button with expandedfalse', foldAX.some(v => v.role === 'button' && v.name === 'Expand 1 more lines' && v.properties.some(p => p.name === 'expanded' && p.value === false)));
    check('every rendered gutter aria-hidden and nonselectable', Object.values(final).flatMap(v => v.rows).every(v => v.gutter.ariaHidden === 'true'));
  });
  await runCase('installed-source-integrity-input-network-and-single-attempt', async check => {
    const changed = []; for (const item of source.files) if (hash(await fs.readFile(item.path)) !== item.sha256) changed.push(item.id);
    E.sourceIntegrity = {allUnchanged: changed.length === 0, changedSourceIDs: changed};
    const events = await evaluate('window.harness.events()'); E.input = {count: events.length, trusted: events.every(v => v.trusted), byType: Object.fromEntries([...new Set(events.map(v => v.type))].map(type => [type, events.filter(v => v.type === type).length])), keydowns: events.filter(v => v.type === 'keydown').map(({id, key, target, trusted}) => ({id, key, target, trusted})), systemClipboardCalls: 0, nativeButtonActivationAssertions: 0};
    E.network = {requests, exceptions, protocolErrors};
    check('all installed/reference/prior/owned runner source bytes unchanged', changed.length === 0, changed);
    check('page file only and no runtime or CDP errors', requests.every(v => !/^https?:|^wss?:/i.test(v.url)) && exceptions.length === 0 && protocolErrors.length === 0, E.network);
    check('observed move click wheel Tab events trusted after ownership gate', E.input.trusted && ['mousemove', 'click', 'wheel', 'keydown'].every(type => E.input.byType[type] > 0) && E.browser.ownership.pass);
    check('single startup exclusive attempt marker no Emulation nativeclipboard or nativeactivation', spawnCount === 1 && E.input.keydowns.every(v => v.key === 'Tab') && Object.keys(methodCounts).every(v => !v.startsWith('Emulation.')) && JSON.parse(await fs.readFile(ATTEMPT, 'utf8')).rerunForbidden === true);
  });
  E.measurements.final = models.map(v => final[v.id]).filter(Boolean);
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
    invariant(norm(PROFILE) === norm(BASE + '-profile') && norm(PROFILE).startsWith(norm(path.join(ROOT, '.devdata/r118-tool-read-'))), 'Wrong profile removal target');
    await noLinks(PROFILE); invariant(norm(await fs.realpath(PROFILE)) === norm(PROFILE), 'Redirected cleanup profile');
    current.profileRemovalTarget = PROFILE; await fs.rm(PROFILE, {recursive: true, force: false}); current.profileRemoved = true;
  }
  current.pass = exitInfo?.code === 0 && current.portReleased && current.allObservedProcessesExited && current.profileRemoved;
}
const mdLink = (name, file) => '[' + name + '](<' + file.replaceAll('\\', '/') + '>)';
function report() {
  const items = E.measurements.final ?? [], get = id => items.find(v => v.id === id), short = get('short'), empty = get('empty'), text = get('empty-text'), window = get('window14'), non = get('noncontiguous');
  return '# Round118 installed ReadBlock reference\n\n' + E.status.toUpperCase() + ': ' + E.summary.passedCases + '/' + E.cases.length + ' cases; ' + E.summary.passedAssertions + '/' + (E.summary.passedAssertions + E.summary.failedAssertions) + ' assertions. ' + mdLink('Compact metrics', OUTPUT.json) + ' · ' + mdLink('Unique installed source/CSS/model table', OUTPUT.source) + ' · ' + mdLink('Runner', SCRIPT) + '.\n\n' + E.scope.description + '\n\n' +
    '| Surface | Verified installed CSS / behavior |\n|---|---|\n' +
    '| Caller | CHAT_READ_MAX_LINES=8; primitive own default16. ToolRow.bodyWrap flex column directly contains ReadBlock; no bodyScroll260 cap. Caller readBody margin4px 0 4px 4px. |\n' +
    '| Card | code font11px/19px; border0; radius-lg16px. Initial data-code-wrap=false. At620px row width, read card616px. |\n' +
    '| Toolbar | 42px high; padding10px 18px 8px 22px; font11px/18px text family; heading/header gap12px; actions gap4px; native controls24×24, glyph14. |\n' +
    '| Language/title/status | Language code family; unknown/absent grammar displays Code block. Title/status text family11px/18px tertiary; title ellipsis, title attribute exact; status nowrap margin-right8px. Status iff returned count<totalLines. |\n' +
    '| Body | padding6px 22px 20px; font11px/19px; overflow-x:auto. No pre element or code block number counter. |\n' +
    '| Read row | flex; min-height:1lh resolves19px, even empty text; white-space:pre. Content min-width0 and primary color. |\n' +
    '| Gutter | max(number digits across full window,3)ch; margin-right12px; right aligned, tertiary, user-select:none and aria-hidden=true. Four-digit window uses4ch for every row. |\n' +
    '| Fold | hidden=returned-maxLines; collapsed head ceil(max/2), tail remainder. At8 cap, head4/fold/tail4. Native button font inherit11/19, height19; width100%; padding-left calc(gutter+12px); border0; tertiary then secondary on hover. |\n' +
    '| Wrap | false initially; trusted click enables pre-wrap plus content overflow-wrap:anywhere, producing multiple19px lines without horizontal overflow. aria-label remains Wrap lines; aria-pressed/tooltip/glyph change. |\n' +
    '| Copy | Full returned window raw text joined LF, including hidden middle and preserved final empty-text line; no gutter/banner/fold chrome. Empty file omits copy; one empty-text row still has Copy. Successful local adapter shows Copied/check for1000ms; feedback flag blocks another click; refused adapter leaves Copy. |\n\n' +
    (short ? 'Measured short: body ' + short.body.box.w + '×' + short.body.box.h + ', card ' + short.block.box.w + '×' + short.block.box.h + ', root ' + short.root.box.h + 'px. ' : '') +
    (empty && text ? 'Empty file body/card ' + empty.body.box.h + '/' + empty.block.box.h + 'px; single blank-text row body ' + text.body.box.h + 'px. ' : '') +
    (non ? 'Gutter3ch=' + short?.rows[0].gutter.box.w + 'px;4ch=' + non.rows[0].gutter.box.w + 'px. ' : '') +
    (window ? '14of100 window retains37..40/fold6/47..50 at rest; expand shows37..50, then collapse control. Expanded body ' + E.measurements.fold?.expandedHeight + 'px and collapsed ' + E.measurements.fold?.collapsedHeight + 'px.\n\n' : '\n\n') +
    'Source-only validation executes the exact emitted readMeta/validReadCall/readCardModel/readCallLine, parser and path helpers, then independently executes the current ui-tool client.js read-model fragment against all ' + source.sourceOnly.readModel.table.length + ' cases. Only a settled root success read with valid object arguments, one text result, exact LF file envelope and valid persisted metadata becomes a read card. Numbering must strictly increase from at least offset and stay within totalLines; gaps are allowed. Metadata text/path controls the card; it is not cross-checked against envelope body/path or argument offset. Empty metadata path and an empty window with offset past total are accepted by installed validation. Table records these facts without imposing new validation. No application model imports or Host execution occur.\n\n' +
    'Installed authority: JZ ReadBlock, r4 CodeToolbar, Vw FoldToggle in the installed web JS; CSS suffix1ay0u/1pq26; ui-tool ToolRow/read-family/read-row/read-card/label adapters and current theme. Read-only reference TypeScript explains symbols; installed bytes/maps are authoritative.\n\n' +
    'The fixture explicitly implements local fold/wrap/copy transitions and a memory clipboard. It does not run React, Shiki/highlightLines, lazy grammar loading, viewport observers, actual motion/presence, Tooltip portals, original callbacks, real file operations, application backend or Host dispatch. CDP Tab only tests focus; no native Enter/Space activation claim. Pending clipboard races are not tested; only the copied success flag duplicate guard. No OS clipboard is accessed.\n\n' +
    'Source bytes unchanged: ' + E.sourceIntegrity?.allUnchanged + '; startup count ' + spawnCount + '; measured DPR ' + E.browser?.viewport?.dpr + '; cleanup ' + E.cleanup?.pass + '. All observed owned Chrome PIDs exited, the exact debugging port released and the owned profile removed when cleanup passes. Protected Electron20012/backend16008/LockApp12516 and GUI3080 untouched. Combined JSON ' + E.compact?.combinedJSONBytes + ' bytes, target<250000.\n\n' +
    Object.entries(E.screenshots).map(([name, v]) => mdLink(name, v.path)).join(' · ') + '\n\n' +
    E.cases.filter(v => v.status !== 'passed').map(v => 'Failure retained: ' + v.id + ': ' + (v.error ?? v.assertions.filter(a => !a.pass).map(a => a.name + ' ' + JSON.stringify(a.detail ?? null)).join('; '))).join('\n\n') + (E.errors.length ? '\n\nErrors: ' + E.errors.join('\n\n') : '') + '\n';
}
function textReport() { return ['Round118 installed ReadBlock: ' + E.status.toUpperCase(), 'Run ' + RUN_ID, 'Cases ' + E.summary.passedCases + '/' + E.cases.length + '; assertions ' + E.summary.passedAssertions + '/' + (E.summary.passedAssertions + E.summary.failedAssertions), ...E.cases.map(v => v.status.toUpperCase() + ' ' + v.id + ' ' + v.assertions.filter(a => a.pass).length + '/' + v.assertions.length), ...E.cases.flatMap(v => v.assertions.filter(a => !a.pass).map(a => 'FAIL ' + v.id + ': ' + a.name + ' ' + JSON.stringify(a.detail ?? null))), ...E.cases.filter(v => v.error).map(v => v.id + ': ' + v.error), 'Combined JSON bytes ' + E.compact.combinedJSONBytes, 'Cleanup ' + JSON.stringify(E.cleanup), 'Errors ' + JSON.stringify(E.errors), ''].join('\n'); }
async function main() {
  await scope(); const x = await extract(), built = fixture(x), inline = built.html.match(/<script>\n([\s\S]*)<\/script>/); invariant(inline, 'Missing inline fixture'); new vm.Script(inline[1], {filename: 'r118-tool-read-inline.js'});
  if (MODE === '--prepare-only') {
    await writeOwned(OUTPUT.source, JSON.stringify(source) + '\n'); await writeOwned(FIXTURE, built.html);
    const prep = {schema: 'r118-tool-read-preflight/v1', status: 'prepared-no-browser', runId: RUN_ID, sourceEvidence: OUTPUT.source, sourceEvidenceSHA256: hash(await fs.readFile(OUTPUT.source)), fixture: FIXTURE, fixtureSHA256: hash(built.html), inlineParsed: true, sourceModelCases: source.sourceOnly.readModel.table.length, chromeNotStarted: true};
    await writeOwned(PREP, JSON.stringify(prep, null, 2) + '\n'); console.log(JSON.stringify({...prep, path: PREP, sourceEvidenceBytes: (await fs.stat(OUTPUT.source)).size})); return;
  }
  const prep = JSON.parse(await fs.readFile(PREP, 'utf8')); invariant(prep.status === 'prepared-no-browser' && prep.runId === RUN_ID && norm(prep.sourceEvidence) === norm(OUTPUT.source), 'Wrong preparation');
  invariant(hash(await fs.readFile(OUTPUT.source)) === prep.sourceEvidenceSHA256 && JSON.stringify(source) === JSON.stringify(JSON.parse(await fs.readFile(OUTPUT.source, 'utf8'))), 'Installed source/runner changed after preparation');
  invariant(prep.fixtureSHA256 === hash(built.html) && hash(await fs.readFile(FIXTURE)) === prep.fixtureSHA256, 'Prepared fixture changed'); E.preflight = {path: PREP, inlineParsed: true, sourceEvidenceVerified: true, fixtureVerified: true};
  await writeOwned(ATTEMPT, JSON.stringify({round: 118, runId: RUN_ID, startedAt: E.startedAt, prepared: PREP, chromeStartupsAllowed: 1, rerunForbidden: true}, null, 2) + '\n');
  try { await start(); await acceptance(built.models); } catch (error) { E.errors.push(error.stack ?? String(error)); console.error(error.stack ?? String(error)); } finally { try { await cleanup(); } catch (error) { E.errors.push('Cleanup ' + (error.stack ?? String(error))); } }
  E.finishedAt = new Date().toISOString(); E.cdpMethods = methodCounts;
  E.summary = {passedCases: E.cases.filter(v => v.status === 'passed').length, failedCases: E.cases.filter(v => v.status === 'failed').length, passedAssertions: E.cases.flatMap(v => v.assertions).filter(v => v.pass).length, failedAssertions: E.cases.flatMap(v => v.assertions).filter(v => !v.pass).length};
  E.status = E.cases.length === 10 && E.summary.failedCases === 0 && E.errors.length === 0 && E.cleanup?.pass ? 'passed' : 'failed';
  E.compact = {styleRecords: E.computedStyles?.length ?? 0, sourceJSONBytes: (await fs.stat(OUTPUT.source)).size, measurementJSONBytes: 0, combinedJSONBytes: 0, targetCombinedJSONBytes: 250000};
  let encoded; for (let i = 0; i < 4; i++) { encoded = JSON.stringify(E) + '\n'; E.compact.measurementJSONBytes = Buffer.byteLength(encoded); E.compact.combinedJSONBytes = E.compact.measurementJSONBytes + E.compact.sourceJSONBytes; } encoded = JSON.stringify(E) + '\n';
  if (Buffer.byteLength(encoded) + E.compact.sourceJSONBytes >= 250000) { E.errors.push('Combined JSON exceeds250KB'); E.status = 'failed'; encoded = JSON.stringify(E) + '\n'; }
  await writeOwned(OUTPUT.text, textReport()); await writeOwned(OUTPUT.report, report()); await writeOwned(OUTPUT.json, encoded);
  console.log(JSON.stringify({status: E.status, ...E.summary, evidence: OUTPUT, combinedJSONBytes: Buffer.byteLength(encoded) + E.compact.sourceJSONBytes, cleanup: E.cleanup?.pass})); if (E.status !== 'passed') process.exitCode = 1;
}
await main();
