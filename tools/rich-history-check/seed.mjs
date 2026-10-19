// All records are synthetic saved states, never outputs of a live Agent.
import fs from 'node:fs/promises'
import path from 'node:path'
import { pathToFileURL } from 'node:url'
import { zstdCompressSync, zstdDecompressSync } from 'node:zlib'
const [root, home, cwd, imagePath, manifestPath] = process.argv.slice(2)
const load = rel => import(pathToFileURL(path.join(root, rel)))
const { Session } = await load('packages/core/session/lib/index.js')
const { sessionFormatCatalog: catalog } = await load('packages/session/session-format-catalog/lib/index.js')
const { saveImageFile } = await load('packages/attachment/attachment-local/lib/index.js')
const backend = await fs.readFile(path.join(root, 'packages/session/session-persistence-jsonl/lib/index.js'), 'utf8')
function helper(name) {
  const start = backend.indexOf('function ' + name + '(')
  if (start < 0) throw Error('missing backend helper ' + name)
  const end = backend.indexOf('\n}', start) + 2
  return Function(backend.slice(start, end) + ';return ' + name + ';')()
}
const projectKey = helper('projectKey'), assertHeader = helper('assertZstdHeaderFrame')
const attachment = await saveImageFile(path.join(home, 'attachments/v1'), { data: await fs.readFile(imagePath), mediaType: 'image/png', name: 'owned-raster.png' }, { maxImageBytes: 20971520, maxImagesPerMessage: 20, maxMessageImageBytes: 209715200, maxImagePixels: 64000000, maxImageDimension: 8192, mediaTypes: ['image/png', 'image/jpeg', 'image/webp', 'image/gif'] }, { maxPixels: 4194304, maxDimension: 8192, maxBytes: 4194304 })
const questions = [{ id: 'choice', question: 'Owned saved decision', header: 'Synthetic question', options: [{ label: 'Yes' }, { label: 'No' }] }]
const answer = { questions, answers: [{ id: 'choice', selected: ['Yes'], custom: 'owned answer' }] }
const headerTools = [{ name: 'ask_user_question', description: 'Saved synthetic timed question schema', parameters: { type: 'object', properties: { timeout: { type: 'number' }, questions: { type: 'array', items: { type: 'object' } } } } }]
const rows = []
const append = (s, type, data, surface = false) => s.append(type, data, surface ? { surfaceOp: 'append' } : undefined)
const user = (s, id, text, extra = []) => append(s, 'user/message', { id, role: 'user', source: { kind: 'user' }, content: [{ type: 'text', text }, ...extra] }, true)
const assistant = (s, id, content) => append(s, 'assistant/message', { turn: 1, step: 1, message: { id, role: 'assistant', source: { kind: 'model', provider: 'synthetic-history', model: 'never-dispatched' }, content }, stream: [] }, true)
const result = (s, callId, content) => append(s, 'tool/result', { turn: 1, step: 1, message: { id: callId + '-result', role: 'tool', source: { kind: 'tool', callId }, toolCallId: callId, content: [{ type: 'text', text: JSON.stringify(content) }] } }, true)
function call(s, id, callId, name, args) {
  const argumentsText = JSON.stringify(args)
  assistant(s, id + '-advertised', [{ type: 'tool-call', id: callId, name, arguments: argumentsText }])
  append(s, 'tool/call', { turn: 1, step: 1, callId, name, arguments: argumentsText })
}
async function save(kind, fill) {
  const id = 'rich-history-fixture-' + kind
  const header = { version: 4, id, cwd, createdAt: 1, isSeeded: false, delegationDepth: 0 }
  const s = new Session(id, undefined, header)
  // Keep these below the follow window without faking model-visible text.
  for (let i = 0; i < 21; i++) append(s, 'job/registered', { jobId: id + '-inert-' + i, owner: id, kind: 'local', label: 'synthetic saved fact', startedAt: 1 })
  append(s, 'turn/start', { turn: 1 })
  user(s, id + '-user', 'Owned saved ' + kind + ' state. No model was called.')
  append(s, 'step/start', { turn: 1, step: 1 })
  await fill(s, id)
  append(s, 'step/end', { turn: 1, step: 1 })
  append(s, 'turn/end', { turn: 1, reason: { kind: 'completed' } })
  const events = Array.from({ length: s.seq }, (_, i) => ({ ...s.eventAt(i), time: 1780000000000 + i }))
  Session.fromRestore(id, events, header, 0, 'detached')
  const strict = catalog.createRestore(catalog.encodeCurrentHeader(header, 0), { recovery: 'strict', validation: 'current' })
  for (const event of events) strict.decodeRow(catalog.encodeCurrentEvent(event))
  strict.finish()
  const head = zstdCompressSync(Buffer.from(JSON.stringify(catalog.encodeCurrentHeader(header, 0)) + '\n'))
  assertHeader(zstdDecompressSync(head))
  const body = zstdCompressSync(Buffer.from(events.map(e => JSON.stringify(catalog.encodeCurrentEvent(e))).join('\n') + '\n'))
  const dir = path.join(home, 'sessions', projectKey(cwd), id)
  await fs.mkdir(dir, { recursive: true })
  await fs.writeFile(path.join(dir, 'session.v4.jsonl.zstd'), Buffer.concat([head, body]), { flag: 'wx' })
  rows.push({ id, kind, events: events.length })
}
await save('text', (s, id) => assistant(s, id + '-assistant', [{ type: 'reasoning', text: '**Owned saved reasoning** with a short explanation.' }, { type: 'text', text: '# Owned Markdown\n\nA **bold** answer and a list:\n\n- alpha\n- beta\n\n```go\nfmt.Println("owned code")\n```\n\n| Item | Value |\n| --- | --- |\n| owned | 42 |' }]))
await save('media', (s, id) => {
  user(s, id + '-image', 'Owned image attachment', [{ type: 'image', attachment }])
  call(s, id, 'owned-present', 'present', { files: [{ path: 'owned-delivery.md' }] })
  result(s, 'owned-present', { ok: true })
  append(s, 'deliverables/presented', { turn: 1, callId: 'owned-present', files: [{ path: 'owned-delivery.md', description: 'Owned saved deliverable, read-only acceptance' }] })
  assistant(s, id + '-assistant', [{ type: 'text', text: 'Owned delivery is available below.' }])
})
await save('workflow', (s, id) => {
  append(s, 'tool-workflow/run-start', { runId: 'owned-workflow', name: 'Owned saved Workflow' })
  append(s, 'tool-workflow/agent-start', { runId: 'owned-workflow', seq: 0, label: 'Recorded completed member', phase: 'Review', childId: 'rich-history-fixture-absent-child' })
  append(s, 'tool-workflow/agent-end', { runId: 'owned-workflow', seq: 0, outcome: 'completed' })
  append(s, 'tool-workflow/agent-start', { runId: 'owned-workflow', seq: 1, label: 'Recorded failed member', phase: '', childId: 'rich-history-fixture-absent-child-2' })
  append(s, 'tool-workflow/agent-end', { runId: 'owned-workflow', seq: 1, outcome: 'failed' })
  append(s, 'tool-workflow/run-end', { runId: 'owned-workflow', stopReason: 'error' })
  assistant(s, id + '-assistant', [{ type: 'text', text: 'Saved failure only; no workflow ran.' }])
})
for (const settled of [false, true]) await save(settled ? 'question-settled' : 'question-continued', (s, id) => {
  append(s, 'request/header', { header: { config: { provider: 'synthetic-history', model: 'never-dispatched' }, tools: headerTools }, reason: 'initial' })
  call(s, id, id + '-question', 'ask_user_question', { questions, timeout: 1 })
  append(s, 'user-questions/asked', { id: id + '-request', callId: id + '-question', questions })
  result(s, id + '-question', { pending: true })
  append(s, 'user-questions/answered', { id: id + '-request', outcome: settled ? 'answered' : 'unavailable', ...(settled ? { answer: { answers: answer.answers } } : {}) })
  if (settled) append(s, 'user/message', { id: id + '-reply', role: 'user', source: { kind: 'user-question-reply', callId: id + '-question', outcome: 'answered' }, content: [{ type: 'text', text: JSON.stringify(answer) }] }, true)
})
await save('approval-audit', (s, id) => {
  append(s, 'approval/asked', { id: id + '-approval', toolName: 'owned-inert-tool', reason: 'Synthetic saved audit; no execution' })
  append(s, 'approval/decided', { id: id + '-approval', outcome: 'rejected' })
  assistant(s, id + '-assistant', [{ type: 'text', text: 'Saved rejected approval audit. No live request exists.' }])
})
await fs.writeFile(manifestPath, JSON.stringify({ rows, attachment }, null, 2))
console.log(JSON.stringify({ count: rows.length, strictSessionRestore: true, physicalV4SeparateFrames: true }))
