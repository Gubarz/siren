// Operator control RPC bindings that do not yet belong to a narrower API module.

export {
  // Beacon lifecycle
  GetBeacon,
  OpenBeaconSession,
  CloseBeaconSession,
  UpdateBeaconIntegrity,

  // Agent reconfigure
  ReconfigureAgent,

  // Credentials
  UpdateCredential,
  GetCredentialByID,
  GetCredentialsByHashType,
  GetPlaintextCredentialsByHashType,
  SniffCredentialHashType,

  // Server utilities
  GetCertificateAuthorityInfo,
  GetCompiler,
  GetCanaries,
  RestartJobs,
  PrimeSpoofMetadataFromPath,
  LogClient,
} from '../../../wailsjs/go/gui/App.js';
