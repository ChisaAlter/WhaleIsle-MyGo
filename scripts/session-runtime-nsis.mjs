// Emit only physical runtime entries. Never recurse through pnpm aliases.
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {inventoryGraph, verifyGraph} from './session-runtime-graph.mjs';
const escape = value => value.replaceAll('$','$$').replaceAll('"','$\\"');
export function emitRuntimeInclude(stage, output) {
  stage=fs.realpathSync(stage);
  const runtime=path.join(stage,'runtime'),vendor=path.join(runtime,'vendor');
  const manifest=JSON.parse(fs.readFileSync(path.join(vendor,'runtime-graph.json'),'utf8'));
  verifyGraph(vendor,manifest,{exclude:['runtime-graph.json','dshd-remote/debug.log']});
  const graph=inventoryGraph(runtime,['vendor/dshd-remote/debug.log']);
  const lines=['; Generated from a verified physical runtime graph; do not edit.'];
  const files=Object.keys(graph.files).sort();
  let current='';
  for(const relative of files) {
    const parent=path.posix.dirname(relative);
    if(parent!==current) {lines.push(`!insertmacro WI_SetPayloadOutPath "$INSTDIR\\runtime${parent==='.'?'':'\\'+escape(parent.replaceAll('/','\\'))}"`);current=parent;}
    // NSIS's compiler also needs extended source paths; a valid runtime file
    // at MAX_PATH otherwise fails before the install-time path helpers run.
    const source=path.toNamespacedPath(path.join(runtime,...relative.split('/')));
    lines.push(`File "${escape(source)}"`,'!insertmacro WI_CheckExtract');
  }
  // Alias parents can be otherwise empty and must exist before restoration.
  for(const relative of graph.directories.sort())lines.push(`CreateDirectory "$INSTDIR\\runtime\\${escape(relative.replaceAll('/','\\'))}"`,'!insertmacro WI_CheckExtract');
  fs.writeFileSync(output,lines.join('\n')+'\n',{flag:'wx'});
  return {files:files.length,links:Object.keys(graph.links).length,include:path.resolve(output)};
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  try {console.log(JSON.stringify(emitRuntimeInclude(...process.argv.slice(2)),null,2));}
  catch(error){console.error(error.message);process.exitCode=1;}
}
