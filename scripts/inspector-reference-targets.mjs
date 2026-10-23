// Execute the installed Inspector's unmodified pure matching functions and the
// read-only source snapshot. No browser, Host, model or installed file is changed.
import {readFileSync, writeFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import path from 'node:path';
import vm from 'node:vm';
import {resolveChatRevealTargets as sourceResolve} from '../reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/src/client/views/chat-node/reveal-target.ts';
import {sessionLogChatTarget as sourceLog} from '../reference/electron/vendor/deepseek-harness/packages/experimental/session-inspector/src/client/views/session-log/chat-target.ts';

const installed = 'C:/软件/Whale Isle/resources/vendor/deepseek-harness/packages/experimental/session-inspector/lib/client.js';
const bundle = readFileSync(installed, 'utf8');
function section(first, end) {
  const start = bundle.indexOf(first), stop = bundle.indexOf(end, start);
  if (start < 0 || stop < start) throw Error('Installed Inspector function region missing');
  return bundle.slice(start, stop);
}
const context = vm.createContext({});
vm.runInContext(section('function partOf(chunk)', '//#region lib/types/client/views/raw-json.js') + '\n' +
  section('function nearest(nodes, seq)', '//#region lib/types/client/views/chat-node/picker.js'), context);
const plain = value => JSON.parse(JSON.stringify(value));
const node = (id, seq, turn, step, kind = 'user', visible = true, rootCallId) => ({id, seq, turn, step, kind, visible, rootCallId});
const browserNodes = nodes => nodes.map(n => ({key: n.id, anchorSeq: n.seq, kind: n.kind, visibility: n.visible ? 'visible' : 'hidden',
  data: n.rootCallId === undefined ? {} : {root: {callId: n.rootCallId}},
  location: n.turn === undefined ? {kind: 'session'} : n.step === undefined ? {kind: 'turn', turn: {turn: n.turn}}
    : {kind: 'step', turn: {turn: n.turn}, step: {step: n.step}}}));
const cases = [
  {name: 'earlier-exact-independent-of-order', target: {anchorSeq: 10}, nodes: [node('later', 30, 1, 1), node('earlier', 10, 1, 1)]},
  {name: 'root-tool-before-exact-in-other-turn', target: {anchorSeq: 90, callId: 'child', rootCallId: 'root', turn: 1, step: 2}, nodes: [node('exact', 90, 1, 2), node('tool', 10, 7, 3, 'tool-call', true, 'root')]},
  {name: 'exact-before-assistant-kind', target: {anchorSeq: 16, nodeKind: 'assistant-step', turn: 1, step: 2}, nodes: [node('assistant', 20, 1, 2, 'assistant-step'), node('exact', 16, 1, 2)]},
  {name: 'assistant-kind-before-nearest', target: {anchorSeq: 15, nodeKind: 'assistant-step', turn: 1, step: 1, groupPart: 'reasoning'}, nodes: [node('hidden', 15, 1, 1, 'assistant-step', false), node('nearby', 16, 1, 1), node('assistant', 20, 1, 1, 'assistant-step')]},
  {name: 'first-kind-not-nearest-kind', target: {anchorSeq: 19, nodeKind: 'assistant-step', turn: 1, step: 1}, nodes: [node('first', 2, 1, 1, 'assistant-step'), node('near', 20, 1, 1, 'assistant-step')]},
  {name: 'same-step-before-nearer-other-turn', target: {anchorSeq: 100, turn: 2, step: 3}, nodes: [node('other-turn', 100, 1, 3), node('same-turn', 99, 2, 4), node('same-step', 40, 2, 3)]},
  {name: 'missing-step-falls-within-turn', target: {anchorSeq: 8, turn: 1, step: 2}, nodes: [node('session', 8), node('same-turn', 7, 1, 1)]},
  {name: 'missing-turn-retains-primary-and-ranks-neighbors', target: {anchorSeq: 8, turn: 9, step: 2}, nodes: [node('near', 7, 1, 1)]},
  {name: 'distance-ties-retain-input-order', target: {anchorSeq: 10}, nodes: [node('first', 8), node('second', 12), node('far', 1)]},
  {name: 'hidden-explicit-enriches-missing-coordinates', target: {nodeKey: 'hidden', groupPart: 'reasoning'}, nodes: [node('other-turn', 100, 3, 3), node('same-turn', 60, 2, 4), node('same-step-after', 101, 2, 3), node('same-step-before', 99, 2, 3), node('hidden', 100, 2, 3, 'assistant-step', false)]},
  {name: 'request-coordinates-override-explicit', target: {nodeKey: 'hidden', anchorSeq: 200, turn: 4, step: 5}, nodes: [node('hidden', 100, 2, 3, 'assistant-step', false), node('requested', 180, 4, 5), node('actual', 100, 2, 3)]},
  {name: 'explicit-session-node-has-no-invented-location', target: {nodeKey: 'session'}, nodes: [node('session', 12)]},
  {name: 'unknown-explicit-without-anchor-has-no-neighbors', target: {nodeKey: 'missing'}, nodes: [node('near', 7, 1, 1)]},
  {name: 'same-node-part-and-bare-fallback-remain-distinct', target: {nodeKey: 'assistant', groupPart: 'response'}, nodes: [node('assistant', 10, 1, 1, 'assistant-step'), node('neighbor', 11, 1, 1)]},
  {name: 'identical-enriched-primary-is-deduplicated', target: {nodeKey: 'explicit'}, nodes: [node('explicit', 10, 1, 1), node('neighbor', 11, 1, 1)]},
  {name: 'zero-anchor-turn-and-step-are-present', target: {anchorSeq: 0, turn: 0, step: 0}, nodes: [node('other', 0, 1, 0), node('zero', 0, 0, 0)]},
];
let seed = 109;
const random = n => {seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0; return seed % n;};
for (let index = 0; index < 160; index++) {
  const nodes = Array.from({length: 2 + random(10)}, (_, i) => {
    const location = random(4), kind = ['user', 'assistant-step', 'tool-call'][random(3)];
    return node('n' + i, random(25), location === 0 ? undefined : random(3), location < 2 ? undefined : random(3), kind, random(5) !== 0, kind === 'tool-call' ? 'call-' + i : undefined);
  });
  const selected = nodes[random(nodes.length)], target = {};
  if (random(4) !== 0) target.anchorSeq = random(25);
  if (random(3) === 0) target.nodeKey = selected.id;
  if (random(2) === 0) target.turn = random(3);
  if (random(2) === 0) target.step = random(3);
  if (random(3) === 0) target.nodeKind = 'assistant-step';
  if (random(3) === 0) target.groupPart = random(2) ? 'reasoning' : 'response';
  if (random(4) === 0) {target.callId = 'nested'; target.rootCallId = selected.rootCallId ?? 'absent';}
  cases.push({name: 'seeded-' + index, target, nodes});
}
for (const item of cases) {
  const nodes = browserNodes(item.nodes);
  item.expected = plain(context.resolveChatRevealTargets(item.target, nodes));
  const fromSource = plain(sourceResolve(item.target, nodes));
  if (JSON.stringify(fromSource) !== JSON.stringify(item.expected)) throw Error('Installed/source mismatch: ' + item.name);
}
const event = (type, data, seq = 10) => ({type, seq, data});
const logs = [
  {name: 'user-no-location', event: event('user/message', {id: 'u'})},
  {name: 'tool-call', event: event('tool/call', {turn: 1, step: 0, callId: 'root'})},
  {name: 'tool-result', event: event('tool/result', {turn: 1, step: 0, message: {source: {callId: 'root'}}}, 20)},
  {name: 'nested-dispatch', event: event('tool/ptc-dispatch-start', {subCallId: 'child', rootCallId: 'root'})},
  {name: 'zero-coordinates', event: event('assistant/attempt', {turn: 0, step: 0}, 0)},
  ...[
    {type: 'reasoning-delta', index: 0, text: 'thinking'}, {type: 'text-delta', index: 1, text: 'answer'},
    {type: 'block-start', index: 0, blockType: 'reasoning'}, {type: 'block-start', index: 1, blockType: 'text'},
    {type: 'block-end', index: 0, block: {type: 'reasoning', text: 'done'}}, {type: 'block-end', index: 1, block: {type: 'text', text: 'done'}},
    {type: 'tool-call-delta', index: 2, id: 'stream-call'}, {type: 'block-end', index: 2, block: {type: 'tool-call', id: 'stream-call'}},
    {type: 'finish', reason: {kind: 'stop'}},
  ].map((chunk, index) => ({name: 'assistant-chunk-' + index, event: event('assistant/attempt', {turn: 1, step: 2}, 15), chunk})),
];
for (const item of logs) {
  item.expected = plain(context.sessionLogChatTarget(item));
  if (JSON.stringify(plain(sourceLog(item))) !== JSON.stringify(item.expected)) throw Error('Installed/source log mismatch: ' + item.name);
}
const payload = {cases, logs};
const output = path.resolve('internal/ui/chat/testdata/inspector-targets.json');
writeFileSync(output, JSON.stringify(payload, null, 2) + '\n');
writeFileSync('docs/review-20261022/evidence/inspector-pass253-reference-targets.json', JSON.stringify({
  installed, installedSha256: createHash('sha256').update(bundle).digest('hex'),
  modelCases: cases.length, logCases: logs.length, installedAndSourceAgree: true,
  fixture: output, fixtureSha256: createHash('sha256').update(readFileSync(output)).digest('hex'),
  method: 'Evaluate verbatim pure-function regions from installed client.js in Node vm; independently import read-only TypeScript source. No DOM/browser claim.',
}, null, 2) + '\n');
console.log(`PASS ${cases.length} model cases and ${logs.length} log-coordinate cases: installed functions equal source snapshot`);
