package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

// TabEntry holds a chromedp context for an open tab.
type TabEntry struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// refCache stores the ref→backendNodeID mapping and the last snapshot nodes per tab.
// Refs are assigned during /snapshot and looked up during /action, avoiding
// a second a11y tree fetch that could drift.
type refCache struct {
	refs  map[string]int64 // "e0" → backendNodeID
	nodes []A11yNode       // last snapshot for diff
}

// Bridge is the central state holder for the Chrome connection, tab contexts,
// and per-tab snapshot caches.
type Bridge struct {
	allocCtx   context.Context
	browserCtx context.Context
	tabs       map[string]*TabEntry
	snapshots  map[string]*refCache
	mu         sync.RWMutex
}

// TabContext returns the chromedp context for a tab and the resolved tabID.
// If tabID is empty, uses the first page target.
// Uses RLock for cache hits, upgrades to Lock only when creating a new entry.
func (b *Bridge) TabContext(tabID string) (context.Context, string, error) {
	if tabID == "" {
		targets, err := b.ListTargets()
		if err != nil {
			return nil, "", fmt.Errorf("list targets: %w", err)
		}
		if len(targets) == 0 {
			return nil, "", fmt.Errorf("no tabs open")
		}
		tabID = string(targets[0].TargetID)
	}

	// Fast path: read lock
	b.mu.RLock()
	if entry, ok := b.tabs[tabID]; ok {
		b.mu.RUnlock()
		return entry.ctx, tabID, nil
	}
	b.mu.RUnlock()

	// Slow path: write lock, double-check
	b.mu.Lock()
	defer b.mu.Unlock()

	if entry, ok := b.tabs[tabID]; ok {
		return entry.ctx, tabID, nil
	}

	ctx, cancel := chromedp.NewContext(b.browserCtx,
		chromedp.WithTargetID(target.ID(tabID)),
	)
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		return nil, "", fmt.Errorf("tab %s not found: %w", tabID, err)
	}

	b.tabs[tabID] = &TabEntry{ctx: ctx, cancel: cancel}
	return ctx, tabID, nil
}

// CleanStaleTabs periodically removes tab entries whose Chrome targets
// no longer exist. Exits when ctx is cancelled.
func (b *Bridge) CleanStaleTabs(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		targets, err := b.ListTargets()
		if err != nil {
			continue
		}

		alive := make(map[string]bool, len(targets))
		for _, t := range targets {
			alive[string(t.TargetID)] = true
		}

		b.mu.Lock()
		for id, entry := range b.tabs {
			if !alive[id] {
				if entry.cancel != nil {
					entry.cancel()
				}
				delete(b.tabs, id)
				delete(b.snapshots, id)
				slog.Info("cleaned stale tab", "id", id)
			}
		}
		b.mu.Unlock()
	}
}

// ListTargets returns all open page targets from Chrome.
func (b *Bridge) ListTargets() ([]*target.Info, error) {
	var targets []*target.Info
	if err := chromedp.Run(b.browserCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			targets, err = target.GetTargets().Do(ctx)
			return err
		}),
	); err != nil {
		return nil, fmt.Errorf("get targets: %w", err)
	}

	pages := make([]*target.Info, 0)
	for _, t := range targets {
		if t.Type == targetTypePage {
			pages = append(pages, t)
		}
	}
	return pages, nil
}
