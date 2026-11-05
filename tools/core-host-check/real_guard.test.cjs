'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const g = require('./real_guard.cjs');

function request(second = false) {
  const text = s => ({ role: 'user', content: [{ type: 'text', text: s }] });
  return { method: 'POST', redirect: 'error', headers: { 'content-type': 'application/json', 'x-api-key': g.SENTINEL }, body: JSON.stringify({ model: 'deepseek-flash', stream: true, max_tokens: 512, thinking: { type: 'disabled' }, system: 'Synthetic QA only.', messages: second ? [text(g.PROMPTS[0]), { role: 'assistant', content: [{ type: 'text', text: 'QA_ONE_OK' }] }, text(g.PROMPTS[1])] : [text(g.PROMPTS[0])] }) };
}
function owned(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'core-real-guard-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const secretFile = path.join(root, 'explicit-fake-secret');
  fs.writeFileSync(secretFile, 'sk-synthetic-no-account-1234', { mode: 0o600 });
  return { root, secretFile, ledgerFile: path.join(root, 'budget.json') };
}
function sse() {
  const event = e => `data: ${JSON.stringify(e)}\n\n`;
  return event({ type: 'message_start', message: { usage: { input_tokens: 80, output_tokens: 0 } } }) +
    event({ type: 'content_block_start', content_block: { type: 'text', text: '' } }) +
    event({ type: 'content_block_delta', delta: { type: 'text_delta', text: 'QA_ONE_OK' } }) +
    event({ type: 'message_delta', delta: { stop_reason: 'end_turn' }, usage: { output_tokens: 24 } }) + event({ type: 'message_stop' });
}
const noLocal = { localAllowed: () => false, localFetch: () => assert.fail('unexpected local fetch') };

test('final payload allowlist rejects model, thinking, limits, tools, auxiliary content and endpoints', () => {
  const facts = g.validatePayload(g.ENDPOINT, request());
  assert.equal(facts.bytes, Buffer.byteLength(request().body));
  assert.equal(facts.systemBytes, Buffer.byteLength(JSON.stringify('Synthetic QA only.')));
  assert.equal(facts.maxTokens, 512);
  assert.equal(facts.turn, 1);
  const mutations = [
    b => { b.model = 'claude-alias'; }, b => { b.thinking.type = 'enabled'; }, b => { b.max_tokens = 513; },
    b => { b.max_tokens = 1.5; }, b => { b.tools = [{ name: 42 }]; }, b => { b.search = {}; }, b => { b.output_config = { effort: 'high' }; },
    b => { b.system = [{ type: 'text', text: 'not a plain system string' }]; },
    b => { b.messages[0].content[0] = { type: 'tool_result', text: 'data' }; },
    b => { b.messages[0].content[0].text = 'a title, user data or another task'; },
    b => { b.system = 'x'.repeat(262144); }
  ];
  for (const mutate of mutations) {
    const r = request(), b = JSON.parse(r.body); mutate(b); r.body = JSON.stringify(b);
    assert.throws(() => g.validatePayload(g.ENDPOINT, r), /^Error: WIRE_/);
  }
  assert.throws(() => g.validatePayload(g.ENDPOINT + '?forward=1', request()), /WIRE_ENDPOINT/);
  assert.throws(() => g.validatePayload('https://api.deepseek.com/anthropic/v1/files', request()), /WIRE_ENDPOINT/);
  const r = request(); r.headers.authorization = 'Bearer never-forward';
  assert.throws(() => g.validatePayload(g.ENDPOINT, r), /WIRE_HEADERS/);
});

test('preflight validates actual bytes but never reads a secret, creates a ledger or calls a transport', async t => {
  const o = owned(t), events = [];
  fs.unlinkSync(o.secretFile);
  const guard = g.createGuardedFetch({ ...o, ...noLocal, preflight: true, audit: e => events.push(e), send: () => assert.fail('preflight network') });
  await assert.rejects(guard.fetch(g.ENDPOINT, request()), /PREFLIGHT_NO_NETWORK/);
  assert.equal(events[0].kind, 'preflight-validated');
  assert.equal(fs.existsSync(o.ledgerFile), false);
  assert.equal(fs.existsSync(o.ledgerFile + '.lock'), false);
  guard.close();
});

test('ledger locks concurrent owners and keeps two attempts across reopening', t => {
  const o = owned(t);
  let ledger = new g.BudgetLedger(o.ledgerFile);
  assert.throws(() => new g.BudgetLedger(o.ledgerFile), /LEDGER_LOCKED/);
  assert.equal(ledger.reserve(), 1);
  assert.equal(JSON.parse(fs.readFileSync(o.ledgerFile)).attempts[0].state, 'unconfirmed');
  ledger.settle(1, 80, 24); ledger.close();
  ledger = new g.BudgetLedger(o.ledgerFile);
  assert.equal(ledger.total(), 352);
  assert.equal(ledger.reserve(), 2);
  ledger.settle(2, 32768, 512); ledger.close();
  ledger = new g.BudgetLedger(o.ledgerFile);
  assert.equal(ledger.total(), 352 + 69632);
  assert.throws(() => ledger.reserve(), /LEDGER_REQUEST_LIMIT/);
  ledger.close();
  assert.equal(JSON.parse(fs.readFileSync(o.ledgerFile)).attempts.length, 2);
});

