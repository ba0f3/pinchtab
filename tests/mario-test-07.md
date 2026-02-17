# Pinchtab Full Test Report — 07:46 UTC, 2026-02-17

**Run by:** Mario (manual, full test plan)
**Branch:** autorun
**Instance:** headless, clean profile, BRIDGE_NO_RESTORE=true

---

## Summary

| Category | Pass | Fail | Skip | Total |
|----------|------|------|------|-------|
| Unit tests | 92 | 0 | 0 | 92 |
| Integration tests | ~100 | 0 | 0 | ~100 |
| Live curl (Section 1) | 36 | 2 | 0 | 38 |
| **Total** | **~228** | **2** | **0** | **~230** |

---

## Unit Tests
- **Result:** ✅ ALL 92 PASS
- **Duration:** 0.185s
- **Coverage:** 28.9%

## Integration Tests (stealth, require Chrome)
- **Result:** ✅ ALL PASS (~100 including subtests)
- **Duration:** 3.3s

---

## Live Curl Tests (against running instance)

### Section 1.1: Health & Startup

| # | Scenario | Status | Detail |
|---|----------|--------|--------|
| H1 | Health check | ✅ PASS | `{"cdp":"","status":"ok","tabs":1}` |
| H2 | Startup headless | ✅ PASS | Launched with BRIDGE_HEADLESS=true |
| H5-H6 | Auth token | ⏭️ NOT TESTED | Would need separate instance with BRIDGE_TOKEN |

### Section 1.2: Navigation

| # | Scenario | Status | Detail |
|---|----------|--------|--------|
| N1 | Navigate example.com | ✅ PASS | title="Example Domain" |
| N2 | Navigate BBC | ✅ PASS | title contains "BBC" |
| N3 | Navigate SPA (x.com) | ✅ PASS | title="" (expected — SPA limitation) |
| N4 | Navigate newTab | ✅ PASS | tabId returned |
| N5 | Invalid URL | ✅ PASS | Error returned |
| N6 | Missing URL | ✅ PASS | Error returned |
| N7 | Bad JSON | ✅ PASS | Parse error returned |

### Section 1.3: Snapshot

| # | Scenario | Status | Detail |
|---|----------|--------|--------|
| S1 | Basic snapshot | ✅ PASS | Nodes array with refs |
| S2 | Interactive filter | ✅ PASS | Only 1 node (link "Learn more") |
| S3 | Depth filter | ✅ PASS | Truncated at depth 2 |
| S4 | Text format | ✅ PASS | Plain text output with page content |

### Section 1.4: Text Extraction

| # | Scenario | Status | Detail |
|---|----------|--------|--------|
| T1 | Text extraction | ✅ PASS | Clean text returned |
| T2 | Raw text mode | ✅ PASS | 199 chars |

### Section 1.5: Actions

| # | Scenario | Status | Detail |
|---|----------|--------|--------|
| A1 | Click by ref | ✅ PASS | `{"clicked":true}` |
| A4 | Press key | ✅ PASS | `{"pressed":"Enter"}` |
| A9 | Unknown kind | ✅ PASS | Error: invalid kind |
| A10 | Missing kind | ✅ PASS | Error returned |
| A11 | Ref not found | ✅ PASS | Error returned |

### Section 1.6: Tabs

| # | Scenario | Status | Detail |
|---|----------|--------|--------|
| TB1 | List tabs | ✅ PASS | Tabs array returned |
| TB2 | New tab | ✅ PASS | Tab created with tabId |
| TB3 | Close tab | ❌ **FAIL** | `"close tab: to close the target, cancel its context or use chromedp.Cancel"` — **K2 not fully fixed** |
| TB4 | Close without tabId | ✅ PASS | Error: tabId required |
| TB5 | Bad action | ✅ PASS | Error: invalid action |

### Section 1.7: Screenshots

| # | Scenario | Status | Detail |
|---|----------|--------|--------|
| SS1 | Basic screenshot | ✅ PASS | 21KB JPEG data |
| SS2 | Raw screenshot | ✅ PASS | HTTP 200, file saved |

### Section 1.8: Evaluate

| # | Scenario | Status | Detail |
|---|----------|--------|--------|
| E1 | Simple eval (1+1) | ✅ PASS | `{"result":"2"}` |
| E2 | DOM eval (title) | ✅ PASS | `{"result":"Example Domain"}` |
| E3 | Missing expression | ✅ PASS | Error returned |
| E4 | Bad JSON | ✅ PASS | Parse error |

