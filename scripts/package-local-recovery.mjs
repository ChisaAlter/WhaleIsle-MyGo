// Offline portable recovery package. Explicit static inputs only; never runs a
// build, installer, Host, model, repair script, or an application executable.
import fs from 'node:fs/promises';
import path from 'node:path';
import crypto from 'node:crypto';
import {fileURLToPath} from 'node:url';
import {createRequire} from 'node:module';
import {cloneStagedSessionRuntime, verifyStagedSessionRuntime} from './session-runtime.mjs';
import {stageComponentSamples} from './component-samples.mjs';

const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const json = async filename => JSON.parse(await fs.readFile(filename, 'utf8'));
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const inside = (root, filename) => {const r=path.relative(root, filename);return r===''||(!path.isAbsolute(r)&&r!=='..'&&!r.startsWith('..'+path.sep));};
const require = createRequire(import.meta.url);
const {readManifest} = require('../internal/desktop/components/generic-source/registry.js');
const {validComponentId} = require('../internal/desktop/components/generic-source/store.js');

async function ordinaryFile(filename) {
  if (!(await fs.lstat(filename)).isFile()) throw Error(`Static input must be an ordinary file: ${filename}`);
  const bytes = await fs.readFile(filename);
  return {bytes, sha256:hash(bytes), size:bytes.length};
}
async function ordinaryTree(source) {
  source=await fs.realpath(source);
  const files=[];
  async function walk(relative='') {
    for (const entry of await fs.readdir(path.join(source,relative),{withFileTypes:true})) {
      const name=path.join(relative,entry.name);
      if(entry.isSymbolicLink())throw Error(`Static asset links are not supported: ${name}`);
      if(entry.isDirectory())await walk(name);
      else if(entry.isFile())files.push({path:name,...await ordinaryFile(path.join(source,name))});
      else throw Error(`Unsupported static asset: ${name}`);
    }
  }
  await walk();
  return files;
}
function requirePE(file, label) {
  const b=file.bytes;
  if(b.length<64||b.toString('ascii',0,2)!=='MZ')throw Error(`${label} is not a built Windows executable`);
  const offset=b.readUInt32LE(60);
  if(offset>b.length-4||b.toString('ascii',offset,offset+4)!=='PE\0\0')throw Error(`${label} has no PE signature`);
}
async function staticInputs(options) {
  const files=[], availability={}, missing=[];
  async function add(source,target,pe=false) {
    const file=await ordinaryFile(source);if(pe)requirePE(file,target);
    files.push({path:target,...file});return file;
  }
  if(options.exe)await add(options.exe,'WhaleIsle.exe',true);else missing.push('exe');
  if(options.preview) {await add(options.preview,'whaleisle-preview.exe',true);availability.preview={verified:true,provenance:'explicit local build; no upstream binary pin'};}
  else missing.push('preview');
  const native=await json(path.join(repo,'internal/desktop/terminalruntime/native-manifest.json'));
  const binding=await json(path.join(repo,'third_party/mygo/plugins/terminal/mygo-plugin.json'));
  const lib=native.libraries.find(x=>x.name==='libghostty-vt');
  const bound=binding.libraries.find(x=>x.name==='libghostty-vt');
  const terminal=lib.files['windows-amd64'];
  if(JSON.stringify(terminal)!==JSON.stringify(bound?.files['windows-amd64'])||lib.source!==bound.source)throw Error('Native terminal binding pin changed');
  if(options.terminal) {
    const file=await add(options.terminal,terminal.name);
    if(file.sha256!==terminal.sha256)throw Error('Native terminal SHA256 mismatch');
    await add(path.join(repo,'internal/desktop/terminalruntime/Ghostty-LICENSE'),terminal.name+'.LICENSE');
    const bytes=Buffer.from(JSON.stringify({...terminal,target:'windows-amd64',source:lib.source},null,2)+'\n');
    files.push({path:'terminal-runtime.json',bytes,sha256:hash(bytes),size:bytes.length});
    availability.terminal={verified:true,sha256:terminal.sha256,source:lib.source};
  } else missing.push('terminal');
  const pet=await json(path.join(repo,'internal/desktop/pet/assets/render-manifest.json'));
  if(options.pet) {
    const root=await fs.realpath(options.pet);
    for(const entry of pet.files) {
      const source=path.resolve(root,entry.path);
      if(!inside(root,source)||!inside(root,await fs.realpath(source)))throw Error('Pet asset escapes selected source');
      const file=await add(source,path.join('pet-render-assets',entry.path));
      if(file.size!==entry.size||file.sha256!==entry.sha256)throw Error(`Pet asset pin mismatch: ${entry.path}`);
    }
    const bytes=Buffer.from(JSON.stringify(pet,null,2)+'\n');
    files.push({path:'pet-render-assets/provenance.json',bytes,size:bytes.length,sha256:hash(bytes)});
    availability.pet={verified:true,commit:pet.commit,files:pet.files.length};
  } else missing.push('pet');
  if(options.components) {
    const root=await fs.realpath(options.components);
    const payload=await ordinaryTree(root);
    let versions=0;
    for(const id of await fs.readdir(root,{withFileTypes:true}))if(id.isDirectory()) {
      if(!validComponentId(id.name)||['whalebridge','whaleisle-host'].includes(id.name))throw Error(`Invalid component ID: ${id.name}`);
      for(const version of await fs.readdir(path.join(root,id.name),{withFileTypes:true}))if(version.isDirectory()) {
        const manifest=readManifest(path.join(root,id.name,version.name));
        if(!manifest||manifest.id!==id.name||manifest.version!==version.name)throw Error(`Invalid component manifest: ${id.name}/${version.name}`);
        versions++;
      }
    }
    if(!versions)throw Error('Component samples catalog is empty');
    availability.components={verified:true,versions,files:payload.length,provenance:'supplied manifest+payload validation; no upstream whole-tree pin'};
  } else missing.push('components');
  // Only checked-in static resources, never a caller's arbitrary profile tree.
  for(const file of await ordinaryTree(path.join(repo,'resources')))files.push({...file,path:path.join('resources',file.path)});
  return {files,availability,missing};
}

