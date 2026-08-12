import {
  GetHTTPC2ProfileByName,
  GetHTTPC2Profiles,
  SaveHTTPC2ProfileJSON,
} from '../../../bindings/siren/cmd/gui/app.js'
import { responseField } from './normalize.js'

export { GetHTTPC2ProfileByName, SaveHTTPC2ProfileJSON }

export async function listHTTPC2Profiles() {
  return responseField(await GetHTTPC2Profiles(), 'Configs', [])
}

export function profileName(profile) {
  return profile?.Name || profile?.name || ''
}
