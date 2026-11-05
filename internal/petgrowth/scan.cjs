/*
MIT License

Copyright (c) 2026 Chisa

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/
'use strict';
// Adapted from WhaleIsle-Electron src/main/pet-growth.js (MIT). Original
// settlement identity and zstd framing retained. The native wrapper supplies
// only its own DSH sessions directory; no model, account or network calls.
const fs = require('node:fs');
const path = require('node:path');
const zlib = require('node:zlib');
const { createHash } = require('node:crypto');
let decodedBytes = 0;
function usageTokens(usage, includeCache = false) {
  if (!usage || typeof usage !== 'object') throw new Error('invalid usage');
  let n = 0;
  // Food is fresh work only. Cached context is already metered separately.
  for (const k of includeCache ? ['inputTokens', 'cacheReadTokens', 'cacheWriteTokens', 'outputTokens'] : ['inputTokens', 'outputTokens']) {
    const v = usage[k] ?? 0;
    if (!Number.isSafeInteger(v) || v < 0 || !Number.isSafeInteger(n + v)) throw new Error('invalid usage count');
    n += v;
  }
  return n;
}
function usageSampleOf(event) {
  const data = event?.data;
  if (!data || typeof data !== 'object') {
    return undefined;
  }
  if (event.type === 'assistant/message') {
    return data.usage !== undefined
      ? { turn: data.turn, step: data.step, usage: data.usage }
      : undefined;
  }
  if (event.type === 'assistant/attempt' || event.type === 'assistant/chunk') {
    const chunk = data.chunk && data.chunk.type === 'usage' ? data.chunk
      : Array.isArray(data.stream) ? [...data.stream].reverse().find((c) => c?.type === 'usage')
        : undefined;
    return chunk && chunk.usage !== undefined
      ? { turn: data.turn, step: data.step, usage: chunk.usage }
      : undefined;
  }
  if (event.type === 'compaction/summary' && data.usage !== undefined) {
    // Compaction usage is billed separately — still real spend, so she can
    // eat it, keyed to its own slot so it never collides with a step.
    return { turn: -1, step: event.seq ?? 0, usage: data.usage };
  }
  return undefined;
}

// Fold one decoded log body into `lastWins`: last sample wins per
// (turn,step) — a retried attempt's later settlement replaces the earlier
// one, and a streamed usage chunk never outranks the final settlement.
function inheritedCut(text) {
  const lines=text.split('\n');let header;
  for(const line of lines){if(!line||line.charCodeAt(0)!==123)continue;try{header=JSON.parse(line);}catch{}break;}
  if(header?.type!=='session'||header.isSeeded!==true)return 0;
  let cut=Number.isSafeInteger(header.seedLength)&&header.seedLength>0?header.seedLength:0;
  for(const line of lines){if(!line.includes('session/end-seed'))continue;let event;try{event=JSON.parse(line);}catch{continue;}
    if(event?.type==='session/end-seed'&&event.data?.inherited===true&&Number.isSafeInteger(event.seq))cut=Math.max(cut,event.seq);
  }
  return cut;
}
function logGeneration(name) {const m=/^session(?:\.v([1-9][0-9]*))?\.jsonl\.zstd$/.exec(name);return m?(m[1]===undefined?0:Number(m[1])):-1;}
function foldLogInto(text, lastWins, cut=inheritedCut(text), cacheInclusive) {
  for (const line of text.split('\n')) {
    if (!line || line.charCodeAt(0) !== 123) {
      continue;
    }
    let event;
    try {
      event = JSON.parse(line);
    } catch {
      throw new Error('incomplete log event');
    }
    if(cut>0&&Number.isSafeInteger(event.seq)&&event.seq<cut)continue;
    const sample = usageSampleOf(event);
    if (sample === undefined) {
      continue;
    }
    if (!Number.isSafeInteger(sample.turn) || !Number.isSafeInteger(sample.step)) throw new Error('invalid settlement identity');
    const key = `${sample.turn}:${sample.step}`;
    lastWins.set(key, usageTokens(sample.usage));
    // Old MyGo scans counted these cache fields as food. Keep a separate,
    // non-crediting amount only to identify a proven old settlement watermark.
    if (cacheInclusive) {
      cacheInclusive.delete(key);
      try { cacheInclusive.set(key, usageTokens(sample.usage, true)); } catch {}
    }
  }
}

function tokensInLog(text) {
  const lastWins = new Map();
  foldLogInto(text, lastWins);
  let total = 0;
  for (const n of lastWins.values()) {
    total += n;
  }
  return total;
}

// Session logs are appended one zstd frame per event batch — a single-frame
// decode only sees the session header. Walk frame boundaries (magic +
// header + blocks + optional checksum) and decompress each frame.
function zstdFrameSize(buf, off) {
  let pos = off + 4; // caller already matched the magic
  if (pos >= buf.length) {
    return 0;
  }
  const fhd = buf[pos];
  pos += 1;
  const fcsFlag = fhd >> 6;
  const singleSeg = (fhd >> 5) & 1;
  const checksum = (fhd >> 2) & 1;
  if (!singleSeg) {
    pos += 1; // window descriptor
  }
  pos += [0, 1, 2, 4][fhd & 3]; // dictionary id
  pos += fcsFlag === 0 ? (singleSeg ? 1 : 0) : [0, 2, 4, 8][fcsFlag];
  for (;;) {
    if (pos + 3 > buf.length) {
      return 0;
    }
    const bh = buf[pos] | (buf[pos + 1] << 8) | (buf[pos + 2] << 16);
    pos += 3;
    const last = bh & 1;
    const type = (bh >> 1) & 3;
    // RLE blocks store a single byte; raw/compressed/treeless store `size`.
    pos += type === 1 ? 1 : (bh >> 3);
    if (last) {
      break;
    }
  }
  if (checksum) {
    pos += 4;
  }
  return pos <= buf.length ? pos - off : 0;
}

function zstdDecompressAll(buf) {
  const parts = [];
  let off = 0;
  while (off + 4 <= buf.length) {
    const magic = buf.readUInt32LE(off);
    if (magic === 0xFD2FB528) {
      const size = zstdFrameSize(buf, off);
      if (!size) throw new Error('incomplete zstd frame');
      const part = zlib.zstdDecompressSync(buf.subarray(off, off + size), { maxOutputLength: 256 * 1024 * 1024 });
      decodedBytes += part.length;
      if (decodedBytes > 512 * 1024 * 1024) throw Object.assign(new Error('scan decode limit'), { decodeLimit: true });
      parts.push(part);
      off += size;
    } else if ((magic & 0xFFFFFFF0) === 0x184D2A50) {
      if (off + 8 > buf.length) throw new Error('incomplete skippable frame');
      off += 8 + buf.readUInt32LE(off + 4); // skippable frame
      if (off > buf.length) throw new Error('incomplete skippable frame');
    } else {
      throw new Error('invalid zstd frame');
    }
  }
  if (off !== buf.length) throw new Error('incomplete zstd frame');
  return Buffer.concat(parts);
}


function scan(root) {
  if (!path.isAbsolute(root) || typeof zlib.zstdDecompressSync !== 'function') throw new Error('scanner unavailable');
  // Resolve only the application-authorized root. Reject symlinks/junctions
  // below it so unrelated application logs cannot enter this local ledger.
  const rootStat = fs.lstatSync(root);
  if (!rootStat.isDirectory() || rootStat.isSymbolicLink()) throw new Error('invalid sessions root');
  const realRoot = fs.realpathSync(root);
  // Session logs only grow, and a directory's settlements depend only on its own
  // log files. The native wrapper passes each directory's signature from its
  // last scan (PET_SCAN_PRIOR); an unchanged directory is reported, not decoded.
  // When the decode budget runs out, the remaining changed directories wait for
  // the next scan instead of discarding the ones already decoded.
  const prior = typeof PET_SCAN_PRIOR === 'object' && PET_SCAN_PRIOR !== null ? PET_SCAN_PRIOR : {};
  const stack = [realRoot], out = [];
  let files = 0, dirs = 0, complete = true;
  while (stack.length) {
    const dir = stack.pop();
    if (++dirs > 50000) throw new Error('scan directory limit');
    const ds = fs.lstatSync(dir), dr = path.relative(realRoot, fs.realpathSync(dir));
    if (!ds.isDirectory() || ds.isSymbolicLink() || path.isAbsolute(dr) || dr === '..' || dr.startsWith('..'+path.sep)) throw new Error('invalid session directory');
    const entries = fs.readdirSync(dir, {withFileTypes:true}).sort((a,b)=>a.name < b.name ? -1 : a.name > b.name ? 1 : 0);
    const logs = [];
    for (const ent of entries) {
      const file = path.join(dir,ent.name);
      if (ent.isSymbolicLink()) continue;
      if (ent.isDirectory()) { stack.push(file); continue; }
      if (!ent.isFile() || logGeneration(ent.name)<0) continue;
      if (++files > 20000) throw new Error('scan file limit');
      const st = fs.lstatSync(file), real = fs.realpathSync(file);
      const rel = path.relative(realRoot, real);
      if (!st.isFile() || st.isSymbolicLink() || path.isAbsolute(rel) || rel === '..' || rel.startsWith('..'+path.sep) || st.size > 128*1024*1024) throw new Error('invalid session file');
      logs.push({ file, st });
    }
    if (logs.length === 0) continue;
    logs.sort((a,b)=>logGeneration(path.basename(b.file))-logGeneration(path.basename(a.file)));
    logs.splice(1); // exactly one authoritative canonical generation per session
    const sig = JSON.stringify(logs.map(({ file, st }) => [path.basename(file), st.size, st.mtimeMs, String(st.ino)]));
    if (prior[dr] === sig) { out.push({ r: dr, s: sig, d: null }); continue; }
    if (!complete) continue;
    const lastWins = new Map(), cacheInclusive = new Map();
    try {
      for (const { file, st } of logs) {
        const fd = fs.openSync(file, fs.constants.O_RDONLY | (fs.constants.O_NOFOLLOW || 0));
        try {
          const actual = fs.fstatSync(fd);
          if (!actual.isFile() || actual.size > 128*1024*1024 || actual.ino !== st.ino || actual.dev !== st.dev) throw new Error('session file changed');
          const body = fs.readFileSync(fd);
          if (body.length > 128*1024*1024) throw new Error('scan file limit');
          foldLogInto(zstdDecompressAll(body).toString('utf8'), lastWins, undefined, cacheInclusive);
        } finally { fs.closeSync(fd); }
      }
    } catch (err) {
      if (err && err.decodeLimit) { complete = false; continue; }
      throw err;
    }
    const samples = {}, cached = {};
    for (const [key,n] of lastWins) {
      const id = createHash('sha256').update(JSON.stringify([path.basename(dir),key])).digest('hex');
      samples[id] = n;
      if (cacheInclusive.get(key) > n) cached[id] = cacheInclusive.get(key);
    }
    out.push({ r: dr, s: sig, d: samples, c: cached });
  }
  return { dirs: out, complete };
}
try {
  process.stdout.write(JSON.stringify(scan(process.argv[2])));
} catch {
  // Do not send local paths, message text or exception bodies to the renderer.
  process.stderr.write('Pet usage scan unavailable or incomplete.');
  process.exitCode = 1;
}
