'use strict';

const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

// Physical storage is independent of Project/workstream identity. Old receipts
// retain their original layout; no directory is moved while an owner may use it.
function worktreePath(home, projectId, workstreamId, layout) {
  if (layout === undefined || layout === null) return path.join(home, 'project-worktrees', projectId, workstreamId);
  if (layout !== 'compact-v1') throw new Error('未知 Project 工作目录布局');
  const profile = crypto.createHash('sha256').update(home.toLowerCase()).digest('hex').slice(0, 16);
  const key = crypto.createHash('sha256').update(JSON.stringify([projectId, workstreamId])).digest('hex').slice(0, 32);
  // Keep the physical cwd away from both long DSH_HOME prefixes and business
  // IDs. A distinct home bucket prevents isolated profiles sharing worktrees.
  return path.join(os.homedir(), '.whaleisle', 'worktrees', profile, `w-${key}`);
}

function checkedWorktreePath(home, projectId, workstreamId, layout) {
  const target = path.resolve(worktreePath(home, projectId, workstreamId, layout));
  const same = (a, b) => process.platform === 'win32' ? a.toLowerCase() === b.toLowerCase() : a === b;
  for (let current = target; ; current = path.dirname(current)) {
    try {
      const info = fs.lstatSync(current);
      if (info.isSymbolicLink() || !info.isDirectory() || !same(fs.realpathSync.native(current), current)) {
        throw new Error('Project 存储路径被链接或文件替换');
      }
    } catch (error) { if (error.code !== 'ENOENT') throw error; }
    if (path.dirname(current) === current) break;
  }
  return target;
}

module.exports = { worktreePath, checkedWorktreePath };
