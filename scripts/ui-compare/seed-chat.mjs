// scripts/ui-compare/seed-chat.mjs — seed one static session history into a
// Host's DSH home, in the real v4 zstd format the session store reads.
//
//   node seed-chat.mjs <harnessRoot> <dshHome> <workspaceCwd> <fixtureJson>
//
// No model calls, no credentials: the events are synthetic saved states built
// with the same Session/create* helpers the apps/web seeded fixtures use.
import fs from 'node:fs/promises'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
import { zstdCompressSync, zstdDecompressSync } from 'node:zlib'

const [root, home, cwd, fixturePath] = process.argv.slice(2)
if (!root || !home || !cwd || !fixturePath) {
  console.error('usage: node seed-chat.mjs <harnessRoot> <dshHome> <workspaceCwd> <fixtureJson>')
  process.exit(2)
}
const load = rel => import(pathToFileURL(path.join(root, rel)))
const { Session } = await load('packages/core/session/lib/index.js')
const { sessionFormatCatalog: catalog } = await load('packages/session/session-format-catalog/lib/index.js')
const {
  createUserMessage, createAssistantMessage, createToolResultMessage,
  createSystemMessage, ToolCallId,
} = await load('packages/llm/llm/lib/index.js')

// projectKey/assertZstdHeaderFrame live unexported in the persistence build —
// reuse them the way tools/rich-history-check's seed does.
const backend = await fs.readFile(path.join(root, 'packages/session/session-persistence-jsonl/lib/index.js'), 'utf8')
function helper(name) {
  const start = backend.indexOf('function ' + name + '(')
  if (start < 0) throw Error('missing backend helper ' + name)
  const end = backend.indexOf('\n}', start) + 2
  return Function(backend.slice(start, end) + ';return ' + name + ';')()
}
const projectKey = helper('projectKey'), assertHeader = helper('assertZstdHeaderFrame')

const fixture = JSON.parse(await fs.readFile(fixturePath, 'utf8'))
const header = { version: 4, id: fixture.id, cwd, createdAt: 1, isSeeded: false, delegationDepth: 0 }
const s = new Session(fixture.id, undefined, header)
const append = (type, data, opts) => s.append(type, data, opts)
const surface = { surfaceOp: 'append' }
const modelSource = { provider: 'synthetic-history', model: 'ui-compare-fixture' }

append('turn/start', { turn: 1 })
append('step/start', { turn: 1, step: 1 })
append('system/message', { turn: 1, step: 1, message: createSystemMessage('ui-compare chat fixture system prompt.') }, surface)
const user = append('user/message', createUserMessage({
  content: [{ type: 'text', text: fixture.user }],
  source: { kind: 'user' },
}), surface)
append('session/title', { title: fixture.title, messageSeqs: [user.seq], source: { kind: 'fallback' } })
append('request/header', { header: { config: { provider: modelSource.provider, model: modelSource.model } }, reason: 'initial' })

const calls = (fixture.tools || []).map((t, i) => ({ ...t, callId: ToolCallId(`${fixture.id}-tool-${i + 1}`) }))
const content = []
if (fixture.thinking) content.push({ type: 'reasoning', text: fixture.thinking })
if (fixture.markdown) content.push({ type: 'text', text: fixture.markdown })
for (const t of calls) content.push({ type: 'tool-call', id: t.callId, name: t.name, arguments: JSON.stringify(t.arguments) })
append('assistant/message', {
  turn: 1, step: 1,
  message: createAssistantMessage({ content, source: modelSource }),
  stream: [],
  usage: { inputTokens: 2000, outputTokens: 240, reasoningTokens: 30 },
}, surface)
for (const t of calls) {
  const call = append('tool/call', { turn: 1, step: 1, callId: t.callId, name: t.name, arguments: JSON.stringify(t.arguments) })
  append('tool/result', {
    turn: 1, step: 1,
    message: createToolResultMessage({
      callId: t.callId,
      content: [{ type: 'text', text: t.result }],
      isError: false,
    }),
  }, { ...surface, sourceEventSeqs: [call.seq] })
}
append('step/end', { turn: 1, step: 1 })
append('turn/end', { turn: 1, reason: { kind: 'completed' } })

const events = Array.from({ length: s.seq }, (_, i) => ({ ...s.eventAt(i), time: 1780000000000 + i }))
// Round-trip through the real restore path so an invalid event shape fails
// the seed rather than producing a history that only renders differently.
Session.fromRestore(fixture.id, events, header, 0, 'detached')
const strict = catalog.createRestore(catalog.encodeCurrentHeader(header, 0), { recovery: 'strict', validation: 'current' })
for (const e of events) strict.decodeRow(catalog.encodeCurrentEvent(e))
strict.finish()
const head = zstdCompressSync(Buffer.from(JSON.stringify(catalog.encodeCurrentHeader(header, 0)) + '\n'))
assertHeader(zstdDecompressSync(head))
const body = zstdCompressSync(Buffer.from(events.map(e => JSON.stringify(catalog.encodeCurrentEvent(e))).join('\n') + '\n'))
const dir = path.join(home, 'sessions', projectKey(cwd), fixture.id)
await fs.mkdir(dir, { recursive: true })
await fs.writeFile(path.join(dir, 'session.v4.jsonl.zstd'), Buffer.concat([head, body]), { flag: 'wx' })
console.log(`seeded ${fixture.id} (${events.length} events) -> ${dir}`)
