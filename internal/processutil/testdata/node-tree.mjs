// Run only beneath a Go processutil.BackgroundTree parent. No positive control.
// Interface: node node-tree.mjs TEST_EXECUTABLE
// Internal:  node node-tree.mjs --child TEST_EXECUTABLE
import { execFile } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const helperRole = 'WHALEISLE_PROCESSUTIL_TEST_HELPER';
const script = fileURLToPath(import.meta.url);

// Deliberately omit windowsHide, detached, shell, and all stdio options.
// execFile's ordinary piped stdout/stderr and default process flags are the
// behavior under test. A nonzero leaf exit is evidence, not a fixture failure.
function capture(file, args, options) {
  return new Promise((resolve, reject) => {
    let child;
    const complete = (error, stdout, stderr) => {
      if (error && (typeof error.code !== 'number' || error.signal || error.killed)) {
        reject(new Error(`fixture child failed to complete: ${String(error.code ?? error.signal ?? 'unknown')}`));
        return;
      }
      resolve({
        pid: child.pid,
        stdout,
        stderr,
        exitCode: error ? error.code : 0,
      });
    };
    child = options === undefined
      ? execFile(file, args, complete)
      : execFile(file, args, options, complete);
  });
}

function leaf(testExecutable) {
  return capture(testExecutable, ['-test.run=^TestBackgroundHelperProcess$'], {
    env: { ...process.env, [helperRole]: 'child_failure' },
  });
}

async function main() {
  const args = process.argv.slice(2);
  const childMode = args[0] === '--child';
  if (args.length !== (childMode ? 2 : 1) || !args.at(-1)) {
    throw new Error('Usage: node node-tree.mjs [--child] TEST_EXECUTABLE');
  }
  const testExecutable = args.at(-1);
  if (childMode) {
    return { nodePid: process.pid, leaf: await leaf(testExecutable) };
  }
  const direct = await leaf(testExecutable);
  const nestedProcess = await capture(process.execPath, [script, '--child', testExecutable]);
  if (nestedProcess.exitCode !== 0) {
    throw new Error(`nested Node exited ${nestedProcess.exitCode}`);
  }
  let nested;
  try {
    nested = JSON.parse(nestedProcess.stdout);
  } catch {
    throw new Error('Nested Node did not emit its JSON report');
  }
  if (nested?.nodePid !== nestedProcess.pid || !nested.leaf) {
    throw new Error('Nested Node report does not match the owned process');
  }
  return { parentPid: process.pid, direct, nested, nestedProcess };
}

try {
  process.stdout.write(JSON.stringify(await main()) + '\n');
} catch (error) {
  process.stderr.write(`node-tree fixture: ${error.message}\n`);
  process.exitCode = 125;
}