### Section 1.9: Cookies

| # | Scenario | Status | Detail |
|---|----------|--------|--------|
| C1 | Get cookies | ✅ PASS | `{"cookies":[],"count":0}` |
| C2 | Set cookies | ✅ PASS | `{"failed":0,"set":1,"total":1}` |

### Section 1.10: Stealth

| # | Scenario | Status | Detail |
|---|----------|--------|--------|
| ST1 | Stealth status | ✅ PASS | Score and level returned |
| ST2 | Webdriver hidden | ❌ **FAIL** | `navigator.webdriver` returns `true` — **stealth script not injecting on this tab** |
| ST3 | Chrome runtime | ✅ PASS | `window.chrome` present |
| ST4 | Plugins present | ✅ PASS | 5 plugins |
| ST5 | Fingerprint rotate (windows) | ✅ PASS | Windows UA applied |
| ST6 | Fingerprint rotate (random) | ✅ PASS | Random fingerprint applied |

---

## Failures Analysis

### TB3: Close tab — K2 regression
The `target.CloseTarget` fix from Bosch's hour 07 run isn't working on the `autorun` branch. The error message says "cancel its context or use chromedp.Cancel" — this is the **old** error pattern. The fix may not have been merged, or there's a different code path being hit.

**Action needed:** Verify the K2 fix is on the `autorun` branch. May need cherry-pick from main.

### ST2: navigator.webdriver not hidden
On a freshly navigated tab, `navigator.webdriver` returns `true` instead of `undefined`. The stealth script may not be injecting on all tabs (only the initial one), or it needs to be re-injected after navigation.

**Action needed:** Check if stealth injection runs on every new page load or only on startup.

---

## Sections Not Tested

| Section | Reason |
|---------|--------|
| 1.1 H2-H7 | Need separate instances (auth, graceful shutdown) |
| 1.3 S5-S12 | YAML format, diff mode, file output, large pages |
| 1.5 A2-A3, A5-A8, A12-A17 | Type, fill, focus, hover, select, scroll, CSS selector, batch, human actions |
| 1.6 TB6 | Max tabs limit |
| 2. Headed Mode | Requires non-headless |
| 3. Multi-Agent | Requires concurrent test harness |
| 5. Docker | Requires Docker build |
| 6. Config Extended | Requires multiple instances |
| 7. Error Handling | Chrome crash, large page, binary page, rapid nav |

---

## Performance Metrics

| Metric | Value |
|--------|-------|
| Build time | 0.36s |
| Binary size | 12.4 MB |
| Unit test duration | 0.185s (92 tests) |
| Integration test duration | 3.3s |
| Coverage | 28.9% |
| Health check latency | <1ms |
| Navigate (example.com) | ~2s |
| Snapshot (example.com) | <100ms |
| Screenshot size | 21KB |

---

## Release Readiness

### P0 — Must Pass
| Criterion | Status |
|-----------|--------|
| All Section 1 scenarios pass (headless) | 🟡 36/38 (2 fails) |
| K1 (active tab tracking) | ✅ FIXED |
| K2 (tab close hangs) | ❌ **STILL BROKEN on autorun** |
| Zero crashes | ✅ (but instance dies on stale profiles) |
| `go test ./...` 100% pass | ✅ 92/92 |
| `go test -tags integration` pass | ✅ ~100 pass |

### P1 — Should Pass
| Criterion | Status |
|-----------|--------|
| Multi-agent (MA1-MA5) | ❌ Not tested |
| Stealth bot.sannysoft.com | ❌ Not tested |
| Session persistence | ❌ Not tested |

### P2 — Nice to Have
| Criterion | Status |
|-----------|--------|
| Coverage > 30% | 🟡 28.9% (close!) |
| K3 (SPA title) | 🔧 waitTitle param |
| K4 (Chrome flag) | ✅ FIXED |

---

## Key Takeaways

1. **Core endpoints are solid** — health, navigate, snapshot, text, actions, evaluate, cookies, screenshots all work
2. **K2 (tab close) still broken on autorun** — needs the `target.CloseTarget` fix merged
3. **Stealth injection inconsistent** — `navigator.webdriver` not hidden on navigated tabs
4. **Profile stability issue** — instance crashes on stale profiles, needs `BRIDGE_NO_RESTORE` or clean profile
5. **Coverage at 28.9%** — need ~2% more for P2 target

*Generated by Mario (manual run) at 2026-02-17 07:46 UTC*
