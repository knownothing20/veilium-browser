export type ManagedStorageLocationStatus = 'present' | 'missing' | 'unexpected-entry' | 'unsafe-link' | 'unavailable'

export interface ManagedStorageLocation {
  id: string
  label: string
  path: string
  kind: 'directory' | 'file'
  status: ManagedStorageLocationStatus
  reasonCode?: string
  description: string
  volume?: string
  onSystemVolume: boolean
}

export interface ManagedStorageLocations {
  dataRoot: string
  dataVolume?: string
  systemVolume?: string
  onSystemVolume: boolean
  generatedAt: string
  locations: ManagedStorageLocation[]
  limitations: string[]
}

type NativeStorageLocationAPI = {
  GetManagedStorageLocations(): Promise<ManagedStorageLocations>
}

function native(): NativeStorageLocationAPI | undefined {
  return (window as unknown as { go?: { main?: { DesktopApp?: NativeStorageLocationAPI } } }).go?.main?.DesktopApp
}

export const storageLocationAPI = {
  isNative: () => Boolean(native()),
  get: async (): Promise<ManagedStorageLocations> => {
    const api = native()
    if (!api) throw new Error('Managed storage locations require the Wails desktop runtime.')
    return api.GetManagedStorageLocations()
  },
}
