export function integrityBadge(integrity) {
  const level = integrity || ''
  const l = level.toLowerCase()
  if (l.includes('system')) return { variant: 'danger', label: level }
  if (l.includes('high')) return { variant: 'warning', label: level }
  if (l.includes('medium')) return { variant: 'info', label: level }
  if (l.includes('low')) return { variant: 'secondary', label: level }
  return { variant: 'secondary', label: level || 'Unknown' }
}
