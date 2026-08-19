# Authenticated proxy bridge

Veilium uses a per-browser, in-process loopback HTTP proxy to adapt upstream proxy authentication for Chromium without placing credentials in command-line arguments, profile metadata, logs, or frontend state.

## Supported upstreams

- `http://host:port` with username/password from the operating-system vault;
- `https://host:port` with TLS 1.2 or newer and vault-backed authentication;
- `socks5://host:port` with RFC 1929 username/password authentication.

VMess, VLESS, Trojan, Shadowsocks, Hysteria2, TUIC, and AnyTLS remain blocked until separate pinned Xray or sing-box adapters are implemented.

## Runtime flow

1. Validate the profile's proxy URL and credential reference.
2. Resolve the username and password from the operating-system vault only when the user starts the profile.
3. Start an HTTP proxy listener on an ephemeral IPv4 loopback port.
4. Perform a loopback health check before launching Chromium.
5. Replace the browser-facing proxy argument with `http://127.0.0.1:<random-port>`.
6. Keep the original upstream address only as a non-secret display value.
7. Close the bridge when startup fails, the browser is stopped, the browser exits naturally, or Veilium shuts down.

## HTTP and CONNECT behavior

Ordinary HTTP requests are forwarded by a dedicated transport configured for the authenticated upstream. Incoming `Proxy-Authorization` and hop-by-hop headers are removed.

HTTPS traffic reaches the bridge through `CONNECT`. The bridge creates an authenticated tunnel through the HTTP/HTTPS proxy or dials the target through the authenticated SOCKS5 upstream. It never performs TLS interception and never sees the encrypted application payload.

## Security boundaries

- the local listener is bound only to `127.0.0.1`;
- the listener uses bounded headers and timeouts and suppresses server error logs;
- inline credentials in proxy URLs remain forbidden;
- password values are not returned through Wails or Bootstrap;
- bridge errors do not include usernames, passwords, or authenticated proxy URLs;
- no plaintext credential fallback is provided;
- a bridge is created per running profile and is not shared between identities.

The loopback listener currently has no additional bearer-token handshake. Its random port is short-lived, accessible only from the same host, and removed with the browser session. A future local-process authentication mechanism may be added if the threat model requires protection from other untrusted local processes.
