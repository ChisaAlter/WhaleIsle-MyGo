#!/usr/bin/env node
// scripts/ui-compare/mock.mjs — dependency-free Messages-compatible HTTP/SSE
// model server for ui-compare card scenarios. Binds 127.0.0.1:0, prints
// PORT=<n>, and answers POST <base>/v1/messages the way the DeepSeek adapter
// expects (same wire shape as the original's test-support/llm-mock-server).
//
//   node mock.mjs --log <path>
//
// Behavior is chosen by REQUEST CONTENT, never by sequence — session-title
// and conversation-reference calls interleave with the real turn:
//   * any request whose messages carry a tool_result block -> fixed final text
//   * the last user text containing "[ui-compare:<scenario>]" -> that script
//   * anything else (title generation, conversation reference, stray) -> aux
// Every response carries fixed usage numbers so both sides render identical
// footers and usage dialogs.
import { createServer } from 'node:http';
import { appendFileSync, mkdirSync } from 'node:fs';
import { dirname } from 'node:path';

const args = process.argv.slice(2);
const logPath = args[args.indexOf('--log') + 1];
mkdirSync(dirname(logPath), { recursive: true });

// The same server also publishes the offline plugin-market index used by the
// market surface: the Host's dsh-plugin-market service fetches
// DSHD_PLUGIN_MARKET_INDEX over plain http, so a fixed document is enough for
// both clients (no registry, no network).
const MARKET_INDEX = {
  format: 1,
  title: { en: 'UI Compare Market', zh: '界面比对示例市场' },
  description: { en: 'Deterministic index for ui-compare', zh: '界面比对的确定性索引' },
  metadata: {
    categories: { tools: { title: { en: 'Tools', zh: '工具' } } },
    labels: { official: { title: { en: 'Official', zh: '官方' } } },
    sortKeys: {},
  },
  plugins: [
    {
      name: '@ui-compare/plugin-alpha',
      title: { en: 'Alpha Plugin', zh: '阿尔法插件' },
      description: { en: 'First deterministic sample plugin.', zh: '第一个确定性示例插件。' },
      author: 'UI Compare',
      metadata: { category: 'tools', keywords: ['sample'], labels: ['official'], sortKeys: {} },
      versions: [{ version: '1.0.0', publishTimestamp: 1762358400, source: { type: 'npm' }, engines: {} }],
    },
    {
      name: '@ui-compare/plugin-beta',
      title: { en: 'Beta Plugin', zh: '贝塔插件' },
      description: { en: 'Second deterministic sample plugin.', zh: '第二个确定性示例插件。' },
      metadata: { keywords: [], labels: [], sortKeys: {} },
      versions: [{ version: '0.2.1', publishTimestamp: 1762963200, source: { type: 'npm' }, engines: {} }],
    },
    {
      name: '@ui-compare/plugin-gamma',
      title: { en: 'Gamma Plugin', zh: '伽马插件' },
      description: { en: 'Incompatible sample plugin.', zh: '不兼容的示例插件。' },
      metadata: { keywords: [], labels: [], sortKeys: {} },
      versions: [{ version: '3.0.0', publishTimestamp: 1762963200, source: { type: 'npm' }, engines: { dsh: '>=99.0.0' } }],
    },
  ],
};

const FIXED_USAGE = { input: 41, cacheRead: 12, cacheWrite: 8, output: 23 };
const sse = (res, payload) => res.write(`event: ${payload.type}\ndata: ${JSON.stringify(payload)}\n\n`);

function toolUse(name, input) {
  return [
    { type: 'content_block_start', index: 0, content_block: { type: 'tool_use', id: 'mock-call-1', name, input: {} } },
    { type: 'content_block_delta', index: 0, delta: { type: 'input_json_delta', partial_json: JSON.stringify(input) } },
    { type: 'content_block_stop', index: 0 },
    { type: 'message_delta', delta: { stop_reason: 'tool_use' }, usage: { output_tokens: FIXED_USAGE.output } },
  ];
}
function text(reply) {
  return [
    { type: 'content_block_start', index: 0, content_block: { type: 'text', text: '' } },
    { type: 'content_block_delta', index: 0, delta: { type: 'text_delta', text: reply } },
    { type: 'content_block_stop', index: 0 },
    { type: 'message_delta', delta: { stop_reason: 'end_turn' }, usage: { output_tokens: FIXED_USAGE.output } },
  ];
}

