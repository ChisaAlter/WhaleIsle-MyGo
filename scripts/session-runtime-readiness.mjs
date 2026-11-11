// Preserve the native visual-preview readiness boundary in a copied released runtime.
// Never applies to unknown bundles or to the source of the runtime copy.
import fs from 'node:fs/promises';
import path from 'node:path';
import crypto from 'node:crypto';

export const guardID = 'mygo-session-runtime-readiness-v1';
export const bundlePath = 'deepseek-harness/packages/client/ui-visual-replies/lib/index.js';
export const recordFile = 'session-runtime-readiness.json';
export const baseSHA256 = 'b58fda1a53692c10f85f07ad1758cfdc5937062f721315e1b9d55427b34372d8';
export const outputSHA256 = '8278107b486c56cd4519232f29e265f84f07943b51d469acd38cc0c6f694ec9b';
const rc4BaseSHA256 = '270c0b7a152a8f137ce255073c76cedce1aec68c2403a0033c17a57ee97ebd23';
const rc4OutputSHA256 = '00dd3a8aed5fe4f98f852d4d5a81d910cd1dbed01922cfa23a1004d7d364abb3';
const hash = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
const original = `function requireEnabled(config) {
\tif (!config.enabled.get()) throw new Error("Visual replies are disabled. Enable Visual replies in Settings to create a preview or publish a page.");
}`;
export const replacement = `// MyGo visual reply readiness gate v1: a saved preference is not process readiness.
async function requireEnabled(config, signal) {
\tif (!config.enabled.get()) throw new Error("Visual replies are disabled. Enable Visual replies in Settings to create a preview or publish a page.");
\tconst url = process.env.DSH_DESKTOP_INSTALL_URL;
\tconst token = process.env.DSH_DESKTOP_INSTALL_TOKEN;
\tif (!url || !token) throw new Error("MyGo visual reply readiness gate v1: desktop runtime is not verified.");
\tconst response = await fetch(new URL("/desktop/html-preview/status", url), {
\t\tmethod: "GET",
\t\theaders: { authorization: \`Bearer \${token}\` },
\t\tsignal: AbortSignal.any([signal, AbortSignal.timeout(3000)]),
\t\tredirect: "error",
\t\tcache: "no-store",
\t});
\tconst status = await response.json();
\tif (!response.ok || status.available !== true) {
\t\tthrow new Error(typeof status.reason === "string" && status.reason ? status.reason.slice(0, 2048) : "MyGo visual reply readiness gate v1: verify HTML preview in Settings for this desktop session.");
\t}
\tsignal.throwIfAborted();
\tif (!config.enabled.get()) throw new Error("Visual replies were disabled while checking desktop readiness.");
}`;
const oldCall = '\t\t\trequireEnabled(config);';
const newCall = '\t\t\tawait requireEnabled(config, exec.signal);';
export const patchSHA256 = hash(replacement + '\n' + oldCall + ' => ' + newCall + '\n7 calls');

export function readinessProfile(version) {
  if (version === '0.2.1-rc.2') return {id: guardID, baseSHA256, patchSHA256, outputSHA256};
  if (version === '0.2.1-rc.4') return {id: guardID, baseSHA256: rc4BaseSHA256, patchSHA256, outputSHA256: rc4OutputSHA256};
  throw new Error(`Unsupported Session runtime readiness version: ${version}`);
}

/** A byte-bound patch of the one released compiled bundle; no tool is executed. */
export function patchSessionReadinessSource(source, version = '0.2.1-rc.2') {
  const profile = readinessProfile(version);
  if (hash(source) !== profile.baseSHA256) throw new Error(`Session runtime readiness expected the exact ${version.replace('0.2.1-', '')} visual-reply bundle`);
  if (source.split(original).length !== 2 || source.split(oldCall).length !== 8) {
    throw new Error('Session runtime readiness requires one function and exactly seven operation checks');
  }
  const output = source.replace(original, replacement).replaceAll(oldCall, newCall);
  if (hash(output) !== profile.outputSHA256) throw new Error('Session runtime readiness output differs from the pinned patch');
  return output;
}

function within(root, candidate) {
  const relative = path.relative(root, candidate);
  return relative !== '' && relative !== '..' && !path.isAbsolute(relative) && !relative.startsWith('..' + path.sep);
}

/** Apply only to the fresh copy identified by its stager, never the source tree. */
export async function applySessionRuntimeReadiness(vendorDir, sourceVendorDir, version = '0.2.1-rc.2') {
  const profile = readinessProfile(version);
  const root = await fs.realpath(path.resolve(vendorDir));
  const source = await fs.realpath(path.resolve(sourceVendorDir));
  if (source === root || within(source, root) || within(root, source)) {
    throw new Error('Session runtime readiness requires disjoint source and runtime-copy directories');
  }
  const filename = path.join(root, bundlePath);
  if (!within(root, await fs.realpath(filename)) || (await fs.lstat(filename)).isSymbolicLink()) {
    throw new Error('Session runtime readiness bundle escapes the runtime copy');
  }
  const bytes = await fs.readFile(filename);
  const recordPath = path.join(root, recordFile);
  let record;
  try {
    if ((await fs.lstat(recordPath)).isSymbolicLink()) throw new Error('Session runtime readiness record must be a file in the runtime copy');
    record = JSON.parse(await fs.readFile(recordPath, 'utf8'));
  } catch (error) {
    if (error.code !== 'ENOENT') throw error;
  }
  if (record !== undefined) {
    if (Object.entries(profile).some(([key, value]) => record[key] !== value) || hash(bytes) !== profile.outputSHA256) {
      throw new Error('Session runtime readiness record does not match the copied bundle');
    }
    return record;
  }
  const output = patchSessionReadinessSource(bytes.toString('utf8'), version);
  record = profile;
  await fs.writeFile(filename, output);
  await fs.writeFile(recordPath, JSON.stringify(record, null, 2) + '\n', {flag: 'wx'});
  return record;
}
