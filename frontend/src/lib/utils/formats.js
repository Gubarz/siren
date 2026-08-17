export function implantFormat(f) {
  return ({ 0: 'shared lib', 1: 'shellcode', 2: 'executable', 3: 'service', 4: 'third-party' })[f] ?? f
}
