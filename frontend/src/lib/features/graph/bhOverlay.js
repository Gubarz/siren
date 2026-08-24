// BloodHound overlay builders for NetworkGraph. Every function takes an
// explicit ctx bag so nothing here reaches into component state.
//
// Each enriched agent gains one BH entity node (kind/owned/tier-zero/
// distance) linked to its agent node. When showEdges is on, the entity's
// shortest path chain to Tier-0 renders as additional dashed nodes/edges.

export const BH_W = 200;
export const BH_H = 64;

function pathChainNode(entity, pathNode, direction) {
  return {
    id: `bh_${pathNode.id}`,
    w: BH_W,
    h: BH_H,
    data: {
      variant: 'bloodhound',
      objectId: pathNode.id,
      label: pathNode.label || pathNode.id,
      kind: pathNode.kind || '',
      owned: Boolean(pathNode.owned),
      tierZero: Boolean(pathNode.tierZero),
      distance: -1,
      chain: true,
      rootEntity: entity.objectId,
      direction,
    },
  };
}

export function addBloodhoundOverlay(rawNodes, rawEdges, ctx) {
  const { agents, enrichment, showEdges, direction } = ctx;
  if (!enrichment) return;

  const existing = new Set(rawNodes.map((n) => n.id));

  for (const agent of agents || []) {
    const enr = enrichment[agent.ID];
    const entity = enr?.entity;
    if (!entity?.objectId) continue;

    const nodeID = `bh_${entity.objectId}`;
    if (!existing.has(nodeID)) {
      existing.add(nodeID);
      rawNodes.push({
        id: nodeID,
        w: BH_W,
        h: BH_H,
        data: {
          variant: 'bloodhound',
          objectId: entity.objectId,
          label: entity.name || entity.objectId,
          kind: entity.kind || '',
          owned: Boolean(enr.owned ?? entity.owned),
          tierZero: Boolean(enr.tierZero ?? entity.tierZero),
          distance: enr.distanceToTierZero ?? -1,
          direction,
        },
      });
    }
    // Correlation edge: agent → its AD entity.
    rawEdges.push({
      id: `e_bh_${agent.ID}_${nodeID}`,
      source: agent.ID,
      target: nodeID,
      style: 'stroke:var(--color-warning-500);stroke-width:1.5;stroke-dasharray:5',
    });

    if (!showEdges) continue;

    const paths = enr.paths || [];
    if (paths.length < 2) continue;

    // Chain from the entity node through the path to Tier-0.
    let previousID = nodeID;
    for (const pathNode of paths) {
      const chainID = `bh_${pathNode.id}`;
      if (!existing.has(chainID)) {
        existing.add(chainID);
        rawNodes.push(pathChainNode(entity, pathNode, direction));
      }
      rawEdges.push({
        id: `e_bhp_${previousID}_${chainID}`,
        source: previousID,
        target: chainID,
        style: 'stroke:var(--color-danger-500);stroke-width:1;stroke-dasharray:4',
      });
      previousID = chainID;
    }
  }
}
