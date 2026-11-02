// Read-only import of the MIT WhaleIsle 0.3.10 runtime; never starts Electron.
// No old Open Intelligent UI application bundle is imported.
import { readFile, mkdir, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { resolve, dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
const archive = process.argv[2];
if (!archive) throw new Error('Pass the absolute path of the owned WhaleIsle 0.3.10 app.asar.');
const bytes = await readFile(archive);
const headerSize = bytes.readUInt32LE(4);
const header = JSON.parse(bytes.subarray(16, 16 + bytes.readUInt32LE(12)).toString('utf8'));
const read = path => {
  let node = header;
  for (const segment of path.split('/')) node = node.files?.[segment];
  if (!node || node.link || !Number.isSafeInteger(node.size)) throw new Error(`Invalid archive entry: ${path}`);
  if (node.unpacked) {
    const data = readFileSync(join(archive + '.unpacked', ...path.split('/')));
    if (data.length !== node.size) throw new Error(`Unpacked entry changed: ${path}`);
    return data;
  }
  const offset = 8 + headerSize + Number(node.offset);
  if (!Number.isSafeInteger(offset) || offset < 0 || offset + node.size > bytes.length) throw new Error('Invalid archive offset');
  return bytes.subarray(offset, offset + node.size);
};
const info = JSON.parse(read('package.json'));
const build = JSON.parse(read('src/renderer/whale-ui/build.json'));
if (info.name !== 'deepseek-harness-desktop' || info.version !== '0.3.10' || info.license !== 'MIT' || build.digest !== '295ba8ffca4711de7cc788d8ffbdd674b906554ab2863d85387b5b04859fc36e') throw new Error('Unsupported owned runtime version or license');
const notices = read('src/renderer/whale-ui/THIRD_PARTY_NOTICES.txt').toString();
if (!notices.includes('Whale UI owns its runtime and component source') || !notices.includes('MIT-licensed dsh-genui')) throw new Error('Missing source ownership and dependency notices');
const products = ['viewer.html','host.cjs','assets/mermaid.js','assets/three.js','assets/echarts-core.js','assets/echarts-full.js','assets/icons.js','assets/standalone-runtime.js','THIRD_PARTY_NOTICES.txt'];
const target = resolve(dirname(fileURLToPath(import.meta.url)), '../../../internal/desktop/openui/viewer/v2');
const assets = [];
for (const name of products) {
  const content = read(`src/renderer/whale-ui/${name}`);
  const dest = join(target, name);
  await mkdir(dirname(dest), { recursive: true });
  await writeFile(dest, content);
  assets.push({ name, bytes: content.length, sha256: createHash('sha256').update(content).digest('hex') });
}
await writeFile(join(target, 'provenance.json'), JSON.stringify({ runtime: 'whale-ui', protocol: 2, source: { product: info.name, version: info.version, license: info.license, digest: build.digest }, excluded: 'Open Intelligent UI application source and old viewer', assets }, null, 2) + '\n');
console.log(`Imported ${assets.length} owned, licensed v2 artifacts. Source installation was read only.`);
