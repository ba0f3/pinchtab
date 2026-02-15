package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

// navigatePage uses raw CDP Page.navigate + polls for a non-blank URL.
// Unlike chromedp.Navigate which waits for the full load event (hangs on SPAs),
// this fires navigation and waits up to 5s for the page to start loading.
func navigatePage(ctx context.Context, url string) error {
	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			p := map[string]any{"url": url}
			var navResult json.RawMessage
			if err := chromedp.FromContext(ctx).Target.Execute(ctx, "Page.navigate", p, &navResult); err != nil {
				return fmt.Errorf("page.navigate: %w", err)
			}

			var resp struct {
				ErrorText string `json:"errorText"`
			}
			if err := json.Unmarshal(navResult, &resp); err == nil && resp.ErrorText != "" {
				return fmt.Errorf("navigate: %s", resp.ErrorText)
			}
			return nil
		}),
		// Brief sleep to let the page start rendering — not a full load wait.
		// Agents should use /snapshot to confirm readiness.
		chromedp.Sleep(500*time.Millisecond),
	)
}

// withElement resolves a backendNodeID to a JS remote object, scrolls it into
// view, and calls the given JS function on it. This is the generic helper for
// all element-targeted actions (click, hover, select, etc.).
func withElement(ctx context.Context, backendNodeID int64, jsFunc string) error {
	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			objectID, err := resolveNodeToObject(ctx, backendNodeID)
			if err != nil {
				return err
			}
			callP := map[string]any{
				"objectId":            objectID,
				"functionDeclaration": jsFunc,
				"arguments":           []any{},
			}
			if err := chromedp.FromContext(ctx).Target.Execute(ctx, "Runtime.callFunctionOn", callP, nil); err != nil {
				return fmt.Errorf("callFunctionOn: %w", err)
			}
			return nil
		}),
	)
}

// withElementArg is like withElement but passes a single string argument to the JS function.
func withElementArg(ctx context.Context, backendNodeID int64, jsFunc string, arg string) error {
	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			objectID, err := resolveNodeToObject(ctx, backendNodeID)
			if err != nil {
				return err
			}
			callP := map[string]any{
				"objectId":            objectID,
				"functionDeclaration": jsFunc,
				"arguments":           []any{map[string]any{"value": arg}},
			}
			if err := chromedp.FromContext(ctx).Target.Execute(ctx, "Runtime.callFunctionOn", callP, nil); err != nil {
				return fmt.Errorf("callFunctionOn: %w", err)
			}
			return nil
		}),
	)
}

// clickByNodeID scrolls an element into view and clicks it.
func clickByNodeID(ctx context.Context, backendNodeID int64) error {
	return withElement(ctx, backendNodeID, "function() { this.scrollIntoViewIfNeeded(); this.click(); }")
}

// typeByNodeID scrolls an element into view, focuses it, and sends keyboard events.
func typeByNodeID(ctx context.Context, backendNodeID int64, text string) error {
	if err := withElement(ctx, backendNodeID, "function() { this.scrollIntoViewIfNeeded(); }"); err != nil {
		return err
	}
	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			p := map[string]any{"backendNodeId": backendNodeID}
			if err := chromedp.FromContext(ctx).Target.Execute(ctx, "DOM.focus", p, nil); err != nil {
				return fmt.Errorf("DOM.focus: %w", err)
			}
			return nil
		}),
		chromedp.KeyEvent(text),
	)
}

// hoverByNodeID scrolls an element into view and dispatches a mouseover event.
func hoverByNodeID(ctx context.Context, backendNodeID int64) error {
	return withElement(ctx, backendNodeID, `function() {
		this.scrollIntoViewIfNeeded();
		this.dispatchEvent(new MouseEvent('mouseover', {bubbles: true}));
		this.dispatchEvent(new MouseEvent('mouseenter', {bubbles: false}));
	}`)
}

// selectByNodeID sets the value of a <select> element and fires change event.
func selectByNodeID(ctx context.Context, backendNodeID int64, value string) error {
	return withElementArg(ctx, backendNodeID, `function(val) {
		this.scrollIntoViewIfNeeded();
		const opt = Array.from(this.options).find(o => o.value === val || o.textContent.trim() === val);
		if (opt) { this.value = opt.value; }
		else { this.value = val; }
		this.dispatchEvent(new Event('change', {bubbles: true}));
	}`, value)
}

// scrollByNodeID scrolls an element into view.
func scrollByNodeID(ctx context.Context, backendNodeID int64) error {
	return withElement(ctx, backendNodeID, "function() { this.scrollIntoViewIfNeeded(); }")
}

// resolveNodeToObject converts a backendNodeID to a JS remote object ID.
func resolveNodeToObject(ctx context.Context, backendNodeID int64) (string, error) {
	p := map[string]any{"backendNodeId": backendNodeID}
	var result json.RawMessage
	if err := chromedp.FromContext(ctx).Target.Execute(ctx, "DOM.resolveNode", p, &result); err != nil {
		return "", fmt.Errorf("DOM.resolveNode: %w", err)
	}
	var resp struct {
		Object struct {
			ObjectID string `json:"objectId"`
		} `json:"object"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", fmt.Errorf("unmarshal resolveNode: %w", err)
	}
	if resp.Object.ObjectID == "" {
		return "", fmt.Errorf("no objectId for node %d", backendNodeID)
	}
	return resp.Object.ObjectID, nil
}