test('unknown delivery, stale atomic writes and malformed ledgers never reset or replay', t => {
  const o = owned(t);
  const ledger = new g.BudgetLedger(o.ledgerFile);
  ledger.reserve(); ledger.close();
  assert.throws(() => new g.BudgetLedger(o.ledgerFile), /LEDGER_UNCONFIRMED/);
  assert.equal(JSON.parse(fs.readFileSync(o.ledgerFile)).attempts[0].reservedMicroCNY, 69632);
  const next = path.join(o.root, 'other.json');
  fs.writeFileSync(next + '.next', '{}');
  assert.throws(() => new g.BudgetLedger(next), /LEDGER_AMBIGUOUS/);
  fs.writeFileSync(next, '{broken'); fs.unlinkSync(next + '.next');
  assert.throws(() => new g.BudgetLedger(next), /LEDGER_INVALID/);
  assert.equal(fs.readFileSync(next, 'utf8'), '{broken');
});

test('fake transport sees only explicit key after reservation; two settled calls and a third denial', async t => {
  const o = owned(t), events = [];
  let calls = 0;
  const guard = g.createGuardedFetch({ ...o, ...noLocal, preflight: false, audit: e => events.push(e), send: async (body, key, signal, observer) => {
    calls++;
    assert.equal(key, 'sk-synthetic-no-account-1234');
    const ledger = JSON.parse(fs.readFileSync(o.ledgerFile));
    assert.equal(ledger.attempts.at(-1).state, 'unconfirmed');
    const raw = Buffer.from(sse());
    observer.push(raw.subarray(0, 27)); observer.push(raw.subarray(27)); observer.end();
    return new Response(raw, { headers: { 'content-type': 'text/event-stream' } });
  } });
  await guard.fetch(g.ENDPOINT, request());
  await assert.rejects(guard.fetch(g.ENDPOINT, request()), /WIRE_TURN_ORDER/);
  // A rejected duplicate is terminal for that guard instance, not a retry route.
  guard.close();
  const secondGuard = g.createGuardedFetch({ ...o, ...noLocal, preflight: false, audit: e => events.push(e), send: async (body, key, signal, observer) => {
    calls++; assert.equal(key, 'sk-synthetic-no-account-1234'); observer.push(Buffer.from(sse())); observer.end(); return new Response(sse());
  } });
  await secondGuard.fetch(g.ENDPOINT, request(true));
  await assert.rejects(secondGuard.fetch(g.ENDPOINT, request(true)), /WIRE_TURN_ORDER/);
  secondGuard.close();
  assert.equal(calls, 2);
  assert.equal(events.filter(e => e.kind === 'settled').length, 2);
  assert.equal(fs.existsSync(o.ledgerFile + '.lock'), false);
  assert.equal(JSON.stringify(events).includes('sk-synthetic'), false);
  assert.equal(JSON.stringify(events).includes(g.PROMPTS[0]), false);
});

test('usage settlement counts cache input conservatively and survives split CRLF without decreasing counters', () => {
  let value;
  const observer = g.usageObserver((input, output) => { value = { input, output }; });
  const raw = sse().replace('"input_tokens":80', '"input_tokens":80,"cache_read_input_tokens":10,"cache_creation_input_tokens":5').replace(/\n/g, '\r\n');
  for (const byte of Buffer.from(raw)) observer.push(Buffer.from([byte]));
  observer.end();
  assert.deepEqual(value, { input: 95, output: 24 });
});

test('non-text/tool/error responses are cut before forwarding the complete offending frame', () => {
  for (const event of [{ type: 'content_block_start', content_block: { type: 'tool_use' } }, { type: 'content_block_start', content_block: { type: 'thinking' } }, { type: 'error', error: { message: 'must not reach Host logs' } }, { type: 'message_delta', delta: { stop_reason: 'tool_use' } }]) {
    const observer = g.usageObserver(() => assert.fail('non-text response settled'));
    assert.throws(() => observer.push(Buffer.from(`data: ${JSON.stringify(event)}\n\n`)), /RESPONSE_NON_TEXT/);
  }
});

