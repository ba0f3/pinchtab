package main

import "context"

// Browser abstracts Chrome DevTools operations for testing.
type Browser interface {
	Navigate(ctx context.Context, url string) error
	Screenshot(ctx context.Context, quality int) ([]byte, error)
	Evaluate(ctx context.Context, expr string) (any, error)
	GetText(ctx context.Context) (string, error)
	GetLocation(ctx context.Context) (url string, title string, err error)
	GetAccessibilityTree(ctx context.Context) ([]rawAXNode, error)
	ClickNode(ctx context.Context, nodeID int64) error
	TypeNode(ctx context.Context, nodeID int64, text string) error
	FocusNode(ctx context.Context, nodeID int64) error
	ClickSelector(ctx context.Context, sel string) error
	TypeSelector(ctx context.Context, sel string, text string) error
	FillSelector(ctx context.Context, sel string, value string) error
	PressKey(ctx context.Context, key string) error
}

// TabManager abstracts tab lifecycle operations for testing.
type TabManager interface {
	Get(tabID string) (ctx context.Context, resolvedID string, err error)
	Create(url string) (tabID string, err error)
	Close(tabID string) error
	List() ([]TabInfo, error)
}

// TabInfo is a simplified tab descriptor returned by TabManager.List.
type TabInfo struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
	Type  string `json:"type"`
}
