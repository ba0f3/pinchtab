# Pinchtab — TODO

**Philosophy**: 12MB binary. HTTP API. Minimal deps. Internal tool, not a product.

---

## DONE

Core HTTP API, 16 endpoints, session persistence, ref caching, action registry,
smart diff, readability `/text`, config file, Dockerfile, YAML/file output,
stealth suite (navigator, WebGL, canvas noise, font metrics, WebRTC, timezone,
plugins, Chrome flags), human interaction (bezier mouse, typing simulation),
fingerprint rotation via CDP (`SetUserAgentOverride`, `SetTimezoneOverride`),
handlers.go split (4 files), architecture docs, image/media blocking
(`BRIDGE_BLOCK_IMAGES`, `BRIDGE_BLOCK_MEDIA`), stealth injection on all tabs,
K1-K9 all fixed, multi-agent concurrency verified (MA1-MA8).
**92 unit tests + ~100 integration tests, 28.9% coverage.**

---

## Open

### ~~P0: Stability~~ — DONE
- [x] **K10 — Profile hang** — Fixed: lock file cleanup, unclean exit detection, 15s Chrome timeout, auto-retry with session clear.
- [ ] **Coverage to 30%** — Add tests for cookie/stealth handler happy paths (~2% gap).

### ~~P1: Token Optimization~~ — DONE
- [x] **`maxTokens` param on `/snapshot`** — Truncates to ~N tokens, returns `truncated: true` flag. Wikipedia 142K → 1.4K with maxTokens=2000.
- [x] **`selector` param on `/snapshot`** — Scopes a11y tree to CSS selector subtree (e.g. `?selector=main`, `?selector=%23content`).
- [x] **`format=compact`** — One-line-per-node format: 56-64% fewer tokens than JSON. All three compose: `?selector=main&maxTokens=2000&format=compact`.

### ~~P2: Bugs~~ — DONE
- [x] **K11 — File output ignores path** — Fixed: `?output=file&path=X` now honors custom path, auto-creates parent dirs.
- [x] **`blockImages` on CreateTab** — Fixed: Global `BRIDGE_BLOCK_IMAGES`/`BRIDGE_BLOCK_MEDIA` now applied on `CreateTab`.

### P3: Multi-Agent
- [ ] **Tab locking** — `POST /tab/lock`, `POST /tab/unlock` with timeout-based deadlock prevention.
- [ ] **Tab ownership tracking** — Show owner in `/tabs` response.

### P4: Quality of Life
- [ ] **Headed mode testing** — Run Section 2 tests to validate non-headless.
- [ ] **Ad blocking** — Basic tracker blocking for cleaner snapshots.
- [ ] **CSS animation disabling** — Faster page loads, more consistent snapshots.
- [ ] **Randomized window sizes** — Avoid automation fingerprint.

### Minor
- [ ] **Dockerfile env vars** — `CHROME_BINARY`/`CHROME_FLAGS` set but not consumed by Go.
- [ ] **humanType global rand** — Accept `*rand.Rand` for reproducible tests.

---

## Not Doing
Desktop app, plugin system, proxy rotation, SaaS, Selenium compat, MCP protocol,
cloud anything, distributed clusters, workflow orchestration.
