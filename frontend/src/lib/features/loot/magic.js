const signatures = [
  { sig: [0x89, 0x50, 0x4E, 0x47], mime: 'image/png', label: 'PNG image' },
  { sig: [0xFF, 0xD8, 0xFF], mime: 'image/jpeg', label: 'JPEG image' },
  { sig: [0x47, 0x49, 0x46, 0x38], mime: 'image/gif', label: 'GIF image' },
  { sig: [0x42, 0x4D], mime: 'image/bmp', label: 'BMP image' },
  { sig: [0x25, 0x50, 0x44, 0x46], mime: 'application/pdf', label: 'PDF document' },
  { sig: [0x50, 0x4B, 0x03, 0x04], mime: 'application/zip', label: 'ZIP archive / Office' },
  { sig: [0x1F, 0x8B], mime: 'application/gzip', label: 'GZIP archive' },
  { sig: [0x7F, 0x45, 0x4C, 0x46], mime: 'application/x-elf', label: 'ELF binary' },
  { sig: [0x4D, 0x5A], mime: 'application/x-dosexec', label: 'PE / DOS executable' },
  { sig: [0xFE, 0xED, 0xFA, 0xCE], mime: 'application/x-mach-o', label: 'Mach-O 32-bit' },
  { sig: [0xFE, 0xED, 0xFA, 0xCF], mime: 'application/x-mach-o', label: 'Mach-O 64-bit' },
  { sig: [0xCE, 0xFA, 0xED, 0xFE], mime: 'application/x-mach-o', label: 'Mach-O 32-bit (reversed)' },
  { sig: [0xCF, 0xFA, 0xED, 0xFE], mime: 'application/x-mach-o', label: 'Mach-O 64-bit (reversed)' },
  { sig: [0x23, 0x21], mime: 'text/x-script', label: 'Shebang script' },
  {
    sig: 'SQLite format 3'.split('').map((c) => c.charCodeAt(0)),
    mime: 'application/x-sqlite3',
    label: 'SQLite3 database',
  },
  {
    sig: [0x30, 0x82],
    mime: 'application/x-pkcs12',
    label: 'PKCS#12 / DER certificate',
    offset: 0,
    conditional(bytes) {
      return bytes.length >= 5 && bytes[2] >= 1 && bytes[3] <= 4
    },
  },
  {
    sig: '-----BEGIN'.split('').map((c) => c.charCodeAt(0)),
    mime: 'application/x-pem-file',
    label: 'PEM certificate/key',
  },
]

function matchesAt(sig, bytes, offset) {
  for (let i = 0; i < sig.length; i++) {
    if (offset + i >= bytes.length) return false
    if (bytes[offset + i] !== sig[i]) return false
  }
  return true
}

export function detectMime(bytes) {
  if (!bytes || bytes.length === 0) return null
  const maxScan = Math.min(bytes.length, 512)
  for (const entry of signatures) {
    const offset = entry.offset ?? 0
    if (offset + entry.sig.length > maxScan) continue
    if (!matchesAt(entry.sig, bytes, offset)) continue
    if (entry.conditional && !entry.conditional(bytes)) continue
    return { mime: entry.mime, label: entry.label }
  }
  return null
}

export function fileTypeLabel(fileType) {
  const labels = { 0: 'TEXT', 1: 'BINARY' }
  return labels[fileType] ?? 'UNKNOWN'
}
