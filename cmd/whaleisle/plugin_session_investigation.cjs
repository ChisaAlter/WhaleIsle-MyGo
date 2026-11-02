'use strict';
// Isolated contract probe of the production session.js with a mock renderer.
// It never opens a real browser, Host, profile, or user session.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const listeners = new Map();
const entries = new Map([['main.conversation', [{options: {}}]]]);
let activeSession = '';
let headerNodes = [];
let composer = null;
let chatNode = null;
const navigation = [];
const bindings = new Map(['fixture-a', 'fixture-b'].map(id => [id, {
  session: {getSnapshot: () => ({openState: 'open'}), subscribe: () => () => {}}
}]));
const roots = new Map([...bindings.keys()].map(id => [id, {
  getAttribute: name => name === 'data-conversation-session' ? id : null,
  querySelectorAll: selector => selector === '[data-slot="conversation.chat.node"]' && chatNode ? [chatNode] : [],
  querySelector: selector => selector === '[data-composer-seat]' ? composer : null,
  contains: node => node === composer || node === chatNode
}]));
const header = {
  querySelectorAll: selector => selector === '[data-slot="conversation.session.header.actions"]' ? headerNodes : [],
  contains: node => headerNodes.includes(node)
};
function node(text, hasControl = true) {
  const attrs = new Map();
  return {
    isConnected: true, textContent: text, hidden: false, scrollHeight: 50,
    querySelector: selector => selector === '[data-slot-error]' ? null : (hasControl ? {} : null),
    querySelectorAll: () => [], closest: () => null,
    setAttribute: (key, value) => attrs.set(key, value),
    removeAttribute: key => attrs.delete(key),
    hasAttribute: key => attrs.has(key)
  };
}
const document = {
  createElement: () => ({textContent: '', remove() {}}),
  head: {appendChild() {}}, documentElement: {},
  querySelectorAll: selector => selector === '[data-conversation-content][data-conversation-session]' && activeSession ? [roots.get(activeSession)] : [],
  querySelector: selector => selector === '[data-slot="main.conversation"]' ? header : null
};
const windows = new Map();
const windowStub = {
  addEventListener: (event, fn) => windows.set(event, fn),
  removeEventListener: event => windows.delete(event)
};
class WebSocketStub {
  static OPEN = 1;
  constructor() { this.readyState = 1; this.events = new Map(); this.messages = []; WebSocketStub.instance = this; }
  addEventListener(event, fn) { this.events.set(event, fn); }
  send(value) { this.messages.push(JSON.parse(value)); }
  close() { this.readyState = 3; }
  dispatch(event, value) { this.events.get(event)?.(value); }
}
class MutationObserverStub { constructor(fn) { this.fn = fn; MutationObserverStub.instance = this; } observe() {} disconnect() {} }
const emptySubscribe = () => () => {};
const ctx = {
  effect(fn) { this.dispose = fn(); },
  slots: {entriesOfSlot: slot => entries.get(slot) || []},
  sessions: {list: {getSnapshot: () => ({byId: {'fixture-a': {}, 'fixture-b': {}}}), subscribe: emptySubscribe}, binding: id => bindings.get(id)},
  layout: {panelInfo: {getSnapshot: () => ({activePanelId: null}), subscribe: emptySubscribe}},
  locale: {getSnapshot: () => ({active: 'en'}), subscribe: emptySubscribe},
  get: name => name === 'uiWorkspace' ? {openSession(id) { activeSession = id; navigation.push(id); }} : undefined,
  on(event, fn) { listeners.set(event, fn); return () => listeners.delete(event); }
};
const source = fs.readFileSync(path.join(__dirname, '../../internal/desktop/pluginui/session.js'), 'utf8');
const sandbox = {URL, URLSearchParams, location: {href: 'http://localhost/?desktopNativeSession=1', search: '?desktopNativeSession=1'}, document, window: windowStub, WebSocket: WebSocketStub, MutationObserver: MutationObserverStub, queueMicrotask};
vm.runInNewContext(source + '\nwhaleBindSessionUI(ctx);', {...sandbox, ctx}, {filename: 'session.js'});
const ws = WebSocketStub.instance;
const settle = () => new Promise(resolve => setImmediate(resolve));
const select = subject => ws.dispatch('message', {data: JSON.stringify({type: 'select', subject})});
const snapshot = () => ws.messages.at(-1);
const notify = () => listeners.get('slots/changed')();
const a = {kind: 'session', sessionId: 'fixture-a', epoch: '1:1'};
const b = {kind: 'session', sessionId: 'fixture-b', epoch: '1:2'};
const entry = {options: {id: 'fixture-action', label: 'Fixture'}, registrant: 'fixture'};

