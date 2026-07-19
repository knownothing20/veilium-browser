import { useEffect, useMemo, useState } from 'react'
import { AdapterRegistry } from './components/AdapterRegistry'
import { CredentialVault } from './components/CredentialVault'
import { MetricCard } from './components/MetricCard'
import { OfficialKernelCard } from './components/OfficialKernelCard'
import { PlanDrawer } from './components/PlanDrawer'
import { ProfileEditor } from './components/ProfileEditor'
import { ProfileTable } from './components/ProfileTable'
import { RuntimePanel } from './components/RuntimePanel'
import { Sidebar, type ViewKey } from './components/Sidebar'
import { backend } from './lib/backend'
import { filterProfiles, groupsOf, profileHealth } from './lib/model'
import { isRuntimeActive, sessionForProfile } from './lib/runtime'
import type {
  AdapterImportRequest,
  AdapterInstallRequest,
  AdapterRecord,
  AdapterValidationReport,
  Bootstrap,
  CredentialSaveRequest,
  KernelImportRequest,
  KernelInstallRequest,
  KernelRecord,
  LaunchPlan,
  Profile,
} from './types'

const emptyBootstrap: Bootstrap = {
  version: 'loading',
  profiles: [],
  providers: [],
  kernels: [],
  adapters: [],
  sessions: [],
  credentials: [],
  credentialProvider: 'Operating-system keyring',
  adapterPins: [],
  kernelPins: [],
  runtimePlatform: 'browser',
  runtimeArch: 'unknown',
}

export default function App() {
  const [data, setData] = useState<Bootstrap>(emptyBootstrap)
  const [view, setView] = useState<ViewKey>('dashboard')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [query, setQuery] = useState('')
  const [group, setGroup] = useState('all')
  const [editing, setEditing] = useState<Profile>()
  const [editorOpen, setEditorOpen] = useState(false)
  const [selectedID, setSelectedID] = useState('')
  const [planProfile, setPlanProfile] = useState<Profile>()
  const [plan, setPlan] = useState<LaunchPlan>()
  const [planError, setPlanError] = useState('')