test('ambiguous transport or incomplete usage consumes one reservation and prohibits a new request', async t => {
  for (const scenario of ['transport', 'partial']) {
    const o = owned(t), events = [];
    const guard = g.createGuardedFetch({ ...o, ...noLocal, preflight: false, audit: e => events.push(e), send: async (body, key, signal, observer) => {
      if (scenario === 'transport') throw new Error('remote error with sensitive text must not appear');
      observer.push(Buffer.from('data: {"type":"message_start","message":{"usage":{"input_tokens":80}}}\n\n'));
      observer.end();
    } });
    await assert.rejects(guard.fetch(g.ENDPOINT, request()), /REQUEST_UNCONFIRMED|USAGE_UNCONFIRMED/);
    await assert.rejects(guard.fetch(g.ENDPOINT, request()), /REQUEST_CONCURRENT/);
    guard.close();
    assert.throws(() => new g.BudgetLedger(o.ledgerFile), /LEDGER_UNCONFIRMED/);
    assert.equal(JSON.stringify(events).includes('sensitive'), false);
    assert.equal(JSON.parse(fs.readFileSync(o.ledgerFile)).attempts.length, 1);
  }
});

function child(script, args = []) {
  const env = {};
  for (const key of ['SystemRoot', 'WINDIR', 'PATH', 'PATHEXT', 'TEMP', 'TMP']) if (process.env[key]) env[key] = process.env[key];
  const r = spawnSync(process.execPath, ['-e', script, path.resolve(__dirname, 'real_guard.cjs'), ...args], { encoding: 'utf8', env, timeout: 15000 });
  assert.equal(r.status, 0, 'isolated Node boundary child failed: ' + r.stderr);
}

test('actual builtin network boundary allows only its owned loopback and blocks alternate egress and process escapes', () => {
  child(`
    const assert=require('node:assert/strict'); const http=require('node:http');
    (async()=>{
      const server=http.createServer((q,s)=>s.end('owned-loopback'));
      await new Promise(r=>server.listen(0,'127.0.0.1',r));
      const port=server.address().port, g=require(process.argv[1]), events=[];
      g.installNetworkBoundary({preflight:true,hostPort:port},e=>events.push(e));
      assert.equal(await (await fetch('http://127.0.0.1:'+port)).text(),'owned-loopback');
      assert.equal(await new Promise((res,rej)=>{const q=http.request({host:'127.0.0.1',port,path:'/owned-path'},r=>{r.resume();r.on('end',()=>res(r.statusCode))});q.on('error',rej);q.end()}),200);
      await assert.rejects(fetch('https://example.invalid/messages'),/WIRE_ENDPOINT/);
      const net=require('node:net'),https=require('node:https'),tls=require('node:tls'),dns=require('node:dns');
      for(const probe of [()=>net.connect(443,'203.0.113.1'),()=>net.connect({host:'127.0.0.1',port,path:'/named-pipe'}),()=>http.request({host:'127.0.0.1',port:port+1,path:'/x'}),()=>https.request('https://api.deepseek.com/anthropic/v1/messages'),()=>tls.connect(443,'203.0.113.1'),()=>http.request('http://127.0.0.1:1'),()=>dns.lookup('example.invalid',()=>{}),()=>require('node:http2').connect('https://example.invalid'),()=>require('node:dgram').createSocket('udp4'),()=>require('node:child_process').spawn('node',[]),()=>new (require('node:worker_threads').Worker)('bad')])assert.throws(probe,/BOUNDARY_EGRESS/);
      await new Promise(r=>server.close(r));
    })().catch(()=>{process.exitCode=1});
  `);
});

test('final builtin HTTPS sender uses a single captured request with no retry, redirect, proxy or ambient authorization', t => {
  const o = owned(t);
  child(`
    const assert=require('node:assert/strict'),fs=require('node:fs'),https=require('node:https');
    const {EventEmitter}=require('node:events'),{Readable}=require('node:stream'); let calls=0;
    https.request=(url,opts,callback)=>{
      calls++; assert.equal(url,'https://api.deepseek.com/anthropic/v1/messages'); assert.equal(opts.agent,false);
      assert.equal(opts.headers['x-api-key'],'sk-synthetic-no-account-1234'); assert.equal(opts.headers.authorization,undefined);
      const req=new EventEmitter(); req.setTimeout=()=>{}; req.destroy=()=>{};
      req.end=body=>{assert.equal(JSON.parse(body).thinking.type,'disabled'); const res=Readable.from([Buffer.from(${JSON.stringify(sse())})]);res.statusCode=200;res.headers={'content-type':'text/event-stream'};callback(res)};
      return req;
    };
    const g=require(process.argv[1]);
    g.installNetworkBoundary({hostPort:1,secretFile:process.argv[2],ledgerFile:process.argv[3],preflight:false},()=>{});
    (async()=>{const r=await fetch(g.ENDPOINT,${JSON.stringify(request())});await r.text();assert.equal(calls,1);assert.equal(JSON.parse(fs.readFileSync(process.argv[3])).attempts[0].state,'settled')})().catch(()=>{process.exitCode=1});
  `, [o.secretFile, o.ledgerFile]);
});
