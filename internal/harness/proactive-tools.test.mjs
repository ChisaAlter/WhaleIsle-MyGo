import test from 'node:test';
import assert from 'node:assert/strict';
import { installRuntimeProvenance, registerKnownToolCallback } from './proactive-provenance.mjs';
import { installToolGate, bindingFor, executionBindingFor } from './proactive-tools.mjs';

function session(id, header = {}, events = []) {
  return { id, header, snapshotEvents: () => events, append(type, data) { events.push({ type, data }); } };
}

function knownFsApply() {}
function knownAcpApply() {}
function thirdPartyApply() {}
registerKnownToolCallback(knownFsApply, 'harness:fs');
registerKnownToolCallback(knownAcpApply, 'desktop:acp');

class MockToolRuntime {
  constructor() {
    this.defs = new Map();
    this.ctx = { fiber: { runtime: { callback: undefined } } };
  }
  withCallback(callback, fn) {
    const prev = this.ctx.fiber.runtime.callback;
    this.ctx.fiber.runtime.callback = callback;
    try { return fn(); } finally { this.ctx.fiber.runtime.callback = prev; }
  }
  register(definition) {
    this.defs.set(definition.name, definition);
    return () => this.defs.delete(definition.name);
  }
  get(name) {
    return this.defs.get(name);
  }
  async dispatchScheduledExecution(exec) {
    return { kind: 'post-result', result: await this.dispatchToolBody(exec) };
  }
  async dispatchToolBody(exec) {
    const def = this.get(exec.name, exec.agent);
    return def.execute(exec.arguments, exec);
  }
}
installRuntimeProvenance(MockToolRuntime.prototype);

function setup() {
  const callbacks = new Map();
  const cleanups = [];
  const tools = new MockToolRuntime();
  const parent = session('parent', { cwd: 'C:\\workspace' }, [
    { type: 'mygo-proactive/binding', data: { taskId: 'task', requestId: 'turn', sessionId: 'parent', boundSessionId: 'parent' } },
  ]);
  installToolGate({
    tools,
    on: (name, fn) => callbacks.set(name, fn),
    effect: fn => { const undo = fn(); if (typeof undo === 'function') cleanups.push(undo); },
    sessions: { get: id => (id === 'parent' ? parent : undefined) },
  });
  return { callbacks, tools, parent, cleanup: () => { for (const fn of cleanups.splice(0)) fn(); } };
}

test('ordinary and forked Sessions keep their original tool path; managed children inherit exact owner', async () => {
  const { callbacks, tools, parent, cleanup } = setup();
  try {
    let calls = 0;
    tools.withCallback(thirdPartyApply, () => tools.register({ name: 'noop', execute: async () => { calls++; return { isError: false, content: [] }; } }));
    await tools.dispatchScheduledExecution({ name: 'noop', callId: 'c1', arguments: {}, agent: { id: 'ordinary', session: session('ordinary') } });
    const fork = session('fork', { parentSession: 'parent' }, parent.snapshotEvents().slice());
    assert.equal(bindingFor(fork), undefined);
    await tools.dispatchScheduledExecution({ name: 'noop', callId: 'c2', arguments: {}, agent: { id: 'fork', session: fork } });
    assert.equal(calls, 2);
    const child = session('child', { origin: 'subagent', parentSession: 'parent' }, parent.snapshotEvents().slice());
    callbacks.get('session/created')(child);
    assert.deepEqual(bindingFor(child), { taskId: 'task', requestId: 'turn', sessionId: 'parent', boundSessionId: 'child' });
  } finally {
    cleanup();
  }
});

test('new ordinary turns release the old task gate while task steers and delegated work retain it', async () => {
  const { callbacks, tools, parent, cleanup } = setup();
  try {
    let effects = 0;
    tools.withCallback(thirdPartyApply, () => tools.register({ name: 'noop', execute: async () => { effects++; return { isError: false, content: [] }; } }));
    parent.append('turn/start', { turn: 1 });
    parent.append('user/message', { source: { rpcId: 'turn' } });
    parent.append('user/message', { source: { rpcId: 'user-steer' } });
    assert.equal(executionBindingFor(parent)?.requestId, 'turn');
    parent.append('turn/end', { turn: 1 });
    parent.append('turn/start', { turn: 2 });
    parent.append('user/message', { source: { rpcId: 'ordinary-next-turn' } });
    assert.equal(executionBindingFor(parent), undefined);
    await tools.dispatchScheduledExecution({ name: 'noop', callId: 'c1', arguments: {}, agent: { id: 'parent', session: parent } });
    assert.equal(effects, 1);
    const ordinaryChild = session('ordinary-child', { origin: 'subagent', parentSession: 'parent' }, parent.snapshotEvents().slice());
    callbacks.get('session/created')(ordinaryChild);
    assert.equal(bindingFor(ordinaryChild), undefined);
    parent.append('mygo-proactive/binding', { taskId: 'task', requestId: 'continue-task', sessionId: 'parent', boundSessionId: 'parent' });
    parent.append('turn/start', { turn: 3 });
    parent.append('user/message', { source: { rpcId: 'continue-task' } });
    assert.equal(executionBindingFor(parent)?.requestId, 'continue-task');
    const taskChild = session('task-child', { origin: 'subagent', parentSession: 'parent' }, parent.snapshotEvents().slice());
    callbacks.get('session/created')(taskChild);
    taskChild.append('turn/start', { turn: 4 });
    taskChild.append('user/message', { source: { rpcId: 'child-prompt' } });
    assert.equal(executionBindingFor(taskChild)?.requestId, 'continue-task');
    // A task-owned call with missing prompt metadata cannot escape its gate.
    parent.append('turn/start', { turn: 4 });
    assert.equal(executionBindingFor(parent)?.requestId, 'continue-task');
  } finally {
    cleanup();
  }
});

