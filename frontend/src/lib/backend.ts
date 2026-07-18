import { defaultProfile } from "./model";
import type {
  AdapterImportRequest,
  AdapterInstallRequest,
  AdapterRecord,
  AdapterValidationReport,
  Bootstrap,
  Capabilities,
  CredentialRecord,
  CredentialSaveRequest,
  EvidenceRun,
  KernelImportRequest,
  KernelRecord,
  LaunchPlan,
  Profile,
  ProviderDescriptor,
  ProxyDiagnosticReport,
  RuntimeSession,
} from "../types";

type WailsDesktopApp = {
  Bootstrap: () => Promise<Bootstrap>;
  ListProfiles: () => Promise<Profile[]>;
  ListSessions: () => Promise<RuntimeSession[]>;
  ListCredentials: () => Promise<CredentialRecord[]>;
  ListAdapters: () => Promise<AdapterRecord[]>;
  Capabilities: (provider: string, version: string) => Promise<Capabilities>;
  CreateProfile: (profile: Profile) => Promise<Profile>;
  UpdateProfile: (profile: Profile) => Promise<Profile>;
  CloneProfile: (id: string, name: string) => Promise<Profile>;
  DeleteProfile: (id: string) => Promise<void>;
  SaveCredential: (request: CredentialSaveRequest) => Promise<CredentialRecord>;
  DeleteCredential: (id: string) => Promise<void>;
  BuildLaunchPlan: (request: {
    profileId: string;
    remoteDebuggingPort: number;
  }) => Promise<LaunchPlan>;
  StartProfile: (profileId: string) => Promise<RuntimeSession>;
  StopProfile: (profileId: string) => Promise<RuntimeSession>;
  RunProxyDiagnostics: (profileId: string) => Promise<ProxyDiagnosticReport>;
  RunEvidence: (profileId: string) => Promise<EvidenceRun>;
  CancelEvidence: (profileId: string) => Promise<void>;
  ListEvidence: (profileId: string) => Promise<EvidenceRun[]>;
  GetEvidence: (id: string) => Promise<EvidenceRun>;
  DeleteEvidence: (id: string) => Promise<void>;
  EvidenceActive: (profileId: string) => Promise<boolean>;
  PickKernelExecutable: () => Promise<string>;
  ImportKernel: (request: KernelImportRequest) => Promise<KernelRecord>;
  VerifyKernel: (id: string) => Promise<KernelRecord>;
  DeleteKernel: (id: string) => Promise<void>;
  PickAdapterExecutable: () => Promise<string>;
  ImportAdapter: (request: AdapterImportRequest) => Promise<AdapterRecord>;
  VerifyAdapter: (id: string) => Promise<AdapterRecord>;
  ValidateAdapter: (id: string) => Promise<AdapterValidationReport>;
  InstallOfficialAdapter: (request: AdapterInstallRequest) => Promise<AdapterRecord>;
  DeleteAdapter: (id: string) => Promise<void>;
};

declare global {
  interface Window {
    go?: { main?: { DesktopApp?: WailsDesktopApp } };
  }
}

const genericCapabilities: Capabilities["capabilities"] = {
  platform: {
    id: "platform",
    status: "unsupported",
    evidenceRequired: false,
    limitation: "custom Chromium reports its own platform",
  },
  "browser-brand": {
    id: "browser-brand",
    status: "unsupported",
    evidenceRequired: false,
    limitation: "custom Chromium reports its own browser brand",
  },
  timezone: {
    id: "timezone",
    status: "unsupported",
    evidenceRequired: false,
    limitation: "custom Chromium uses the host or browser-configured timezone",
  },
  "surface-seed": {
    id: "surface-seed",
    status: "unsupported",
    evidenceRequired: false,
    limitation: "no reviewed surface-seed contract exists",
  },
  "surface-controls": {
    id: "surface-controls",
    status: "unsupported",
    evidenceRequired: false,
    limitation: "no reviewed surface-control contract exists",
  },
  "hardware-concurrency": {
    id: "hardware-concurrency",
    status: "unsupported",
    evidenceRequired: false,
    limitation: "no reviewed CPU override contract exists",
  },
  "device-memory": {
    id: "device-memory",
    status: "unsupported",
    evidenceRequired: false,
    limitation: "no reviewed device-memory contract exists",
  },
  "custom-gpu": {
    id: "custom-gpu",
    status: "unsupported",
    evidenceRequired: false,
    limitation: "no reviewed GPU override contract exists",
  },
  "webrtc-policy": {
    id: "webrtc-policy",
    status: "unverified",
    evidenceRequired: true,
    limitation: "command-line policy has not completed the Phase 4 evidence chain",
  },
};

function sample(
  provider: string,
  trustStatus: Capabilities["trustStatus"],
  capabilities: Capabilities["capabilities"] = genericCapabilities,
): Capabilities {
  return {
    schemaVersion: 2,
    provider,
    revision: 1,
    trustStatus,
    majorVersion: 148,
    capabilities,
    limitations: [
      "binary integrity does not establish reviewed provider trust",
    ],
  };
}

