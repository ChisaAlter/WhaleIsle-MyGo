#!/usr/bin/env node
/** Round119: installed DiffBlock source VM validation and one owned Chrome acceptance. */
import fs from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';
import {createHash, randomUUID} from 'node:crypto';
import net from 'node:net';
import vm from 'node:vm';
import {spawn} from 'node:child_process';

const ROOT = 'C:\\AI\\WhaleIsle-MyGo';
const SCRIPT = path.join(ROOT, 'scripts/tool-diff-reference-chrome.mjs');
const INSTALL = 'C:\\软件\\Whale Isle\\resources\\vendor\\deepseek-harness';
const TOOL = path.join(INSTALL, 'packages/client/ui-tool/lib');
const REF = path.join(ROOT, 'reference/electron/vendor/deepseek-harness/packages/client/ui-primitives/src');
const SOURCE_PATHS = {
  runner: SCRIPT, webJS: path.join(INSTALL, 'apps/web/dist/assets/index-wKUmVWbt.js'), webCSS: path.join(INSTALL, 'apps/web/dist/assets/index-l5jToo6l.css'),
  tool: path.join(TOOL, 'client.js'), row: path.join(TOOL, 'types/client/tool/components/ToolRow.js'),
  diff: path.join(TOOL, 'types/client/tool/models/diff-card-model.js'), labels: path.join(TOOL, 'types/client/tool/models/primitive-labels.js'),
  raw: path.join(TOOL, 'types/client/tool/models/raw-tool-call.js'), recorded: path.join(TOOL, 'types/client/tool/models/recorded-path.js'),
  mutation: path.join(TOOL, 'types/client/tool/toolviews/file-mutation-row.js'), generic: path.join(TOOL, 'types/client/tool/toolviews/GenericToolCard.js'),
  paths: path.join(INSTALL, 'packages/util/workspace-path/lib/types/index.js'), language: path.join(INSTALL, 'packages/util/code-language/lib/types/index.js'),
  theme: path.join(INSTALL, 'packages/client/ui-theme/lib/client.js'), locale: path.join(INSTALL, 'packages/client/ui-conversation/lib/types/client/locales.js'),
  commonEN: path.join(INSTALL, 'packages/client/locale/lib/types/locales/en.js'), commonZH: path.join(INSTALL, 'packages/client/locale/lib/types/locales/zh.js'),
  refDiff: path.join(REF, 'DiffBlock.tsx'), refDiffCSS: path.join(REF, 'DiffBlock.module.css'), refCardCSS: path.join(REF, 'CodeCard.module.css'),
  refToolbar: path.join(REF, 'CodeToolbar.tsx'), refFold: path.join(REF, 'FoldToggle.tsx'), icons: path.join(REF, 'icons/index.tsx'),
  priorRead: path.join(ROOT, 'scripts/tool-read-reference-chrome.mjs'), priorSummary: path.join(ROOT, 'scripts/tool-summary-reference-chrome.mjs'),
};
const CHROME = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const MODE = process.argv[2];
const suppliedPrep = MODE === '--accept-prepared' ? process.argv[3] : null;
const suppliedRun = suppliedPrep && path.basename(suppliedPrep).match(/^r119-tool-diff-([0-9a-fTZ-]+)-preflight\.json$/)?.[1];
const RUN_ID = suppliedRun ?? new Date().toISOString().replace(/[:.]/g, '-') + '-' + randomUUID().slice(0, 8);
const BASE = path.join(ROOT, '.devdata/r119-tool-diff-' + RUN_ID);
const FIXTURE = BASE + '.html', PROFILE = BASE + '-profile', PREP = BASE + '-preflight.json';
const ATTEMPT = path.join(ROOT, '.devdata/r119-tool-diff-acceptance-attempt.json');
const PREFIX = path.join(ROOT, 'docs/review-20261022/evidence/tool-diff-pass263-reference-' + RUN_ID);
const OUTPUT = {source: PREFIX + '-source.json', json: PREFIX + '.json', report: PREFIX + '-report.md', text: PREFIX + '.txt', light: PREFIX + '-light.png', dark: PREFIX + '-dark.png', copied: PREFIX + '-copied.png', expanded: PREFIX + '-expanded.png', wrapped: PREFIX + '-wrapped.png'};
const PROTECTED = new Set([20012, 16008, 12516]);
const norm = value => path.resolve(value).toLowerCase();
const hash = value => createHash('sha256').update(value).digest('hex');
const invariant = (value, message) => { if (!value) throw Error(message); };
const near = (a, b) => Math.abs(a - b) < 0.04;
const delay = ms => { invariant(ms <= 1000, 'Long blocking wait forbidden'); return new Promise(resolve => setTimeout(resolve, ms)); };
const allowed = new Set([FIXTURE, PREP, ATTEMPT, ...Object.values(OUTPUT)].map(norm));
const E = {
  schema: 'tool-diff-pass263-reference/v1', round: 119, runId: RUN_ID, startedAt: new Date().toISOString(), status: 'running',
  attemptPolicy: 'One exclusive round-wide owned Chrome acceptance after static preflight. One startup, each case once; retain failures with no accepting rerun.',
  scope: {
    script: SCRIPT, fixture: FIXTURE, profile: PROFILE,
    description: 'Manual installed DiffBlock/CodeToolbar/FoldToggle DOM, direct ToolRow.bodyWrap containment, verbatim installed CSS and current light/dark tokens, synthetic diffs and explicit local handlers. Exact installed jsdiff/localHunks/buildRows/copyText and emitted/bundled diffCardModel execute source-only in VM.',
    input: 'CDP only within own spawned PID/profile/port-verified headless Chrome; no hardware input, original Electron/GUI assertion or native Enter/Space activation claim.',
    network: 'No server; file CSP denies network; CDP blocks HTTP/HTTPS/WS/WSS. Only owned loopback debugging endpoint contacted.',
    protected: {electronPID: 20012, backendPID: 16008, lockAppPID: 12516, guiPort: 3080, touched: false},
  },
  compactPolicy: 'Unique installed CSS/functions/maps/hashes in source sidecar; intern computed styles; canonical fixture geometry once; scalar transitions. Combined source and metrics JSON strictly below250000 bytes.',
  sourceEvidence: OUTPUT.source, cases: [], measurements: {}, screenshots: {}, errors: [], writes: [],
  limitations: [
    'Installed JS/CSS/ui-tool artifacts are authoritative. Older read-only reference DiffBlock/CodeToolbar/FoldToggle/CSS guides symbol mapping only; installed source/maps are verified.',
    'Manual DOM and explicit local handlers do not execute React, hooks/useMemo/useState/useCallback, reconciliation/remounts, actual CodeToolbar/FoldToggle callbacks, Shiki/grammar loading, Host dispatch or application backend.',
    'Current DiffBlock itself renders plain text without invoking Shiki; language only labels the toolbar. Pure supportsHighlighting alias lookup is tested source-only; no highlighter runs.',
    'Copy success/refusal uses only an in-memory fixture adapter. No OS/application clipboard access. Copied flag guards clicks after success; pending-write races are not certified.',
    'Motion/presence, tooltips/portals/timers, file opening, native operations and original app are unexecuted. Fold/copy local timers certify the fixture adapter only.',
    'DPR1 measured from startup flags; no Emulation, monitor or physical display claim. System-resolved installed font stacks only.',
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
const digestText = text => ({chars: text.length, bytes: Buffer.byteLength(text), sha256: hash(text)});
function compactDiffs(diffs) {
  return diffs.map(d => ({path: d.path, oldText: d.oldText === null ? null : d.oldText.length < 500 ? d.oldText : digestText(d.oldText), newText: d.newText.length < 500 ? d.newText : digestText(d.newText)}));
}
function modelTable(pure, bundled) {
  const edit = {file_path: 'src/file.ts', old_string: 'old', new_string: 'new'};
  const write = {file_path: 'src/file.ts', content: 'new'};
  const applied = [{path: 'C:\\fixture\\src\\file.ts', oldText: 'applied-old', newText: 'applied-new'}];
  const samples = [];
  const add = (id, patch, accepted, expected) => samples.push({id, patch, accepted, expected});
  add('running-edit', {}, true, {path: edit.file_path, oldText: 'old', newText: 'new'});
  add('running-write', {name: 'write', args: write}, true, {path: write.file_path, oldText: null, newText: 'new'});
  add('preparing-no-card', {phase: 'preparing'}, false);
  add('root-child', {parentCallId: 'p'}, false);
  add('null-parent-still-child', {parentCallId: null}, false);
  add('args-invalid-json', {argsRaw: '{'}, false);
  add('args-array', {argsRaw: '[]'}, false);
  add('args-null', {argsRaw: 'null'}, false);
  add('args-string', {argsRaw: '"text"'}, false);
  add('empty-path', {args: {...edit, file_path: ''}}, false);
  add('whitespace-path', {args: {...edit, file_path: '  '}}, false);
  add('numeric-path', {args: {...edit, file_path: 7}}, false);
  add('relative-path-retained', {args: {...edit, file_path: './a.ts'}}, true, {path: './a.ts', oldText: 'old', newText: 'new'});
  add('empty-old-becomes-null', {args: {...edit, old_string: ''}}, true, {path: edit.file_path, oldText: null, newText: 'new'});
  add('empty-new-delete', {args: {...edit, new_string: ''}}, true, {path: edit.file_path, oldText: 'old', newText: ''});
  add('edit-old-missing', {args: {file_path: 'a', new_string: 'new'}}, false);
  add('edit-new-nonstring', {args: {...edit, new_string: 1}}, false);
  add('replace-all-true', {args: {...edit, replace_all: true}}, true);
  add('replace-all-false', {args: {...edit, replace_all: false}}, true);
  add('replace-all-null', {args: {...edit, replace_all: null}}, false);
  add('replace-all-string', {args: {...edit, replace_all: 'true'}}, false);
  add('write-empty-content', {name: 'write', args: {...write, content: ''}}, true, {path: write.file_path, oldText: null, newText: ''});
  add('write-content-missing', {name: 'write', args: {file_path: 'a'}}, false);
  add('write-content-nonstring', {name: 'write', args: {...write, content: null}}, false);
  add('other-tool', {name: 'read'}, false);
  add('str-create-default-empty', {name: 'str_replace_editor', args: {command: 'create', path: 'a.ts'}}, true, {path: 'a.ts', oldText: null, newText: ''});
  add('str-create-text', {name: 'str_replace_editor', args: {command: 'create', path: 'a.ts', file_text: 'a'}}, true, {path: 'a.ts', oldText: null, newText: 'a'});
  add('str-create-nonstring', {name: 'str_replace_editor', args: {command: 'create', path: 'a.ts', file_text: null}}, false);
  add('str-replace-defaults', {name: 'str_replace_editor', args: {command: 'str_replace', path: 'a.ts'}}, true, {path: 'a.ts', oldText: null, newText: ''});
  add('str-replace-empty-old-retained', {name: 'str_replace_editor', args: {command: 'str_replace', path: 'a.ts', old_str: '', new_str: 'b'}}, true, {path: 'a.ts', oldText: '', newText: 'b'});
  add('str-replace-bad-old', {name: 'str_replace_editor', args: {command: 'str_replace', path: 'a.ts', old_str: 1}}, false);
  add('str-replace-bad-new', {name: 'str_replace_editor', args: {command: 'str_replace', path: 'a.ts', new_str: 1}}, false);
  add('str-view-no-card', {name: 'str_replace_editor', args: {command: 'view', path: 'a.ts'}}, false);
  add('str-empty-path', {name: 'str_replace_editor', args: {command: 'create', path: ' '}}, false);
  add('settled-str-create-generic', {name: 'str_replace_editor', args: {command: 'create', path: 'a.ts'}, settled: true}, false);
  add('settled-str-replace-generic', {name: 'str_replace_editor', args: {command: 'str_replace', path: 'a.ts'}, settled: true}, false);
  add('settled-edit-applied-remap', {settled: true, meta: {path: applied[0].path, diffs: applied}}, true, {path: edit.file_path, oldText: 'applied-old', newText: 'applied-new'});
  add('settled-error-edit', {settled: true, isError: true, meta: {diffs: applied}}, false);
  add('settled-error-write', {name: 'write', args: write, settled: true, isError: true}, false);
  for (const [id, meta] of [['missing', undefined], ['null', null], ['array', []], ['no-diffs', {}], ['empty-diffs', {diffs: []}], ['nonarray-diffs', {diffs: {}}], ['null-hunk', {diffs: [null]}], ['numeric-path', {diffs: [{path: 7, oldText: null, newText: ''}]}], ['missing-old', {diffs: [{path: '', newText: ''}]}], ['bad-new', {diffs: [{path: '', oldText: null, newText: 7}]}]]) {
    add('settled-edit-' + id, {settled: true, meta}, false);
    add('settled-write-fallback-' + id, {settled: true, name: 'write', args: write, meta}, true, {path: write.file_path, oldText: null, newText: 'new'});
  }
  add('applied-empty-path-accepted', {settled: true, meta: {diffs: [{path: '', oldText: null, newText: ''}]}}, true, {path: '', oldText: null, newText: ''});
  add('applied-other-path-retained', {settled: true, meta: {path: 'C:\\fixture\\different.ts', diffs: applied}}, true, applied[0]);
  add('relative-recorded-path-no-remap', {settled: true, meta: {path: applied[0].path.slice(11), diffs: applied}}, true, applied[0]);
  add('applied-extra-fields-discarded', {settled: true, meta: {diffs: [{path: 'a.ts', oldText: null, newText: 'a', ignored: 'extra'}]}}, true, {path: 'a.ts', oldText: null, newText: 'a'});
  add('invalid-second-hunk-rejects-all', {settled: true, meta: {diffs: [applied[0], {path: 'bad', oldText: false, newText: ''}]}}, false);
  return samples.map(({id, patch, accepted, expected}) => {
    const name = patch.name ?? 'edit', args = patch.args ?? edit, call = {name, argsRaw: patch.argsRaw ?? JSON.stringify(args)};
    const block = patch.settled ? {kind: 'tool_result', call, isError: patch.isError ?? false, content: [], meta: patch.meta} : {phase: patch.phase ?? 'start', ...call};
    if (Object.hasOwn(patch, 'parentCallId')) block.parentCallId = patch.parentCallId;
    const actual = pure.diffCardModel(block), other = bundled.diffCardModel(block);
    const same = JSON.stringify(actual) === JSON.stringify(other), selected = actual?.card.diffs[0];
    const pass = same && (actual !== null) === accepted && (!expected || JSON.stringify(selected) === JSON.stringify(expected));
    return {id, input: block, expectedAccepted: accepted, ...(expected ? {expectedFirst: expected} : {}), result: actual, emittedMatchesBundled: same, pass};
  });
}
function diffTable(pure) {
  const d = (oldText, newText, file = 'fixture.txt') => ({path: file, oldText, newText});
  const contextOld = Array.from({length: 24}, (_, i) => 'line-' + i), contextNew = [...contextOld]; contextNew[2] = 'changed-2'; contextNew[20] = 'changed-20';
  const bounded = n => ['shared-prefix', ...Array.from({length: n}, (_, i) => 'old-' + i), 'shared-suffix'].join('\n');
  const changed = n => ['shared-prefix', ...Array.from({length: n}, (_, i) => 'new-' + i), 'shared-suffix'].join('\n');
  const largeOld = ['共享开头🧭', ...Array.from({length: 130}, (_, i) => '旧' + i + '漢字🧪e\u0301'.repeat(80)), '共享结尾🧭'].join('\n');
  const largeNew = ['共享开头🧭', ...Array.from({length: 130}, (_, i) => '新' + i + '仮名🚀o\u0308'.repeat(80)), '共享结尾🧭'].join('\n');
  const samples = [
    ['empty-array', [], [0, 0], []], ['empty-create-path-only', [d(null, '')], [0, 0], ['path']],
    ['unchanged-path-only', [d('a\nb\n', 'a\nb\n')], [0, 0], ['path']],
    ['terminator-only', [d('a\nb', 'a\nb\n')], [0, 0], ['path']],
    ['create-two', [d(null, 'a\nb\n')], [2, 0], ['path', 'add', 'add']],
    ['delete-two', [d('a\nb\n', '')], [0, 2], ['path', 'del', 'del']],
    ['replacement-context', [d('a\nb\nc\n', 'a\nB\nc\n')], [1, 1], ['path', 'context', 'del', 'add', 'context']],
    ['blank-add', [d(null, '\n')], [1, 0], ['path', 'add']],
    ['interior-blank', [d('a\nb', 'a\n\nb')], [1, 0]],
    ['two-terminal-LF', [d('a\n', 'a\n\n')], [1, 0]],
    ['same-CRLF', [d('a\r\nb\r\n', 'a\r\nb\r\n')], [0, 0]],
    ['CRLF-versus-LF', [d('a\r\nb\r\n', 'a\nb\n')], [2, 2]],
    ['bare-CR-not-split', [d('a\rb', 'a\rc')], [1, 1]],
    ['literal-sign-content', [d('+old\n-same\n', '+new\n-same\n')], [1, 1]],
    ['ambiguous-LCS', [d('a\nb\na\n', 'a\na\nb\n')], [1, 1], ['path', 'context', 'del', 'context', 'add']],
    ['distant-context-gap', [d(contextOld.join('\n'), contextNew.join('\n'))], [2, 2]],
    ['same-file-gap-and-new-path', [d('a', 'b', 'a.ts'), d('c', 'd', 'a.ts'), d('e', 'f', 'b.ts')], [3, 3], ['path', 'del', 'add', 'gap', 'del', 'add', 'path', 'del', 'add']],
    ['nine-rows', [d(null, Array.from({length: 8}, (_, i) => 'a' + i).join('\n'))], [8, 0]],
    ['ten-rows', [d(null, Array.from({length: 9}, (_, i) => 'a' + i).join('\n'))], [9, 0]],
    ['exact256-keeps-shared-context', [d(bounded(128), changed(128))], [128, 128]],
    ['over256-full-fragment-fallback', [d(bounded(129), changed(129))], [131, 131]],
    ['large-unicode-full-fragments', [d(largeOld, largeNew, '目录/🧭.ts')], [132, 132]],
  ];
  return samples.map(([id, diffs, expected, kinds]) => {
    const rows = pure.eV(diffs), totals = pure.QZ(diffs), copy = pure.tV(rows), hunks = diffs.map(value => pure.Bw(value));
    const counts = Object.fromEntries(['path', 'gap', 'context', 'del', 'add'].map(kind => [kind, rows.filter(row => row.kind === kind).length]));
    let pass = totals.added === expected[0] && totals.removed === expected[1] && (!kinds || rows.map(row => row.kind).join() === kinds.join());
    if (id === 'blank-add') pass &&= rows[1].text === '' && copy === 'fixture.txt\n+ ';
    if (id === 'distant-context-gap') pass &&= hunks[0].length === 2 && counts.gap === 1 && !rows.some(row => row.text === 'line-12');
    if (id === 'ambiguous-LCS') pass &&= copy === 'fixture.txt\n  a\n- b\n  a\n+ b';
    if (id === 'over256-full-fragment-fallback') pass &&= counts.context === 0 && rows[1].text === 'shared-prefix' && rows.at(-1).text === 'shared-suffix';
    if (id === 'large-unicode-full-fragments') {
      const oldRows = rows.filter(row => row.kind === 'del').map(row => row.text), newRows = rows.filter(row => row.kind === 'add').map(row => row.text);
      pass &&= oldRows.join('\n') === largeOld && newRows.join('\n') === largeNew && oldRows[65].includes('🧪e\u0301') && newRows[65].includes('🚀o\u0308');
    }
    return {id, inputs: compactDiffs(diffs), ...(id.includes('unicode') ? {recipe: '132 LF-joined lines: shared Chinese/emoji prefix/suffix,130 indexed old/new lines each with80 repeats of CJK+astral emoji+combining sequence; full strings executed without truncation.'} : {}), expected: {added: expected[0], removed: expected[1]}, totals, counts, hunkCounts: hunks.map(v => v.length), rows: rows.length < 30 ? rows : {count: rows.length, sha256: hash(JSON.stringify(rows))}, copy: copy.length < 700 ? copy : digestText(copy), pass};
  });
}
async function extract() {
  const texts = Object.fromEntries(await Promise.all(Object.entries(SOURCE_PATHS).map(async ([id, file]) => [id, await fs.readFile(file, 'utf8')])));
  const {webJS, webCSS, tool, theme, row} = texts;
  const allRules = [...webCSS.matchAll(/([^{}]+)\{([^{}]*)\}/g)], classCSS = suffix => allRules.filter(v => v[1].includes('_' + suffix + '_')).map(v => v[0]).join('');
  const diffCSS = classCSS('17vb8'), cardCSS = classCSS('1pq26');
  const rowLiteral = tool.match(/^\t\tconst css\$3 = ("(?:[^"\\]|\\.)*");\r?$/m); invariant(rowLiteral, 'Missing installed ToolRow CSS');
  const rowMap = fragment(tool, 'var ToolRow_module_css_default = ', '\n\t\t//#endregion', 'tool');
  const disclosureCSS = fragment(webCSS, '._root_1rdzk_1{', '._button_ll2mm_2{', 'webCSS');
  const maps = {diff: cssMap(diffCSS, '17vb8', ['block', 'body', 'line', 'path', 'gap', 'del', 'add', 'context', 'expand']), card: cssMap(cardCSS, '1pq26', ['card', 'header', 'heading', 'language', 'actions', 'action']), row: Object.fromEntries([...rowMap.text.matchAll(/"([^"]+)": "([^"]+)"/g)].map(v => [v[1], v[2]])), disclosure: cssMap(disclosureCSS.text, '1j0v6', ['root', 'header', 'row', 'leading', 'title']), shimmer: cssMap(disclosureCSS.text, '1rdzk', ['root', 'content', 'text'])};
  const css = {diff: {source: 'webCSS', text: diffCSS}, card: {source: 'webCSS', text: cardCSS}, row: {source: 'tool', line: tool.slice(0, rowLiteral.index).split('\n').length, text: JSON.parse(rowLiteral[1])}, disclosure: disclosureCSS, ...Object.fromEntries(['base', 'motion', 'corner_shape', 'design_platform', 'focus', 'scrollbar', 'gradient_shadow_text'].map(name => ['theme_' + name, literal(theme, name)]))};
  const installedJS = {
    jsdiff: fragment(webJS, 'class QL{', 'const oT=', 'webJS'), localHunks: fragment(webJS, 'function Bw(', 'function QZ(', 'webJS'),
    totals: fragment(webJS, 'function QZ(', 'function eV(', 'webJS'), buildRows: fragment(webJS, 'function eV(', 'function L8(', 'webJS'),
    contentLines: fragment(webJS, 'function L8(', 'function tV(', 'webJS'), copyText: fragment(webJS, 'function tV(', 'function nV(', 'webJS'),
    assertNever: fragment(webJS, 'function XZ(', 'const M8=', 'webJS'), diffBlock: fragment(webJS, 'function nV(', 'const vn=', 'webJS'),
    toolbar: fragment(webJS, 'function r4(', 'const VZ=', 'webJS'), fold: fragment(webJS, 'function Vw(', 'function r4(', 'webJS'),
    language: fragment(webJS, '__=new Map(Object.entries(', 'const BM=', 'webJS'), highlightingAliases: fragment(webJS, 'GH=new Map(', 'const qH=', 'webJS'),
  };
  invariant(installedJS.diffBlock.text.includes('maxLines:r=16') && installedJS.diffBlock.text.includes('if(n.length===0)return null') && installedJS.diffBlock.text.includes('I=Math.ceil(r/2),k=r-I') && installedJS.diffBlock.text.includes('Jo(tV(n))') && installedJS.diffBlock.text.includes('1e3'), 'DiffBlock installed behavior changed');
  invariant(installedJS.localHunks.text.includes('context:3,maxEditLength:YZ') && diffCSS.includes('grid-template-columns:minmax(max-content,1fr)') && diffCSS.includes('padding:6px 0 20px') && diffCSS.includes('inset 3px 0 0'), 'Diff CSS/algorithm changed');
  invariant(texts.diff.includes('CHAT_DIFF_MAX_LINES = 9;') && row.includes('maxLines: CHAT_DIFF_MAX_LINES, className: css.diffBody') && texts.mutation.includes('const diff = diffCardModel(block);') && texts.generic.includes('const diff = diffCardModel(block);'), 'Installed diff callsites changed');
  invariant(css.row.text.includes('.' + maps.row.diffBody + ',') || css.row.text.includes(',' + '.' + maps.row.diffBody), 'Diff caller CSS missing');
  for (const value of Object.values(maps).flatMap(Object.values)) invariant(webJS.includes('"' + value + '"') || tool.includes('"' + value + '"'), 'CSS map absent from installed JS ' + value);
  const diffCode = installedJS.jsdiff.text + '\nconst YZ=256;\n' + ['localHunks', 'totals', 'buildRows', 'contentLines', 'copyText', 'assertNever'].map(key => installedJS[key].text).join('\n') + '\n({Bw,QZ,eV,L8,tV})';
  const pure = vm.runInNewContext(diffCode, Object.create(null), {timeout: 1500}), diffTests = diffTable(pure);
  invariant(diffTests.every(v => v.pass), 'Exact diff source VM table differs: ' + JSON.stringify(diffTests.filter(v => !v.pass)));
  const rawCode = pureFunction(texts.raw, 'parsedToolCall');
  const recordedCode = pureFunction(texts.recorded, 'recordedAbsolutePath');
  const pathCode = ['isWindowsStylePath', 'isAbsoluteWorkspacePath'].map(name => pureFunction(texts.paths, name)).join('\n');
  const modelCode = ['narrowDiffs', 'intendedDiff', 'appliedDiffs', 'diffCardModel'].map(name => pureFunction(texts.diff, name)).join('\n');
  const bundledModel = fragment(tool, '\t\tfunction narrowDiffs(', '\n\t\t//#endregion', 'tool');
  const modelVM = code => vm.runInNewContext('const parsedCalls=new WeakMap();\n' + rawCode + '\n' + pathCode + '\n' + recordedCode + '\n' + code + '\n({diffCardModel})', Object.create(null), {timeout: 1000});
  const modelTests = modelTable(modelVM(modelCode), modelVM(bundledModel.text));
  invariant(modelTests.every(v => v.pass), 'Exact model source VM table differs: ' + JSON.stringify(modelTests.filter(v => !v.pass)));
  const languageCode = texts.language.replace(/^export /gm, '').replace(/^\/\/# sourceMappingURL=.*$/m, '');
  const emittedLang = vm.runInNewContext(languageCode + '\n({languageForPath,CODE_HIGHLIGHT_EXTENSIONS})', Object.create(null), {timeout: 1000});
  const bundledLang = vm.runInNewContext('const ' + installedJS.language.text + '\n({languageForPath:A2})', Object.create(null), {timeout: 1000});
  const support = vm.runInNewContext('const ' + installedJS.highlightingAliases.text + '\n({Iw})', Object.create(null), {timeout: 1000});
  const languageTests = ['src/a.ts', 'src/b.TSX', 'a.py', 'a.mjs', 'C:\\dir\\MAIN.PS1', '.env', 'foo.makefile', 'Makefile', 'a.unknown', 'a.constructor', 'a.__proto__', 'a.lock', 'dir.with.dot/file', 'foo.jsonl'].map(file => {
    const value = emittedLang.languageForPath(file), other = bundledLang.languageForPath(file);
    return {path: file, language: value ?? null, supported: support.Iw(value), pass: value === other && (value === undefined || support.Iw(value))};
  });
  invariant(languageTests.every(v => v.pass), 'Language source table differs');
  const dictionary = {};
  for (const language of ['en', 'zh']) {
    const start = texts.locale.indexOf('export const ' + language + ' = {'), end = texts.locale.indexOf('\n};', start) + 3; invariant(start >= 0 && end > start, 'Missing conversation dictionary ' + language);
    dictionary[language] = Object.fromEntries(['diff.collapseAria', 'diff.expandAria', 'diff.expandRest'].map(key => [key, localeLabel(texts.locale.slice(start, end), key)]));
    for (const key of ['copy', 'copied', 'collapse', 'codeBlock.title', 'codeBlock.wrap', 'codeBlock.unwrap']) dictionary[language][key] = localeLabel(texts[language === 'en' ? 'commonEN' : 'commonZH'], key);
  }
  const labelsCode = ['codeToolbarLabels', 'diffBlockLabels'].map(name => pureFunction(texts.labels, name)).join('\n');
  const labels = Object.fromEntries(Object.entries(dictionary).map(([language, values]) => [language, vm.runInNewContext(labelsCode + '\n({diffBlockLabels})', Object.create(null), {timeout: 1000}).diffBlockLabels((key, args) => { invariant(Object.hasOwn(values, key), 'Unknown locale key ' + key); return Object.entries(args ?? {}).reduce((out, [k, v]) => out.replaceAll('{' + k + '}', String(v)), values[key]); })]));
  const artwork = {};
  for (const [key, name] of [['up', 'IconChevronUpOutlineArtwork'], ['wrap', 'IconWrapFillArtwork'], ['nowrap', 'IconNowrapFillArtwork'], ['copy', 'IconCopyOutlineArtwork'], ['check', 'IconCheckOutlineArtwork']]) {
    const start = texts.icons.indexOf(name + ' = '), end = texts.icons.indexOf('\n)', start) + 2; invariant(start >= 0 && end > start, 'Missing icon guide ' + name);
    const shapes = [...texts.icons.slice(start, end).matchAll(/<(path|rect)\s+([^>]+)\/>/g)].map(v => ({tag: v[1], attributes: Object.fromEntries([...v[2].matchAll(/([\w-]+)="([^"]+)"/g)].map(a => [a[1], a[2]]))}));
    invariant(shapes.length && shapes.every(s => s.tag !== 'path' || webJS.includes(s.attributes.d)), 'Icon path not in installed JS');
    for (const s of shapes.filter(v => v.tag === 'rect')) invariant(webJS.includes(Object.entries(s.attributes).map(([k, v]) => k + ':' + JSON.stringify(v)).join(',')), 'Icon rect not in installed JS');
    artwork[key] = {source: 'icons', name, shapes, installedVerified: true};
  }
  const themeTokens = {};
  const tokenNames = ['--dsw-alias-state-success-primary', '--dsw-alias-state-error-primary', '--dsw-alias-code-diff-added', '--dsw-alias-code-diff-deleted', '--dsw-alias-markdown-code-block', '--dsw-alias-label-secondary', '--dsw-alias-label-tertiary'];
  for (const mode of ['light', 'dark']) {
    const tokens = {};
    for (const r of css.theme_design_platform.text.matchAll(/([^{}]+)\{([^{}]*)\}/g)) if ([':root', 'body'].includes(r[1]) || (mode === 'dark' && r[1] === 'body[data-ds-dark-theme]')) for (const v of r[2].matchAll(/(--[\w-]+):([^;}]+)/g)) tokens[v[1]] = v[2];
    themeTokens[mode] = Object.fromEntries(tokenNames.map(name => { let value = tokens[name]; for (let depth = 0; /^var\(/.test(value) && depth < 20; depth++) value = tokens[value.slice(4, -1)]; invariant(/^#[0-9a-f]{6}([0-9a-f]{2})?$/i.test(value), 'Cannot resolve theme token ' + name); return [name, value]; }));
  }
  source = {
    schema: 'tool-diff-pass263-reference-source/v1', runId: RUN_ID, authority: INSTALL,
    files: await Promise.all(Object.entries(SOURCE_PATHS).map(async ([id, file]) => ({id, path: file, bytes: (await fs.stat(file)).size, sha256: hash(await fs.readFile(file))}))),
    css, maps, installedJS, dictionary, themeTokens, artwork,
    sourceOnly: {diff: {symbols: {jsdiff: 'QL/.../rT', localHunks: 'Bw', buildRows: 'eV', contentLines: 'L8', copyText: 'tV', totals: 'QZ', maxEditLength: 256, context: 3}, table: diffTests}, model: {source: 'diff', functions: modelCode, dependencies: {raw: rawCode, recorded: recordedCode, paths: pathCode}, bundledVerification: {source: 'tool', startUtf16Offset: bundledModel.startUtf16Offset, endUtf16Offset: bundledModel.endUtf16Offset, sha256: hash(bundledModel.text), method: 'Execute emitted and exact client.js model fragments independently for every table entry; same parser/path helpers, no app import.'}, table: modelTests}, language: {source: 'language', emittedSHA256: hash(languageCode), table: languageTests}, labels: {source: 'labels', functions: labelsCode, en: {expand1: labels.en.expand(1), expandAria1: labels.en.expandAria(1)}, zh: {expand1: labels.zh.expand(1), expandAria1: labels.zh.expandAria(1)}},
      caller: {source: 'row', lines: [38, 45, 56, 68, 97, 101, 102], maxLines: 9, primitiveDefault: 16, className: maps.row.diffBody, containment: 'ToolRow.bodyWrap flex column directly contains DiffBlock; no bodyScroll260 cap. FileMutationRow and GenericToolCard both derive diffCardModel.', margin: '4px 0 4px 4px'},
      semantics: {rows: 'Each new path starts path row; consecutive same-path fragment starts ⋯ gap; distant local hunks also insert ⋯. Identical/empty single fragment retains path-only; empty diffs renders null.', normalization: 'Empty text zero lines; remove one terminal LF only; CR remains. Normalize each content line by appending LF before bounded jsdiff.', fallback: 'Over256 edits replaces complete old/new fragments including shared prefix/suffix; no source/text truncation.', copy: 'Entire flat row list including hidden middle. del=-space, add=+space, context=two spaces, path/gap verbatim; LF joined no final extra LF.', language: 'Common canonical languageForPath across every diff else undefined; toolbar supportsHighlighting guards fallback. DiffBlock plain text has no Shiki render.', fold: 'hidden=all rows-max9 including paths/gaps; head ceil9/2=5,tail4; expanded all rows then collapse control.', copyState: 'copied flag guard after success,1000ms feedback; refusal leaves Copy; pending-write race not certified.'}},
    referenceVerification: 'Read-only guides locate symbols and icon artwork. Verbatim current installed functions/CSS/maps/tokens govern all assertions; no original/dependency/native/build/model-test changes.',
  };
  return {css, maps, dictionary, labels, artwork, rows: diffs => JSON.parse(JSON.stringify(pure.eV(diffs))), raw: rows => pure.tV(rows), lang: diffs => { const first = diffs[0] ? emittedLang.languageForPath(diffs[0].path) : undefined; return diffs.every(d => emittedLang.languageForPath(d.path) === first) && support.Iw(first) ? first : undefined; }};
}
const esc = v => String(v).replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;');
function makeModels(x) {
  const d = (oldText, newText, file = 'src/file.ts') => ({path: file, oldText, newText});
  return [
    {id: 'short', x: 40, y: 110, w: 620, diffs: [d('alpha\nold\nomega\n', 'alpha\nnew\nomega\n')]},
    {id: 'nine', x: 40, y: 410, w: 620, diffs: [d(null, Array.from({length: 8}, (_, i) => 'a' + i).join('\n'))]},
    {id: 'ten', x: 40, y: 750, w: 620, diffs: [d(null, Array.from({length: 9}, (_, i) => 'a' + i).join('\n'))]},
    {id: 'unchanged', x: 40, y: 1090, w: 620, diffs: [d('same\n', 'same\n', 'same.txt')]},
    {id: 'empty', x: 40, y: 1260, w: 620, diffs: []},
    {id: 'blank', x: 40, y: 1360, w: 620, diffs: [d(null, '\n', 'blank.txt')]},
    {id: 'multi', x: 720, y: 110, w: 720, diffs: [d('  first\nold\nend\n', '  first\nnew\nend\n', 'src/a.ts'), d('old fragment', 'new fragment', 'src/a.ts'), d('head\nremove\ntail\n', 'head\nadded\ntail\n', 'src/b.ts')]},
    {id: 'wrap', x: 720, y: 650, w: 340, diffs: [d('OLD-' + 'abcdefghij-'.repeat(16), 'NEW-' + 'ABCDEFGHIJ-'.repeat(16), 'src/long.ts')]},
    {id: 'common-language', x: 1100, y: 650, w: 340, diffs: [d(null, 'first', 'src/a.ts'), d(null, 'second', 'src/b.TSX')]},
    {id: 'mixed-language', x: 1100, y: 960, w: 340, diffs: [d(null, 'first', 'src/a.ts'), d(null, 'second', 'src/b.py')]},
    {id: 'no-language', x: 1100, y: 1270, w: 340, diffs: [d(null, 'plain', 'README')]},
    {id: 'zh', x: 720, y: 1270, w: 340, locale: 'zh', diffs: [d(null, Array.from({length: 9}, (_, i) => '行' + i).join('\n'), 'zh.txt')]},
  ].map(v => { const rows = x.rows(v.diffs); return {...v, maxLines: 9, locale: v.locale ?? 'en', language: x.lang(v.diffs) ?? null, rows, raw: x.raw(rows)}; });
}
function fixture(x) {
  const models = makeModels(x), m = x.maps;
  const svg = key => '<svg data-icon="' + key + '" width="14" height="14" viewBox="0 0 16 16" fill="none" aria-hidden="true" stroke-width="1">' + x.artwork[key].shapes.map(s => '<' + s.tag + ' ' + Object.entries(s.attributes).map(([k, v]) => k + '="' + esc(v) + '"').join(' ') + '/>').join('') + '</svg>';
  const rows = (values, all) => values.map(v => '<div class="' + m.diff.line + ' ' + m.diff[v.kind] + '" data-diff-line data-kind="' + v.kind + '" data-index="' + all.indexOf(v) + '">' + esc(v.text) + '</div>').join('');
  const block = v => {
    if (v.rows.length === 0) return '';
    const L = x.labels[v.locale], hidden = v.rows.length - v.maxLines, capped = hidden > 0;
    return '<div class="' + m.card.card + ' ' + m.diff.block + ' ' + m.row.diffBody + '" data-diff="" data-code-wrap="false"><div class="' + m.card.header + '" data-code-block-banner><div class="' + m.card.heading + '" data-heading><span class="' + m.card.language + '" data-language>' + esc(v.language ?? L.codeLabel) + '</span></div><div class="' + m.card.actions + '" data-actions><button type="button" class="' + m.card.action + '" data-wrap aria-label="' + esc(L.wrapLabel) + '" aria-pressed="false" data-fixture-tooltip-label="' + esc(L.wrapLabel) + '">' + svg('wrap') + '</button><button type="button" class="' + m.card.action + '" data-copy aria-label="' + esc(L.copy) + '" data-fixture-tooltip-label="' + esc(L.copy) + '">' + svg('copy') + '</button></div></div><div class="' + m.diff.body + '" data-diff-body>' + rows(capped ? v.rows.slice(0, 5) : v.rows, v.rows) + (capped ? '<button type="button" class="' + m.diff.expand + '" data-fold aria-expanded="false" aria-label="' + esc(L.expandAria(hidden)) + '">' + esc(L.expand(hidden)) + '</button>' + rows(v.rows.slice(-4), v.rows) : '') + '</div></div>';
  };
  const scenario = v => '<section class="scenario" style="left:' + v.x + 'px;top:' + v.y + 'px;width:' + v.w + 'px"><p class="caption">' + esc(v.id) + ' · ' + v.rows.length + ' total rows</p><div class="' + m.row.root + '" data-own-card="' + v.id + '" data-variant="edit" data-tool="edit" data-state="ok"><div class="' + m.disclosure.root + '"><div class="' + m.disclosure.header + '"><div class="' + m.disclosure.row + ' ' + m.row.row + '" data-disclosure-row data-expandable role="button" tabindex="0" aria-expanded="true"><span class="' + m.disclosure.leading + ' ' + m.row.leading + '">' + svg('up') + '</span><span class="' + m.shimmer.root + '"><span class="' + m.shimmer.content + '"><span class="' + m.shimmer.text + ' ' + m.disclosure.title + ' ' + m.row.title + '">Edited</span><span class="' + m.row.sep + '" aria-hidden="true"></span><span class="' + m.row.summary + '"><span class="' + m.shimmer.text + '">Fixture ' + esc(v.id) + '</span></span></span></span></div></div><div data-fixture-motion data-dsh-motion="fade" data-state="open"><div class="' + m.row.bodyWrap + '" data-body-wrap>' + block(v) + '</div></div></div></div></section>';
  const sheets = ['base', 'motion', 'corner_shape', 'design_platform', 'focus', 'scrollbar', 'gradient_shadow_text'].map(name => x.css['theme_' + name].text).concat([x.css.disclosure.text, x.css.card.text, x.css.diff.text, x.css.row.text]);
  const data = {models, maps: m, labels: Object.fromEntries(Object.entries(x.labels).map(([locale, values]) => [locale, Object.fromEntries(Object.entries(values).filter(([, value]) => typeof value === 'string'))])), dictionary: x.dictionary, artwork: x.artwork};
  const html = '<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src \'none\';style-src \'unsafe-inline\';script-src \'unsafe-inline\';connect-src \'none\';img-src data:"><title>Round119 installed DiffBlock</title>' + sheets.map((text, i) => '<style data-installed-css="' + i + '">' + text + '</style>').join('') + '<style>html,body{margin:0;min-height:100%}body{font-family:var(--dsw-font-family);font-size:14px;color:var(--dsw-alias-label-primary);background:var(--dsw-alias-bg-base)}#intro{position:absolute;left:40px;top:20px;font-size:12px;color:var(--dsw-alias-label-secondary)}h1{font-size:18px;font-weight:500;margin:0 0 5px}#intro p{margin:0}.scenario{position:absolute}.caption{font-size:12px;line-height:18px;color:var(--dsw-alias-label-secondary);margin:0 0 8px}</style></head><body data-platform="windows"><header id="intro"><h1>Installed DiffBlock · isolated Chrome DPR1</h1><p>Verbatim installed CSS; manual DOM, local handlers and memory clipboard. Source-only exact installed VM. No React, Shiki, Host, motion or hardware input.</p></header>' + models.map(scenario).join('') + '<script>\n' + inlineFixture.toString() + '\ninlineFixture(' + JSON.stringify(data).replaceAll('<', '\\u003c') + ');\n</script></body></html>';
  return {html, models};
}
function inlineFixture(data) {
  'use strict';
  const {models, maps: m, labels, dictionary, artwork} = data;
  const events = [], clipboard = [], feedback = [], folds = [], styles = [], styleIDs = new Map(); let clipboardOK = true;
  const own = id => document.querySelector('[data-own-card="' + id + '"]');
  const rect = element => { if (!element) return null; const b = element.getBoundingClientRect(); return {x: b.x, y: b.y, w: b.width, h: b.height}; };
  const keys = ['display', 'boxSizing', 'fontFamily', 'fontSize', 'lineHeight', 'fontWeight', 'color', 'backgroundColor', 'boxShadow', 'whiteSpace', 'overflowWrap', 'overflowX', 'overflowY', 'minHeight', 'minWidth', 'gridTemplateColumns', 'alignItems', 'alignSelf', 'justifyItems', 'width', 'paddingTop', 'paddingRight', 'paddingBottom', 'paddingLeft', 'marginTop', 'marginRight', 'marginBottom', 'marginLeft', 'borderTopWidth', 'borderRadius', 'columnGap', 'outlineStyle', 'outlineWidth', 'outlineColor', 'outlineOffset'];
  const style = element => element ? Object.fromEntries(keys.map(key => [key, getComputedStyle(element)[key]])) : null;
  const intern = element => { if (!element) return null; const value = style(element), serialized = JSON.stringify(value); if (!styleIDs.has(serialized)) { styleIDs.set(serialized, 'S' + (styles.length + 1)); styles.push({id: styleIDs.get(serialized), value}); } return styleIDs.get(serialized); };
  const scroll = e => e ? {x: e.scrollLeft, y: e.scrollTop, sw: e.scrollWidth, sh: e.scrollHeight, cw: e.clientWidth, ch: e.clientHeight} : null;
  const box = (e, withScroll = false) => e ? {box: rect(e), style: intern(e), ...(withScroll ? {scroll: scroll(e)} : {})} : null;
  const label = (locale, key, count) => dictionary[locale][key].replaceAll('{count}', String(count));
  const svg = key => { const e = document.createElementNS('http://www.w3.org/2000/svg', 'svg'); for (const [name, value] of Object.entries({'data-icon': key, width: '14', height: '14', viewBox: '0 0 16 16', fill: 'none', 'aria-hidden': 'true', 'stroke-width': '1'})) e.setAttribute(name, value); for (const s of artwork[key].shapes) { const shape = document.createElementNS(e.namespaceURI, s.tag); for (const [name, value] of Object.entries(s.attributes)) shape.setAttribute(name, value); e.append(shape); } return e; };
  const rowElement = (row, index) => { const e = document.createElement('div'); e.className = m.diff.line + ' ' + m.diff[row.kind]; e.dataset.diffLine = ''; e.dataset.kind = row.kind; e.dataset.index = String(index); e.textContent = row.text; return e; };
  for (const type of ['mousemove', 'pointerdown', 'pointerup', 'mousedown', 'mouseup', 'click', 'keydown', 'keyup', 'wheel']) document.addEventListener(type, event => events.push({type, trusted: event.isTrusted, id: event.target.closest('[data-own-card]')?.dataset.ownCard ?? null, key: event.key ?? null, target: event.target.closest('[data-wrap]') ? 'wrap' : event.target.closest('[data-copy]') ? 'copy' : event.target.closest('[data-fold]') ? 'fold' : event.target.closest('[data-diff-body]') ? 'body' : event.target.closest('[data-disclosure-row]') ? 'row' : null}), true);
  for (const value of models) {
    const root = own(value.id), block = root.querySelector('[data-diff]'); if (!block) continue;
    const body = root.querySelector('[data-diff-body]'), wrap = root.querySelector('[data-wrap]'), copy = root.querySelector('[data-copy]'), fold = root.querySelector('[data-fold]'), L = labels[value.locale];
    // Explicit LOCAL handlers corresponding to recorded installed state transitions.
    wrap.addEventListener('click', () => { const wrapped = block.dataset.codeWrap !== 'true'; block.dataset.codeWrap = String(wrapped); wrap.setAttribute('aria-pressed', String(wrapped)); wrap.dataset.fixtureTooltipLabel = L[wrapped ? 'unwrapLabel' : 'wrapLabel']; wrap.replaceChildren(svg(wrapped ? 'nowrap' : 'wrap')); });
    if (fold) {
      let expanded = false;
      fold.addEventListener('click', () => {
        expanded = !expanded; const hidden = value.rows.length - value.maxLines;
        fold.setAttribute('aria-expanded', String(expanded)); fold.setAttribute('aria-label', expanded ? L.collapseAria : label(value.locale, 'diff.expandAria', hidden)); fold.textContent = expanded ? L.collapse : label(value.locale, 'diff.expandRest', hidden);
        const indices = expanded ? value.rows.map((_, i) => i) : [0, 1, 2, 3, 4];
        body.replaceChildren(...indices.map(i => rowElement(value.rows[i], i)), fold, ...(expanded ? [] : value.rows.slice(-4).map((row, i) => rowElement(row, value.rows.length - 4 + i))));
        folds.push({id: value.id, expanded, indices: [...body.querySelectorAll('[data-diff-line]')].map(e => +e.dataset.index), at: performance.now()});
      });
    }
    let copied = false;
    copy.addEventListener('click', () => {
      if (copied) return;
      const ok = clipboardOK; clipboard.push({id: value.id, text: value.raw, ok, adapter: 'in-memory fixture only'});
      void Promise.resolve(ok).then(success => {
        if (!success) return; copied = true; copy.setAttribute('aria-label', L.copied); copy.dataset.fixtureTooltipLabel = L.copied; copy.replaceChildren(svg('check')); feedback.push({id: value.id, event: 'copied', at: performance.now()});
        window.setTimeout(() => { copied = false; copy.setAttribute('aria-label', L.copy); copy.dataset.fixtureTooltipLabel = L.copy; copy.replaceChildren(svg('copy')); feedback.push({id: value.id, event: 'reset', at: performance.now()}); }, 1000);
      });
    });
  }
  function quick(id) {
    const root = own(id), block = root.querySelector('[data-diff]'), body = root.querySelector('[data-diff-body]'), fold = root.querySelector('[data-fold]'), wrap = root.querySelector('[data-wrap]'), copy = root.querySelector('[data-copy]');
    const rows = [...root.querySelectorAll('[data-diff-line]')], focused = document.activeElement;
    return {exists: !!block, wrapped: block?.dataset.codeWrap, language: root.querySelector('[data-language]')?.textContent, rowExpanded: root.querySelector('[data-disclosure-row]').getAttribute('aria-expanded'), indices: rows.map(e => +e.dataset.index), kinds: rows.map(e => e.dataset.kind), texts: rows.map(e => e.textContent), lineWidths: rows.map(e => e.getBoundingClientRect().width), lineHeights: rows.map(e => e.getBoundingClientRect().height), bodyHeight: body?.getBoundingClientRect().height, blockHeight: block?.getBoundingClientRect().height, bodyScroll: scroll(body), fold: fold ? {text: fold.textContent, label: fold.getAttribute('aria-label'), expanded: fold.getAttribute('aria-expanded'), position: [...body.children].indexOf(fold)} : null, wrapPressed: wrap?.getAttribute('aria-pressed'), wrapLabel: wrap?.getAttribute('aria-label'), wrapTooltip: wrap?.dataset.fixtureTooltipLabel, wrapIcon: wrap?.querySelector('svg')?.dataset.icon, copyLabel: copy?.getAttribute('aria-label'), copyIcon: copy?.querySelector('svg')?.dataset.icon, focused: focused === wrap ? 'wrap' : focused === copy ? 'copy' : focused === fold ? 'fold' : focused === root.querySelector('[data-disclosure-row]') ? 'row' : null, focusVisible: focused?.matches(':focus-visible') ?? false};
  }
  function one(id) {
    const root = own(id), block = root.querySelector('[data-diff]'), header = root.querySelector('[data-code-block-banner]'), body = root.querySelector('[data-diff-body]'), fold = root.querySelector('[data-fold]');
    return {id, root: box(root), row: box(root.querySelector('[data-disclosure-row]')), bodyWrap: box(root.querySelector('[data-body-wrap]')), block: box(block), header: box(header), heading: box(root.querySelector('[data-heading]')), language: root.querySelector('[data-language]') ? {...box(root.querySelector('[data-language]')), text: root.querySelector('[data-language]').textContent} : null, actions: box(root.querySelector('[data-actions]')), buttons: [...(header?.querySelectorAll('button') ?? [])].map(e => ({kind: e.hasAttribute('data-wrap') ? 'wrap' : 'copy', ...box(e), tag: e.tagName, type: e.type, label: e.getAttribute('aria-label'), pressed: e.getAttribute('aria-pressed'), text: e.textContent, icon: {key: e.querySelector('svg')?.dataset.icon, box: rect(e.querySelector('svg'))}})), body: box(body, true), rows: [...root.querySelectorAll('[data-diff-line]')].map(e => ({index: +e.dataset.index, kind: e.dataset.kind, text: e.textContent, ...box(e), before: getComputedStyle(e, '::before').content})), fold: fold ? {...box(fold), text: fold.textContent, label: fold.getAttribute('aria-label'), expanded: fold.getAttribute('aria-expanded'), position: [...body.children].indexOf(fold)} : null, hasBodyScrollAncestor: !!block?.closest('.' + m.row.bodyScroll)};
  }
  function metrics(id) { const root = own(id); return Object.fromEntries(['[data-diff]', '[data-disclosure-row]', '[data-body-wrap]', '[data-code-block-banner]', '[data-language]', '[data-actions]', '[data-wrap]', '[data-copy]', '[data-diff-body]', '[data-diff-line]', '[data-kind="add"]', '[data-kind="del"]', '[data-kind="context"]', '[data-kind="path"]', '[data-kind="gap"]', '[data-fold]'].map(selector => [selector, style(root.querySelector(selector))])); }
  function early() {
    const root = own('short'), body = root.querySelector('[data-diff-body]'), row = root.querySelector('[data-diff-line]'), block = root.querySelector('[data-diff]'), header = root.querySelector('[data-code-block-banner]');
    const pick = (e, names) => Object.fromEntries(names.map(k => [k, getComputedStyle(e)[k]]));
    return {theme: document.body.hasAttribute('data-ds-dark-theme') ? 'dark' : 'light', dpr: devicePixelRatio, card: {...pick(block, ['fontFamily', 'fontSize', 'lineHeight', 'backgroundColor', 'borderRadius']), width: rect(block).w, height: rect(block).h}, header: {...pick(header, ['fontSize', 'lineHeight', 'paddingTop', 'paddingRight', 'paddingBottom', 'paddingLeft']), height: rect(header).h}, body: {...pick(body, ['display', 'gridTemplateColumns', 'paddingTop', 'paddingRight', 'paddingBottom', 'paddingLeft', 'overflowX']), width: rect(body).w, height: rect(body).h}, row: {...pick(row, ['minHeight', 'paddingLeft', 'paddingRight', 'whiteSpace']), width: rect(row).w, height: rect(row).h}, add: pick(root.querySelector('[data-kind="add"]'), ['color', 'backgroundColor', 'boxShadow']), del: pick(root.querySelector('[data-kind="del"]'), ['color', 'backgroundColor', 'boxShadow']), context: pick(root.querySelector('[data-kind="context"]'), ['color']), path: pick(root.querySelector('[data-kind="path"]'), ['color', 'fontWeight'])};
  }
  window.harness = {ready: true, quick, one, metrics, early, styles: () => styles, events: () => events, clipboard: () => clipboard, feedback: () => feedback, folds: () => folds, setClipboardOK: value => { clipboardOK = value; }, setTheme: dark => document.body.toggleAttribute('data-ds-dark-theme', dark), focus: (id, selector) => own(id).querySelector(selector).focus({preventScroll: true}), box: (id, selector) => rect(own(id).querySelector(selector))};
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
async function theme(dark) { await evaluate('window.harness.setTheme(' + dark + ')'); await frames(); }
async function acceptance(models) {
  await move(5, 5); E.browser.viewport = await evaluate('({dpr:devicePixelRatio,innerWidth,innerHeight})');
  E.fixtureData = models.map(v => ({id: v.id, totalRows: v.rows.length, maxLines: v.maxLines, locale: v.locale, language: v.language, raw: digestText(v.raw)}));
  const final = {};
  await runCase('early-light-dark-installed-metrics', async check => {
    const light = await evaluate('window.harness.early()'); await capture('light'); await theme(true); const dark = await evaluate('window.harness.early()'); await capture('dark');
    E.measurements.early = {light, dark}; console.log('EARLY_METRICS ' + JSON.stringify(E.measurements.early));
    check('DPR1 both themes and code11/19 header42', [light, dark].every(v => v.dpr === 1 && v.card.fontSize === '11px' && v.card.lineHeight === '19px' && near(v.header.height, 42)));
    check('body grid stretch width616 pad6 0 20 rows pad22 height19', [light, dark].every(v => v.body.display === 'grid' && v.body.gridTemplateColumns === '616px' && v.body.paddingTop === '6px' && v.body.paddingRight === '0px' && v.body.paddingBottom === '20px' && v.body.paddingLeft === '0px' && v.row.paddingLeft === '22px' && v.row.paddingRight === '22px' && near(v.row.width, v.body.width) && near(v.row.height, 19)));
    check('light add22c55e del ec1313 alpha20/255 and inset3px', light.add.color === 'rgb(34, 197, 94)' && light.del.color === 'rgb(236, 19, 19)' && light.add.backgroundColor === 'rgba(34, 197, 94, 0.08)' && light.del.backgroundColor === 'rgba(236, 19, 19, 0.08)' && [light.add, light.del].every(v => v.boxShadow === v.color + ' 3px 0px 0px 0px inset'), light);
    check('dark add22c55e del f25a5a alpha31/255 inset3px', dark.add.color === 'rgb(34, 197, 94)' && dark.del.color === 'rgb(242, 90, 90)' && dark.add.backgroundColor === 'rgba(34, 197, 94, 0.12)' && dark.del.backgroundColor === 'rgba(242, 90, 90, 0.12)' && [dark.add, dark.del].every(v => v.boxShadow === v.color + ' 3px 0px 0px 0px inset'), dark);
    check('card light f9fafb dark1b1b1c context secondary and path500', light.card.backgroundColor === 'rgb(249, 250, 251)' && dark.card.backgroundColor === 'rgb(27, 27, 28)' && light.context.color === 'rgb(97, 102, 107)' && dark.context.color === 'rgb(207, 211, 214)' && light.path.fontWeight === '500' && dark.path.fontWeight === '500');
    await theme(false);
  });
  await runCase('exact-source-vm-jsdiff-model-language-labels', async check => {
    check('all exact installed source diff cases', source.sourceOnly.diff.table.length >= 22 && source.sourceOnly.diff.table.every(v => v.pass));
    check('all emitted and bundled model table rows agree', source.sourceOnly.model.table.length >= 53 && source.sourceOnly.model.table.every(v => v.pass && v.emittedMatchesBundled), {count: source.sourceOnly.model.table.length});
    check('model settled edit metadata strict write fallback and legacy Generic', ['settled-edit-missing', 'settled-edit-empty-diffs', 'settled-str-create-generic'].every(id => source.sourceOnly.model.table.find(v => v.id === id).result === null) && source.sourceOnly.model.table.find(v => v.id === 'settled-write-fallback-null').result.card.diffs[0].newText === 'new');
    check('line endings ambiguous LCS gaps cap256 large Unicode', ['CRLF-versus-LF', 'ambiguous-LCS', 'distant-context-gap', 'exact256-keeps-shared-context', 'over256-full-fragment-fallback', 'large-unicode-full-fragments'].every(id => source.sourceOnly.diff.table.find(v => v.id === id).pass));
    check('language emitted equals bundled supports map', source.sourceOnly.language.table.every(v => v.pass));
    check('English and Chinese exact installed labels', source.dictionary.en['diff.expandAria'] === 'Expand {count} more diff lines' && source.dictionary.en['diff.collapseAria'] === 'Collapse diff' && source.dictionary.zh['diff.expandAria'] === '展开其余 {count} 行差异' && source.dictionary.zh.copy === '复制');
    check('caller9 primitive16 direct body no260cap', source.sourceOnly.caller.maxLines === 9 && source.sourceOnly.caller.primitiveDefault === 16 && source.sourceOnly.caller.containment.includes('no bodyScroll260'));
  });
  await runCase('short-toolbar-body-grid-prefix-metrics', async check => {
    const s = await one('short'), k = await metrics('short'); final.short = s;
    check('root620 row24 card616 margin4 0 4 4', near(s.root.box.w, 620) && near(s.row.box.h, 24) && near(s.block.box.w, 616) && k['[data-diff]'].marginTop === '4px' && k['[data-diff]'].marginRight === '0px' && k['[data-diff]'].marginBottom === '4px' && k['[data-diff]'].marginLeft === '4px');
    check('direct bodyWrap flex no outer bodyScroll', k['[data-body-wrap]'].display === 'flex' && !s.hasBodyScrollAncestor);
    check('header42 pad10 18 8 22 font11/18', near(s.header.box.h, 42) && k['[data-code-block-banner]'].paddingTop === '10px' && k['[data-code-block-banner]'].paddingRight === '18px' && k['[data-code-block-banner]'].paddingBottom === '8px' && k['[data-code-block-banner]'].paddingLeft === '22px' && k['[data-code-block-banner]'].fontSize === '11px' && k['[data-code-block-banner]'].lineHeight === '18px');
    check('toolbar only language typescript and two24px icon14 actions gap4', s.language.text === 'typescript' && s.buttons.length === 2 && s.buttons.every(v => v.tag === 'BUTTON' && v.type === 'button' && near(v.box.w, 24) && near(v.box.h, 24) && near(v.icon.box.w, 14) && near(v.icon.box.h, 14) && v.text === '') && k['[data-actions]'].columnGap === '4px');
    check('five19px rows fill content width body121 card163 root195', s.rows.length === 5 && s.rows.every(v => near(v.box.h, 19) && near(v.box.w, 616)) && near(s.body.box.h, 121) && near(s.block.box.h, 163) && near(s.root.box.h, 195));
    check('CSS before prefixes del-space add-space context-two-spaces', s.rows.find(v => v.kind === 'del').before === '"- "' && s.rows.find(v => v.kind === 'add').before === '"+ "' && s.rows.find(v => v.kind === 'context').before === '"  "' && s.rows.find(v => v.kind === 'path').before === 'none');
    check('row min19 pre borderbox padding22 and code card radius16', k['[data-diff-line]'].minHeight === '19px' && k['[data-diff-line]'].whiteSpace === 'pre' && k['[data-diff-line]'].boxSizing === 'border-box' && k['[data-diff-line]'].paddingLeft === '22px' && k['[data-diff]'].borderRadius === '16px');
  });
  await runCase('empty-array-path-only-blank-and-languages', async check => {
    for (const id of ['empty', 'unchanged', 'blank', 'common-language', 'mixed-language', 'no-language']) final[id] = await one(id);
    const empty = await quick('empty'), unchanged = await quick('unchanged'), blank = await quick('blank');
    check('empty diffs renders no block or toolbar', !empty.exists && final.empty.block === null && final.empty.buttons.length === 0);
    check('unchanged diff keeps one path row and copy body45 card87', unchanged.exists && unchanged.kinds.join() === 'path' && unchanged.copyLabel === 'Copy' && near(unchanged.bodyHeight, 45) && near(unchanged.blockHeight, 87));
    check('blank add still19px plus prefix body64 card106', blank.kinds.join() === 'path,add' && blank.texts[1] === '' && near(blank.lineHeights[1], 19) && near(blank.bodyHeight, 64) && near(blank.blockHeight, 106) && final.blank.rows[1].before === '"+ "');
    check('common ts+TSX canonical typescript; mixed and absent fallback', final['common-language'].language.text === 'typescript' && final['mixed-language'].language.text === 'Code block' && final['no-language'].language.text === 'Code block');
  });
  await runCase('threshold9-10-head5-tail4-chinese-AX', async check => {
    final.nine = await one('nine'); final.ten = await one('ten'); final.zh = await one('zh');
    const nine = await quick('nine'), ten = await quick('ten'), zh = await quick('zh'), k = await metrics('ten');
    check('exact9 rows no fold body197 card239', nine.indices.join() === '0,1,2,3,4,5,6,7,8' && nine.fold === null && near(nine.bodyHeight, 197) && near(nine.blockHeight, 239));
    check('10 total shows indices0..4 fold then6..9', ten.indices.join() === '0,1,2,3,4,6,7,8,9' && ten.fold.position === 5 && ten.fold.text === '… 1 more lines' && ten.fold.label === 'Expand 1 more diff lines');
    check('fold19px full grid width pad22 body216 card258', near(final.ten.fold.box.h, 19) && near(final.ten.fold.box.w, final.ten.body.box.w) && k['[data-fold]'].fontSize === '11px' && k['[data-fold]'].lineHeight === '19px' && k['[data-fold]'].paddingLeft === '22px' && near(ten.bodyHeight, 216) && near(ten.blockHeight, 258));
    const ax = await accessibility('[data-own-card="ten"] [data-fold]'), zhAX = await accessibility('[data-own-card="zh"] [data-wrap]'); E.measurements.initialAX = {fold: ax, zhWrap: zhAX};
    check('AX fold named native button expandedfalse', ax.some(v => v.role === 'button' && v.name === ten.fold.label && v.properties.some(p => p.name === 'expanded' && p.value === false)));
    check('Chinese labels installed exact fold/copy/wrap', zh.fold.label === '展开其余 1 行差异' && zh.fold.text === '… 其余 1 行' && zh.copyLabel === '复制' && zh.wrapLabel === '自动换行' && zhAX.some(v => v.name === '自动换行'));
  });
  await runCase('full-row-copy-prefix-path-gap-feedback-guard-refusal', async check => {
    const value = models.find(v => v.id === 'multi'), before = await quick('multi'), beforeAX = await accessibility('[data-own-card="multi"] [data-copy]');
    await click(await targetBox('multi', '[data-copy]')); const copied = await quick('multi'), afterAX = await accessibility('[data-own-card="multi"] [data-copy]');
    await click(await targetBox('multi', '[data-copy]')); const guarded = (await evaluate('window.harness.clipboard()')).length; await capture('copied');
    await waitFor(async () => (await quick('multi')).copyLabel === 'Copy', 'copy feedback reset', 2500);
    const feedback = await evaluate('window.harness.feedback()'), start = feedback.find(v => v.event === 'copied'), reset = feedback.find(v => v.event === 'reset');
    await evaluate('window.harness.setClipboardOK(false)'); await click(await targetBox('multi', '[data-copy]')); const refused = await quick('multi'); await evaluate('window.harness.setClipboardOK(true)');
    const records = await evaluate('window.harness.clipboard()');
    E.measurements.copy = {beforeAX, afterAX, copiedLabel: copied.copyLabel, guardedWriteCount: guarded, feedbackElapsedMS: reset.at - start.at, refusedLabel: refused.copyLabel, records: records.map(v => ({id: v.id, ok: v.ok, ...digestText(v.text), adapter: v.adapter}))};
    check('full exact source copy includes paths gap hidden rows and two-space context', records[0].text === value.raw && records[0].text.includes('src/a.ts\n    first\n- old\n+ new\n  end\n⋯\n- old fragment\n+ new fragment\nsrc/b.ts') && before.indices.length < value.rows.length);
    check('copy omits fold labels and preserves plus/minus space prefixes', !records[0].text.includes('more lines') && records[0].text.includes('\n- remove\n+ added\n  tail'));
    check('Copy to Copied/check AX feedback and duplicate guard', copied.copyLabel === 'Copied' && copied.copyIcon === 'check' && guarded === 1 && beforeAX.some(v => v.name === 'Copy') && afterAX.some(v => v.name === 'Copied'));
    check('feedback1000ms reset and refused adapter no success state', reset.at - start.at >= 995 && reset.at - start.at < 1600 && records.length === 2 && !records[1].ok && refused.copyLabel === 'Copy' && refused.copyIcon === 'copy' && feedback.filter(v => v.event === 'copied').length === 1, {elapsedMS: reset.at - start.at});
    check('memory only and row/fold state retained', records.every(v => v.adapter === 'in-memory fixture only') && copied.fold.expanded === 'false' && refused.rowExpanded === 'true');
  });
  await runCase('multiple-file-fold-expand-collapse-path-gap', async check => {
    const value = models.find(v => v.id === 'multi'), before = await quick('multi'); await click(await targetBox('multi', '[data-fold]')); const expanded = await quick('multi'); await capture('expanded');
    await click(await targetBox('multi', '[data-fold]')); const collapsed = await quick('multi');
    const fold = await targetBox('multi', '[data-fold]'); await move(fold.x + fold.w / 2, fold.y + fold.h / 2); final.multi = await one('multi'); const k = await metrics('multi');
    E.measurements.fold = {totalRows: value.rows.length, hidden: value.rows.length - 9, expandedIndices: expanded.indices, expandedKinds: expanded.kinds, expandedHeight: expanded.bodyHeight, expandedControlPosition: expanded.fold.position, expandedLabel: expanded.fold.label, collapsedIndices: collapsed.indices, collapsedHeight: collapsed.bodyHeight, transitions: await evaluate('window.harness.folds()')};
    check('flat row cap counts paths and gaps with5head4tail', before.indices.join() === '0,1,2,3,4,' + value.rows.slice(-4).map((_, i) => value.rows.length - 4 + i).join() && before.fold.position === 5);
    check('expand full rows exact kinds/text paths gap and collapse afterward', expanded.indices.length === value.rows.length && expanded.texts.join('\n') === value.rows.map(v => v.text).join('\n') && expanded.kinds.includes('gap') && expanded.kinds.filter(v => v === 'path').length === 2 && expanded.fold.position === value.rows.length && expanded.fold.label === 'Collapse diff' && expanded.fold.text === 'Collapse');
    check('expanded body uncapped above260 and delta hidden*19', expanded.bodyHeight > 260 && near(expanded.bodyHeight - collapsed.bodyHeight, (value.rows.length - 9) * 19) && !final.multi.hasBodyScrollAncestor);
    check('collapse restores original indices and row remains open', collapsed.indices.join() === before.indices.join() && collapsed.fold.expanded === 'false' && collapsed.rowExpanded === 'true');
    check('hover fold secondary color and full grid width', k['[data-fold]'].color === k['[data-kind="path"]'].color && near(final.multi.fold.box.w, final.multi.rows[0].box.w));
  });
  await runCase('horizontal-scroll-grid-background-fill-versus-wrap', async check => {
    const before = await quick('wrap'), beforeK = await metrics('wrap'); await wheel(await targetBox('wrap', '[data-diff-body]'), 0, 190); const scrolled = await quick('wrap');
    await click(await targetBox('wrap', '[data-wrap]')); const wrapped = await quick('wrap'), k = await metrics('wrap'); await capture('wrapped'); final.wrap = await one('wrap');
    E.measurements.wrap = {unwrapped: {bodyScroll: before.bodyScroll, rowWidths: before.lineWidths, lineHeights: before.lineHeights, afterWheelX: scrolled.bodyScroll.x, gridTemplateColumns: beforeK['[data-diff-body]'].gridTemplateColumns}, wrapped: {bodyScroll: wrapped.bodyScroll, rowWidths: wrapped.lineWidths, lineHeights: wrapped.lineHeights, gridTemplateColumns: k['[data-diff-body]'].gridTemplateColumns, pressed: wrapped.wrapPressed, icon: wrapped.wrapIcon, tooltip: wrapped.wrapTooltip}};
    check('unwrapped false pre rows19 body horizontal overflow trusted wheel', before.wrapped === 'false' && beforeK['[data-diff-line]'].whiteSpace === 'pre' && before.lineHeights.every(v => near(v, 19)) && before.bodyScroll.sw > before.bodyScroll.cw && scrolled.bodyScroll.x > 0);
    check('all row backgrounds stretch to content grid width beyond viewport', before.lineWidths.every(v => near(v, before.lineWidths[0])) && before.lineWidths[0] > before.bodyScroll.cw && near(before.lineWidths[0], Number.parseFloat(beforeK['[data-diff-body]'].gridTemplateColumns)), E.measurements.wrap.unwrapped);
    check('wrapped grid viewport width pre-wrap anywhere no horizontal overflow', wrapped.wrapped === 'true' && k['[data-diff-line]'].whiteSpace === 'pre-wrap' && k['[data-diff-line]'].overflowWrap === 'anywhere' && wrapped.bodyScroll.sw === wrapped.bodyScroll.cw && wrapped.bodyScroll.x === 0 && wrapped.lineWidths.every(v => near(v, wrapped.bodyScroll.cw)));
    check('wrapped add/del multiline19px increments backgrounds fill rows', wrapped.lineHeights.slice(1).every(v => v > 19 && near(v % 19, 0)) && final.wrap.rows.slice(1).every(v => near(v.box.w, wrapped.bodyScroll.cw)) && k['[data-kind="add"]'].boxShadow.includes('3px 0px 0px 0px inset'));
    check('stable aria-label Wrap lines pressedtrue nowrap glyph tooltip', before.wrapLabel === 'Wrap lines' && wrapped.wrapLabel === 'Wrap lines' && wrapped.wrapPressed === 'true' && wrapped.wrapIcon === 'nowrap' && wrapped.wrapTooltip === 'Do not wrap lines');
  });
  await runCase('Tab-focus-visible-and-named-toolbar', async check => {
    await focus('short', '[data-disclosure-row]'); await key('Tab', 'Tab', 9); const wrap = await quick('short'), k = await metrics('short'); await key('Tab', 'Tab', 9); const copy = await quick('short');
    const ax = await accessibility('[data-own-card="short"] [data-wrap]'); E.measurements.focus = {firstTab: wrap.focused, secondTab: copy.focused, focusVisible: wrap.focusVisible && copy.focusVisible, wrapAX: ax};
    check('Tab row then wrap then copy focus-visible', wrap.focused === 'wrap' && copy.focused === 'copy' && wrap.focusVisible && copy.focusVisible);
    check('installed focus ring1px offset2 solid', k['[data-wrap]'].outlineStyle === 'solid' && k['[data-wrap]'].outlineWidth === '1px' && k['[data-wrap]'].outlineOffset === '2px');
    check('wrap AX native button stable name and pressedfalse', ax.some(v => v.role === 'button' && v.name === 'Wrap lines' && v.properties.some(p => p.name === 'pressed' && p.value === 'false')));
  });
  await runCase('source-integrity-input-network-exclusive-attempt', async check => {
    const changed = []; for (const item of source.files) if (hash(await fs.readFile(item.path)) !== item.sha256) changed.push(item.id); E.sourceIntegrity = {allUnchanged: changed.length === 0, changedSourceIDs: changed};
    const events = await evaluate('window.harness.events()'); E.input = {count: events.length, trusted: events.every(v => v.trusted), byType: Object.fromEntries([...new Set(events.map(v => v.type))].map(type => [type, events.filter(v => v.type === type).length])), keydowns: events.filter(v => v.type === 'keydown').map(({id, key, target, trusted}) => ({id, key, target, trusted})), systemClipboardCalls: 0, nativeButtonActivationAssertions: 0}; E.network = {requests, exceptions, protocolErrors};
    check('all installed guide prior and runner source bytes unchanged', changed.length === 0, changed);
    check('only fixture file network no runtime/CDP errors', requests.every(v => !/^https?:|^wss?:/i.test(v.url)) && exceptions.length === 0 && protocolErrors.length === 0, E.network);
    check('all observed input trusted after ownership verification', E.input.trusted && ['mousemove', 'click', 'wheel', 'keydown'].every(type => E.input.byType[type] > 0) && E.browser.ownership.pass);
    check('one startup exclusive marker no Emulation or hardware input', spawnCount === 1 && E.input.keydowns.every(v => v.key === 'Tab') && Object.keys(methodCounts).every(v => !v.startsWith('Emulation.')) && JSON.parse(await fs.readFile(ATTEMPT, 'utf8')).rerunForbidden === true);
  });
  E.measurements.final = models.map(v => final[v.id]).filter(Boolean); E.computedStyles = await evaluate('window.harness.styles()');
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
    invariant(norm(PROFILE) === norm(BASE + '-profile') && norm(PROFILE).startsWith(norm(path.join(ROOT, '.devdata/r119-tool-diff-'))), 'Wrong profile removal target');
    await noLinks(PROFILE); invariant(norm(await fs.realpath(PROFILE)) === norm(PROFILE), 'Redirected cleanup profile');
    current.profileRemovalTarget = PROFILE; await fs.rm(PROFILE, {recursive: true, force: false}); current.profileRemoved = true;
  }
  current.pass = exitInfo?.code === 0 && current.portReleased && current.allObservedProcessesExited && current.profileRemoved;
}

const mdLink = (name, file) => '[' + name + '](<' + file.replaceAll('\\', '/') + '>)';
function report() {
  return '# Round119 installed DiffBlock reference\n\n' + E.status.toUpperCase() + ': ' + E.summary.passedCases + '/' + E.cases.length + ' cases; ' + E.summary.passedAssertions + '/' + (E.summary.passedAssertions + E.summary.failedAssertions) + ' assertions. ' + mdLink('Metrics', OUTPUT.json) + ' · ' + mdLink('Unique installed source', OUTPUT.source) + ' · ' + mdLink('Runner', SCRIPT) + '.\n\n' + E.scope.description + '\n\n' +
    '| Surface | Installed behavior and measured geometry |\n|---|---|\n' +
    '| Caller/card | ToolRow direct bodyWrap, no bodyScroll260 cap. Caller cap9; primitive16. Margin4 0 4 4; code11px/19px; radius16; border0. |\n' +
    '| Toolbar |42px; pad10 18 8 22;font11/18; controls24x24/icons14/actions gap4. Common extension canonical language else localized Code block; no title/status passed by DiffBlock. |\n' +
    '| Body/rows |Grid minmax(max-content,1fr);pad6 0 20;overflow-x:auto. All rows stretch to common content width. Row border-box,min-height19,pad0 22,pre; blank addition still19. Wrapped grid minmax(0,1fr),pre-wrap/anywhere. |\n' +
    '| Light colors |Add #22c55e bg#22c55e14;del#ec1313 bg#ec131314;card#f9fafb;context/path#61666b. |\n' +
    '| Dark colors |Add #22c55e bg#22c55e1f;del#f25a5a bg#f25a5a1f;card#1b1b1c;context/path#cfd3d6. Both themes add/del inset3px stripe same as foreground. |\n' +
    '| Folding |Flatten paths/gaps/content first;9 no fold;10 head5/fold1/tail4. At9 body197/card239;10 folded body216/card258. Expand all then collapse control; no outer260 cap. |\n' +
    '| Copy |Full source VM row text includes folded paths/gaps and every line. del=-space,add=+space,context=two spaces;path/gap verbatim;LF joined. Local memory success Copied/check1000ms, copied guard;refusal leaves Copy. |\n' +
    '| Empty/language |Empty diffs renders null. Unchanged and empty-create single diff retain path-only with Copy. Same ts/TSX label typescript;different/absent languages localized fallback. |\n\n' +
    'Source-only VM runs exact installed jsdiff/localHunks/buildRows/contentLines/copyText/totals against ' + source.sourceOnly.diff.table.length + ' cases including LF/CRLF/bareCR, ambiguous LCS, distant context gaps,256 edit boundary,full-fragment fallback and large CJK/astral/combining Unicode without truncation. Diff-card model runs ' + source.sourceOnly.model.table.length + ' rows independently through emitted functions and current ui-tool client.js fragment. It validates preparation/root/running/settled dispatch,write fallback,edit applied metadata,legacy Generic settlement,recorded-path remapping and malformed metadata; accepted current edge cases are preserved. ' + source.sourceOnly.language.table.length + ' emitted/bundled language cases and current en/zh label adapters also execute.\n\n' +
    'The reference is manual DOM with verbatim installed CSS and explicit local handlers. No React,Shiki,Host,file operations,OS clipboard,native button activation,motion,tooltips or application backend executes. Browser input is CDP only after exact PID/profile/port verification. Protected Electron20012/backend16008/LockApp12516/GUI3080 untouched. One Chrome startup;no accepting rerun.\n\n' +
    'Source integrity ' + E.sourceIntegrity?.allUnchanged + ';DPR ' + E.browser?.viewport?.dpr + ';cleanup ' + E.cleanup?.pass + ';combined source+metrics JSON ' + E.compact.combinedJSONBytes + ' bytes (<250000).\n\n' +
    Object.entries(E.screenshots).map(([name, value]) => mdLink(name, value.path)).join(' · ') + '\n\n' +
    E.cases.filter(v => v.status !== 'passed').map(v => 'Failure retained: ' + v.id + ': ' + (v.error ?? v.assertions.filter(a => !a.pass).map(a => a.name + ' ' + JSON.stringify(a.detail ?? null)).join('; '))).join('\n\n') + (E.errors.length ? '\n\nErrors: ' + E.errors.join('\n\n') : '') + '\n';
}
function textReport() { return ['Round119 installed DiffBlock: ' + E.status.toUpperCase(), 'Run ' + RUN_ID, 'Cases ' + E.summary.passedCases + '/' + E.cases.length + '; assertions ' + E.summary.passedAssertions + '/' + (E.summary.passedAssertions + E.summary.failedAssertions), ...E.cases.map(v => v.status.toUpperCase() + ' ' + v.id + ' ' + v.assertions.filter(a => a.pass).length + '/' + v.assertions.length), ...E.cases.flatMap(v => v.assertions.filter(a => !a.pass).map(a => 'FAIL ' + v.id + ': ' + a.name + ' ' + JSON.stringify(a.detail ?? null))), ...E.cases.filter(v => v.error).map(v => v.id + ': ' + v.error), 'Combined JSON bytes ' + E.compact.combinedJSONBytes, 'Cleanup ' + JSON.stringify(E.cleanup), 'Errors ' + JSON.stringify(E.errors), ''].join('\n'); }
async function main() {
  await scope(); const x = await extract(), built = fixture(x), inline = built.html.match(/<script>\n([\s\S]*)<\/script>/); invariant(inline, 'Missing inline fixture'); new vm.Script(inline[1], {filename: 'r119-tool-diff-inline.js'});
  if (MODE === '--prepare-only') {
    await writeOwned(OUTPUT.source, JSON.stringify(source) + '\n'); await writeOwned(FIXTURE, built.html);
    const prep = {schema: 'r119-tool-diff-preflight/v1', status: 'prepared-no-browser', runId: RUN_ID, sourceEvidence: OUTPUT.source, sourceEvidenceSHA256: hash(await fs.readFile(OUTPUT.source)), fixture: FIXTURE, fixtureSHA256: hash(built.html), inlineParsed: true, sourceDiffCases: source.sourceOnly.diff.table.length, sourceModelCases: source.sourceOnly.model.table.length, sourceLanguageCases: source.sourceOnly.language.table.length, chromeNotStarted: true};
    await writeOwned(PREP, JSON.stringify(prep, null, 2) + '\n'); console.log(JSON.stringify({...prep, path: PREP, sourceEvidenceBytes: (await fs.stat(OUTPUT.source)).size})); return;
  }
  const prep = JSON.parse(await fs.readFile(PREP, 'utf8')); invariant(prep.status === 'prepared-no-browser' && prep.runId === RUN_ID && norm(prep.sourceEvidence) === norm(OUTPUT.source), 'Wrong preparation');
  invariant(hash(await fs.readFile(OUTPUT.source)) === prep.sourceEvidenceSHA256 && JSON.stringify(source) === JSON.stringify(JSON.parse(await fs.readFile(OUTPUT.source, 'utf8'))), 'Installed source/runner changed after preparation');
  invariant(prep.fixtureSHA256 === hash(built.html) && hash(await fs.readFile(FIXTURE)) === prep.fixtureSHA256, 'Prepared fixture changed'); E.preflight = {path: PREP, inlineParsed: true, sourceEvidenceVerified: true, fixtureVerified: true};
  await writeOwned(ATTEMPT, JSON.stringify({round: 119, runId: RUN_ID, startedAt: E.startedAt, prepared: PREP, chromeStartupsAllowed: 1, rerunForbidden: true}, null, 2) + '\n');
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
