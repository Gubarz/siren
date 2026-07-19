import { GetEventHistory } from '../../../wailsjs/go/main/App.js'

export async function listEvents({ since = 0, limit = 300 } = {}) {
  return await GetEventHistory(since, limit)
}