const legacyPatchedCapabilities: Capabilities["capabilities"] = Object.fromEntries(
  Object.entries(genericCapabilities).map(([id, declaration]) => [
    id,
    id === "device-memory"
      ? declaration
      : {
          ...declaration,
          status: "unverified",
          evidenceRequired: true,
          limitation: "legacy command-line claim lacks reviewed real-browser evidence",
        },
  ]),
) as Capabilities["capabilities"];

const providers: ProviderDescriptor[] = [
  {
    id: "custom-chromium",
    name: "Custom local Chromium",
    description:
      "Locally imported Chromium with generic launch support and no Veilium-reviewed fingerprint claims.",
    versions: ["148.0.0", "144.0.0", "142.0.0"],
    samples: [sample("custom-chromium", "custom")],
  },
  {
    id: "native-chromium",
    name: "Legacy native Chromium",
    description:
      "Compatibility definition for records created before Provider Contract v2.",
    versions: ["148.0.0", "144.0.0", "142.0.0"],
    samples: [sample("native-chromium", "legacy")],
  },
  {
    id: "patched-chromium",
    name: "Legacy patched Chromium",
    description:
      "Compatibility definition for pre-v2 records; former boolean claims remain unverified.",
    versions: ["148.0.0", "144.0.0", "142.0.0"],
    samples: [sample("patched-chromium", "legacy", legacyPatchedCapabilities)],
  },
];

let mockKernels: KernelRecord[] = [];
let mockAdapters: AdapterRecord[] = [];
let mockProfiles: Profile[] = [];
let mockSessions: RuntimeSession[] = [];
const native = (): WailsDesktopApp | undefined => window.go?.main?.DesktopApp;
const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T;

