import {
  AddTrafficEncoder,
  GetTrafficEncoderMap,
  RemoveTrafficEncoder,
} from '../../../bindings/siren/cmd/gui/app.js'
import { responseField } from './normalize.js'

export { AddTrafficEncoder, RemoveTrafficEncoder }

export async function listTrafficEncoders() {
  const encoders = responseField(await GetTrafficEncoderMap(), 'Encoders', {})
  return Object.entries(encoders)
    .map(([fallbackName, encoder]) => normalizeEncoder(fallbackName, encoder))
    .sort((a, b) => a.name.localeCompare(b.name))
}

function normalizeEncoder(fallbackName, encoder = {}) {
  const wasm = encoder.Wasm || encoder.wasm || {}
  const data = wasm.Data || wasm.data || []
  return {
    ...encoder,
    name: wasm.Name || wasm.name || fallbackName,
    id: encoder.ID ?? encoder.id ?? 0,
    size: data.length || 0,
    wasm,
  }
}
