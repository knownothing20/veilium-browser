# Supervised sing-box provider

Veilium includes a constrained sing-box configuration provider for Hysteria2, TUIC, and AnyTLS profiles. It uses only a locally imported, integrity-verified sing-box executable and never downloads or updates the binary automatically.

## Runtime sequence

1. Re-verify the managed sing-box executable.
2. Resolve the selected credential from the operating-system vault.
3. Strictly parse the profile URL and reject unknown, duplicate, or unsafe options.
4. Generate a private per-session JSON configuration.
5. Start sing-box under Veilium's Unix process-group or Windows Job Object supervision.
6. Require a real SOCKS5 no-auth handshake on a random IPv4 loopback port.
7. Give Chromium only `socks5://127.0.0.1:<port>`.
8. Stop sing-box and remove its configuration/log directory when the browser exits, startup fails, or Veilium shuts down.

## Supported profile forms

Every URL requires an explicit host and port, contains no inline user information, path, or fragment, and uses only the documented options below. TLS certificate verification is always enabled.

### Hysteria2

```text
hysteria2://server.example:443?sni=server.example&alpn=h3&upMbps=50&downMbps=200&network=udp
```

The vault secret may be the password directly. Salamander obfuscation requires strict JSON:

```json
{
  "password": "hysteria-password",
  "obfsPassword": "obfuscation-password"
}
```

Use `obfs=salamander` in the URL when that field is present.

### TUIC

```text
tuic://server.example:443?serverName=server.example&alpn=h3&congestionControl=bbr&udpRelayMode=quic&network=udp
```

The vault secret must be strict JSON:

```json
{
  "uuid": "2dd61d93-75d8-4da4-ac0e-6aece7eac365",
  "password": "tuic-password"
}
```

Zero-RTT is deliberately disabled. Accepted congestion controls are `cubic`, `new_reno`, and `bbr`; accepted UDP relay modes are `native` and `quic`.

### AnyTLS

```text
anytls://server.example:443?sni=server.example&alpn=h2,http%2F1.1&idleCheck=30s&idleTimeout=45s&minIdle=3
```

The vault secret may be the password directly. AnyTLS requires a declared managed sing-box version of 1.12.0 or newer.

## TLS and DNS boundary

- TLS is mandatory and `insecure` mode is unavailable.
- IP endpoints must provide `sni` or `serverName`.
- ALPN is limited to `h3`, `h2`, and `http/1.1`.
- The private configuration uses a local sing-box DNS server and routes the browser-facing SOCKS inbound to the single reviewed outbound.

## Deliberate limits

This provider does not accept arbitrary sing-box JSON, remote rule sets, TUN inbounds, mixed/listen-all interfaces, multiplexing, ECH, client certificates, custom DNS servers, insecure TLS, endpoint objects, or every ecosystem share-link alias. Unknown fields fail closed. Hosted CI validates generated configuration and runtime lifecycle with a managed test process; it does not connect to a real external sing-box server.
