# Veilium development rules

- Never commit directly to `main`; use reviewed branches and Draft PRs.
- Do not copy source code from Donut Browser, Ant Browser, VirtualBrowser, or their browser kernels.
- New fingerprint fields require a provider/version capability contract and tests.
- Do not claim a setting is applied until an integration test verifies the selected browser binary.
- Local APIs must bind to loopback and require authentication by default.
- Never log proxy passwords, cookies, tokens, or decrypted browser data.
- Do not add workflow write permissions, automatic merging, or deployment without explicit review.
- Keep platform-specific runtime code behind interfaces and test the portable policy layer on Linux and Windows.
