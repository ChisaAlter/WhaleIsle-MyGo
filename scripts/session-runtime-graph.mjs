// Closed runtime graph utilities. No code is loaded, repaired, or executed.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
export const inside = (root, filename) => {const relative=path.relative(root,filename);return relative===''||(!path.isAbsolute(relative)&&relative!=='..'&&!relative.startsWith('..'+path.sep));};
export function safeChild(root, relative) {
  if(typeof relative!=='string'||!relative||relative.includes('\\')||relative.split('/').some(part=>!part||part==='.'||part==='..'||part.includes(':')))throw Error(`Unsafe runtime graph path: ${relative}`);
  const result=path.resolve(root,...relative.split('/'));if(!inside(root,result)||result===root)throw Error('Runtime graph path escapes root');return result;
}
export const shaFile = filename => crypto.createHash('sha256').update(fs.readFileSync(filename)).digest('hex');
export function inventoryGraph(directory, excluded=[]) {
  const root=fs.realpathSync(directory), files={}, links={}, directories=[];
  function walk(dir) {
    for(const entry of fs.readdirSync(dir,{withFileTypes:true})) {
      const filename=path.join(dir,entry.name),relative=path.relative(root,filename).split(path.sep).join('/');
      if(excluded.includes(relative))continue;
      if(entry.isSymbolicLink()) {
        const real=fs.realpathSync(filename);
        if(!inside(root,real)||inside(real,filename))throw Error(`External or ancestor runtime link: ${relative}`);
        const stat=fs.statSync(real);if(!stat.isFile()&&!stat.isDirectory())throw Error(`Unsupported runtime link: ${relative}`);
        links[relative]={target:path.relative(root,real).split(path.sep).join('/'),kind:stat.isDirectory()?'directory':'file'};
      } else if(entry.isDirectory()) {directories.push(relative);walk(filename);}
      else if(entry.isFile())files[relative]=shaFile(filename);
      else throw Error(`Unsupported runtime entry: ${relative}`);
    }
  }
  walk(root);return {root,files,links,directories};
}
export function cloneGraph(source,destination,{exclude=[],hardlink=false}={}) {
  const graph=inventoryGraph(source,exclude),target=path.resolve(destination);
  if(inside(graph.root,target)||inside(target,graph.root)||fs.existsSync(target))throw Error('Runtime graph clone requires a fresh disjoint destination');
  fs.mkdirSync(target,{recursive:true});
  for(const relative of graph.directories)fs.mkdirSync(safeChild(target,relative),{recursive:true});
  for(const relative of Object.keys(graph.files)) {const from=safeChild(graph.root,relative),to=safeChild(target,relative);hardlink?fs.linkSync(from,to):fs.copyFileSync(from,to);}
  for(const [relative,link] of Object.entries(graph.links)) {const to=safeChild(target,relative),from=safeChild(target,link.target);fs.symlinkSync(process.platform==='win32'&&link.kind==='directory'?from:path.relative(path.dirname(to),from),to,link.kind==='directory'?(process.platform==='win32'?'junction':'dir'):'file');}
  return graph;
}
export function verifyGraph(directory,manifest,{exclude=[],allowMaterialized=false}={}) {
  if(!manifest||manifest.schemaVersion!==1||typeof manifest.files!=='object'||typeof manifest.links!=='object')throw Error('Invalid runtime graph manifest');
  const actual=inventoryGraph(directory,exclude);
  for(const [relative,expected] of Object.entries(manifest.files)) {
    safeChild(actual.root,relative);
    if(!/^[a-f0-9]{64}$/.test(expected)||actual.files[relative]!==expected)throw Error(`Runtime graph file changed or missing: ${relative}`);
  }
  for(const [relative,expected] of Object.entries(manifest.links)) {
    const filename=safeChild(actual.root,relative),target=safeChild(actual.root,expected.target);
    if(!['file','directory'].includes(expected.kind))throw Error('Invalid runtime graph link kind');
    const link=actual.links[relative];
    if(link?.target===expected.target&&link.kind===expected.kind)continue;
    if(!allowMaterialized||link||!fs.existsSync(filename)||!fs.existsSync(target))throw Error(`Runtime graph link changed or missing: ${relative}`);
    if(expected.kind==='file') {if(shaFile(filename)!==shaFile(target))throw Error(`Materialized runtime alias changed: ${relative}`);}
    else throw Error(`Runtime graph directory alias must be preserved: ${relative}`);
  }
  for(const relative of Object.keys(actual.files))if(!manifest.files[relative])throw Error(`Untracked runtime graph file: ${relative}`);
  for(const relative of Object.keys(actual.links))if(!manifest.links[relative])throw Error(`Untracked runtime graph link: ${relative}`);
  return actual;
}
