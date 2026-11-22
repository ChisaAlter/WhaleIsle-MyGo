// scripts/ui-compare/seed-schedule.mjs — seed Host schedule tasks into a DSH
// home, in the real per-record JSON layout the schedule domain reads.
//
//   node seed-schedule.mjs <dshHome> <fixtureJson>
//
// Layout (vendor/deepseek-harness/packages/storage/storage-json/src/
// per-record-unit.ts): <home>/storages/<unit>/<table>/<key>.json, each
// document {"version":<unit version>,"record":<value>} pretty-printed with a
// trailing newline (format.ts serializeRecord). The schedule unit is
// version 2 (packages/schedule/schedule/src/storage.ts); the record value is
// the stored ScheduleTask (storage.ts scheduleTaskSchema).
import { mkdirSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { readFileSync } from 'node:fs'

const [home, fixturePath] = process.argv.slice(2)
if (!home || !fixturePath) {
  console.error('usage: node seed-schedule.mjs <dshHome> <fixtureJson>')
  process.exit(2)
}
const fixture = JSON.parse(readFileSync(fixturePath, 'utf8'))
const dir = join(home, 'storages', 'schedule', 'tasks')
mkdirSync(dir, { recursive: true })
for (const { id, ...task } of fixture.tasks) {
  writeFileSync(join(dir, `${id}.json`), `${JSON.stringify({ version: 2, record: task }, null, 2)}\n`)
}
console.log(`seeded ${fixture.tasks.length} schedule task(s) -> ${dir}`)
