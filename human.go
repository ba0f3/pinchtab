package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

// humanMouseMove performs natural mouse movement with bezier curves
func humanMouseMove(ctx context.Context, fromX, fromY, toX, toY float64) error {

	distance := math.Sqrt((toX-fromX)*(toX-fromX) + (toY-fromY)*(toY-fromY))

	baseDuration := 100 + (distance/2000)*200
	duration := baseDuration + float64(rand.Intn(100))

	steps := int(duration / 20)
	if steps < 5 {
		steps = 5
	}
	if steps > 30 {
		steps = 30
	}

	cp1X := fromX + (toX-fromX)*0.25 + (rand.Float64()-0.5)*50
	cp1Y := fromY + (toY-fromY)*0.25 + (rand.Float64()-0.5)*50
	cp2X := fromX + (toX-fromX)*0.75 + (rand.Float64()-0.5)*50
	cp2Y := fromY + (toY-fromY)*0.75 + (rand.Float64()-0.5)*50

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)

		oneMinusT := 1 - t
		x := oneMinusT*oneMinusT*oneMinusT*fromX +
			3*oneMinusT*oneMinusT*t*cp1X +
			3*oneMinusT*t*t*cp2X +
			t*t*t*toX

		y := oneMinusT*oneMinusT*oneMinusT*fromY +
			3*oneMinusT*oneMinusT*t*cp1Y +
			3*oneMinusT*t*t*cp2Y +
			t*t*t*toY

		x += (rand.Float64() - 0.5) * 2
		y += (rand.Float64() - 0.5) * 2

		if err := chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return input.DispatchMouseEvent(input.MouseMoved, x, y).Do(ctx)
			}),
		); err != nil {
			return err
		}

		delay := time.Duration(16+rand.Intn(8)) * time.Millisecond
		time.Sleep(delay)
	}

	return nil
}

// humanClick performs a click with natural mouse movement and timing
func humanClick(ctx context.Context, x, y float64) error {

	startOffsetX := (rand.Float64()-0.5)*200 + 50
	startOffsetY := (rand.Float64()-0.5)*200 + 50
	startX := x + startOffsetX
	startY := y + startOffsetY

	distance := math.Sqrt(startOffsetX*startOffsetX + startOffsetY*startOffsetY)
	if distance > 30 {

		if err := chromedp.Run(ctx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return input.DispatchMouseEvent(input.MouseMoved, startX, startY).Do(ctx)
			}),
		); err != nil {
			return err
		}

		if err := humanMouseMove(ctx, startX, startY, x, y); err != nil {
			return err
		}
	}

	time.Sleep(time.Duration(50+rand.Intn(150)) * time.Millisecond)

	if err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return input.DispatchMouseEvent(input.MousePressed, x, y).
				WithButton(input.Left).
				WithClickCount(1).
				Do(ctx)
		}),
	); err != nil {
		return err
	}

	time.Sleep(time.Duration(30+rand.Intn(90)) * time.Millisecond)

	releaseX := x + (rand.Float64()-0.5)*2
	releaseY := y + (rand.Float64()-0.5)*2

	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return input.DispatchMouseEvent(input.MouseReleased, releaseX, releaseY).
				WithButton(input.Left).
				WithClickCount(1).
				Do(ctx)
		}),
	)
}

// humanClickElement clicks an element with human-like behavior
func humanClickElement(ctx context.Context, nodeID cdp.NodeID) error {
	// Get element center coordinates
	var box *dom.BoxModel
	if err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			box, err = dom.GetBoxModel().WithNodeID(nodeID).Do(ctx)
			return err
		}),
	); err != nil {
		return err
	}

	if len(box.Content) < 4 {
		return fmt.Errorf("invalid box model")
	}

	centerX := (box.Content[0] + box.Content[2]) / 2
	centerY := (box.Content[1] + box.Content[5]) / 2

	offsetX := (rand.Float64() - 0.5) * 10
	offsetY := (rand.Float64() - 0.5) * 10

	return humanClick(ctx, centerX+offsetX, centerY+offsetY)
}

// humanType types text with natural timing and occasional corrections
func humanType(text string, fast bool) []chromedp.Action {
	actions := []chromedp.Action{}

	baseDelay := 80
	if fast {
		baseDelay = 40
	}

	chars := []rune(text)
	for i, char := range chars {

		actions = append(actions, chromedp.KeyEvent(string(char)))

		delay := baseDelay + rand.Intn(baseDelay/2)

		if rand.Float64() < 0.05 {
			delay += rand.Intn(500)
		}

		if i > 0 && chars[i-1] == char {
			delay = delay / 2
		}

		actions = append(actions, chromedp.Sleep(time.Duration(delay)*time.Millisecond))

		if rand.Float64() < 0.03 && i < len(chars)-1 {

			wrongChar := rune('a' + rand.Intn(26))
			actions = append(actions,
				chromedp.KeyEvent(string(wrongChar)),
				chromedp.Sleep(time.Duration(50+rand.Intn(100))*time.Millisecond),
				chromedp.KeyEvent("\b"),
				chromedp.Sleep(time.Duration(30+rand.Intn(70))*time.Millisecond),
			)
		}
	}

	return actions
}
