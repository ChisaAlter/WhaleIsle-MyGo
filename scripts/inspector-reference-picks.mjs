// Execute installed reverse matchers verbatim. VM substitutes only their loaded
// window/row stores; no browser, Host, network or model is used.
import {readFileSync, writeFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {stripTypeScriptTypes} from 'node:module';
import vm from 'node:vm';
import path from 'node:path';
import {matchChatNodeRow as sourceChat} from '../reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/src/client/views/chat-node/pick-match.ts';

const installed = 'C:/软件/Whale Isle/resources/vendor/deepseek-harness/packages/experimental/session-inspector/lib/client.js';
const bundle = readFileSync(installed, 'utf8');
const reference = 'reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/src/client/views/session-log/model.ts';
const source = readFileSync(reference, 'utf8');
function section(text, first, end) {
  const start = text.indexOf(first), stop = text.indexOf(end, start);
  if (start < 0 || stop < start) throw Error('Reverse matcher region missing: ' + first);
  return text.slice(start, stop);
}
const installedPick = section(bundle, 'pick(target) {\n\t\t\t\tconst window = this.source', '\n\t\t\t/**');
const sourcePick = section(source, 'pick(target: InspectorPickTarget)', '\n\n  /**');
const context = vm.createContext({});
vm.runInContext('globalThis.installedLog = (class {' + installedPick + '}).prototype.pick;\n' +
  section(bundle, 'function matchChatNodeRow(target, rows)', '\n\t\t//#endregion'), context);
vm.runInContext(stripTypeScriptTypes('globalThis.sourceLog = (class {' + sourcePick + '}).prototype.pick'), context);
const plain = value => JSON.parse(JSON.stringify(value ?? null));
function logPick(fn, c) {
  const records = new Map(c.rows.map(row => [row.key, {getSnapshot: () => ({value: row.value})}]));
  return plain(fn.call({source: {getSnapshot: () => ({entries: c.entries})}, accept: () => {},
    settledRoots: new Map(c.settledRoots ?? []), rows: {getSnapshot: () => c.rows}, records}, c.target));
}
const event = (seq, type, data = {}) => ({type: 'event', event: {seq, type, data}});
const live = (seq, id, turn, step) => ({type: 'transient', event: {seq, type: 'assistant/live-chunk', data: {attemptId: id, turn, step}}});
const child = (key, parent, part) => ({key, parent, value: {type: 'block-start', index: 9, blockType: part}});
const roots = [event(0, 'turn/start', {turn: 0}), event(1, 'step/start', {turn: 0, step: 0}),
  event(2, 'tool/call', {turn: 0, step: 0, callId: 'call'}), event(3, 'assistant/attempt', {turn: 0, step: 0}),
  event(4, 'assistant/message', {turn: 0, step: 0}), event(5, 'step/start', {turn: 0, step: 1}),
  event(6, 'assistant/message', {turn: 0, step: 1}), live(7, 'live', 0, 1), event(8, 'custom/future', {turn: 2, step: 0}),
  event(9, 'session/presentation', null)];
const rows = [child('event:4/chunk:0', 'event:4', 'text'), child('nested/reasoning', 'event:4/chunk:0', 'reasoning'),
  child('event:4/chunk:2', 'event:4', 'reasoning'), child('event:4/chunk:4', 'event:4', 'reasoning'),
  child('attempt:live/chunk:0', 'attempt:live', 'reasoning')];
const logCases = [];
function log(name, target, entries = roots, children = rows, settledRoots = []) {
  logCases.push({name, target, entries, rows: children, settledRoots});
}
log('call-before-global-anchor', {callId: 'call', anchorSeq: 4, turn: 2, step: 9});
log('anchor-before-last-assistant', {anchorSeq: 3, nodeKind: 'assistant-step', turn: 0, step: 0});
log('anchor-zero-without-turn', {anchorSeq: 0});
log('anchor-zero-overrides-position', {anchorSeq: 0, turn: 2, step: 0});
log('last-assistant-in-exact-step-zero', {nodeKind: 'assistant-step', turn: 0, step: 0});
log('last-assistant-in-whole-turn-is-live', {nodeKind: 'assistant-step', turn: 0});
log('live-reasoning', {nodeKind: 'assistant-step', turn: 0, step: 1, groupPart: 'reasoning'});
log('reasoning-first-direct-child', {anchorSeq: 4, groupPart: 'reasoning'});
log('response-is-root', {anchorSeq: 4, groupPart: 'response'});
log('missing-reasoning-keeps-root', {anchorSeq: 3, groupPart: 'reasoning'});
log('step-before-turn', {turn: 0, step: 0});
log('turn-before-first-same-step', {turn: 0, step: 8});
log('first-same-step-without-boundaries', {turn: 2, step: 0});
log('step-without-turn-does-not-place', {step: 0});
log('no-identities-does-not-pick-zero', {});
log('null-data-does-not-invent-placement', {turn: 9, step: 0});
log('missing-call-keeps-anchor', {callId: 'missing', anchorSeq: 4});
log('root-call-alone-does-not-pick', {rootCallId: 'call'});
log('call-does-not-use-stream-child', {callId: 'preparing'}, [live(2, 'live', 0, 0)], [{key: 'attempt:live/chunk:0', parent: 'attempt:live', value: {type: 'tool-call-delta', id: 'preparing'}}]);
log('settled-attempt-key-retained', {anchorSeq: 4, groupPart: 'reasoning'}, [event(4, 'assistant/attempt', {turn: 0, step: 0})], [child('attempt:stable/chunk:3', 'attempt:stable', 'reasoning')], [[4, 'attempt:stable']]);
log('settled-key-evicted-reloaded', {anchorSeq: 4}, [event(4, 'assistant/attempt', {turn: 0, step: 0})], []);
let seed = 110254;
const rnd = n => {seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0; return seed % n;};
for (let i = 0; i < 160; i++) {
  const target = {};
  if (rnd(2)) target.turn = rnd(4);
  if (rnd(2)) target.step = rnd(3);
  if (rnd(2)) target.anchorSeq = rnd(11);
  if (rnd(2)) target.nodeKind = 'assistant-step';
  if (rnd(2)) target.groupPart = rnd(2) ? 'reasoning' : 'response';
  if (rnd(4) === 0) target.callId = rnd(2) ? 'call' : 'absent';
  log('seeded-log-' + i, target, roots.filter(() => rnd(5) !== 0), rows);
}
for (const c of logCases) {
  c.expected = logPick(context.installedLog, c);
  if (logPick(context.sourceLog, c) !== c.expected) throw Error('Installed/source log pick mismatch: ' + c.name);
}
const chatRows = [['reason', {nodeKey: 'assistant', groupPart: 'reasoning', turn: 0, step: 0}],
  ['response', {nodeKey: 'assistant', groupPart: 'response', turn: 0, step: 0}], ['bare', {nodeKey: 'assistant', turn: 0, step: 0}],
  ['group', {groupKey: 'process'}], ['other-step', {nodeKey: 'other', turn: 0, step: 1}], ['other-turn', {nodeKey: 'last', turn: 1, step: 0}]];
const chatCases = [];
function chat(name, target, table = chatRows) {chatCases.push({name, target, rows: table});}
chat('exact-part', {nodeKey: 'assistant', groupPart: 'response'});
chat('bare-part-is-exact', {nodeKey: 'assistant'});
chat('unknown-part-first-node', {nodeKey: 'assistant', groupPart: 'future'});
chat('node-before-conflicting-group', {nodeKey: 'assistant', groupPart: 'reasoning', groupKey: 'process', turn: 1});
chat('missing-node-then-group', {nodeKey: 'missing', groupKey: 'process', turn: 0});
chat('missing-group-then-step', {groupKey: 'missing', turn: 0, step: 1});
chat('turn-zero-fallback', {turn: 0, step: 8});
chat('step-zero-fallback', {turn: 1, step: 0});
chat('no-turn-no-position', {step: 0});
chat('no-identity', {anchorSeq: 0, callId: 'call'});
for (let i = 0; i < 80; i++) {
  const target = {};
  if (rnd(2)) target.nodeKey = ['assistant', 'other', 'missing'][rnd(3)];
  if (rnd(2)) target.groupPart = ['response', 'reasoning', 'future'][rnd(3)];
  if (rnd(2)) target.groupKey = rnd(2) ? 'process' : 'missing';
  if (rnd(2)) target.turn = rnd(3);
  if (rnd(2)) target.step = rnd(3);
  chat('seeded-chat-' + i, target, chatRows.filter(() => rnd(5) !== 0));
}
for (const c of chatCases) {
  c.expected = plain(context.matchChatNodeRow(c.target, new Map(c.rows)));
  if (plain(sourceChat(c.target, new Map(c.rows))) !== c.expected) throw Error('Installed/source Chat pick mismatch: ' + c.name);
}
const output = path.resolve('internal/ui/frame/testdata/inspector-picks.json');
writeFileSync(output, JSON.stringify({logCases, chatCases}, null, 2) + '\n');
writeFileSync('docs/review-20261022/evidence/inspector-pass254-reference-picks.json', JSON.stringify({installed,
  installedSha256: createHash('sha256').update(bundle).digest('hex'), logCases: logCases.length, chatCases: chatCases.length,
  installedAndSourceAgree: true, fixture: output, fixtureSha256: createHash('sha256').update(readFileSync(output)).digest('hex'),
  method: 'Verbatim installed log method/Chat matcher in Node VM; independent TypeScript source method/matcher, loaded entries/row stores substituted. No browser DOM claim.'}, null, 2) + '\n');
console.log(`PASS ${logCases.length} Log picks and ${chatCases.length} Chat picks: installed functions equal source snapshot`);
