import './types'

declare module './types' {
  interface FingerprintConfig {
    windowWidth?: number
    windowHeight?: number
    deviceScaleFactor?: number
  }

  interface EvidenceRun {
    consistencyInputDigest?: string
    consistencyRulesRevision?: string
  }
}
