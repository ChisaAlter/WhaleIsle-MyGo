'use strict';

// Opt-in final-wire guard. No proxy, SDK, ambient key or automatic recovery.
const fs = require('node:fs');
const path = require('node:path');
const { AsyncLocalStorage } = require('node:async_hooks');
const { syncBuiltinESMExports } = require('node:module');
const https = require('node:https');

const ENDPOINT = 'https://api.deepseek.com/anthropic/v1/messages';
const SENTINEL = 'qa-real-flow-no-secret';
const RESERVATION = 69632;
const CAP = 1000000;
const PROMPTS = [
  'Reply with the plain text QA_ONE_OK only. Do not use tools or search.',
  'Reply with the plain text QA_TWO_OK only. Do not use tools or search.'
];
const fail = (code, detail) => { const e = new Error(code); e.code = code; if (typeof detail === 'string' && detail.length <= 200) e.detail = detail; throw e; };
const integer = x => Number.isSafeInteger(x) && x >= 0;
const plain = x => x !== null && typeof x === 'object' && !Array.isArray(x);

function validatePayload(url, init) {
  if (String(url) !== ENDPOINT || init?.method !== 'POST' || init.redirect !== 'error') fail('WIRE_ENDPOINT');
  // Bounds the real Host envelope (baseline system/context block is ~140KB even
  // with an empty preset); still refuses pathological or replayed payloads.
  if (typeof init.body !== 'string' || Buffer.byteLength(init.body) > 262144) fail('WIRE_BYTES');
  let b;
  try { b = JSON.parse(init.body); } catch { fail('WIRE_JSON'); }
  const allowed = new Set(['model', 'stream', 'messages', 'max_tokens', 'thinking', 'system', 'temperature', 'stop_sequences', 'tools']);
  if (!plain(b)) fail('WIRE_FIELDS');
  const unknown = Object.keys(b).filter(k => !allowed.has(k));
  if (unknown.length) fail('WIRE_FIELDS', 'fields:' + unknown.map(k => k + '(' + Buffer.byteLength(JSON.stringify(b[k]) ?? '') + ')').join(','));
  if (b.model !== 'deepseek-flash' || b.stream !== true || !Number.isSafeInteger(b.max_tokens) || b.max_tokens < 1 || b.max_tokens > 512) fail('WIRE_MODEL_LIMIT');
  if (!plain(b.thinking) || Object.keys(b.thinking).length !== 1 || b.thinking.type !== 'disabled') fail('WIRE_THINKING');
  if ('system' in b && typeof b.system !== 'string') fail('WIRE_TEXT');
  if ('temperature' in b && (typeof b.temperature !== 'number' || !Number.isFinite(b.temperature))) fail('WIRE_FIELDS');
  if ('stop_sequences' in b && (!Array.isArray(b.stop_sequences) || b.stop_sequences.some(s => typeof s !== 'string'))) fail('WIRE_FIELDS');
  // The production tool catalog rides on the real wire; bound its size and
  // schema shape — descriptions/parameters are static plugin manifests.
  if ('tools' in b) {
    if (!Array.isArray(b.tools) || b.tools.length > 64 || Buffer.byteLength(JSON.stringify(b.tools)) > 32768) fail('WIRE_TOOLS');
    for (const t of b.tools) {
      if (!plain(t) || typeof t.name !== 'string' || Object.keys(t).some(k => !['name', 'description', 'input_schema', 'type', 'strict', 'cache_control'].includes(k))) fail('WIRE_TOOLS');
      if ('description' in t && typeof t.description !== 'string') fail('WIRE_TOOLS');
      if ('input_schema' in t && !plain(t.input_schema)) fail('WIRE_TOOLS');
      if ('type' in t && typeof t.type !== 'string') fail('WIRE_TOOLS');
    }
  }
  if (!Array.isArray(b.messages) || b.messages.length < 1 || b.messages.length > 4) fail('WIRE_MESSAGES');
  for (let i = 0; i < b.messages.length; i++) {
    const m = b.messages[i];
    if (!plain(m) || Object.keys(m).some(k => k !== 'role' && k !== 'content') || m.role !== (i % 2 ? 'assistant' : 'user')) fail('WIRE_TEXT');
    if (!Array.isArray(m.content) || m.content.length === 0 || m.content.some(c => !plain(c) || Object.keys(c).some(k => k !== 'type' && k !== 'text') || c.type !== 'text' || typeof c.text !== 'string')) fail('WIRE_TEXT');
  }
  if (b.messages.at(-1).role !== 'user') fail('WIRE_MESSAGES');
  const userMessages = b.messages.filter(m => m.role === 'user');
  const userTexts = userMessages.map(m => m.content.map(c => c.text).join(''));
  // The production envelope wraps the prompt with host-generated context; only
  // the exact synthetic prompt inside a bounded wrapper keeps this synthetic.
  if (userTexts.some((t, i) => t.length > 8192 || typeof PROMPTS[i] !== 'string' || !t.includes(PROMPTS[i]))) {
    // Lengths plus a containment flag for the expected synthetic prompt; no
    // message content is ever serialized into the audit.
    fail('WIRE_SYNTHETIC_ONLY', 'userTexts:' + userTexts.map((t, i) => t.length + (typeof PROMPTS[i] === 'string' && t.includes(PROMPTS[i]) ? '*' : '')).join(','));
  }
  const h = new Headers(init.headers);
  const headerNames = new Set(['content-type', 'x-api-key', 'accept', 'anthropic-version', 'user-agent', 'content-length', 'accept-encoding', 'connection', 'host', 'x-deepseek-harness-session-id', 'x-deepseek-harness-user-id']);
  const badHeaders = [...h.keys()].filter(k => !headerNames.has(k));
  if (badHeaders.length) fail('WIRE_HEADERS', 'headers:' + badHeaders.join(','));
  if (h.get('x-api-key') !== SENTINEL || h.get('content-type') !== 'application/json') fail('WIRE_HEADERS', 'values');
  return { bytes: Buffer.byteLength(init.body), systemBytes: 'system' in b ? Buffer.byteLength(JSON.stringify(b.system)) : 0, messagesBytes: Buffer.byteLength(JSON.stringify(b.messages)), maxTokens: b.max_tokens, turn: userMessages.length };
}

