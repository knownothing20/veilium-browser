# Proxy health diagnostics

Veilium can run an on-demand, non-persistent health report for any browser profile. The report uses the same credential resolution and loopback authentication bridge used by the browser runtime.

## Measured checks

The desktop application performs one HTTPS request through the selected route to the configured exit-IP probe. The default endpoint is the universal JSON endpoint from ipify:

```text
https://api64.ipify.org?format=json
```

The request contains no profile name, credential ID, username, password, cookies, or browser fingerprint data.

The report measures:

- whether the selected route can complete an HTTPS request;
- time until the first response byte;
- total request duration;
- the public IPv4 or IPv6 address observed by the probe;
- temporary bridge startup health for vault-backed proxies.

Diagnostic reports are held only in frontend memory. They are not written to `profiles.json`, `credentials.json`, runtime logs, or a history database.

## DNS assessment

For HTTP, HTTPS, and SOCKS5 proxy routes, Veilium passes destination hostnames to the upstream proxy instead of resolving the diagnostic target in the local diagnostic client.

This is a protocol-path assessment. It is not a third-party DNS-leak test using a unique delegated domain, and it does not identify the recursive resolver used by the proxy provider.

Direct profiles are marked as using the operating-system resolver.

## WebRTC assessment

The report audits the profile setting:

- `disabled`: pass;
- `proxy-only`: pass;
- `default` while using a proxy: warning;
- direct route: skipped.

This is not a live STUN test inside the launched Chromium profile. A future browser-integrated test page or controlled extension is required before Veilium can claim real WebRTC leak detection.

## Supported routes

- direct connection baseline;
- unauthenticated HTTP and HTTPS proxies;
- unauthenticated SOCKS5 proxies;
- vault-backed HTTP, HTTPS, and SOCKS5 proxies through the loopback bridge.

Xray and sing-box protocol families remain unsupported and return a failed adapter check.

## Security boundaries

- production probe URLs must use HTTPS;
- plain HTTP probe URLs are accepted only for loopback integration tests;
- redirects are rejected;
- response bodies are size-limited;
- TLS requires version 1.2 or newer;
- environment proxy variables are ignored;
- vault material is redacted from any returned error text;
- temporary diagnostic bridges are always closed after a report;
- only one report may run per profile at a time.

The exit IP itself is network-sensitive information and is shown only to the local desktop user.
