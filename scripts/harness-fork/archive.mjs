// Read an upstream tarball into the exact entries the pin's trees are made of.
//
// The record is compared by tree hash, so every byte that goes into a tree has to
// come from the archive: path bytes, the mode class the tar header states, and the
// content. Asking the filesystem afterwards is what made the replay depend on the
// host — Git for Windows records no execute bit, tar tools disagree about Windows
// paths and about non-ASCII header bytes, and an unprivileged account cannot
// create the links a tag ships. None of that is an identity question.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import zlib from 'node:zlib';
import {spawnSync} from 'node:child_process';

const BLOCK = 512;
const META = new Set(['x', 'g', 'L', 'K']);

function field(block, start, length) {
  const raw = block.subarray(start, start + length);
  const end = raw.indexOf(0);
  return raw.subarray(0, end === -1 ? raw.length : end);
}

function numeric(block, start, length) {
  if (block[start] & 0x80) {
    let value = 0;
    for (let index = start + 1; index < start + length; index++) value = value * 256 + block[index];
    return value;
  }
  const text = field(block, start, length).toString('latin1').trim();
  return text ? parseInt(text, 8) || 0 : 0;
}

function paxRecords(content) {
  const records = new Map();
  let offset = 0;
  while (offset < content.length) {
    const space = content.indexOf(0x20, offset);
    if (space === -1) break;
    const length = parseInt(content.subarray(offset, space).toString('latin1'), 10);
    if (!Number.isFinite(length) || length <= 0) break;
    const record = content.subarray(space + 1, offset + length - 1);
    const equals = record.indexOf(0x3d);
    if (equals !== -1) records.set(record.subarray(0, equals).toString('latin1'), record.subarray(equals + 1));
    offset += length;
  }
  return records;
}

const padding = size => (BLOCK - (size % BLOCK)) % BLOCK;

function header(block) {
  const prefix = field(block, 345, 155).toString('utf8');
  const name = field(block, 0, 100).toString('utf8');
  return {
    name: prefix ? `${prefix}/${name}` : name,
    mode: numeric(block, 100, 8),
    size: numeric(block, 124, 12),
    type: String.fromCharCode(block[156] || 0x30),
    linkname: field(block, 157, 100).toString('utf8'),
  };
}

/**
 * Walk a tar (or tar.gz) once. `onEntry` gets every non-directory entry and may
 * return `{write, close}` to receive its content; returning nothing discards it.
 */
export function scan(archive, onEntry) {
  return new Promise((resolve, reject) => {
    const raw = fs.createReadStream(archive);
    const stream = archive.endsWith('.tar') ? raw : raw.pipe(zlib.createGunzip());
    let pending = Buffer.alloc(0);
    let state = 'header';
    let sink = null;
    let size = 0;
    let left = 0;
    let meta = new Map();
    let globalMeta = new Map();
    let longName = null;
    let longLink = null;
    let metaEntry = null;
    let metaBuffer = Buffer.alloc(0);

    const settle = () => {
      if (sink) {
        sink.close();
        sink = null;
      }
    };

    const fail = error => {
      settle();
      stream.destroy();
      reject(error);
    };

    const closeMeta = () => {
      if (metaEntry.type === 'x') meta = paxRecords(metaBuffer);
      else if (metaEntry.type === 'g') globalMeta = paxRecords(metaBuffer);
      else if (metaEntry.type === 'L') longName = field(metaBuffer, 0, metaBuffer.length).toString('utf8');
      else longLink = field(metaBuffer, 0, metaBuffer.length).toString('utf8');
      metaEntry = null;
      metaBuffer = Buffer.alloc(0);
    };

    const consume = () => {
      for (;;) {
        if (state === 'header') {
          if (pending.length < BLOCK) return;
          const block = pending.subarray(0, BLOCK);
          pending = pending.subarray(BLOCK);
          if (block.every(value => value === 0)) {
            settle();
            continue;
          }
          const entry = header(block);
          size = entry.size;
          if (META.has(entry.type)) {
            const effective = new Map([...globalMeta, ...meta]);
            if (effective.has('size')) size = Number(effective.get('size').toString('latin1'));
            metaEntry = entry;
            state = 'meta';
            left = size;
            if (left === 0) {
              closeMeta();
              state = 'pad';
              left = 0;
            }
            continue;
          }
          const effective = new Map([...globalMeta, ...meta]);
          if (effective.has('path')) entry.name = effective.get('path').toString('utf8');
          if (effective.has('size')) size = Number(effective.get('size').toString('latin1'));
          if (effective.has('linkpath')) entry.linkname = effective.get('linkpath').toString('utf8');
          if (longName !== null) entry.name = longName;
          if (longLink !== null) entry.linkname = longLink;
          meta = new Map();
          longName = null;
          longLink = null;
          try {
            settle();
            sink = onEntry({name: entry.name, mode: entry.mode, type: entry.type, size, linkname: entry.linkname});
          } catch (error) {
            fail(error);
            return;
          }
          state = sink ? (size > 0 ? 'body' : 'pad') : 'skip';
          left = sink ? (size > 0 ? size : padding(size)) : size + padding(size);
          if (left === 0) {
            settle();
            state = 'header';
          }
          continue;
        }

        if (state === 'body' || state === 'meta') {
          const take = Math.min(left, pending.length);
          if (take) {
            if (state === 'body') sink.write(pending.subarray(0, take));
            else metaBuffer = Buffer.concat([metaBuffer, pending.subarray(0, take)]);
            left -= take;
            pending = pending.subarray(take);
          }
          if (left > 0) return;
          if (state === 'body') settle();
          else closeMeta();
          state = 'pad';
          left = padding(size);
          if (left === 0) state = 'header';
          continue;
        }

        if (state === 'pad' || state === 'skip') {
          const take = Math.min(left, pending.length);
          left -= take;
          pending = pending.subarray(take);
          if (left > 0) return;
          state = 'header';
          continue;
        }
      }
    };

    stream.on('data', chunk => {
      pending = pending.length ? Buffer.concat([pending, chunk]) : chunk;
      consume();
    });
    stream.on('error', fail);
    stream.on('end', () => {
      settle();
      resolve();
    });
  });
}