function atomicWrite(file, value) {
  const temporary = file + '.next';
  let fd;
  try {
    fd = fs.openSync(temporary, 'wx', 0o600);
    fs.writeFileSync(fd, JSON.stringify(value) + '\n');
    fs.fsyncSync(fd);
    fs.closeSync(fd); fd = undefined;
    fs.renameSync(temporary, file);
  } catch {
    if (fd !== undefined) fs.closeSync(fd);
    // A leftover .next is intentionally not silently recovered.
    fail('LEDGER_WRITE');
  }
}

class BudgetLedger {
  constructor(file) {
    if (!path.isAbsolute(file)) fail('LEDGER_PATH');
    this.file = file;
    this.lock = file + '.lock';
    if (fs.existsSync(file + '.next')) fail('LEDGER_AMBIGUOUS');
    try { this.fd = fs.openSync(this.lock, 'wx', 0o600); } catch { fail('LEDGER_LOCKED'); }
    try {
      if (fs.existsSync(file)) {
        if (!fs.lstatSync(file).isFile() || fs.lstatSync(file).isSymbolicLink() || fs.statSync(file).size > 16384) fail('LEDGER_INVALID');
        try { this.value = JSON.parse(fs.readFileSync(file, 'utf8')); } catch { fail('LEDGER_INVALID'); }
      } else {
        this.value = { schema: 1, model: 'deepseek-official/deepseek-flash', capMicroCNY: CAP, reservationMicroCNY: RESERVATION, attempts: [] };
        atomicWrite(file, this.value);
      }
      this.validate();
    } catch (e) { this.close(); throw e; }
  }
  validate() {
    const v = this.value;
    if (!plain(v) || v.schema !== 1 || v.model !== 'deepseek-official/deepseek-flash' || v.capMicroCNY !== CAP || v.reservationMicroCNY !== RESERVATION || !Array.isArray(v.attempts) || v.attempts.length > 2) fail('LEDGER_INVALID');
    for (let i = 0; i < v.attempts.length; i++) {
      const a = v.attempts[i];
      if (!plain(a) || a.number !== i + 1 || a.reservedMicroCNY !== RESERVATION || a.state !== 'settled' || !integer(a.inputTokens) || !integer(a.outputTokens) || a.inputTokens > 32768 || a.outputTokens > 512 || a.chargedMicroCNY !== a.inputTokens * 2 + a.outputTokens * 8) fail('LEDGER_UNCONFIRMED');
    }
    if (this.total() > CAP) fail('LEDGER_BUDGET');
  }
  total() { return this.value.attempts.reduce((n, a) => n + (a.state === 'settled' ? a.chargedMicroCNY : a.reservedMicroCNY), 0); }
  reserve() {
    this.validate();
    if (this.value.attempts.length >= 2) fail('LEDGER_REQUEST_LIMIT');
    if (this.total() + RESERVATION > CAP) fail('LEDGER_BUDGET');
    const a = { number: this.value.attempts.length + 1, reservedMicroCNY: RESERVATION, state: 'unconfirmed' };
    this.value.attempts.push(a);
    atomicWrite(this.file, this.value);
    return a.number;
  }
  settle(number, inputTokens, outputTokens) {
    if (!integer(inputTokens) || !integer(outputTokens) || inputTokens > 32768 || outputTokens > 512) fail('USAGE_LIMIT');
    const a = this.value.attempts[number - 1];
    if (!a || a.state !== 'unconfirmed') fail('LEDGER_UNCONFIRMED');
    Object.assign(a, { state: 'settled', inputTokens, outputTokens, chargedMicroCNY: inputTokens * 2 + outputTokens * 8 });
    atomicWrite(this.file, this.value);
  }
  close() {
    if (this.fd === undefined) return;
    fs.closeSync(this.fd); this.fd = undefined;
    fs.unlinkSync(this.lock);
  }
}

