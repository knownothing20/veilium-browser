import { useEffect, useMemo, useState } from "react";
import { applyKernel, capabilityFor, defaultProfile } from "../lib/model";
import type {
  AdapterRecord,
  CredentialRecord,
  KernelRecord,
  Profile,
  ProviderDescriptor,
} from "../types";

function requiredAdapterKind(raw?: string): AdapterRecord["kind"] | undefined {
  const scheme = (raw || "").split(":", 1)[0].trim().toLowerCase();
  if (["vmess", "vless", "trojan", "ss", "shadowsocks"].includes(scheme))
    return "xray";
  if (["hysteria2", "tuic", "anytls"].includes(scheme)) return "sing-box";
  return undefined;
}

export function ProfileEditor({
  open,
  profile,
  providers,
  kernels,
  adapters,
  credentials,
  onClose,
  onSave,
}: {
  open: boolean;
  profile?: Profile;
  providers: ProviderDescriptor[];
  kernels: KernelRecord[];
  adapters: AdapterRecord[];
  credentials: CredentialRecord[];
  onClose: () => void;
  onSave: (profile: Profile) => Promise<void>;
}) {
  const [draft, setDraft] = useState<Profile>(() =>
    profile ? structuredClone(profile) : defaultProfile(providers[0]),
  );
  const [tags, setTags] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    const next = profile
      ? structuredClone(profile)
      : defaultProfile(providers[0]);
    setDraft(next);
    setTags((next.tags || []).join(", "));
    setError("");
  }, [profile, providers, open]);
  const selectedProvider =
    providers.find((item) => item.id === draft.kernel.provider) || providers[0];
  const capability = useMemo(
    () => capabilityFor(providers, draft.kernel.provider, draft.kernel.version),
    [providers, draft.kernel.provider, draft.kernel.version],
  );
  const verifiedKernels = kernels.filter((item) => item.status === "verified");
  const adapterKind = requiredAdapterKind(draft.proxy.url);
  const compatibleAdapters = adapters.filter(
    (item) => item.status === "verified" && item.kind === adapterKind,
  );
  if (!open) return null;
  const update = <K extends keyof Profile>(key: K, value: Profile[K]) =>
    setDraft((current) => ({ ...current, [key]: value }));
  const updateFingerprint = (
    key: keyof Profile["fingerprint"],
    value: string | number,
  ) =>
    setDraft((current) => ({
      ...current,
      fingerprint: { ...current.fingerprint, [key]: value },
    }));
  const updateKernel = (key: keyof Profile["kernel"], value: string) =>
    setDraft((current) => ({
      ...current,
      kernel: { ...current.kernel, [key]: value },
    }));
  const updateProxy = (key: keyof Profile["proxy"], value: string) =>
    setDraft((current) => ({
      ...current,
      proxy: { ...current.proxy, [key]: value },
    }));
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError("");
    try {
      await onSave({
        ...draft,
        tags: tags
          .split(",")
          .map((item) => item.trim())
          .filter(Boolean),
      });
      onClose();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setSaving(false);
    }
  }
  return (
    <div className="overlay" onMouseDown={onClose}>
      <form
        className="editor-panel"
        onSubmit={submit}
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className="editor-header">
          <div>
            <span className="eyebrow">Isolated identity</span>
            <h2>
              {profile ? "Edit browser profile" : "Create browser profile"}
            </h2>
          </div>
          <button type="button" className="close-button" onClick={onClose}>
            ×
          </button>
        </header>
        <div className="editor-scroll">
          <section className="form-section">
            <div className="section-heading">
              <span>01</span>
              <div>
                <h3>Identity</h3>
                <p>
                  Human-readable organization only. Runtime state stays
                  separate.
                </p>
              </div>
            </div>
            <div className="form-grid two">
              <label>
                Name
                <input
                  required
                  value={draft.name}
                  onChange={(event) => update("name", event.target.value)}
                />
              </label>
              <label>
                Group
                <input
                  value={draft.group || ""}
                  onChange={(event) => update("group", event.target.value)}
                />
              </label>
            </div>
            <label>
              Tags
              <input
                value={tags}
                onChange={(event) => setTags(event.target.value)}
              />
              <small>Separate tags with commas.</small>
            </label>
            <label>
              Notes
              <textarea
                value={draft.notes || ""}
                onChange={(event) => update("notes", event.target.value)}
              />
            </label>
          </section>
          <section className="form-section">
            <div className="section-heading">
              <span>02</span>
              <div>
                <h3>Kernel contract</h3>
                <p>
                  Registered kernels are copied into managed storage and
                  verified by SHA-256.
                </p>
              </div>
            </div>
            <label>
              Registered kernel
              <select
                value={draft.kernel.id || ""}
                onChange={(event) => {
                  const record = verifiedKernels.find(
                    (item) => item.id === event.target.value,
                  );
                  if (record)
                    setDraft((current) => applyKernel(current, record));
                  else
                    setDraft((current) => ({
                      ...current,
                      kernel: { ...current.kernel, id: undefined },
                    }));
                }}
              >
                <option value="">Legacy manual executable</option>
                {verifiedKernels.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name} · Chromium {item.version.split(".")[0]}
                  </option>
                ))}
              </select>
              <small>Modified or missing kernels cannot be selected.</small>
            </label>
            <div className="form-grid two">
              <label>
                Provider
                <select
                  disabled={Boolean(draft.kernel.id)}
                  value={draft.kernel.provider}
                  onChange={(event) => {
                    const provider =
                      providers.find(
                        (item) => item.id === event.target.value,
                      ) || providers[0];
                    const defaults = defaultProfile(provider);
                    setDraft((current) => ({
                      ...current,
                      kernel: defaults.kernel,
                      fingerprint: defaults.fingerprint,
                    }));
                  }}
                >
                  {providers.map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Chromium version
                <select
                  disabled={Boolean(draft.kernel.id)}
                  value={draft.kernel.version}
                  onChange={(event) =>
                    updateKernel("version", event.target.value)
                  }
                >
                  {selectedProvider?.versions.map((version) => (
                    <option key={version}>{version}</option>
                  ))}
                </select>
              </label>
            </div>
            <label>
              Executable path
              <input
                required
                readOnly={Boolean(draft.kernel.id)}
                value={draft.kernel.executable}
                onChange={(event) =>
                  updateKernel("executable", event.target.value)
                }
              />
            </label>
            <div className="capability-strip">
              <span className={capability?.canSeedSurfaces ? "on" : ""}>
                Seeded surfaces
              </span>
              <span className={capability?.canDisableSurfaces ? "on" : ""}>
                Surface controls
              </span>
              <span className={capability?.canSetCustomGpu ? "on" : ""}>
                Custom GPU
              </span>
              <span className={capability?.canSetHardwareThreads ? "on" : ""}>
                CPU override
              </span>
            </div>
          </section>
          <section className="form-section">
            <div className="section-heading">
              <span>03</span>
              <div>
                <h3>Identity consistency</h3>
                <p>
                  Platform, locale, timezone and screen settings should describe
                  one plausible device.
                </p>
              </div>
            </div>
            <div className="form-grid three">
              <label>
                Platform
                <select
                  value={draft.fingerprint.platform}
                  disabled={!capability?.canSetPlatform}
                  onChange={(event) =>
                    updateFingerprint("platform", event.target.value)
                  }
                >
                  <option value="windows">Windows</option>
                  <option value="linux">Linux</option>
                  <option value="macos">macOS</option>
                </select>
              </label>
              <label>
                Brand
                <select
                  value={draft.fingerprint.brand}
                  disabled={!capability?.canSetBrand}
                  onChange={(event) =>
                    updateFingerprint("brand", event.target.value)
                  }
                >
                  <option>Chromium</option>
                  <option>Chrome</option>
                  <option>Edge</option>
                  <option>Opera</option>
                  <option>Vivaldi</option>
                </select>
              </label>
              <label>
                Language
                <input
                  value={draft.fingerprint.language}
                  onChange={(event) =>
                    updateFingerprint("language", event.target.value)
                  }
                />
              </label>
              <label>
                Timezone
                <input
                  value={draft.fingerprint.timezone}
                  disabled={!capability?.canSetTimezone}
                  onChange={(event) =>
                    updateFingerprint("timezone", event.target.value)
                  }
                />
              </label>
              <label>
                Screen width
                <input
                  type="number"
                  min="800"
                  max="7680"
                  value={draft.fingerprint.screenWidth}
                  onChange={(event) =>
                    updateFingerprint("screenWidth", Number(event.target.value))
                  }
                />
              </label>
              <label>
                Screen height
                <input
                  type="number"
                  min="600"
                  max="4320"
                  value={draft.fingerprint.screenHeight}
                  onChange={(event) =>
                    updateFingerprint(
                      "screenHeight",
                      Number(event.target.value),
                    )
                  }
                />
              </label>
              <label>
                CPU threads
                <input
                  type="number"
                  min="2"
                  max="128"
                  disabled={!capability?.canSetHardwareThreads}
                  value={draft.fingerprint.hardwareConcurrency || 0}
                  onChange={(event) =>
                    updateFingerprint(
                      "hardwareConcurrency",
                      Number(event.target.value),
                    )
                  }
                />
              </label>
              <label>
                WebRTC
                <select
                  value={draft.fingerprint.webrtcPolicy}
                  onChange={(event) =>
                    updateFingerprint("webrtcPolicy", event.target.value)
                  }
                >
                  <option value="proxy-only">Proxy only</option>
                  <option value="disabled">Disabled</option>
                  <option value="default">Default</option>
                </select>
              </label>
              <label>
                GPU profile
                <select
                  value={draft.fingerprint.gpuProfile}
                  onChange={(event) =>
                    updateFingerprint("gpuProfile", event.target.value)
                  }
                >
                  <option value="auto">Auto-consistent</option>
                  <option value="native">Native host</option>
                  {capability?.canSetCustomGpu && (
                    <option value="custom">Custom metadata</option>
                  )}
                </select>
              </label>
            </div>
          </section>
          <section className="form-section">
            <div className="section-heading">
              <span>04</span>
              <div>
                <h3>Network route</h3>
                <p>
                  Passwords remain in the operating-system vault. Advanced
                  routes also bind an integrity-verified local adapter.
                </p>
              </div>
            </div>
            <label>
              Proxy URL
              <input
                value={draft.proxy.url || ""}
                onChange={(event) => updateProxy("url", event.target.value)}
                placeholder="direct://, http://proxy.example:8080, or vless://…"
              />
            </label>
            <label>
              Credential
              <select
                value={draft.proxy.credentialRef || ""}
                onChange={(event) =>
                  updateProxy("credentialRef", event.target.value)
                }
              >
                <option value="">No credential</option>
                {credentials.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name} · {item.username}
                  </option>
                ))}
              </select>
              <small>
                HTTP, HTTPS and SOCKS5 credentials use the built-in loopback
                bridge. For VLESS/VMess, store the UUID as the vault secret;
                Trojan and Shadowsocks use the password.
              </small>
            </label>
            {adapterKind && (
              <label>
                Managed {adapterKind} adapter
                <select
                  required
                  value={draft.proxy.adapterRef || ""}
                  onChange={(event) =>
                    updateProxy("adapterRef", event.target.value)
                  }
                >
                  <option value="">Select a verified adapter</option>
                  {compatibleAdapters.map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name} · {item.version}
                    </option>
                  ))}
                </select>
                <small>
                  The adapter binary is verified now and again before runtime.
                  Xray routes run through a private per-session SOCKS5 endpoint;
                  sing-box routes remain fail-closed.
                </small>
              </label>
            )}
            {!adapterKind && draft.proxy.adapterRef && (
              <div className="info-banner">
                <strong>Adapter reference will be rejected</strong>
                <p>
                  Only VMess, VLESS, Trojan, Shadowsocks, Hysteria2, TUIC and
                  AnyTLS routes use managed adapters.
                </p>
              </div>
            )}
          </section>
        </div>
        {error && <div className="form-error">{error}</div>}
        <footer className="editor-footer">
          <button type="button" className="button secondary" onClick={onClose}>
            Cancel
          </button>
          <button className="button primary" disabled={saving}>
            {saving ? "Saving…" : profile ? "Save changes" : "Create profile"}
          </button>
        </footer>
      </form>
    </div>
  );
}
