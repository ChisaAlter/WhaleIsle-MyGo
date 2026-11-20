// A download that fails must say which proxy it went through — this machine's
// HTTPS_PROXY listens but cannot reach codeload, and a message that reads like a
// record failure sends the next person after the pin instead of the network.
import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {download} from './fork.mjs';

const url = 'https://codeload.github.com/deepseek-ai/deepseek-harness/tar.gz/ddac96760b6edf8fa0fe17cbdcf0b512f4c12b2c';
const refused = {status: 7, stderr: 'curl: (7) Failed to connect to codeload.github.com port 443 after 12 ms'};
const destination = t => path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'harness-fork-download-')), 'archive.tar.gz');

test('a failed download names the proxy curl picked up from the environment', t => {
  assert.throws(
    () => download(url, destination(t), {runner: () => refused, environment: {HTTPS_PROXY: 'http://127.0.0.1:10808'}}),
    error => {
      assert.match(error.message, /went through the proxy at http:\/\/127\.0\.0\.1:10808 \(HTTPS_PROXY\)/);
      assert.match(error.message, /retry with --no-proxy/);
      assert.match(error.message, /curl: \(7\) Failed to connect/);
      return true;
    },
  );
});

test('a failed download names the proxy it was told to use', t => {
  assert.throws(
    () => download(url, destination(t), {proxy: 'http://127.0.0.1:7890', runner: () => refused, environment: {}}),
    error => {
      assert.match(error.message, /the proxy at http:\/\/127\.0\.0\.1:7890 \(--proxy\)/);
      return true;
    },
  );
});

test('without a proxy the failure asks for one', t => {
  assert.throws(
    () => download(url, destination(t), {runner: () => refused, environment: {}}),
    error => {
      assert.match(error.message, /Pass --proxy <url> when this network needs one/);
      assert.doesNotMatch(error.message, /no-proxy/);
      return true;
    },
  );
});

test('--no-proxy downloads directly and does not blame a proxy', t => {
  const seen = [];
  const target = destination(t);
  const result = download(url, target, {
    noProxy: true,
    environment: {HTTPS_PROXY: 'http://127.0.0.1:10808'},
    runner: (command, args) => {
      seen.push({command, args});
      return {status: 0, stderr: ''};
    },
  });
  assert.equal(result, target);
  assert.equal(seen.length, 1);
  assert.equal(seen[0].command, 'curl');
  assert.equal(seen[0].args[seen[0].args.indexOf('--noproxy') + 1], '*', 'curl must ignore the environment proxy, not just skip --proxy');
  assert.ok(!seen[0].args.includes('--proxy'));
});

test('a failure with --no-proxy does not blame a proxy the operator bypassed', t => {
  assert.throws(
    () => download(url, destination(t), {noProxy: true, runner: () => refused, environment: {HTTPS_PROXY: 'http://127.0.0.1:10808'}}),
    error => {
      assert.match(error.message, /directly failed/);
      assert.match(error.message, /This network may need --proxy <url>/);
      assert.doesNotMatch(error.message, /127\.0\.0\.1:10808/);
      return true;
    },
  );
});
