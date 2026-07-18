# Managed proxy adapter runtime

Veilium treats Xray and sing-box executables as explicit, locally imported dependencies. It does not silently download, update, or execute an unverified binary.

## Managed import flow

1. The user selects an existing local Xray or sing-box executable in the Wails desktop application.
2. Veilium requires a declared version, HTTPS source URL, and SPDX license identifier.
3. The source must be a regular file; symbolic links, directories, and empty files are rejected.
4. The binary is copied into Veilium's private adapter directory through a temporary file.
5. Veilium records the SHA-256 digest, byte size, adapter kind, declared version, source URL, license identifier, supported protocol family, and verification timestamps.
6. Importing the same digest, kind, and version is idempotent.

The original source path is not persisted. Remote downloads are deliberately unavailable in this phase.

## Adapter families

The registry maps protocol schemes to one managed adapter family:

- Xray: `vmess`, `vless`, `trojan`, `ss`, and `shadowsocks`;
- sing-box: `hysteria2`, `tuic`, and `anytls`.

A profile using one of these schemes must reference a verified adapter of the matching kind. HTTP, HTTPS, SOCKS5, and direct routes must not reference an Xray or sing-box adapter.

## Integrity states

- `verified`: digest and byte size still match the imported record;
- `modified`: the managed binary changed, became a symbolic link, or is no longer a regular file;
- `missing`: the managed binary no longer exists.

Veilium re-verifies the adapter when a profile is saved, when a launch plan is reviewed, and immediately before runtime preparation. A modified or missing adapter is blocked.

## License and provenance boundary

The source URL and SPDX value begin as user-declared provenance metadata. Veilium now compares imported executables with an embedded official release manifest. Exact kind, version, executable SHA-256, and byte-size matches receive an `official` identity and canonical release metadata; all other files remain clearly labeled `custom local`.

The official identity still does not prove reproducible source provenance or replace publisher signing. Users remain responsible for obtaining binaries lawfully and reviewing upstream licenses, notices, and redistribution obligations. See [`OFFICIAL_ADAPTER_VALIDATION.md`](OFFICIAL_ADAPTER_VALIDATION.md).

## Runtime provider boundary

The trusted registry exposes a typed provider interface. Veilium includes constrained Xray and sing-box providers. Unreviewed protocols, options, and transport combinations still fail closed with explicit errors.

A provider must:

- accept only a verified managed adapter record;
- generate private, per-session configuration files;
- bind browser-facing inbound listeners only to loopback;
- avoid writing credentials or share-link secrets to logs;
- delete temporary configuration and stop the adapter process with the browser session;
- return a browser-safe local proxy endpoint only after readiness checks pass.

## Deletion safety

An adapter cannot be removed while any profile references its ID. Removal first moves its managed directory to a private quarantine path, persists metadata deletion, and then deletes the quarantined files. Persistence failure attempts to restore the directory and record.

## Executable providers

Runtime providers are documented in [`XRAY_PROVIDER.md`](XRAY_PROVIDER.md) and [`SING_BOX_PROVIDER.md`](SING_BOX_PROVIDER.md). Both remain constrained to explicitly reviewed protocol and option subsets.