function git(args, {cwd, env, input, allow = false} = {}) {
  const result = spawnSync('git', args, {
    cwd,
    env: {...process.env, ...env},
    input,
    windowsHide: true,
    maxBuffer: 256 * 1024 * 1024,
  });
  if (result.error) throw result.error;
  if (result.status !== 0 && !allow) {
    throw new Error(`git ${args.join(' ')} failed: ${(result.stderr?.toString() ?? '').trim() || `exit ${result.status}`}`);
  }
  return result;
}
const escaping = /^([/\]|[A-Za-z]:)|(^|\/)\.\.(\/|$)/;
const config = ['-c', 'core.autocrlf=false', '-c', 'core.eol=lf', '-c', 'core.safecrlf=false', '-c', 'core.precomposeunicode=false', '-c', 'core.symlinks=false'];

/**
 * Put an archive's entries on disk under the paths the record names. Links are
 * reported, never created: whether a host can hold one is a materialize question,
 * and the identity must not depend on the answer.
 */
export async function readArchive({archive, destination, strip = 0, prefix = '', label = 'archive', log = () => {}}) {
  fs.rmSync(destination, {recursive: true, force: true});
  fs.mkdirSync(destination, {recursive: true});
  const entries = [];
  const folded = new Map();
  await scan(archive, entry => {
    if (entry.type === '5' || entry.name.endsWith('/')) return null;
    if (entry.type === '1') throw new Error(`${label} contains a hard link (${entry.name}); the record cannot describe one`);
    const parts = entry.name.split('/');
    if (parts.length <= strip) return null;
    let relative = parts.slice(strip).join('/');
    if (prefix) {
      if (!relative.startsWith(prefix)) return null;
      relative = relative.slice(prefix.length);
    }
    if (!relative) return null;
    if (escaping.test(relative)) throw new Error(`${label} has an escaping path: ${entry.name}`);
    const name = relative.normalize('NFC');
    const key = name.toLowerCase();
    if (folded.has(key) && folded.get(key) !== name) {
      throw new Error(`${label} has two paths that differ only by case (${folded.get(key)}, ${name}); a host that folds case cannot hold both`);
    }
    folded.set(key, name);
    if (entry.type === '2') {
      log(`${label}: link ${name} -> ${entry.linkname}`);
      entries.push({path: name, mode: '120000', target: entry.linkname});
      return null;
    }
    if (entry.type !== '0') throw new Error(`${label} has an unsupported entry type ${entry.type} at ${entry.name}`);
    entries.push({path: name, mode: entry.mode & 0o111 ? '100755' : '100644'});
    const file = path.join(destination, ...name.split('/'));
    fs.mkdirSync(path.dirname(file), {recursive: true});
    const fd = fs.openSync(file, 'w');
    return {write: chunk => fs.writeSync(fd, chunk), close: () => fs.closeSync(fd)};
  });
  return {entries};
}

