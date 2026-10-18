'use strict';

// Only the original runtime's public device fields cross the private bridge.
// Never forward store records, raw errors, device secrets, or token hashes.
function publicPhoneDevices(snapshot) {
  if (!Array.isArray(snapshot?.devices)) return [];
  return snapshot.devices.filter(device =>
    device && typeof device.id === 'string' && device.id.length > 0 && device.id.length <= 256
  ).map(device => ({
    id: device.id,
    name: typeof device.name === 'string' ? device.name.slice(0, 120) : device.id,
    createdAt: typeof device.createdAt === 'string' ? device.createdAt.slice(0, 64) : '',
    lastSeenAt: typeof device.lastSeenAt === 'string' ? device.lastSeenAt.slice(0, 64) : '',
  }));
}

// This adapter invokes DshdRemote's public API only. Rotation renews the offer;
// it never deletes older bootstrap tokens or changes sticky device identities.
async function managePhoneDevice(remote, command, hooks) {
  const action = command.operation;
  if (!['rename', 'unbind', 'rotate'].includes(action)) throw new Error('invalid-action');
  if (!remote.daemon) throw new Error('runtime-offline');
  if (action !== 'rotate') {
    if (typeof command.deviceId !== 'string' || !publicPhoneDevices(remote.snapshot()).some(device => device.id === command.deviceId)) {
      throw new Error('device-missing');
    }
  }
  if (action === 'rename') {
    if (typeof command.name !== 'string' || !command.name.trim() || command.name.trim().length > 120 || /[\u0000-\u001f\u007f]/.test(command.name)) {
      throw new Error('invalid-name');
    }
    await remote.renameDevice(command.deviceId, command.name.trim());
    const renamed = publicPhoneDevices(remote.snapshot()).find(device => device.id === command.deviceId);
    if (!renamed || renamed.name !== command.name.trim()) throw new Error('rename-failed');
    return;
  }
  hooks.hideOffer();
  if (action === 'rotate') {
    await remote.rotateToken();
    return;
  }
  // DshdRemote owns this daemon. Stopping closes already authenticated sockets
  // and prevents its in-memory store from overwriting the revocation. Other
  // paired devices keep their credentials and may reconnect after sync.
  hooks.revokeGit();
  await remote.stopDaemon();
  await remote.unbindDevice(command.deviceId);
  if (publicPhoneDevices(remote.snapshot()).some(device => device.id === command.deviceId)) {
    throw new Error('unbind-failed');
  }
  hooks.applied();
  await remote.sync();
}