test('tool provenance, single-use authorization, argument integrity and ACP rejection bridge', async () => {
  const { tools, parent, cleanup } = setup();
  const nativeTools = globalThis[Symbol.for('whaleisle.nativeTaskTools')];
  const previous = globalThis.fetch;
  const oldURL = process.env.DSH_DESKTOP_INSTALL_URL, oldToken = process.env.DSH_DESKTOP_INSTALL_TOKEN;
  process.env.DSH_DESKTOP_INSTALL_URL = 'http://127.0.0.1:1';
  process.env.DSH_DESKTOP_INSTALL_TOKEN = 'synthetic-tool-test';
  let prepared = 0, effects = 0, denyNext = false;
  const actions = [], rejectedAcp = [];
  const undoRejector = nativeTools.setAcpRejector(async exec => { rejectedAcp.push(exec.callId); });
  globalThis.fetch = async (_url, opts) => {
    const { action, payload } = JSON.parse(opts.body);
    actions.push({ action, payload });
    let value;
    if (action === 'tool/prepare') {
      value = denyNext ? { status: 'denied', reason: 'Rejected in task inbox' } : { status: ++prepared === 1 ? 'pending' : 'ready' };
    } else if (action === 'tool/claim') {
      value = { status: 'allow', lease: 'single-use' };
    } else {
      value = { ok: true };
    }
    return { ok: true, json: async () => value };
  };
  try {
    let authFirst, authRepeat;
    tools.withCallback(thirdPartyApply, () => tools.register({
      name: 'write',
      execute: async (_args, exec) => {
        effects++;
        authFirst = nativeTools.consumeAuthorization(exec);
        authRepeat = nativeTools.consumeAuthorization(exec);
        return { isError: false, content: [] };
      },
    }));
    const args = { recipient: 'named-recipient', body: 'report\n'.repeat(20000) };
    const agent = { id: 'parent', session: parent };
    const exec = { agent, callId: 'call', name: 'write', arguments: args, signal: new AbortController().signal };
    const out = await tools.dispatchScheduledExecution(exec);
    assert.equal(out.result.isError, false);
    assert.equal(effects, 1);
    assert.equal(authFirst, true);
    assert.equal(authRepeat, false);
    assert.equal(nativeTools.consumeAuthorization(exec), false);
    // An unclassified plugin naming its tool `write` never receives `harness:fs` provenance.
    const claimedCall = actions.find(x => x.action === 'tool/claim').payload.call;
    assert.equal(claimedCall.source, 'plugin:unclassified');
    assert.deepEqual(claimedCall.arguments, args);
    assert.equal(actions.at(-1).payload.status, 'succeeded');
    assert.equal(actions.at(-1).payload.lease, 'single-use');

    // Genuine tool-fs callback receives `harness:fs` provenance.
    tools.withCallback(knownFsApply, () => tools.register({ name: 'read', execute: async () => ({ isError: false, content: [] }) }));
    await tools.dispatchScheduledExecution({ agent, callId: 'fs-read', name: 'read', arguments: { file_path: 'a.txt' }, signal: new AbortController().signal });
    assert.equal(actions.find(x => x.action === 'tool/prepare' && x.payload.call.callId === 'parent/fs-read').payload.call.source, 'harness:fs');

    // Argument mutation after approval is blocked before tool body execution.
    tools.withCallback(thirdPartyApply, () => tools.register({ name: 'mutated', execute: async () => ({ isError: false, content: [] }) }));
    const originalDispatch = MockToolRuntime.prototype.dispatchToolBody;
    const tamperedExec = { agent, callId: 'tamper', name: 'mutated', arguments: { target: 'safe' }, signal: new AbortController().signal };
    const origPrepare = globalThis.fetch;
    globalThis.fetch = async (url, opts) => {
      const { action } = JSON.parse(opts.body);
      if (action === 'tool/claim') tamperedExec.arguments = { target: 'evil' };
      return origPrepare(url, opts);
    };
    const tamperedOut = await tools.dispatchScheduledExecution(tamperedExec);
    globalThis.fetch = origPrepare;
    void originalDispatch;
    assert.equal(tamperedOut.result.isError, true);
    assert.match(tamperedOut.result.error.message, /changed after task approval/);

    // Denied `desktop:acp` tool triggers the registered ACP rejector so external permission does not hang.
    tools.withCallback(knownAcpApply, () => tools.register({ name: 'acp_edit', execute: async () => ({ isError: false, content: [] }) }));
    denyNext = true;
    const deniedAcp = await tools.dispatchScheduledExecution({ agent, callId: 'acp-call-1', name: 'acp_edit', arguments: { path: 'x.txt' }, signal: new AbortController().signal });
    assert.equal(deniedAcp.result.isError, true);
    assert.deepEqual(rejectedAcp, ['acp-call-1']);
  } finally {
    undoRejector();
    cleanup();
    globalThis.fetch = previous;
    if (oldURL === undefined) delete process.env.DSH_DESKTOP_INSTALL_URL; else process.env.DSH_DESKTOP_INSTALL_URL = oldURL;
    if (oldToken === undefined) delete process.env.DSH_DESKTOP_INSTALL_TOKEN; else process.env.DSH_DESKTOP_INSTALL_TOKEN = oldToken;
  }
});