/** The same read, for consumers that need real files rather than a tree. */
export async function extractTo({archive, destination, strip = 0, prefix = ''}) {
  const {entries} = await readArchive({archive, destination, strip, prefix});
  return {files: entries.filter(entry => !entry.target).length};
}

/**
 * Turn one archive into the tree the pin records for it. The mode class and the
 * link targets come from the tar headers; the blob of each regular file comes from
 * git applying the archive's own .gitattributes, so a `.cmd` the tag stores as
 * CRLF still lands as the LF blob the record holds.
 * @returns {Promise<{tree: string, entries: number}>}
 */
export async function importArchiveTree({archive, repo, strip = 0, prefix = '', expectTree, label, work, log = () => {}}) {
  const scratch = path.join(work, 'import', `${label.replace(/[^\w.-]+/g, '-')}-${process.pid}-${Date.now()}`);
  const tree = path.join(scratch, 'tree');
  const staged = path.join(os.tmpdir(), `harness-fork-staged-${process.pid}-${Date.now()}.index`);
  const recorded = path.join(os.tmpdir(), `harness-fork-record-${process.pid}-${Date.now()}.index`);
  const base = {GIT_DIR: path.join(repo, '.git')};
  fs.rmSync(scratch, {recursive: true, force: true});
  fs.rmSync(staged, {force: true});
  fs.rmSync(recorded, {force: true});
  try {
    const {entries} = await readArchive({archive, destination: tree, strip, prefix, label, log});
    git([...config, 'add', '-A', '-f', '.'], {cwd: tree, env: {...base, GIT_WORK_TREE: tree, GIT_INDEX_FILE: staged}});
    const stagedBlobs = new Map();
    for (const line of git(['-c', 'core.quotePath=false', 'ls-files', '-s'], {cwd: tree, env: {...base, GIT_INDEX_FILE: staged}}).stdout.toString().split('\n').filter(Boolean)) {
      const match = /^\d+ ([0-9a-f]{40}) \d+\t(.*)$/.exec(line);
      if (match) stagedBlobs.set(match[2], match[1]);
    }
    const links = entries.filter(entry => entry.target !== undefined);
    const linkBlobs = new Map();
    if (links.length) {
      const list = path.join(scratch, 'links.txt');
      const files = links.map((entry, position) => {
        const file = path.join(scratch, `link-${position}`);
        fs.writeFileSync(file, entry.target);
        return file;
      });
      fs.writeFileSync(list, `${files.join('\n')}\n`);
      const shas = git(['hash-object', '-w', '--stdin-paths'], {cwd: repo, env: base, input: fs.readFileSync(list)})
        .stdout.toString().trim().split('\n');
      links.forEach((entry, position) => linkBlobs.set(entry, shas[position]));
    }
    const missing = [];
    const info = entries.map(entry => {
      const sha = entry.target === undefined ? stagedBlobs.get(entry.path) : linkBlobs.get(entry);
      if (!sha) {
        missing.push(entry.path);
        return null;
      }
      return `${entry.mode} ${sha}\t${entry.path}`;
    }).filter(Boolean).join('\n') + '\n';
    if (missing.length) throw new Error(`${label}: no blob for ${missing.length} path(s), first ${missing[0]}`);
    git(['update-index', '--add', '--index-info'], {cwd: repo, env: {...base, GIT_INDEX_FILE: recorded}, input: Buffer.from(info, 'utf8')});
    const result = git(['write-tree'], {cwd: repo, env: {...base, GIT_INDEX_FILE: recorded}}).stdout.toString().trim();
    if (expectTree && result !== expectTree) {
      const modes = entries.reduce((counts, entry) => ({...counts, [entry.mode]: (counts[entry.mode] ?? 0) + 1}), {});
      throw new Error(`${label} content is not the recorded tree: the archive imports ${result} with ${entries.length} entries (${JSON.stringify(modes)}), the pin records ${expectTree}. The download is not the source this fork was recorded against; do not record over it.`);
    }
    return {tree: result, entries: entries.length};
  } finally {
    fs.rmSync(staged, {force: true});
    fs.rmSync(recorded, {force: true});
    fs.rmSync(scratch, {recursive: true, force: true});
  }
}

export {git};
