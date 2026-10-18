import test from 'node:test';
import assert from 'node:assert/strict';
import {validatePreviewEntitlements, signingTeam} from './macos-preview-bundle.mjs';

test('preview entitlement allowlist rejects every extra grant and nonboolean values', () => {
  assert.doesNotThrow(() => validatePreviewEntitlements({'com.apple.security.app-sandbox': true}));
  for (const value of [{}, null, [], {'com.apple.security.app-sandbox': false}, {'com.apple.security.app-sandbox': 'true'},
    ...['network.client', 'network.server', 'files.user-selected.read-only', 'inherit', 'get-task-allow', 'temporary-exception.files.absolute-path.read-only']
      .map(key => ({'com.apple.security.app-sandbox': true, ['com.apple.security.' + key]: true}))]) {
    assert.throws(() => validatePreviewEntitlements(value));
  }
});
test('signing team rejects absent, ad-hoc, malformed and ambiguous identities', () => {
  assert.equal(signingTeam('Identifier=org.whaleisle.preview\nTeamIdentifier=AB12345678\n'), 'AB12345678');
  for (const value of ['', 'TeamIdentifier=not set', 'TeamIdentifier=A" or true',
    'TeamIdentifier=AB12345678\nTeamIdentifier=AB12345678']) assert.throws(() => signingTeam(value));
});
