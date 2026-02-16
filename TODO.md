# Pinchtab — TODO

**Philosophy**: 12MB binary. HTTP API. Zero deps. Internal tool, not a product.
If a feature needs a GUI, complex config, or "target users" — it's probably wrong.

## Completed (P0–P5)
Safety, file split, Go idioms, testability, features — all done.
38 tests passing, 0 lint issues. See git history for details.

Key deliverables: session persistence, stealth mode, ref caching, action registry,
hover/select/scroll actions, smart diff, text format, readability /text,
BridgeAPI interface, handler tests, nil guard, deprecated flag removal.

---

## Bugs & In Progress
- [x] **Navigate timeout on some SPAs** — Fixed: `navigatePage` now polls `document.readyState` until interactive/complete (10s timeout)
- [x] **Restore navigates all tabs at once** — Fixed: Added concurrency limiting (max 3 concurrent tab creations, max 2 concurrent navigations) 
- [x] **Screenshot base64 returns raw bytes** — Fixed: Already working correctly, returns proper base64 string encoding in JSON

## P6: Next Up — COMPLETE! ✅
- [x] **Action chaining** — `POST /actions` batch multiple actions in one call (big token saver for agents)
- [x] **`/cookies` endpoint** — read/set cookies (auth debugging)
- [x] **LaunchAgent/systemd** — auto-start on boot
- [x] **Config file** — `~/.pinchtab/config.json` (alternative to env vars)

## P7: Nice to Have — COMPLETE! ✅
- [x] **File-based output** — `?output=file` saves snapshot to disk, returns path
- [x] **Compact format** — YAML or indented text instead of JSON
- [x] **Docker image** — `docker run pinchtab` with bundled Chromium

## P8: Stealth & Anti-Detection — COMPLETE! ✅
- [x] **Enhanced stealth mode** — Fix additional headless detection vectors (pointer type, viewport handling, content isolations)
- [x] **Human-like interactions** — Natural mouse movement algorithm with timing variations for clicks/types
  - `POST /action {"kind": "humanClick", "ref": "e5", "delay": "random"}`
  - `POST /action {"kind": "humanType", "text": "hello", "typing_delay": "natural"}`
- [x] **Fingerprint rotation** — Randomize navigator properties, screen sizes, WebGL parameters between sessions
  - `POST /fingerprint/rotate {"os": "random", "screen": "random", "webgl": "spoof"}`
- [x] **Stealth profiles API** — Configurable via fingerprint rotation endpoint
- [x] **Anti-fingerprinting core** — Spoof navigator device/OS/hardware properties, screen resolution, WebGL context
- [x] **Font spoofing** — System-appropriate fonts with randomized metrics to prevent font fingerprinting
- [x] **Memory optimization** — Chrome flags reduce memory footprint
- [x] **Request header spoofing** — User-Agent rotation matches navigator properties
- [x] **DNS leak prevention** — WebRTC blocked to prevent IP leaks
- [x] **Stealth status endpoint** — `GET /stealth/status` to check current anti-detection configuration

## P9: Multi-Agent Coordination
**Current status**: Multiple agents can share one Pinchtab instance but may conflict on ref caches and navigation. Best practice: use separate tabs per agent (`newTab: true`) and always specify `tabId`.

- [ ] **Tab locking mechanism** — Prevent conflicts when multiple agents access same Pinchtab instance
  - `POST /tab/lock {"tabId": "tab1", "lockId": "agent-uuid", "timeoutMs": 30000}`
  - `POST /tab/unlock {"tabId": "tab1", "lockId": "agent-uuid"}`
  - All operations require lockId when tab is locked
- [ ] **Agent sessions (optional)** — Isolated browser contexts per agent for complex workflows
  - `POST /session/create {"agentId": "agent1", "profile": "isolated"}` 
  - `GET /session/list` — show active agent sessions
  - `DELETE /session/{sessionId}` — cleanup agent session
- [ ] **Ref cache versioning** — Prevent stale ref conflicts between agents taking overlapping snapshots
- [ ] **Tab ownership tracking** — Show which agent/session owns which tabs in `/tabs` response
- [ ] **Concurrent action queuing** — Queue conflicting actions on same tab instead of failing
- [ ] **Agent-aware cleanup** — Clean up tabs/caches when agent sessions expire

**Implementation priority**: Start with tab locking (solves 80% of conflicts with minimal complexity), then add optional sessions for advanced use cases.

## P10: Quality of Life  
- [ ] **Built-in ad blocking** — Integrate basic ad/tracker blocking for cleaner automation (less noise in snapshots)
- [ ] **CSS animation disabling** — Skip animations for faster page loads and more consistent snapshots
- [ ] **Debloated Chrome launch** — Strip unnecessary Chrome features for lower memory usage and faster startup
- [ ] **Non-default window sizes** — Randomize initial window dimensions to avoid common automation fingerprints
- [ ] **Custom user data directory management** — Better session isolation and cleanup options

## Future: Maybe Never
- Desktop app — Pinchtab is a 12MB HTTP server. Agents don't need GUIs.
- Profile switching — Just use `pinchtab --profile=testing` or config file
- Performance dashboard — `GET /monitor/performance` returns JSON. Done.
- If humans need monitoring, serve a simple `/debug/` page, not a desktop app

Keep it headless. Keep it simple. The HTTP API is the product.

## Not Doing
- Plugin system
- Proxy rotation (IP-level)
- Multi-tenant SaaS (different from multi-agent coordination)
- Selenium compatibility
- React UI
- Cloud anything
- MCP protocol (HTTP is the interface)
- Machine learning / AI integration
- External fingerprint services
- Distributed browser clusters
- Complex workflow orchestration (agents should handle their own logic)
