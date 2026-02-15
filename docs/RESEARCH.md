# Browser Bridge — Interface Research

## The Question: Why Screenshots?

Screenshots are the worst way for an AI agent to interact with a browser. Here's why, and what's better.

## Three Approaches Compared

### 1. Screenshots (Vision-based)
**How it works:** Render page → PNG → base64 → send to vision model → model interprets pixels

**Problems:**
- **Expensive:** A single screenshot = ~1,000-2,000 tokens (image tokens). A 10-step task burns 10-20k tokens just on screenshots
- **Slow:** Render → encode → transfer → vision model inference. Each step adds latency
- **Unreliable:** Vision models hallucinate button positions, misread text, can't see off-screen content
- **No element refs:** Model says "click the blue button at (342, 187)" — coordinates drift between renders
- **Requires vision model:** Not all LLMs have vision. Limits which models can use the bridge

**When it's actually useful:**
- Visual verification (does this page *look* right?)
- CAPTCHA solving
- Canvas/WebGL content (no DOM equivalent)
- Debugging — human wants to see what the agent sees

### 2. Accessibility Tree Snapshots (What we already have)
**How it works:** CDP `Accessibility.getFullAXTree()` → structured tree → flat JSON with refs

**Advantages:**
- **~4x cheaper than screenshots** (Playwright team benchmarks: ~27k vs ~114k tokens for same task via MCP)
- **Fast:** Pure data, no rendering/encoding
- **Deterministic refs:** `e0`, `e1`, `e2`... — stable element references the agent can click/type
- **Works with any LLM:** Text-only, no vision required
- **Semantic:** Role + name + value tells the agent what things *are*, not what they look like

**Limitations:**
- Doesn't capture visual layout (relative positioning, colors, spacing)
- Custom widgets with poor ARIA labeling become invisible
- Very large pages = very large trees (need pruning)

### 3. Playwright CLI Approach (Files on Disk) — NEW
**How it works:** Snapshots/screenshots saved to disk files, agent reads only what it needs

**Key insight from Playwright CLI (Microsoft, Feb 2026):**
- MCP returns full snapshot inline → bloats context window
- CLI saves to `.playwright-cli/snapshot.yaml` → agent reads if needed
- **4x token reduction** vs MCP for same tasks (27k vs 114k tokens)
- Screenshots never enter context unless agent explicitly reads the file

**Implication for Browser Bridge:**
- Add `/snapshot?format=file` — save to disk, return path only
- Add `/screenshot?save=true` — save PNG to disk, return path
- Agent decides whether to read the file or just act on previous knowledge

## Hybrid Strategy (Recommended)

The best approach combines methods based on the task:

| Task | Best Method |
|------|------------|
| Navigate + interact | Accessibility snapshot (default) |
| Verify visual appearance | Screenshot (on demand) |
| Fill forms, click buttons | Snapshot refs (`e0`, `e1`) |
| Read page content | Snapshot + `/evaluate` for innerText |
| Debug / human review | Screenshot saved to disk |

### Cost Comparison (Estimated per 10-step task)

| Method | Tokens | Cost (Claude) | Latency |
|--------|--------|---------------|---------|
| Screenshot every step | ~20,000 | ~$0.06 | ~500ms/step |
| A11y snapshot every step (inline) | ~5,000 | ~$0.015 | ~100ms/step |
| A11y snapshot (file, read 2 of 10) | ~1,500 | ~$0.005 | ~80ms/step |
| Hybrid (snapshot + 1 screenshot) | ~7,000 | ~$0.02 | ~150ms avg |

## What the Industry Is Doing

### Playwright MCP (Microsoft)
- **Default:** Accessibility snapshots, no screenshots
- "Bypasses the need for screenshots or visually-tuned models"
- Flat YAML format with element refs
- Vision mode available but opt-in

### Browser Use (open source, 8k+ stars)
- Uses accessibility tree + optional screenshots
- Their benchmark: 100 hard tasks, best models hit ~60% success
- Cost: $10-100 per 100-task evaluation run depending on model
- Key finding: structured data >> pixel interpretation for reliability

### OpenClaw (what we use)
- Already does this! `browser` tool uses accessibility snapshots by default
- `refs="aria"` for stable cross-call references
- Screenshots available but secondary

### Computer Use (Anthropic/OpenAI)
- Full screenshot-based approach (most expensive)
- Works for anything but costs ~10x more tokens
- Designed for general computer control, not browser-specific tasks

## Key Takeaways

1. **Accessibility tree is the primary interface** — already implemented in our `/snapshot`
2. **Screenshots should be opt-in, not default** — keep `/screenshot` but don't use it as the main interaction loop
3. **File-based output reduces tokens dramatically** — add save-to-disk option for both snapshots and screenshots
4. **Ref resolution is the hardest part** — our current `resolveRef()` uses heuristics. Need to improve this
5. **Pruning large trees** — add depth/filter params to `/snapshot` to keep token count low

## Sources
- Playwright MCP: github.com/microsoft/playwright-mcp
- Playwright CLI benchmarks: testcollab.com/blog/playwright-cli (Feb 2026)
- Browser Use benchmark: browser-use.com/posts/ai-browser-agent-benchmark
- "Building Browser Agents" paper: arxiv.org/abs/2511.19477 (Vardanyan, 2025)
