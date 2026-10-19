// Profile-local rc.2 registration of the original dshbot room composition.
import * as original from __ORIGINAL_DSHBOT_MODULE__;
export * from __ORIGINAL_DSHBOT_MODULE__;

export const inject = [...original.inject, 'agentPresets'];
export const roomPresetDefinition = __ORIGINAL_ROOM_DEFINITION__;

export async function apply(ctx, config) {
  // The original global tool registrations must precede eager preset mounting:
  // tools.restrict() refuses unknown names. Await registration before this
  // plugin becomes ACTIVE and its original catalog-restore listener starts.
  const result = original.apply(ctx, config);
  const unregister = await ctx.agentPresets.register(roomPresetDefinition);
  ctx.effect(() => unregister, 'mygo.dshbot-room-preset');
  return result;
}