export const backend = {
  isNative: () => Boolean(native()),

  async bootstrap(): Promise<Bootstrap> {
    const api = native();
    return api
      ? api.Bootstrap()
      : {
          version: "0.14.0-browser-preview",
          profiles: clone(mockProfiles),
          providers: clone(providers),
          kernels: clone(mockKernels),
          adapters: clone(mockAdapters),
          sessions: clone(mockSessions),
          credentials: [],
          credentialProvider: "Desktop operating-system keyring",
          adapterPins: [],
          runtimePlatform: "browser",
          runtimeArch: "unknown",
        };
  },

  async listSessions(): Promise<RuntimeSession[]> {
    const api = native();
    return api ? api.ListSessions() : clone(mockSessions);
  },

  async capabilities(provider: string, version: string): Promise<Capabilities> {
    const api = native();
    if (api) return api.Capabilities(provider, version);
    const descriptor = providers.find((item) => item.id === provider);
    const major = Number.parseInt(version, 10);
    const selected =
      descriptor?.samples.find((item) => item.majorVersion === major) ??
      descriptor?.samples[0];
    if (!selected) throw new Error(`Unknown provider ${provider}`);
    return clone(selected);
  },

  async createProfile(input: Profile): Promise<Profile> {
    const api = native();
    if (api) return api.CreateProfile(input);
    const created = {
      ...clone(input),
      id: crypto.randomUUID(),
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    mockProfiles = [...mockProfiles, created];
    return clone(created);
  },

  async updateProfile(input: Profile): Promise<Profile> {
    const api = native();
    if (api) return api.UpdateProfile(input);
    const updated = { ...clone(input), updatedAt: new Date().toISOString() };
    mockProfiles = mockProfiles.map((item) =>
      item.id === input.id ? updated : item,
    );
    return clone(updated);
  },

  async cloneProfile(id: string, name: string): Promise<Profile> {
    const api = native();
    if (api) return api.CloneProfile(id, name);
    const source = mockProfiles.find((item) => item.id === id);
    if (!source) throw new Error("Profile not found");
    const cloned = {
      ...clone(source),
      id: crypto.randomUUID(),
      name: name || `${source.name} Copy`,
      createdAt: new Date().toISOString(),
    };
    mockProfiles = [...mockProfiles, cloned];
    return clone(cloned);
  },

  async deleteProfile(id: string): Promise<void> {
    const api = native();
    if (api) return api.DeleteProfile(id);
    mockProfiles = mockProfiles.filter((item) => item.id !== id);
  },

  async saveCredential(
    request: CredentialSaveRequest,
  ): Promise<CredentialRecord> {
    const api = native();
    if (!api)
      throw new Error(
        "The operating-system credential vault is available only in the Wails desktop application",
      );
    return api.SaveCredential(request);
  },

  async deleteCredential(id: string): Promise<void> {
    const api = native();
    if (!api)
      throw new Error(
        "The operating-system credential vault is available only in the Wails desktop application",
      );
    return api.DeleteCredential(id);
  },

  async buildLaunchPlan(
    profileId: string,
    remoteDebuggingPort = 0,
  ): Promise<LaunchPlan> {
    const api = native();
    if (api) return api.BuildLaunchPlan({ profileId, remoteDebuggingPort });
    const profile = mockProfiles.find((item) => item.id === profileId);
    if (!profile) throw new Error("Profile not found");
    return {
      executable: profile.kernel.executable,
      proxyDisplay: profile.proxy.url || "direct://",
      requiresBridge: Boolean(profile.proxy.credentialRef),
      bridgeKind: profile.proxy.adapterRef
        ? "managed-adapter"
        : profile.proxy.credentialRef
          ? "local-auth-bridge"
          : undefined,
      args: [
        `--user-data-dir=${profile.userDataDir}`,
        "--remote-debugging-address=127.0.0.1",
      ],
      warnings: ["Browser preview mode: no native process will be launched."],
    };
  },

  async startProfile(profileId: string): Promise<RuntimeSession> {
    const api = native();
    if (!api)
      throw new Error(
        "Browser process execution is available only in the Wails desktop application",
      );
    return api.StartProfile(profileId);
  },

  async stopProfile(profileId: string): Promise<RuntimeSession> {
    const api = native();
    if (!api)
      throw new Error(
        "Browser process execution is available only in the Wails desktop application",
      );
    return api.StopProfile(profileId);
  },

  async runProxyDiagnostics(profileId: string): Promise<ProxyDiagnosticReport> {
    const api = native();
    if (!api)
      throw new Error(
        "Proxy diagnostics are available only in the Wails desktop application",
      );
    return api.RunProxyDiagnostics(profileId);
  },

  async runEvidence(profileId: string): Promise<EvidenceRun> {
    const api = native();
    if (!api)
      throw new Error(
        "Real-browser evidence is available only in the Wails desktop application",
      );
    return api.RunEvidence(profileId);
  },

  async cancelEvidence(profileId: string): Promise<void> {
    const api = native();
    if (!api)
      throw new Error(
        "Real-browser evidence is available only in the Wails desktop application",
      );
    return api.CancelEvidence(profileId);
  },

  async listEvidence(profileId: string): Promise<EvidenceRun[]> {
    const api = native();
    return api ? api.ListEvidence(profileId) : [];
  },

  async getEvidence(id: string): Promise<EvidenceRun> {
    const api = native();
    if (!api)
      throw new Error(
        "Evidence reports are available only in the Wails desktop application",
      );
    return api.GetEvidence(id);
  },

  async deleteEvidence(id: string): Promise<void> {
    const api = native();
    if (!api)
      throw new Error(
        "Evidence reports are available only in the Wails desktop application",
      );
    return api.DeleteEvidence(id);
  },

  async evidenceActive(profileId: string): Promise<boolean> {
    const api = native();
    return api ? api.EvidenceActive(profileId) : false;
  },

  async pickKernelExecutable(): Promise<string> {
    const api = native();
    if (!api)
      throw new Error(
        "File selection is available only in the Wails desktop application",
      );
    return api.PickKernelExecutable();
  },

  async importKernel(request: KernelImportRequest): Promise<KernelRecord> {
    const api = native();
    if (!api)
      throw new Error(
        "Kernel import is available only in the Wails desktop application",
      );
    return api.ImportKernel(request);
  },

  async verifyKernel(id: string): Promise<KernelRecord> {
    const api = native();
    if (api) return api.VerifyKernel(id);
    const record = mockKernels.find((item) => item.id === id);
    if (!record) throw new Error("Kernel not found");
    return clone(record);
  },

  async deleteKernel(id: string): Promise<void> {
    const api = native();
    if (api) return api.DeleteKernel(id);
    if (mockProfiles.some((item) => item.kernel.id === id))
      throw new Error("Kernel is used by a profile");
    mockKernels = mockKernels.filter((item) => item.id !== id);
  },

  async pickAdapterExecutable(): Promise<string> {
    const api = native();
    if (!api)
      throw new Error(
        "Adapter file selection is available only in the Wails desktop application",
      );
    return api.PickAdapterExecutable();
  },

  async importAdapter(request: AdapterImportRequest): Promise<AdapterRecord> {
    const api = native();
    if (!api)
      throw new Error(
        "Adapter import is available only in the Wails desktop application",
      );
    return api.ImportAdapter(request);
  },

  async verifyAdapter(id: string): Promise<AdapterRecord> {
    const api = native();
    if (api) return api.VerifyAdapter(id);
    const record = mockAdapters.find((item) => item.id === id);
    if (!record) throw new Error("Proxy adapter not found");
    return clone(record);
  },

  async validateAdapter(id: string): Promise<AdapterValidationReport> {
    const api = native();
    if (!api)
      throw new Error(
        "Official adapter validation is available only in the Wails desktop application",
      );
    return api.ValidateAdapter(id);
  },

  async installOfficialAdapter(
    request: AdapterInstallRequest,
  ): Promise<AdapterRecord> {
    const api = native();
    if (!api)
      throw new Error(
        "Official adapter installation is available only in the Wails desktop application",
      );
    return api.InstallOfficialAdapter(request);
  },

  async deleteAdapter(id: string): Promise<void> {
    const api = native();
    if (api) return api.DeleteAdapter(id);
    if (mockProfiles.some((item) => item.proxy.adapterRef === id))
      throw new Error("Proxy adapter is used by a profile");
    mockAdapters = mockAdapters.filter((item) => item.id !== id);
  },
};

export { providers, defaultProfile };
