import {
  EncodeShellcode,
  GenerateShellcodeRDI,
  GetShellcodeEncoderMap,
} from '../../../wailsjs/go/gui/App.js'
import { responseField } from './normalize.js'

export { EncodeShellcode, GenerateShellcodeRDI }

export async function listShellcodeEncoders() {
  const archMap = responseField(await GetShellcodeEncoderMap(), 'Encoders', {})
  return Object.entries(archMap)
    .flatMap(([arch, value]) => normalizeArch(arch, value))
    .sort((a, b) => `${a.arch}:${a.name}`.localeCompare(`${b.arch}:${b.name}`))
}

function normalizeArch(arch, archInfo = {}) {
  const encoders = archInfo.Encoders || archInfo.encoders || {}
  const descriptions = archInfo.Descriptions || archInfo.descriptions || {}
  return Object.entries(encoders).map(([name, value]) => ({
    arch,
    name,
    value,
    description: descriptions[name] || '',
  }))
}
