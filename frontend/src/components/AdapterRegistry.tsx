import { useMemo, useState } from "react";
import type {
  AdapterImportRequest,
  AdapterRecord,
  AdapterReleasePin,
  AdapterValidationReport,
} from "../types";

const defaults: Record<
  AdapterRecord["kind"],
  Pick<AdapterImportRequest, "licenseSpdx" | "sourceUrl">
> = {
  xray: {
    licenseSpdx: "MPL-2.0",
    sourceUrl: "https://github.com/XTLS/Xray-core",
  },
  "sing-box": {
    licenseSpdx: "GPL-3.0-or-later",
    sourceUrl: "https://github.com/SagerNet/sing-box",
  },
};

export function AdapterRegistry({
  records,
  pins,
  reports,
  nativeMode,
  busy,
  error,
  onPick,
  onImport,
  onVerify,
  onValidate,
  onDelete,
}: {
  records: AdapterRecord[];
  pins: AdapterReleasePin[];
  reports: Record<string, AdapterValidationReport>;
  nativeMode: boolean;
  busy: boolean;
  error?: string;
  onPick: () => Promise<string>;
  onImport: (request: AdapterImportRequest) => Promise<void>;
  onVerify: (record: AdapterRecord) => Promise<void>;
  onValidate: (record: AdapterRecord) => Promise<void>;
  onDelete: (record: AdapterRecord) => Promise<void>;
}) {
  const [request, setRequest] = useState<AdapterImportRequest>({
    name: "",
    kind: "xray",
    version: pins.find((pin) => pin.kind === "xray")?.version ?? "",
    sourcePath: "",
    ...defaults.xray,
  });
  const protocols = useMemo(
    () =>
      request.kind === "xray"
        ? ["VMess", "VLESS", "Trojan", "Shadowsocks"]
        : ["Hysteria2", "TUIC", "AnyTLS"],
    [request.kind],
  );
  const releasePins = useMemo(() => {
    const unique = new Map<string, AdapterReleasePin>();
    for (const pin of pins) unique.set(`${pin.kind}@${pin.version}`, pin);
    return Array.from(unique.values());
  }, [pins]);

  async function pick() {
    const path = await onPick();
    if (path)
      setRequest((current) => ({
        ...current,
        sourcePath: path,
        name: current.name || path.split(/[\\/]/).pop() || current.kind,
      }));
  }

  async function submit() {
    await onImport(request);
    setRequest((current) => ({ ...current, name: "", sourcePath: "" }));
  }

  function changeKind(kind: AdapterRecord["kind"]) {
    const pin = pins.find((item) => item.kind === kind);
    setRequest((current) => ({
      ...current,
      kind,
      version: pin?.version ?? "",
      ...defaults[kind],
    }));
  }

  return (
    <>
      <section className="panel official-release-panel">
        <div className="panel-heading">
          <div>
            <h2>Pinned official releases</h2>
            <p>
              Veilium recognizes only these exact release assets as official.
              Custom local binaries remain usable but are clearly labeled and
              cannot pass the official configuration check.
            </p>
          </div>
        </div>
        <div className="official-pin-grid">
          {releasePins.map((pin) => (
            <article key={`${pin.kind}-${pin.version}`}>
              <strong>{pin.kind === "xray" ? "Xray" : "sing-box"}</strong>
              <span>{pin.tag}</span>
              <code>{pin.repository}</code>
              <small>
                Linux and Windows amd64 · archive and executable SHA-256 pinned
              </small>
            </article>
          ))}
        </div>
      </section>

      <section className="panel adapter-import">
        <div className="panel-heading">
          <div>
            <h2>Register local adapter</h2>
            <p>
              Veilium hashes the imported executable and automatically matches
              it against the embedded official release manifest. User-entered
              provenance never overrides a digest mismatch.
            </p>
          </div>
        </div>
        <div className="adapter-import-grid">
          <label>
            Name
            <input
              value={request.name}
              onChange={(event) =>
                setRequest((current) => ({ ...current, name: event.target.value }))
              }
              placeholder="Official or custom runtime"
            />
          </label>
          <label>
            Adapter kind
            <select
              value={request.kind}
              onChange={(event) =>
                changeKind(event.target.value as AdapterRecord["kind"])
              }
            >
              <option value="xray">Xray</option>
              <option value="sing-box">sing-box</option>
            </select>
          </label>
          <label>
            Version
            <input
              value={request.version}
              onChange={(event) =>
                setRequest((current) => ({ ...current, version: event.target.value }))
              }
              placeholder="Declared upstream version"
            />
          </label>
          <label>
            SPDX license
            <input
              value={request.licenseSpdx}
              onChange={(event) =>
                setRequest((current) => ({
                  ...current,
                  licenseSpdx: event.target.value,
                }))
              }
            />
          </label>
          <label className="adapter-source">
            Source URL
            <input
              value={request.sourceUrl}
              onChange={(event) =>
                setRequest((current) => ({ ...current, sourceUrl: event.target.value }))
              }
            />
          </label>
          <label className="adapter-path">
            Executable path
            <div className="path-picker">
              <input
                readOnly
                value={request.sourcePath}
                placeholder="Choose a local adapter executable…"
              />
              <button
                type="button"
                className="button secondary"
                onClick={() => void pick()}
                disabled={!nativeMode || busy}
              >
                Browse
              </button>
            </div>
          </label>
          <button
            className="button primary adapter-import-button"
            onClick={() => void submit()}
            disabled={
              !nativeMode ||
              busy ||
              !request.name.trim() ||
              !request.version.trim() ||
              !request.sourcePath.trim() ||
              !request.licenseSpdx.trim() ||
              !request.sourceUrl.trim()
            }
          >
            {busy ? "Working…" : "Import and identify"}
          </button>
        </div>
        <div className="adapter-capabilities">
          <span>Reviewed capability family</span>
          {protocols.map((protocol) => (
            <strong key={protocol}>{protocol}</strong>
          ))}
        </div>
        {!nativeMode && (
          <div className="info-banner">
            <strong>Desktop runtime required</strong>
            <p>Browser preview mode cannot read or execute local binaries.</p>
          </div>
        )}
        {error && <div className="form-error">{error}</div>}
      </section>

      <div className="adapter-list">
        {records.length === 0 ? (
          <section className="panel empty-state">
            <div className="empty-icon">⇄</div>
            <h3>No managed proxy adapters</h3>
            <p>Import Xray or sing-box before assigning advanced protocols.</p>
          </section>
        ) : (
          records.map((record) => {
            const report = reports[record.id];
            return (
              <article className="adapter-card" key={record.id}>
                <div className="adapter-card-head">
                  <div className="adapter-logo">
                    {record.kind === "xray" ? "X" : "S"}
                  </div>
                  <div>
                    <h2>{record.name}</h2>
                    <code>
                      {record.kind} · {record.version}
                    </code>
                  </div>
                  <div className="adapter-status-stack">
                    <span className={`kernel-status ${record.status}`}>
                      {record.status}
                    </span>
                    <span className={record.official ? "official-badge" : "custom-badge"}>
                      {record.official
                        ? `Official ${record.officialTag}`
                        : "Custom local"}
                    </span>
                  </div>
                </div>
                <div className="adapter-protocols">
                  {record.protocols.map((protocol) => (
                    <span key={protocol}>{protocol}</span>
                  ))}
                </div>
                <dl>
                  <div>
                    <dt>Executable SHA</dt>
                    <dd title={record.sha256}>
                      {record.sha256.slice(0, 16)}…{record.sha256.slice(-8)}
                    </dd>
                  </div>
                  <div>
                    <dt>Identity</dt>
                    <dd>
                      {record.official
                        ? `${record.officialAsset} · ${record.officialPlatform}/${record.officialArch}`
                        : "No embedded official release match"}
                    </dd>
                  </div>
                  <div>
                    <dt>License</dt>
                    <dd>{record.licenseSpdx}</dd>
                  </div>
                  <div>
                    <dt>Source</dt>
                    <dd title={record.sourceUrl}>{record.sourceUrl}</dd>
                  </div>
                  <div>
                    <dt>Managed path</dt>
                    <dd title={record.executable}>{record.executable}</dd>
                  </div>
                </dl>
                {report && (
                  <div className="official-validation-report">
                    <div>
                      <strong>Official configuration check passed</strong>
                      <span>{report.versionText.split("\n")[0]}</span>
                    </div>
                    <ul>
                      {report.checks.map((check) => (
                        <li key={check.id}>✓ {check.label}</li>
                      ))}
                    </ul>
                  </div>
                )}
                <div className="kernel-actions">
                  <button
                    className="button secondary"
                    disabled={busy}
                    onClick={() => void onVerify(record)}
                  >
                    Integrity check
                  </button>
                  <button
                    className="button secondary"
                    disabled={busy || !nativeMode || !record.official || record.status !== "verified"}
                    title={record.official ? "Run the official binary configuration checks" : "Only an exact pinned official binary can run this check"}
                    onClick={() => void onValidate(record)}
                  >
                    Official check
                  </button>
                  <button
                    className="button secondary danger-text"
                    disabled={busy}
                    onClick={() => void onDelete(record)}
                  >
                    Remove
                  </button>
                </div>
              </article>
            );
          })
        )}
      </div>
    </>
  );
}