const SCENARIOS = {
  approval: () => toolUse('pwsh', {
    command: 'echo hi',
    description: 'ui-compare approval probe',
    sandbox_permissions: 'danger-full-access',
    justification: 'ui-compare approval probe',
  }),
  question: () => toolUse('ask_user_question', {
    questions: [
      {
        id: 'color', header: 'Choice', question: 'Pick a color?', multi_select: false,
        options: [
          { label: 'Red', description: 'Warm tone for headers.' },
          { label: 'Blue', description: 'Cool tone for accents.' },
          { label: 'Green', description: 'Neutral tone for surfaces.' },
        ],
      },
      {
        id: 'toppings', header: 'Toppings', question: 'Pick toppings?', multi_select: true,
        options: [{ label: 'Cheese' }, { label: 'Mushroom' }],
      },
    ],
  }),
  usage: () => text('ui-compare usage turn complete'),
  // /plan arms plan mode through the real composer on both sides first; the
  // exit_plan_mode call then goes through the shared review waterfall, so the
  // pending plan-review card is what the scenarios compare.
  plan: () => toolUse('exit_plan_mode', {
    plan: '# Ship plan mode UI\n\nSettle the plan-review card against the original.\n\n## Steps\n- Enter plan mode\n- Draft `exit_plan_mode` payload\n- Render the review card',
  }),
};
const AUX = () => text('mock reply ok');
const FINAL = () => text('Done.');

function choose(body) {
  const messages = body?.messages;
  if (!Array.isArray(messages)) return 'aux';
  for (const m of messages) {
    for (const c of [].concat(m?.content || [])) {
      if (c && c.type === 'tool_result') return 'final';
    }
  }
  // The scenario marker can sit in a steered user message that is not the
  // last message of the request (plan-mode context follows it). Scan every
  // message, but skip auxiliary calls that merely embed prior user text —
  // title and conversation-reference generations carry it inside JSON.
  let behavior = 'aux';
  for (const m of messages) {
    const text = [].concat(m?.content || []).map(c => (c && c.type === 'text' ? c.text : '') || '').join('\n');
    if (text.startsWith('Conversation reference data') || text.startsWith('Generate the session title')) continue;
    const hit = /\[ui-compare:([a-z-]+)\]/.exec(text);
    if (hit && SCENARIOS[hit[1]]) behavior = hit[1];
  }
  return behavior;
}

const server = createServer((req, res) => {
  if (req.method === 'GET' && (req.url.split('?')[0].replace(/\/+$/, '') === '/market-index.json')) {
    appendFileSync(logPath, JSON.stringify({ ts: new Date().toISOString(), behavior: 'market-index' }) + '\n');
    res.writeHead(200, { 'content-type': 'application/json' });
    res.end(JSON.stringify(MARKET_INDEX) + '\n');
    return;
  }
  if (req.method !== 'POST' || !req.url.replace(/\/+$/, '').endsWith('/v1/messages')) {
    res.writeHead(404); res.end(); return;
  }
  let raw = '';
  req.on('data', c => raw += c);
  req.on('end', () => {
    let body = null;
    try { body = JSON.parse(raw); } catch {}
    const behavior = choose(body);
    const events = (behavior === 'final' ? FINAL : behavior === 'aux' ? AUX : SCENARIOS[behavior])();
    appendFileSync(logPath, JSON.stringify({
      ts: new Date().toISOString(), behavior,
      lastUser: (() => { const ms = body?.messages || []; const t = [].concat(ms[ms.length - 1]?.content || []).map(c => c?.text || '').join(' '); return t.slice(0, 160); })(),
    }) + '\n');
    res.writeHead(200, { 'content-type': 'text/event-stream', 'cache-control': 'no-cache', connection: 'keep-alive' });
    sse(res, { type: 'message_start', message: { id: 'mock-message', type: 'message', role: 'assistant', model: 'mock-model', content: [], usage: { input_tokens: FIXED_USAGE.input, output_tokens: 0, cache_read_input_tokens: FIXED_USAGE.cacheRead, cache_creation_input_tokens: FIXED_USAGE.cacheWrite } } });
    for (const ev of events) sse(res, ev);
    sse(res, { type: 'message_stop' });
    res.end();
  });
});
server.listen(0, '127.0.0.1', () => console.log('PORT=' + server.address().port));
