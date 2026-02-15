# Steel Browser Analysis

**Repo:** github.com/steel-dev/steel-browser  
**Codebase:** ~12,000 lines TypeScript (API only), plus React UI  
**Stack:** Fastify + Puppeteer-core + LevelDB + DuckDB + React  

## What Steel Actually Does

Steel is a **scraping/extraction** service that happens to use a browser. It is NOT an agent interaction layer.

### Their 4 Endpoints (that's it)
1. `POST /scrape` — Navigate to URL, extract HTML/markdown/readability/cleaned_html
2. `POST /screenshot` — Navigate to URL, return JPEG screenshot
3. `POST /pdf` — Navigate to URL, return PDF
4. `POST /search` — Scrape Brave Search results for a query

### What They DON'T Have
- ❌ **No accessibility tree / snapshot** — Zero references to `accessibility`, `snapshot`, `a11y`, or `AXTree` in the entire codebase
- ❌ **No element interaction** — No click, type, fill, hover, select
- ❌ **No action API** — Can't interact with pages at all
- ❌ **No JS evaluation** — Can't run arbitrary JavaScript
- ❌ **No tab management** — Single primary page, not multi-tab

### What They DO Have (that we don't)
- ✅ Plugin system (BasePlugin with lifecycle hooks)
- ✅ Session management (isolated browser contexts, cookies, localStorage persistence via LevelDB)
- ✅ Proxy chain support (per-request proxy rotation)
- ✅ Anti-detection / stealth (fingerprinting, user-agent management)
- ✅ Selenium WebDriver compatibility layer
- ✅ File upload/download management
- ✅ Request logging with DuckDB storage
- ✅ PDF-to-HTML conversion
- ✅ HTML-to-Markdown (Turndown + Readability/Defuddle)
- ✅ React UI for session monitoring
- ✅ OpenAPI spec (Scalar docs)
- ✅ Docker packaging

## Architecture Issues

### Overengineered for what it does
- 12,000 lines of TypeScript for 4 endpoints
- Plugin system with 7 lifecycle hooks — but ships with no plugins
- DuckDB + LevelDB + in-memory storage abstractions — for request logs
- Selenium compatibility layer — adds complexity for legacy support
- WebSocket proxy infrastructure — for live viewing, not agent control

### Missing the core problem
Steel solves "how do I scrape web pages reliably" — which is a solved problem (Firecrawl, Crawlee, etc). It does NOT solve "how does an AI agent interact with a web page" — which is the actual hard problem.

**No accessibility tree = no agent interaction.** An agent can't fill a form, click a button, or navigate a SPA with Steel. It can only fetch content.

### Cloud-first mindset baked in
- Session management designed for multi-tenant cloud (session IDs, isolation)
- Proxy support for bypassing rate limits at scale
- Anti-detection for scraping hostile sites
- All of this is irrelevant for local agent use

## What Browser Bridge Should Be (First Principles)

### The actual problem
An AI agent needs to:
1. **See** what's on a page (cheaply, structured)
2. **Act** on elements (click, type, select)
3. **Navigate** between pages
4. **Extract** specific data
5. **Verify** results

### Minimum viable surface
```
GET  /tabs                          # What's open
GET  /snapshot?tabId=X              # Accessibility tree (primary interface)
POST /action    {kind, ref, text}   # Click/type/fill by a11y ref
POST /navigate  {url}               # Go somewhere  
POST /evaluate  {expression}        # Run JS (escape hatch)
GET  /screenshot?tabId=X            # Visual verification (opt-in)
GET  /text?tabId=X                  # Readable text extraction
```

That's 7 endpoints. Not 4 wrong ones + 30 supporting files.

### What we should steal from Steel
1. **HTML-to-Markdown extraction** — Useful for content reading. But as a `/text` endpoint, not the primary interface
2. **Docker packaging** — Single `docker run` to start
3. **OpenAPI spec** — Good for any HTTP client to auto-discover

### What we should NOT copy
1. Session management (overkill for local use)
2. Plugin system (YAGNI — add when needed)
3. Selenium compatibility (legacy baggage)
4. Proxy chains (not our problem)
5. Anti-detection (not our problem)
6. React UI (curl is the UI)

## Competitive Position

| Feature | Steel | Browser Bridge | Playwright MCP |
|---------|-------|---------------|----------------|
| A11y tree snapshot | ❌ | ✅ | ✅ |
| Element interaction | ❌ | ✅ | ✅ |
| Content extraction | ✅ | ⚡ TODO | ❌ |
| Screenshot | ✅ | ✅ | ✅ (opt-in) |
| JS evaluation | ❌ | ✅ | ✅ |
| Multi-tab | ❌ | ✅ | ❌ |
| Protocol | HTTP | HTTP | MCP (JSON-RPC) |
| Any agent can use | ✅ | ✅ | ❌ (MCP only) |
| Docker | ✅ | TODO | ❌ |
| Lines of code | ~12,000 | ~350 | ~5,000 |
| Zero config | ❌ | ✅ | ❌ |

**Browser Bridge is already more useful for AI agents than Steel, in 350 lines vs 12,000.**

Steel is a scraping tool marketed as a browser API. We're building an actual agent-browser interface.

## Conclusion

Don't copy Steel. Don't compete with Steel. They solve a different problem (web scraping infrastructure) and solve it with 30x more code than necessary.

Our angle: **the simplest possible bridge between any AI agent and a real browser.** Accessibility-first, HTTP API, zero config, works with curl.

The closest competitor is actually **Playwright MCP** — but that requires MCP protocol support. We're HTTP, which means anything can use us.