function usageObserver(complete) {
  let pending = '', input, output, stopped = false, finished = false, bad = false, bytes = 0;
  const decoder = new TextDecoder();
  const frame = text => {
    const data = text.split('\n').filter(x => x.startsWith('data:')).map(x => x.slice(5).trimStart()).join('\n');
    if (!data || data === '[DONE]') return;
    let e;
    try { e = JSON.parse(data); } catch { bad = true; return; }
    if (e.type === 'error' || e.type === 'content_block_start' && e.content_block?.type !== 'text' || e.type === 'content_block_delta' && e.delta?.type !== 'text_delta') fail('RESPONSE_NON_TEXT');
    const u = e.message?.usage ?? e.usage;
    if (u) {
      if ('input_tokens' in u) {
        const fields = [u.input_tokens, u.cache_creation_input_tokens ?? 0, u.cache_read_input_tokens ?? 0];
        if (fields.some(x => !integer(x))) bad = true; else input = Math.max(input ?? 0, fields.reduce((a, b) => a + b, 0));
      }
      if ('output_tokens' in u) { if (!integer(u.output_tokens)) bad = true; else output = Math.max(output ?? 0, u.output_tokens); }
    }
    if (e.type === 'message_delta' && e.delta?.stop_reason != null) {
      if (e.delta.stop_reason === 'end_turn') finished = true; else fail('RESPONSE_NON_TEXT');
    }
    if (e.type === 'message_stop') stopped = true;
  };
  return {
    push(chunk) {
      bytes += chunk.length;
      if (bytes > 1024 * 1024) fail('RESPONSE_LIMIT');
      pending = (pending + decoder.decode(chunk, { stream: true })).replace(/\r\n/g, '\n');
      let at;
      while ((at = pending.indexOf('\n\n')) >= 0) { frame(pending.slice(0, at)); pending = pending.slice(at + 2); }
      if (pending.length > 65536) fail('RESPONSE_LIMIT');
    },
    end() {
      pending += decoder.decode();
      if (pending.trim()) frame(pending);
      if (bad || !stopped || !finished || !integer(input) || !integer(output)) fail('USAGE_UNCONFIRMED');
      complete(input, output);
    }
  };
}