export async function inventoryLocalRecoverySources(options) {
  const runtime=await verifyStagedSessionRuntime(path.join(options.stage,'vendor'));
  const inputs=await staticInputs(options);
  return {schemaVersion:1,missing:inputs.missing,assets:inputs.availability,
    runtime:{cliVersion:runtime.cliVersion,harnessPin:runtime.harnessPin,node:runtime.node,relocatedLinks:runtime.relocatedLinks},
    files:inputs.files.map(({path,size,sha256})=>({path:path.split(path.sep).join('/'),size,sha256}))};
}

export async function packageLocalRecovery(options) {
  const absolute=path.resolve(options.out);
  const destination=path.join(await fs.realpath(path.dirname(absolute)),path.basename(absolute));
  try {await fs.lstat(destination);throw Error('Portable package destination must be fresh');}
  catch(error){if(error.code!=='ENOENT')throw error;}
  for(const source of [options.stage,options.exe,options.preview,options.terminal,options.pet,options.components].filter(Boolean)) {
    const real=await fs.realpath(source);
    if(inside(real,destination)||inside(destination,real))throw Error('Portable package destination overlaps an input');
  }
  const inputs=await staticInputs(options);
  if(inputs.missing.length)throw Error(`Missing explicit package inputs: ${inputs.missing.join(', ')}`);
  await verifyStagedSessionRuntime(path.join(options.stage,'vendor'));
  await fs.mkdir(destination);
  const runtime=await cloneStagedSessionRuntime(options.stage,path.join(destination,'runtime'));
  for(const file of inputs.files) {
    const target=path.join(destination,file.path);
    await fs.mkdir(path.dirname(target),{recursive:true});
    await fs.writeFile(target,file.bytes,{flag:'wx'});
  }
  const components=stageComponentSamples(options.components,path.join(destination,'runtime/components/samples'));
  const manifest={schemaVersion:1,kind:'whaleisle-isolated-portable',userData:'data',runtime:'runtime',entry:'WhaleIsle.exe',
    provenance:{runtime:{cliVersion:runtime.cliVersion,harnessPin:runtime.harnessPin,node:runtime.node,relocatedLinks:runtime.relocatedLinks},
      assets:inputs.availability,components,files:inputs.files.map(({path,size,sha256})=>({path:path.split(path.sep).join('/'),size,sha256}))}};
  // Marker last: incomplete copying never advertises a launchable package.
  await fs.writeFile(path.join(destination,'portable-package.json'),JSON.stringify(manifest,null,2)+'\n',{flag:'wx'});
  return manifest;
}

if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  try {
    const [command,...args]=process.argv.slice(2), options={};
    if(!['inventory','package'].includes(command)||args.length%2)throw Error('Usage: package-local-recovery.mjs inventory|package --stage DIR [--exe FILE --preview FILE --terminal DLL --pet DIR --components DIR --out FRESH_DIR]');
    for(let i=0;i<args.length;i+=2) {
      const key=args[i].slice(2);
      if(!args[i].startsWith('--')||!['stage','exe','preview','terminal','pet','components','out'].includes(key)||options[key])throw Error(`Invalid option: ${args[i]}`);
      options[key]=args[i+1];
    }
    if(!options.stage||(command==='package'&&!options.out))throw Error('--stage and package --out are required');
    console.log(JSON.stringify(command==='inventory'?await inventoryLocalRecoverySources(options):await packageLocalRecovery(options),null,2));
  }catch(error){console.error(`Portable package: ${error.message}`);process.exitCode=1;}
}
