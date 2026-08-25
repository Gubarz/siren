// Pure helpers for the sessions/local-admin sections in the agent BloodHound
// tab. Direction itself is decided server-side from the same entity-kind
// string; these helpers only shape display data and copy.

export function isComputerKind(kind) {
  return kind === 'Computer';
}

export function sessionHeading(kind) {
  return isComputerKind(kind) ? 'Users with sessions on this host' : 'Sessions';
}

export function adminHeading(kind) {
  return isComputerKind(kind) ? 'Local admins of this host' : 'Local admin on';
}

// uniqueEntities returns graph nodes deduplicated by ID (group expansion can
// yield duplicate paths), preserving first-seen order. When excludeId is set,
// that node is dropped so a queried entity never lists itself among its own
// relations.
export function uniqueEntities(graph, excludeId = '') {
  const seen = new Set();
  return (Array.isArray(graph?.nodes) ? graph.nodes : []).filter((n) => {
    if (excludeId && (n?.id === excludeId || n?.objectId === excludeId)) return false;
    if (!n?.id || seen.has(n.id)) return false;
    seen.add(n.id);
    return true;
  });
}

// toActionEntity adapts a graph node to the shape actionsForEntity expects.
export function toActionEntity(node) {
  return { ...node, objectId: node?.objectId || node?.id || '', name: node?.label || '' };
}
