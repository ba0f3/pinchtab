package main

import (
	"context"

	"github.com/chromedp/cdproto/target"
)

type BridgeAPI interface {
	TabContext(tabID string) (ctx context.Context, resolvedID string, err error)
	ListTargets() ([]*target.Info, error)
	CreateTab(url string) (tabID string, ctx context.Context, cancel context.CancelFunc, err error)
	CloseTab(tabID string) error

	GetRefCache(tabID string) *refCache
	SetRefCache(tabID string, cache *refCache)
	DeleteRefCache(tabID string)
}

type TabInfo struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
	Type  string `json:"type"`
}
