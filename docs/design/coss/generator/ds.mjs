import fs from 'node:fs'; import path from 'node:path';
import { __unstable__loadDesignSystem } from 'tailwindcss';
const SRC = path.resolve('../ui-b2336c45f2b7355c42678ce786b6f14d62b819be');
export async function loadDS(){
  let css = fs.readFileSync(path.join(SRC,'packages/ui/src/styles/globals.css'),'utf8');
  css = css.replace(/^@source.*$/mg,'');
  return __unstable__loadDesignSystem(css, { base: SRC, loadStylesheet: async (id, base) => {
    let p = id==='tailwindcss' ? path.resolve('node_modules/tailwindcss/index.css') : path.resolve(base, id);
    if (id.startsWith('tailwindcss/')) p = path.resolve('node_modules', id.replace(/^tailwindcss\//,'tailwindcss/')) ;
    return { path:p, base: path.dirname(p), content: fs.readFileSync(p,'utf8') };
  }});
}
