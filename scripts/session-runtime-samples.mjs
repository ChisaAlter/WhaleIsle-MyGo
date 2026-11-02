// Optional original sample catalog; explicit or existing inputs stay strict.
import fs from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {stageComponentSamples} from './component-samples.mjs';
export function stageOptionalSamples(source,destination,explicit=false) {
 if(!explicit&&!fs.existsSync(source))return {versions:0,files:[],reason:'Selected distribution contains no optional bundled component samples'};
 return stageComponentSamples(source,destination);
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
 try {
  const [source,destination,explicit]=process.argv.slice(2);
  if(!source||!destination||!['true','false'].includes(explicit))throw Error('Usage: session-runtime-samples.mjs SOURCE DESTINATION true|false');
  console.log(JSON.stringify(stageOptionalSamples(source,destination,explicit==='true')));
 } catch(error){console.error(error.message);process.exitCode=1;}
}
