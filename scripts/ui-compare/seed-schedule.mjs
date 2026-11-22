// scripts/ui-compare/seed-schedule.mjs — seed Host schedule tasks into a DSH
// home, in the real whole-unit JSON layout the schedule domain reads.
//
//   node seed-schedule.mjs <dshHome> <fixtureJson>
//
// Layout (vendor/deepseek-harness/packages/storage/storage-json/src/
// format.ts serialize + single-unit.ts): <home>/storages/<unit>.json, one
// document {unit:{name,version},global,tables:{<table>:{<key>:<value>}}}
// pretty-printed with a trailing newline. The schedule unit declares no
// layout (the default `single`) and is version 1 in the pinned runtime
// (runtime-rc4: storage.ts scheduleTaskSchema requires record, sessionId,
// and status active|inactive); each table value is the stored ScheduleTask.
import { mkdirSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { readFileSync } from 'node:fs'

const [home, fixturePath] = process.argv.slice(2)
if (!home || !fixturePath) {
  console.error('usage: node seed-schedule.mjs <dshHome> <fixtureJson>')
  process.exit(2)
}
const fixture = JSON.parse(readFileSync(fixturePath, 'utf8'))
const dir = join(home, 'storages')
mkdirSync(dir, { recursive: true })
const tasks = {}
for (const { id, ...task } of fixture.tasks) tasks[id] = task
const document = { unit: { name: 'schedule', version: 1 }, global: null, tables: { tasks } }
writeFileSync(join(dir, 'schedule.json'), `${JSON.stringify(document, null, 2)}\n`)
console.log(`seeded ${fixture.tasks.length} schedule task(s) -> ${join(dir, 'schedule.json')}`)
