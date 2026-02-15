# Browser Bridge

Local browser control service for AI agents. Runs on your machine, connects to your Chrome, survives restarts.

## Why

Agents need a browser that:
- **Persists sessions** — cookies, auth, tabs survive Chrome/agent restarts
- **Is directly controllable** — any agent talks to it via HTTP, no SDK needed
- **Uses accessibility trees** — not screenshots. 4x cheaper, works with any LLM
- **Stays out of the way** — zero config, single process, no cloud

## Architecture

```
┌─────────────┐     HTTP :18800    ┌──────────────┐      CDP       ┌─────────┐
│  Any Agent  │ ──────────────►   │ Browser      │ ─────────────► │ Chrome  │
│  (OpenClaw, │  snapshot, act,   │ Bridge       │  DevTools      │         │
│   PicoClaw, │  navigate, eval   │              │  Protocol      │         │
│   curl)     │                   │  sessions +  │                │         │
└─────────────┘                   │  tab state   │                └─────────┘
                                  └──────────────┘
```

## API

```bash
# What's open
curl localhost:18800/tabs

# See the page (accessibility tree — cheap, structured, actionable)
curl localhost:18800/snapshot?tabId=X

# Interact
curl -X POST localhost:18800/action -d '{"tabId":"X","kind":"click","ref":"e5"}'
curl -X POST localhost:18800/action -d '{"tabId":"X","kind":"type","ref":"e12","text":"hello"}'

# Navigate
curl -X POST localhost:18800/navigate -d '{"tabId":"X","url":"https://example.com"}'

# Run JS
curl -X POST localhost:18800/evaluate -d '{"tabId":"X","expression":"document.title"}'

# Visual check (opt-in, expensive)
curl localhost:18800/screenshot?tabId=X

# Readable text (like reader mode)
curl localhost:18800/text?tabId=X
```

## Sessions

Browser Bridge saves tab state (URLs, positions) to `~/.browser-bridge/sessions.json`. When Chrome restarts, it restores your tabs. When the bridge restarts, it reconnects.

Auth cookies, localStorage, sessionStorage — all live in Chrome's profile. Bridge doesn't touch them, they just persist naturally.

## Quick Start

```bash
# 1. Start Chrome with remote debugging
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222

# 2. Start the bridge  
node server.js

# 3. Use it
curl localhost:18800/health
```

## Not Goals

- Not a product, not a cloud service
- Not a scraping tool
- Not trying to replace Playwright MCP
- No plugin system, no SDK, no React UI
