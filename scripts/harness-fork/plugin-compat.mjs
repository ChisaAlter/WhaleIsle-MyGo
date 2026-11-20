// The carried desktop plugins whose harness compatibility this stage takes
// responsibility for.
//
// The desktop plugins are copied from 本体's vendor tree, so the dsh versions
// each one accepts are 本体's declaration. A stage that ships a newer harness
// than those declarations admit has to widen them, and widening is an
// assertion about 本体's plugin that 本体 did not make. This table is that
// assertion's record: one row per plugin, naming the exact peer requirements
// the stage widens. A carried plugin whose requirement is neither satisfied by
// the shipped version nor named here stops the stage, so an unrecorded
// assertion never reaches a payload. The reason and the proof for each row live
// in scripts/runtime-composition/README.md under "Carried plugin compatibility
// is widened at the stage".
import fs from 'node:fs';
import path from 'node:path';

export const CARRIED_COMPATIBILITY = [
  {
    plugin: 'dsh-whale',
    requirements: [
      '@deepseek-ai/dsh-agent',
      '@deepseek-ai/dsh-session',
      '@deepseek-ai/dsh-settings',
      '@deepseek-ai/dsh-system-prompt',
    ],
  },
  {
    plugin: 'dsh-remote',
    requirements: [
      '@deepseek-ai/dsh-commands',
      '@deepseek-ai/dsh-host-webserver',
      '@deepseek-ai/dsh-system-prompt',
      '@deepseek-ai/dsh-tools',
    ],
  },
  {
    plugin: 'dshbot',
    requirements: [
      '@deepseek-ai/dsh-agent',
      '@deepseek-ai/dsh-llm',
      '@deepseek-ai/dsh-scope',
      '@deepseek-ai/dsh-session',
      '@deepseek-ai/dsh-settings',
      '@deepseek-ai/dsh-subagent',
      '@deepseek-ai/dsh-system-prompt',
      '@deepseek-ai/dsh-tools',
    ],
  },
];

const DSH = '@deepseek-ai/dsh';

const isDshRequirement = name => name === DSH || name.startsWith(`${DSH}-`);

// The Host reads a plugin's `engines.dsh` alone when it sets one and its dsh
// peers otherwise (packages/boot/app-boot/src/plugin-compatibility.ts). No
// carried plugin sets engines.dsh today; a bump that starts to needs a decision
// about that field rather than a peer widened for nothing.
const names = (range, version) =>
  range.trim() === '*' ||
  range.split('||').some(alternative => alternative.trim().replace(/^[~^=]/, '') === version);

/**
 * Widen one carried plugin's dsh requirements to admit the shipped version.
 * @param manifest parsed plugin package.json, mutated in place.
 * @param version exact dsh version the stage ships, e.g. `0.2.2-rc.5`.
 * @param table recorded rows; defaults to CARRIED_COMPATIBILITY.
 * @returns one `requirement: before -> after` line per widened range.
 * @throws when a requirement needs widening but no row records it.
 */
export function widenPluginCompatibility(manifest, version, table = CARRIED_COMPATIBILITY) {
  if (typeof manifest.engines?.dsh === 'string') {
    throw new Error(`${manifest.name} declares engines.dsh ${manifest.engines.dsh}; the Host reads that field alone and this stage has no record for it`);
  }
  const row = table.find(entry => entry.plugin === manifest.name);
  const changes = [];
  for (const [name, range] of Object.entries(manifest.peerDependencies ?? {})) {
    if (!isDshRequirement(name) || names(range, version)) continue;
    if (!row?.requirements.includes(name)) {
      throw new Error(
        `${manifest.name} declares ${name} ${range}, which does not admit ${version}; add it to CARRIED_COMPATIBILITY with the reason widening it is safe`,
      );
    }
    manifest.peerDependencies[name] = `${range} || ${version}`;
    changes.push(`${name}: ${range} -> ${manifest.peerDependencies[name]}`);
  }
  return changes;
}

/**
 * Widen every carried plugin manifest in a stage's vendor directory.
 * @param vendor the stage's vendor directory.
 * @param carried plugin directory names copied from the desktop, in carry order.
 * @param version exact dsh version the stage ships.
 * @returns one `name@version: requirement -> range` line per widened manifest.
 */
export function carryPluginCompatibility(vendor, carried, version) {
  const widened = [];
  for (const name of carried) {
    const manifestPath = path.join(vendor, name, 'package.json');
    if (!fs.existsSync(manifestPath)) continue;
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
    const changes = widenPluginCompatibility(manifest, version);
    if (changes.length === 0) continue;
    fs.writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
    widened.push(`${manifest.name}@${manifest.version}: ${changes.join(', ')}`);
  }
  return widened;
}