function readExplicitSecret(file) {
  if (!path.isAbsolute(file)) fail('SECRET_FILE');
  let key;
  try {
    const st = fs.lstatSync(file);
    if (!st.isFile() || st.isSymbolicLink() || st.size < 8 || st.size > 512) fail('SECRET_FILE');
    key = fs.readFileSync(file, 'utf8').trim();
  } catch { fail('SECRET_FILE'); }
  if (!/^[A-Za-z0-9_-]{8,256}$/.test(key) || key === SENTINEL) fail('SECRET_FILE');
  return key;
}

function createGuardedFetch({ preflight, secretFile, ledgerFile, audit, send, localFetch, localAllowed }) {
  let ledger, preflightCount = 0, busy = false;
  const close = () => { if (ledger) ledger.close(); };
  const guarded = async (url, init) => {
    if (localAllowed(url)) return localFetch(url, init);
    try {
      const facts = validatePayload(url, init);
      if (busy) fail('REQUEST_CONCURRENT');
      busy = true;
      if (preflight) {
        if (++preflightCount > 1) fail('PREFLIGHT_REQUEST_LIMIT');
        audit({ kind: 'preflight-validated', ...facts });
        fail('PREFLIGHT_NO_NETWORK');
      }
      if (!ledger) ledger = new BudgetLedger(ledgerFile);
      if (facts.turn !== ledger.value.attempts.length + 1) fail('WIRE_TURN_ORDER');
      // Read only this explicit file, after the entire actual wire request passed.
      const key = readExplicitSecret(secretFile);
      const number = ledger.reserve();
      const requestLedger = ledger;
      audit({ kind: 'reserved', number, reservedMicroCNY: RESERVATION, ...facts });
      const observer = usageObserver((inputTokens, outputTokens) => {
        requestLedger.settle(number, inputTokens, outputTokens);
        audit({ kind: 'settled', number, inputTokens, outputTokens, chargedMicroCNY: inputTokens * 2 + outputTokens * 8 });
        requestLedger.close();
        if (ledger === requestLedger) ledger = undefined;
        busy = false;
      });
      return await send(init.body, key, init.signal, observer);
    } catch (e) {
      // Fixed codes plus request metadata only: never serialize bodies,
      // credentials or provider errors. Byte counts bound the attempted call.
      const code = typeof e?.code === 'string' && /^(WIRE_|LEDGER_|USAGE_|SECRET_|REQUEST_|PREFLIGHT_)/.test(e.code) ? e.code : 'REQUEST_UNCONFIRMED';
      if (code !== 'PREFLIGHT_NO_NETWORK') {
        const row = { kind: 'rejected', code };
        if (typeof init?.body === 'string') row.bytes = Buffer.byteLength(init.body);
        if (typeof e?.detail === 'string') row.detail = e.detail;
        audit(row);
      }
      fail(code);
    }
  };
  return { fetch: guarded, close };
}

