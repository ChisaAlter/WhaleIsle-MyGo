// Build only original adapters against v2 runtime hooks; SDK/React never bundle.
import { build } from 'esbuild';
import { readFile } from 'node:fs/promises';
const sdk=`const h=globalThis.__whaleMyGo;export const defineComponent=x=>x;export const useIsStreaming=h.isStreaming;export const useStateField=(id,value)=>h.field('selected',value);export const reactive=x=>x;`;
const react=`const r=globalThis.__whaleMyGo.React;export default r;export const {createElement,createContext,useContext,useEffect,useLayoutEffect,useMemo,useRef,useState,useId}=r;`;
// Schema declarations are retained by the immutable upstream registry. Adapter
// components use those already validated props, while their own schema builders
// are not installed as a second registry.
const schema=`const s=new Proxy(function(){return s},{get:(_,k)=>k==='then'?undefined:s,apply:()=>s});export const z=s;`;
await build({entryPoints:['src/v2-native-components.tsx'],outfile:'../../internal/desktop/openui/viewer/v2/native-adapter.js',bundle:true,platform:'browser',target:['es2022'],format:'iife',jsx:'transform',minify:true,plugins:[{name:'owned-v2-hooks',setup(b){b.onResolve({filter:/\/bridge$/},a=>({path:'bridge',namespace:'v2-hook'}));b.onResolve({filter:/^(react|@openuidev\/react-lang|zod\/v4)$/},a=>({path:a.path,namespace:'v2-hook'}));b.onLoad({filter:/.*/,namespace:'v2-hook'},a=>({contents:a.path==='bridge'?`export const byteLength=v=>new TextEncoder().encode(v).length;export const LIMITS={program:262144};`:a.path==='react'?react:a.path==='zod/v4'?schema:sdk,loader:'js'}));}}],logLevel:'warning'});
const path='../../internal/desktop/openui/viewer/v2/native-adapter.js';const result=await readFile(path,'utf8');if(result.includes('react.production')||result.includes('createLibrary'))throw Error('Adapter bundled a second interpreter/React');console.log('Built v2 map, native YouTube, image and isolated HTML adapters.');
