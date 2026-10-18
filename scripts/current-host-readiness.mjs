// Auditable MyGo-only safety patch on a clean, pinned build COPY.
// Never writes the official source checkout or user settings.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
export const guardID = 'mygo-visual-readiness-v1';
export const toolsPath = 'deepseek-harness/packages/client/ui-visual-replies/src/tools.ts';
export const indexPath = 'deepseek-harness/packages/client/ui-visual-replies/lib/index.js';
export const baseSHA256 = 'b8bf5270c9db57134ba448b198236186ccdf8f1b548f217b5c0e2210634c295e';
export const marker = 'MyGo visual reply readiness gate v1';
const hash = value => crypto.createHash('sha256').update(value).digest('hex');
const original = `function requireEnabled(config: Config): void {
  if (!config.enabled.get()) throw new Error('Visual replies are disabled. Enable Visual replies in Settings to create a preview or publish a page.')
}`;
export const replacement = `// MyGo visual reply readiness gate v1: a saved preference is not process readiness.
async function requireEnabled(config: Config, signal: AbortSignal): Promise<void> {
  if (!config.enabled.get()) throw new Error('Visual replies are disabled. Enable Visual replies in Settings to create a preview or publish a page.')
  const url = process.env.DSH_DESKTOP_INSTALL_URL
  const token = process.env.DSH_DESKTOP_INSTALL_TOKEN
  if (!url || !token) throw new Error('MyGo visual reply readiness gate v1: desktop runtime is not verified.')
  const response = await fetch(new URL('/desktop/html-preview/status', url), {
    method: 'GET',
    headers: { authorization: \`Bearer \${token}\` },
    signal: AbortSignal.any([signal, AbortSignal.timeout(3000)]),
    redirect: 'error',
    cache: 'no-store',
  })
  const status = await response.json() as { available?: boolean; reason?: string }
  if (!response.ok || status.available !== true) {
    throw new Error(typeof status.reason === 'string' && status.reason ? status.reason.slice(0, 2048) : 'MyGo visual reply readiness gate v1: verify HTML preview in Settings for this desktop session.')
  }
  signal.throwIfAborted()
  if (!config.enabled.get()) throw new Error('Visual replies were disabled while checking desktop readiness.')
}`;
export const patchSHA256 = hash(replacement + '\nrequireEnabled(config) => await requireEnabled(config, exec.signal)');
export function patchSource(source) {
 if(hash(source)!==baseSHA256) throw new Error('Visual-reply safety patch expected the exact fixed official tools.ts');
 if(!source.includes(original)) throw new Error('Visual-reply safety patch anchor is missing');
 return source.replace(original,replacement).replaceAll('      requireEnabled(config)','      await requireEnabled(config, exec.signal)');
}
export function applyReadinessGuard(work) {
 const file=path.join(work,toolsPath);
 const source=fs.readFileSync(file,'utf8');
 const recordPath=path.join(work,'visual-readiness-patch.json');
 if(fs.existsSync(recordPath)) {
  const record=JSON.parse(fs.readFileSync(recordPath,'utf8'));
  if(record.id===guardID && record.patchSHA256===patchSHA256 && record.sourceSHA256===hash(source)) return record;
  throw new Error('Existing visual readiness patch record does not match the source');
 }
 const patched=patchSource(source);
 const record={id:guardID,baseSHA256,patchSHA256,sourceSHA256:hash(patched)};
 fs.writeFileSync(file,patched);
 fs.writeFileSync(recordPath,JSON.stringify(record,null,2));
 return record;
}
export function inspectReadinessGuard(root) {
 const record=JSON.parse(fs.readFileSync(path.join(root,'visual-readiness-patch.json'),'utf8'));
 if(record.id!==guardID || record.baseSHA256!==baseSHA256 || record.patchSHA256!==patchSHA256 || record.sourceSHA256!==hash(fs.readFileSync(path.join(root,toolsPath)))) throw new Error('Visual readiness safety patch provenance mismatch');
 const built=fs.readFileSync(path.join(root,indexPath),'utf8');
 if(!built.includes(marker) || !built.includes('/desktop/html-preview/status')) throw new Error('Visual readiness source patch has not been compiled into the Host artifact');
 return record;
}
