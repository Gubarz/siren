import { GetEventHistory, SetEventsAcknowledged } from '../../../bindings/siren/cmd/gui/app.js'

export async function listEvents({ since = 0, limit = 300 } = {}) {
  return await GetEventHistory(since, limit)
}

export async function setEventsAcknowledged(seqs, acked) {
  return await SetEventsAcknowledged(seqs, acked)
}