function installNetworkBoundary(config, audit) {
  const net = require('node:net'), tls = require('node:tls'), http = require('node:http');
  const dns = require('node:dns'), dgram = require('node:dgram'), http2 = require('node:http2');
  const cp = require('node:child_process'), workers = require('node:worker_threads');
  const permit = {}, authority = new AsyncLocalStorage();
  const originalRequest = https.request.bind(https), originalFetch = globalThis.fetch;
  const localPorts = new Set([config.hostPort]);
  if (process.env.DSH_DESKTOP_INSTALL_URL) {
    const u = new URL(process.env.DSH_DESKTOP_INSTALL_URL);
    if (u.protocol !== 'http:' || u.hostname !== '127.0.0.1') fail('BOUNDARY_CONTROL_URL');
    localPorts.add(Number(u.port));
  }
  const loopback = host => ['127.0.0.1', '::1', '[::1]'].includes(host);
  const localAllowed = input => {
    try { const u = new URL(typeof input === 'string' || input instanceof URL ? input : input.url); return u.protocol === 'http:' && loopback(u.hostname) && localPorts.has(Number(u.port)); } catch { return false; }
  };
  // Destinations only (host:port); never paths, bodies or headers.
  const targetOf = args => {
    let o = args[0];
    if (Array.isArray(o)) o = o[0];
    if (typeof o === 'string' || o instanceof URL) {
      try { const u = new URL(o); return u.hostname + ':' + (u.port || (u.protocol === 'https:' ? 443 : 80)); } catch { return 'unparsed-url'; }
    }
    if (plain(o)) return String(o.hostname ?? o.host ?? '?') + (o.port != null ? ':' + o.port : '');
    if (typeof o === 'number' || /^\d+$/.test(String(o))) return String(args[1]) + ':' + o;
    return undefined;
  };
  const blocked = (kind = 'SOCKET', target) => { const code = 'BOUNDARY_EGRESS_' + kind; audit(target ? { kind: 'rejected', code, target } : { kind: 'rejected', code }); fail(code); };
  const hostOf = o => {
    const h = String(o.hostname ?? o.host ?? '');
    if (h.startsWith('[')) return h.slice(0, h.indexOf(']') + 1);
    const i = h.indexOf(':');
    return i < 0 ? h : h.slice(0, i);
  };
  const portOf = o => {
    if (o.port != null) return Number(o.port);
    const h = String(o.host ?? '');
    const i = h.lastIndexOf(':');
    return i > 0 ? Number(h.slice(i + 1)) : NaN;
  };
  const connectionAllowed = args => {
    if (authority.getStore() === permit) return true;
    let o = args[0];
    if (Array.isArray(o)) o = o[0];
    if (plain(o)) return !o.path && loopback(hostOf(o)) && localPorts.has(portOf(o));
    return (typeof o === 'number' || /^\d+$/.test(String(o))) && loopback(args[1]) && localPorts.has(Number(o));
  };
  // http.request option objects carry `path` as the request path on an already
  // owned server; it is not a socket path and must not disqualify owned calls.
  const requestAllowed = first => {
    if (typeof first === 'string' || first instanceof URL) return localAllowed(first);
    return plain(first) && loopback(hostOf(first)) && localPorts.has(portOf(first));
  };
  const connect = net.Socket.prototype.connect;
  net.Socket.prototype.connect = function (...args) { if (!connectionAllowed(args)) return blocked('SOCKET', targetOf(args)); return connect.apply(this, args); };
  for (const mod of [http, https]) {
    const request = mod.request;
    mod.request = function (...args) {
      const first = args[0];
      let allowed = requestAllowed(first);
      const overrides = args.find(a => plain(a));
      if (allowed && overrides && overrides !== first && ('host' in overrides || 'hostname' in overrides || 'port' in overrides)) {
        allowed = loopback(hostOf(overrides)) && localPorts.has(portOf(overrides));
      }
      if (!allowed) return blocked('HTTP', targetOf(args));
      return request.apply(this, args);
    };
    mod.get = function (...args) { const req = mod.request(...args); req.end(); return req; };
  }
  const tlsConnect = tls.connect;
  tls.connect = function (...args) { if (!connectionAllowed(args)) return blocked('TLS', targetOf(args)); return tlsConnect.apply(this, args); };
  const lookup = dns.lookup;
  dns.lookup = function (host, ...args) { if (!(authority.getStore() === permit && host === 'api.deepseek.com') && !loopback(host)) return blocked('DNS', String(host)); return lookup.call(this, host, ...args); };
  const promiseLookup = dns.promises.lookup;
  dns.promises.lookup = function (host, ...args) { if (!(authority.getStore() === permit && host === 'api.deepseek.com') && !loopback(host)) return blocked('DNS', String(host)); return promiseLookup.call(this, host, ...args); };
  for (const name of Object.keys(dns)) if (name.startsWith('resolve') || name === 'reverse') dns[name] = () => blocked('DNS');
  for (const name of Object.keys(dns.promises)) if (name.startsWith('resolve') || name === 'reverse') dns.promises[name] = () => blocked('DNS');
  dns.Resolver = dns.promises.Resolver = class { constructor() { blocked('DNS'); } };
  dgram.createSocket = () => blocked('UDP'); dgram.Socket.prototype.send = () => blocked('UDP');
  http2.connect = authority => blocked('HTTP2', targetOf([authority]));
  // Program basename only — arguments and command lines are never serialized.
  const procTarget = args => {
    try {
      let f = args[0];
      if (plain(f)) f = f.file;
      const t = path.basename(String(f ?? ''));
      return t || undefined;
    } catch { return undefined; }
  };
  for (const name of ['spawn', 'spawnSync', 'execFile', 'execFileSync', 'fork']) cp[name] = (...a) => blocked('SUBPROCESS', procTarget(a));
  for (const name of ['exec', 'execSync']) cp[name] = (...a) => blocked('SUBPROCESS', procTarget([String(a[0] ?? '').trim().split(/\s+/)[0]]));
  cp.ChildProcess.prototype.spawn = function (...a) { return blocked('SUBPROCESS', procTarget(a)); };
  if (typeof process.execve === 'function') process.execve = (...a) => blocked('SUBPROCESS', procTarget(a));
  workers.Worker = class { constructor() { blocked('WORKER'); } };
  syncBuiltinESMExports();
  const send = (body, key, signal, observer) => authority.run(permit, () => new Promise((resolve, reject) => {
    let req;
    const failed = () => reject(Object.assign(new Error('REQUEST_UNCONFIRMED'), { code: 'REQUEST_UNCONFIRMED' }));
    try {
      // One low-level POST, no redirects, connection pooling, retry, or ambient proxy.
      req = originalRequest(ENDPOINT, { method: 'POST', agent: false, signal, headers: { 'content-type': 'application/json', accept: 'text/event-stream', 'anthropic-version': '2023-06-01', 'x-api-key': key, 'content-length': Buffer.byteLength(body) } }, res => {
        if (res.statusCode !== 200 || !String(res.headers['content-type']).startsWith('text/event-stream')) { res.destroy(); failed(); return; }
        let ended = false;
        const stream = new ReadableStream({
          start(controller) {
            res.on('data', chunk => { try { observer.push(chunk); controller.enqueue(new Uint8Array(chunk)); } catch { res.destroy(); controller.error(new Error('USAGE_UNCONFIRMED')); } });
            res.on('end', () => { try { observer.end(); ended = true; controller.close(); } catch { controller.error(new Error('USAGE_UNCONFIRMED')); } });
            res.on('error', () => { if (!ended) controller.error(new Error('REQUEST_UNCONFIRMED')); });
            res.on('aborted', () => { if (!ended) controller.error(new Error('REQUEST_UNCONFIRMED')); });
          },
          cancel() { res.destroy(); req.destroy(); }
        });
        resolve(new Response(stream, { status: 200, headers: { 'content-type': 'text/event-stream' } }));
      });
      req.on('error', failed);
      req.setTimeout(60000, () => req.destroy());
      req.end(body);
    } catch { if (req) req.destroy(); failed(); }
  }));
  const guard = createGuardedFetch({ ...config, audit, send, localFetch: originalFetch, localAllowed });
  globalThis.fetch = guard.fetch;
  process.once('exit', guard.close);
  return guard;
}

module.exports = { ENDPOINT, SENTINEL, RESERVATION, CAP, PROMPTS, validatePayload, BudgetLedger, usageObserver, createGuardedFetch, installNetworkBoundary };
if (process.env.WHALE_REAL_GUARD_CONFIG) {
  let config;
  try { config = JSON.parse(fs.readFileSync(process.env.WHALE_REAL_GUARD_CONFIG, 'utf8')); } catch { fail('BOUNDARY_CONFIG'); }
  const audit = value => fs.appendFileSync(config.auditFile, JSON.stringify({ ...value, pid: process.pid }) + '\n', { mode: 0o600 });
  installNetworkBoundary(config, audit);
  audit({ kind: 'boundary-installed', preflight: config.preflight });
}
