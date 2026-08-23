// Pure chip derivation for enrichment payloads. Kept component-free so the
// sessions table and the per-agent tab share one definition.

// Returns null when there is nothing to show (no entity resolved or no
// connection), otherwise a chip descriptor:
//   { kind: 'tierZero'|'owned'|'unreached', label, title }
export function enrichmentChip(enrichment) {
  if (!enrichment?.entity?.objectId) return null;
  if (enrichment.tierZero || enrichment.distanceToTierZero === 0) {
    return {
      kind: 'tierZero',
      label: 'T0',
      title: 'This entity is Tier-0',
    };
  }
  if (enrichment.owned) {
    return {
      kind: 'owned',
      label: 'OWNED',
      title: 'BloodHound marks this entity as owned',
    };
  }
  if (enrichment.distanceToTierZero > 0) {
    return {
      kind: 'tierZero',
      label: `T0·${enrichment.distanceToTierZero}`,
      title: pathTitle(enrichment.paths),
    };
  }
  return { kind: 'unreached', label: '—', title: 'No path to Tier-0 found' };
}

export function pathTitle(paths) {
  if (!paths || paths.length === 0) return 'No path to Tier-0 found';
  const labels = paths.map((n) => n.label || n.id).join(' → ');
  return `Path to Tier-0: ${labels}`;
}
