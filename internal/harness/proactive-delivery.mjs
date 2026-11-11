import {desktopCall} from './tools.mjs';

const routeFields = ['channel','botId','actor','chatId','incarnation'];
function tasksService(ctx) {
  const service = ctx.get('whaleTasks');
  if (!service?.available || service.closed) throw Error('请先启用鲸鱼娘并连接已授权的飞书任务会话。');
  return service;
}
function routeFor(task) {
  const route = task?.origin?.route;
  if (!route || route.channel !== 'feishu' || routeFields.some(key => !route[key])) return;
  return {id:task.id,title:task.goal || task.id,...Object.fromEntries(routeFields.map(key => [key,route[key]]))};
}

export async function deliveryRoutes(ctx) {
  const service = tasksService(ctx), routes = [], seen = new Set();
  for (const task of service.list()) {
    const route = routeFor(task);
    if (!route) continue;
    const key = JSON.stringify(routeFields.map(key=>route[key]));
    if (seen.has(key)) continue;
    try {
      service.assertImRoute(task.origin.route);
      const adapter = service.channels.get(service.channelKey(task.origin.route));
      if (!adapter || await adapter.status(task.origin.route) !== 'ready') continue;
      service.assertImRoute(task.origin.route);
      if (service.channels.get(service.channelKey(task.origin.route)) !== adapter) continue;
      seen.add(key); routes.push(route);
    } catch { /* Revoked/offline routes cannot be offered for a new approval. */ }
  }
  return routes;
}

export async function sendDelivery(ctx, approvalId) {
  // Claim returns the immutable user-approved destination and text. The caller
  // can supply only an ID; duplicate dispatch cannot claim a second lease.
  const approval = await desktopCall('delivery/claim',{approvalId,hostHome:process.env.DSH_HOME});
  const delivery = approval.delivery;
  let admitted = false, status = 'failed', error = '', messageIds = [], permit;
  try {
    permit = ctx.get('desktopTaskAdmission').begin('native proactive delivery');
    const service = tasksService(ctx);
    const sourceTask = service.store.get(delivery.route.id);
    const route = sourceTask.origin.route;
    const current = routeFor(sourceTask);
    if (!current || routeFields.some(key=>current[key] !== delivery.route[key])) throw Error('飞书会话归属已变化，请重新选择并审批。');
    const key = service.channelKey(route), epoch = service.ownerEpochs.get(key) ?? 0;
    const adapter = service.channels.get(key);
    if (!adapter) throw Error('飞书渠道未连接。');
    service.assertImRoute(route,epoch);
    if (await adapter.status(route) !== 'ready') throw Error('飞书会话已离线或授权已撤销。');
    const admit = () => {
      permit.assertCurrent();
      service.assertImRoute(route,epoch);
      if (ctx.get('whaleTasks') !== service || service.channels.get(key) !== adapter) throw Object.assign(Error('飞书连接已更换。'),{knownUnsent:true});
      admitted = true;
    };
    const receipt = await adapter.send(route,approvalId,delivery.text,admit);
    messageIds = receipt?.providerMessageIds ?? [];
    status = receipt?.outcome === 'sent' && messageIds.length ? 'sent' : 'unknown';
    if (status === 'unknown') error = '飞书发送回执不完整，请核对目标会话。';
  } catch (cause) {
    status = admitted && cause?.knownUnsent !== true ? 'unknown' : 'failed';
    error = String(cause?.message ?? cause);
  }
  try { await desktopCall('delivery/finish',{approvalId,lease:delivery.lease,status,error,messageIds}); }
  finally { permit?.done(); }
  return {status};
}
