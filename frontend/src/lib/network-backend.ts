import type {
  NetworkCompatibilityMatrix,
  NetworkEvidenceRun,
  NetworkProbeConfiguration,
  NetworkProbeSet,
} from '../network-types'

type NetworkDesktopApp = {
  GetNetworkProbeSet: () => Promise<NetworkProbeConfiguration>
  SaveNetworkProbeSet: (set: NetworkProbeSet) => Promise<NetworkProbeSet>
  DeleteNetworkProbeSet: () => Promise<void>
  RunNetworkEvidence: (profileId: string) => Promise<NetworkEvidenceRun>
  CancelNetworkEvidence: (profileId: string) => Promise<void>
  ListNetworkEvidence: (profileId: string) => Promise<NetworkEvidenceRun[]>
  GetNetworkEvidence: (id: string) => Promise<NetworkEvidenceRun>
  DeleteNetworkEvidence: (id: string) => Promise<void>
  NetworkEvidenceActive: (profileId: string) => Promise<boolean>
  NetworkCompatibilityMatrix: () => Promise<NetworkCompatibilityMatrix>
}

function native(): NetworkDesktopApp | undefined {
  const root = window as unknown as {
    go?: { main?: { DesktopApp?: NetworkDesktopApp } }
  }
  return root.go?.main?.DesktopApp
}

function requireNative(): NetworkDesktopApp {
  const api = native()
  if (!api) throw new Error('Network evidence is available only in the Wails desktop application')
  return api
}

export const networkBackend = {
  configured: async (): Promise<NetworkProbeConfiguration> => requireNative().GetNetworkProbeSet(),
  saveProbeSet: async (set: NetworkProbeSet): Promise<NetworkProbeSet> => requireNative().SaveNetworkProbeSet(set),
  deleteProbeSet: async (): Promise<void> => requireNative().DeleteNetworkProbeSet(),
  run: async (profileId: string): Promise<NetworkEvidenceRun> => requireNative().RunNetworkEvidence(profileId),
  cancel: async (profileId: string): Promise<void> => requireNative().CancelNetworkEvidence(profileId),
  list: async (profileId: string): Promise<NetworkEvidenceRun[]> => requireNative().ListNetworkEvidence(profileId),
  get: async (id: string): Promise<NetworkEvidenceRun> => requireNative().GetNetworkEvidence(id),
  delete: async (id: string): Promise<void> => requireNative().DeleteNetworkEvidence(id),
  active: async (profileId: string): Promise<boolean> => requireNative().NetworkEvidenceActive(profileId),
  matrix: async (): Promise<NetworkCompatibilityMatrix> => requireNative().NetworkCompatibilityMatrix(),
}
