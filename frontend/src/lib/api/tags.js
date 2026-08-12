// Client-side operator tags — persisted per teamserver on the Go side, no
// sliver-server RPC involved. Backing service is internal/tags/tags.go.

import {
  GetAgentTags,
  GetEntityTags,
  SetAgentTags,
  SetEntityTags,
  GetAllAgentTags,
  GetAllEntityTags,
  ListKnownTags,
  GetAllAgentColors,
  GetAllEntityColors,
  GetEntityColor,
  SetAgentColor,
  SetEntityColor,
} from '../../../bindings/siren/cmd/gui/app.js';

export {
  GetAgentTags,
  GetEntityTags,
  SetAgentTags,
  SetEntityTags,
  GetAllAgentTags,
  GetAllEntityTags,
  ListKnownTags,
  GetAllAgentColors,
  GetAllEntityColors,
  GetEntityColor,
  SetAgentColor,
  SetEntityColor,
};
