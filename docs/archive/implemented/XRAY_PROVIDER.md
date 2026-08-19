# Supervised Xray provider

Veilium can run a locally imported, integrity-verified Xray executable for a constrained set of advanced proxy profiles. The browser never receives the remote server credential or the original advanced-protocol URL. It connects only to a temporary SOCKS5 listener on `127.0.0.1`.

## Runtime sequence

1. Re-verify the Profile's managed Xray executable by SHA-256 and size.
2. Resolve the selected credential from the operating-system vault.
3. Parse the constrained Profile URL and reject unknown or ambiguous options.
4. Allocate a temporary loopback SOCKS5 port.
5. Generate a private Xray JSON file in a per-session directory with mode `0600`.
6. Start Xray through the same Unix process-group or Windows Job Object ownership used for Chromium.
7. Perform a real SOCKS5 no-auth handshake against the loopback listener.
8. Start Chromium with only `--proxy-server=socks5://127.0.0.1:<port>`.
9. Stop Chromium if the Xray process exits unexpectedly.
10. Stop Xray and remove the private configuration and log directory when the browser exits or Veilium shuts down.

The Xray executable, private configuration, and browser process are all managed separately. A healthy Xray listener is required before Chromium starts.

## Supported protocol subset

- VLESS
- VMess
- Trojan
- Shadowsocks (`ss` or `shadowsocks`)

Supported transports:

- RAW (`type=raw` or `type=tcp`)
- WebSocket (`type=ws` or `type=websocket`)
- gRPC (`type=grpc`)

Supported transport security:

- TLS
- REALITY for VLESS and Trojan with RAW or gRPC
- `none` only for VMess in the current reviewed subset

Veilium intentionally rejects insecure VLESS/Trojan public routes, unknown query parameters, VMess+REALITY, WebSocket+REALITY, invalid ALPN values, inline URL user information, paths/fragments outside the approved query schema, and `allowInsecure` behavior.

## Constrained Profile URL format

Veilium does not yet claim compatibility with every ecosystem share-link format. A Profile stores only the remote endpoint and non-secret transport metadata.

Examples:

```text
vless://server.example:443?security=tls&type=raw&sni=server.example&encryption=none
vmess://server.example:443?security=tls&type=ws&sni=server.example&path=%2Fsocket&host=cdn.example&cipher=auto
trojan://server.example:443?security=tls&type=grpc&sni=server.example&serviceName=proxy
ss://server.example:8388?method=aes-256-gcm
```

The URL must not contain a UUID, password, REALITY key, inline user information, or a fragment.

## Credential-vault values

The Profile references a metadata-only credential record. Its secret remains inside the operating-system vault.

Simple secrets:

- VLESS and VMess: canonical UUID
- Trojan: password
- Shadowsocks: password; the credential username may hold the encryption method when `method` is omitted from the Profile URL

REALITY needs both the protocol identity/password and the server X25519 public value. Store them as a strict JSON object in the vault secret:

```json
{
  "id": "5783a3e7-e373-51cd-8642-c83782b807c5",
  "realityPassword": "SERVER_X25519_PUBLIC_VALUE"
}
```

For Trojan REALITY, use `password` instead of `id`. Unknown JSON fields are rejected. The REALITY value is never accepted in the persisted Profile URL.

## Private configuration boundary

The generated Xray JSON contains network credentials, so Veilium:

- creates it only after runtime credential resolution;
- writes it to a unique directory with private permissions;
- never returns it through Bootstrap, Wails bindings, diagnostics, or browser arguments;
- removes it after normal stop, process exit, startup failure, or application shutdown.

An abrupt operating-system or power failure can leave a `0600` file in the private application directory. Automated stale-session recovery requires a future single-instance lock and crash-recovery design so one Veilium process cannot delete another active process's configuration.

## Remaining limits

This feature does not provide:

- XHTTP, HTTPUpgrade, mKCP, Hysteria transport, or arbitrary Xray JSON passthrough;
- legacy VMess base64 share-link import;
- automatic Xray download, publisher signature verification, or provenance proof;
- ML-DSA-65 REALITY verification fields;
- advanced routing rules, multiplexing, custom DNS, TUN, or remote control;
- sing-box execution;
- a real external Xray server in hosted CI.

CI validates generated configuration structure, secret boundaries, the real managed-process lifecycle, SOCKS5 readiness behavior, browser handoff, cleanup, and Windows/Linux builds. A user-supplied Xray binary and authorized endpoint are still required for real-world acceptance testing.
