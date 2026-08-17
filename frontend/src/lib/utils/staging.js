import { implantFormat } from './formats.js'

export const STAGER_JOB_DESCRIPTION = 'Raw TCP listener (stager only)'

export function isStagerJob(job) {
  return (job?.Description ?? job?.description) === STAGER_JOB_DESCRIPTION
}

export function stagerListeners(jobs = []) {
  return jobs.filter(isStagerJob)
}

export function stagedBuildRows(builds = []) {
  return builds
    .filter((build) => build.staged)
    .map((build) => ({
      _rowKey: build.name,
      _name: build.name,
      _nonce: build.nonce ?? build.Nonce ?? null,
      _osArch: `${build.GOOS ?? build.goos ?? '?'}/${build.GOARCH ?? build.goarch ?? '?'}`,
      _format: implantFormat(build.Format ?? build.format),
      _type: (build.IsBeacon ?? build.isBeacon) ? 'beacon' : 'session',
    }))
}

export function stagerListenerRows(jobs = []) {
  return stagerListeners(jobs).map((job, index) => ({
    _rowKey: job.ID ?? job.id ?? index,
    _id: job.ID ?? job.id ?? '-',
    _port: job.Port ?? job.port ?? '-',
    _profile: job.ProfileName ?? job.profileName ?? '-',
  }))
}