(async () => {
  ws.dispatch('open');
  await settle();
  select(a);
  await settle();
  assert.equal(snapshot().subject.sessionId, a.sessionId);
  assert.equal(snapshot().outlets.length, 0, 'no registered contributions should produce no outlets');
  assert.deepEqual(navigation, ['fixture-a'], 'discovery navigates the renderer even when nothing is registered');

  entries.set('conversation.session.header.actions', [entry]);
  headerNodes = [node('Fixture action')];
  notify();
  await settle();
  let state = snapshot();
  assert.equal(state.entries[0].id, 'fixture-action');
  assert.equal(state.outlets.length, 1);
  assert.equal(state.outlets[0].available, true, 'registered and mounted contribution must be discoverable');
  const oldRevision = state.revision;
  const oldHandle = state.outlets[0].handle;
  ws.dispatch('message', {data: JSON.stringify({type: 'sessionSurface', revision: oldRevision, subject: a, handle: oldHandle})});
  await settle();
  assert.equal(snapshot().surface, 'conversation.session.header.actions');

  // Withdrawal while the outlet is being shown must revoke that surface, even
  // when the old DOM node has not yet been removed by the renderer.
  entries.delete('conversation.session.header.actions');
  notify();
  await settle();
  assert.equal(snapshot().outlets.length, 0);
  assert.equal(snapshot().surface, '');
  assert.equal(headerNodes[0].hasAttribute('data-whale-session-surface'), false);
  ws.dispatch('message', {data: JSON.stringify({type: 'sessionSurface', revision: oldRevision, subject: a, handle: oldHandle})});
  await settle();
  assert.equal(snapshot().surface, '', 'withdrawn old handle cannot reopen its surface');
  entries.set('conversation.session.header.actions', [entry]);
  notify();
  await settle();
  assert.equal(snapshot().outlets[0].available, true, 'reloading a contribution restores its outlet');

  headerNodes = [];
  select(b);
  await settle();
  state = snapshot();
  assert.equal(state.subject.sessionId, b.sessionId);
  assert.equal(state.surface, '', 'switch must clear the previous surface');
  assert.equal(state.outlets.length, 1);
  assert.equal(state.outlets[0].available, false, 'a registered but unmounted contribution is not actionable');
  ws.dispatch('message', {data: JSON.stringify({type: 'sessionSurface', revision: oldRevision, subject: a, handle: oldHandle})});
  await settle();
  assert.equal(snapshot().surface, '', 'an earlier session cannot reopen its surface');

  entries.delete('conversation.session.header.actions');
  notify();
  await settle();
  assert.equal(snapshot().outlets.length, 0, 'withdrawal clears available and unavailable outlets');

  // The chain fallback is ordinary composer UI: declaration alone does not
  // establish that this chain entry was elected for the current session.
  entries.set('conversation.composer', [{options: {label: 'Declining plugin'}, registrant: 'fixture'}]);
  composer = node('Ordinary fallback composer');
  notify();
  await settle();
  state = snapshot();
  assert.equal(state.outlets.length, 1);
  assert.equal(state.outlets[0].available, true, 'current collector counts ordinary fallback as an outlet');
  assert.equal(state.outlets[0].slot, 'conversation.composer');

  // A keyed chat wrapper can be dispatching a different kind than the only
  // registered plugin key. The wrapper does not identify its selected entry.
  entries.delete('conversation.composer');
  entries.set('conversation.chat.node', [{options: {key: 'plugin-message', label: 'Plugin message'}, registrant: 'fixture'}]);
  chatNode = node('Other message kind fallback');
  notify();
  await settle();
  state = snapshot();
  assert.equal(state.entries[0].key, 'plugin-message');
  assert.equal(state.outlets[0].slot, 'conversation.chat.node');
  assert.equal(state.outlets[0].available, true, 'keyed fallback also looks available to the collector');
  assert.deepEqual(navigation, ['fixture-a', 'fixture-b']);
  ctx.dispose();
  console.log('PASS: isolated session.js discovery, load, clear, switch, stale surface, chain and keyed fallback ambiguity');
})().catch(error => { console.error(error); process.exitCode = 1; });
