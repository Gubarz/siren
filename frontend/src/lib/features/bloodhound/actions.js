// Action bridge: maps BloodHound findings to Sliver operations. Pure
// functions of the ctx bag — no store reads, no side effects at import time.
//
// Every returned action is operator-confirmed (one click minimum); runCommand
// opens the pre-filled command modal rather than executing autonomously.

function isWindowsAgent(agent) {
  return (agent?.OS || '').toLowerCase() === 'windows';
}

function ownedSession(ctx) {
  const { agent, enrichment } = ctx;
  if (!agent) return false;
  return Boolean(enrichment?.owned) && isWindowsAgent(agent);
}

function moveAction(ctx) {
  const { agent, entity } = ctx;
  const owned = Boolean(ownedSession(ctx));
  return {
    label: `Move to ${entity.name || 'host'} from ${agent?.Hostname || 'this agent'}`,
    icon: 'network-wired',
    disabled: !owned,
    reason: owned ? '' : 'Requires an owned Windows session',
    on: () => { if (owned) ctx.runCommand?.(agent.ID, 'psexec'); },
  };
}

// actionsForEntity returns the operator actions for a BH entity in the
// context of the current agent tab.
export function actionsForEntity(ctx) {
  const { entity, enrichment, kerberoastableIDs } = ctx;
  if (!entity?.objectId) return [];
  const items = [];

  if (entity.kind === 'User' && kerberoastableIDs?.has(entity.objectId)) {
    const owned = ownedSession(ctx);
    items.push({
      label: `Kerberoast from ${ctx.agent?.Hostname || 'this agent'}`,
      icon: 'key',
      disabled: !owned,
      reason: owned ? '' : 'Requires an owned Windows session',
      on: () => { if (owned) ctx.runCommand?.(ctx.agent.ID, 'kerberoast'); },
    });
  }

  if (entity.kind === 'Computer' && !entity.owned) {
    items.push(moveAction(ctx));
  }

  if (enrichment?.tierZero) {
    items.push({
      label: 'Add to case',
      icon: 'folder',
      on: () => ctx.addToCase?.open({
        collection: 'bloodhound',
        itemID: entity.objectId,
        label: entity.name || entity.objectId,
      }),
    });
  }

  items.push(
    { label: 'Tag', icon: 'tag', on: () => ctx.openTags?.('bloodhound', entity.objectId, entity.name) },
    { label: 'Comment', icon: 'message-square', on: () => ctx.openComments?.('bloodhound', entity.objectId, entity.name) },
  );
  return items;
}

// actionsForEdge returns actions for a relationship the operator clicked on
// the attack-path graph. Movement edges (AdminTo/HasSession) toward an
// un-owned computer yield the lateral-movement action.
export function actionsForEdge(ctx) {
  const kind = ctx.edge?.label || ctx.edge?.kind || '';
  const movement = kind === 'AdminTo' || kind === 'HasSession';
  if (!movement) return [];
  if (!ctx.entity?.objectId || ctx.entity.kind !== 'Computer' || ctx.entity.owned) return [];
  return [moveAction(ctx)];
}
