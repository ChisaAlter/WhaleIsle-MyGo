// Offline staging only. Does not download or execute the neural model/runtime.
// node scripts/pet-render-assets.mjs SOURCE_DIRECTORY EXECUTABLE_DIRECTORY
import { readFile, realpath, stat, mkdir, writeFile } from 'node:fs/promises';
import { resolve, relative, dirname, join, isAbsolute } from 'node:path';
import { createHash } from 'node:crypto';

const manifest = JSON.parse(await readFile(new URL('../internal/desktop/pet/assets/render-manifest.json', import.meta.url), 'utf8'));
const [sourceArg, targetArg, ...extra] = process.argv.slice(2);
if (!sourceArg || !targetArg || extra.length) throw new Error('Usage: node scripts/pet-render-assets.mjs SOURCE_DIRECTORY EXECUTABLE_DIRECTORY');
const source = await realpath(sourceArg), target = resolve(targetArg, 'pet-render-assets');
const selected=[];
// Verify everything before touching destination. No arbitrary source enumeration.
for (const file of manifest.files) {
  const name = await realpath(join(source, file.path));
  const rel = relative(source, name);
  if (rel.startsWith('..') || isAbsolute(rel)) throw new Error(`Resource escapes source: ${file.path}`);
  const info=await stat(name);
  if (!info.isFile() || info.size!==file.size) throw new Error(`Missing/changed resource: ${file.path}`);
  const data=await readFile(name);
  if(createHash('sha256').update(data).digest('hex')!==file.sha256)throw new Error(`Checksum mismatch: ${file.path}`);
  selected.push({file,data});
}
// Write the already verified bytes, avoiding a verify/copy source race.
for (const {file,data} of selected) {
  const name=join(target,file.path);
  await mkdir(dirname(name),{recursive:true});
  // Destination paths may not redirect writes through pre-existing symlinks.
  const parent=await realpath(dirname(name)), root=await realpath(target);
  const rel=relative(root,parent);
  if(rel.startsWith('..')||isAbsolute(rel))throw new Error('Destination escapes staging root');
  try {const existing=await realpath(name);if(existing!==name)throw new Error('Destination file is a symlink');}catch(e){if(e.code!=='ENOENT')throw e;}
  await writeFile(name,data);
}
await writeFile(join(target,'provenance.json'),JSON.stringify(manifest,null,2)+'\n');
await writeFile(join(resolve(targetArg),'.pet-render-staged'),manifest.commit+'\n');
console.log(`Staged ${selected.length} verified THA4 files beside executable at ${target}`);
