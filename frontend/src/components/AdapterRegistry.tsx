import { useMemo, useState } from "react";
import type { AdapterImportRequest, AdapterRecord } from "../types";

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
  nativeMode,
  busy,
  error,
  onPick,
  onImport,
  onVerify,
  onDelete,
}: {
  records: AdapterRecord[];
  nativeMode: boolean;
  busy: boolean;
  error?: string;
  onPick: () => Promise<string>;
  onImport: (request: AdapterImportRequest) => Promise<void>;
  onVerify: (record: AdapterRecord) => Promise<void>;
  onDelete: (record: AdapterRecord) => Promise<void>;
}) {
  const [request, setRequest] = useState<AdapterImportRequest>({
    name: "",
    kind: "xray",
    version: "",
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
    setRequest((current) => ({
      ...current,
      name: "",
      version: "",
      sourcePath: "",
    }));
  }
  return (
    <>
      <section className="panel adapter-import">
        <div className="panel-heading">
          <div>
            <h2>Register local adapter</h2>
            <p>
              Veilium copies the selected executable into private managed
              storage and records its provenance. Registered Xray binaries can
              run supported profiles; sing-box remains registry-only.
            </p>
          </div>
        </div>
        <div className="adapter-import-grid">
          <label>
            Name
            <input
              value={request.name}
              onChange={(event) =>
                setRequest((current) => ({
                  ...current,
                  name: event.target.value,
                }))
              }
              placeholder="Xray local runtime"
            />
          </label>
          <label>
            Adapter kind
            <select
              value={request.kind}
              onChange={(event) => {
                const kind = event.target.value as AdapterRecord["kind"];
                setRequest((current) => ({
                  ...current,
                  kind,
                  ...defaults[kind],
                }));
              }}
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
                setRequest((current) => ({
                  ...current,
                  version: event.target.value,
                }))
              }
              placeholder="Upstream version"
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
                setRequest((current) => ({
                  ...current,
                  sourceUrl: event.target.value,
                }))
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
            {busy ? "Working…" : "Import and verify"}
          </button>
        </div>
        <div className="adapter-capabilities">
          <span>Declared capability family</span>
          {protocols.map((protocol) => (
            <strong key={protocol}>{protocol}</strong>
          ))}
        </div>
        {!nativeMode && (
          <div className="info-banner">
            <strong>Desktop runtime required</strong>
            <p>Browser preview mode cannot read or copy local executables.</p>
          </div>
        )}
        {error && <div className="form-error">{error}</div>}
      </section>
      <div className="adapter-list">
        {records.length === 0 ? (
          <section className="panel empty-state">
            <div className="empty-icon">⇄</div>
            <h3>No managed proxy adapters</h3>
            <p>
              Import an Xray or sing-box executable before assigning advanced
              protocols to a profile.
            </p>
          </section>
        ) : (
          records.map((record) => (
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
                <span className={`kernel-status ${record.status}`}>
                  {record.status}
                </span>
              </div>
              <div className="adapter-protocols">
                {record.protocols.map((protocol) => (
                  <span key={protocol}>{protocol}</span>
                ))}
              </div>
              <dl>
                <div>
                  <dt>SHA-256</dt>
                  <dd title={record.sha256}>
                    {record.sha256.slice(0, 16)}…{record.sha256.slice(-8)}
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
              <div className="kernel-actions">
                <button
                  className="button secondary"
                  disabled={busy}
                  onClick={() => void onVerify(record)}
                >
                  Verify now
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
          ))
        )}
      </div>
    </>
  );
}
