# Browser Bridge — TODO

## Goal
Better browser for our agents: session persistence across restarts + direct controllability via HTTP.

## P0: Working Foundation
- [ ] **Test with Chrome CDP** — restart Chrome with `--remote-debugging-port=9222`
- [ ] **Session persistence** — save tab state (URLs, scroll position) to `~/.browser-bridge/sessions.json` on shutdown, restore on startup
- [ ] **Auto-reconnect** — when Chrome restarts, bridge reconnects without dying
- [ ] **Improve ref resolution** — current `resolveRef()` is fragile. Use CDP `DOM.resolveNode` from a11y node backendDOMNodeId
- [ ] **`/text` endpoint** — readable text extraction (Readability/Defuddle style) for content reading tasks

## P1: Daily Driver Quality  
- [ ] **Snapshot pruning** — `?filter=interactive` (buttons/links/inputs only), `?depth=N`, `?maxNodes=N`
- [ ] **Smart diff** — `?diff=true` returns only changes since last snapshot for that tab. Massive token savings on multi-step tasks
- [ ] **Launch helper** — `node server.js --launch` starts Chrome with CDP flag if not already running
- [ ] **Graceful shutdown** — save state on SIGTERM/SIGINT, clean disconnect
- [ ] **Tab restore on Chrome restart** — detect Chrome restart, reopen saved tabs
- [ ] **Error messages that help** — "Chrome not running with CDP" → "Run: /Applications/Google Chrome.app/... --remote-debugging-port=9222"

## P2: Nice to Have
- [ ] **File-based output** — `?output=file` saves snapshot to disk, returns path. Agent reads only if needed (Playwright CLI approach, 4x token savings)
- [ ] **Compact format** — YAML or indented text instead of JSON for snapshots
- [ ] **Action chaining** — `POST /actions` batch multiple actions in one call
- [ ] **Docker image** — `docker run browser-bridge` with bundled Chromium
- [ ] **Config file** — `~/.browser-bridge/config.json` for port, Chrome path, auth token
- [ ] **LaunchAgent/systemd** — auto-start on boot

## Not Doing
- Plugin system
- Proxy rotation / anti-detection  
- Session isolation / multi-tenant
- Selenium compatibility
- React UI
- Cloud anything
- MCP protocol (HTTP is the interface)
