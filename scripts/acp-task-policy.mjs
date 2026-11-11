// Native task turns govern external ACP permissions downward-only: ACP
// auto-allow cannot bypass the task inbox, client-side mutations cannot skip
// Host tool dispatch, and task rejections settle waiting ACP permission calls.
const TASK_TOOLS = Symbol.for('whaleisle.nativeTaskTools');

export function nativeTaskTools() {
  return globalThis[TASK_TOOLS];
}

export function isTaskBound(agent) {
  return Boolean(nativeTaskTools()?.binding?.(agent));
}

export function effectiveAutoAllow(conn) {
  if (isTaskBound(conn?.dshAgent)) return false;
  return conn?.agentConfig?.permissionPolicy === 'auto-allow';
}

export function taskRuntimePolicy(conn) {
  const autoAllow = effectiveAutoAllow(conn);
  return {
    runtimeMode: autoAllow ? 'full-access' : 'approval-required',
    cwd: conn?.cwd,
    approvalPolicy: autoAllow ? 'never' : 'on-request',
    sandboxPolicy: autoAllow ? { type: 'dangerFullAccess' } : { type: 'workspaceWrite' },
  };
}

export function guardTaskClientMutation(conn, operation) {
  if (isTaskBound(conn?.dshAgent)) {
    throw new Error(`Task-governed sessions cannot run unapproved ACP client ${operation}`);
  }
}

export function settleTaskPermission(exec, targetAgent, perm) {
  if (!isTaskBound(targetAgent)) return false;
  const approved = Boolean(nativeTaskTools()?.consumeAuthorization?.(exec));
  perm.resolve(approved);
  return true;
}

export function rejectTrackedAcpTool(pendingToolExecutions, exec) {
  const callId = exec?.callId;
  if (!callId) return;
  const tracked = pendingToolExecutions.get(callId);
  pendingToolExecutions.delete(callId);
  const entry = tracked?.entry;
  if (!entry) return;
  entry.taskRejected = true;
  if (entry.pendingPermission) {
    const perm = entry.pendingPermission;
    entry.pendingPermission = null;
    perm.resolve(false);
  }
  if (Array.isArray(entry.waiters)) {
    for (const waiter of entry.waiters.splice(0)) waiter();
  }
}

export function registerTaskAcpRejector(ctx, pendingToolExecutions) {
  if (typeof ctx?.effect !== 'function') return;
  ctx.effect(() => nativeTaskTools()?.setAcpRejector?.((exec) => {
    rejectTrackedAcpTool(pendingToolExecutions, exec);
  }));
}
