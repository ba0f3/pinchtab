package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

var bridge Bridge

func main() {
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		slog.Error("cannot create state dir", "err", err)
		os.Exit(1)
	}

	var allocCancel context.CancelFunc

	if cdpURL != "" {
		slog.Info("connecting to Chrome", "url", cdpURL)
		bridge.allocCtx, allocCancel = chromedp.NewRemoteAllocator(context.Background(), cdpURL)
	} else {
		if err := os.MkdirAll(profileDir, 0755); err != nil {
			slog.Error("cannot create profile dir", "err", err)
			os.Exit(1)
		}
		slog.Info("launching Chrome", "profile", profileDir, "headless", headless)

		opts := []chromedp.ExecAllocatorOption{
			// Profile & basics
			chromedp.UserDataDir(profileDir),
			chromedp.NoFirstRun,
			chromedp.NoDefaultBrowserCheck,

			// Stealth: hide automation indicators
			// Note: --disable-blink-features=AutomationControlled removed — deprecated
			// in Chrome 144+, triggers warning banner. stealth.js handles navigator.webdriver.
			chromedp.Flag("exclude-switches", "enable-automation"),
			chromedp.Flag("disable-infobars", true),

			// Performance & networking
			chromedp.Flag("disable-background-networking", false),
			chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
			chromedp.Flag("disable-popup-blocking", true),
			chromedp.Flag("disable-default-apps", false),
			chromedp.Flag("no-first-run", true),

			// UI: suppress crash bar
			chromedp.Flag("disable-session-crashed-bubble", true),
			chromedp.Flag("hide-crash-restore-bubble", true),

			// Identity
			chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"),
			chromedp.WindowSize(1440, 900),
		}

		if headless {
			opts = append(opts, chromedp.Headless)
		} else {
			opts = append(opts, chromedp.Flag("headless", false))
		}

		markCleanExit()
		bridge.allocCtx, allocCancel = chromedp.NewExecAllocator(context.Background(), opts...)
	}
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(bridge.allocCtx)
	defer browserCancel()

	// Inject stealth script (embedded from stealth.js)
	if err := chromedp.Run(browserCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(stealthScript).Do(ctx)
			return err
		}),
	); err != nil {
		slog.Error("cannot start Chrome", "err", err)
		os.Exit(1)
	}

	bridge.browserCtx = browserCtx
	bridge.tabs = make(map[string]*TabEntry)
	bridge.snapshots = make(map[string]*refCache)

	// Register the initial tab
	initTargetID := chromedp.FromContext(browserCtx).Target.TargetID
	bridge.tabs[string(initTargetID)] = &TabEntry{ctx: browserCtx}
	slog.Info("initial tab", "id", string(initTargetID))

	if !noRestore {
		// Restore in background so it doesn't block the HTTP server
		go bridge.RestoreState()
	}

	// Background tab cleanup
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	go bridge.CleanStaleTabs(cleanupCtx, 30*actionTimeout)

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", bridge.handleHealth)
	mux.HandleFunc("GET /tabs", bridge.handleTabs)
	mux.HandleFunc("GET /snapshot", bridge.handleSnapshot)
	mux.HandleFunc("GET /screenshot", bridge.handleScreenshot)
	mux.HandleFunc("GET /text", bridge.handleText)
	mux.HandleFunc("POST /navigate", bridge.handleNavigate)
	mux.HandleFunc("POST /action", bridge.handleAction)
	mux.HandleFunc("POST /evaluate", bridge.handleEvaluate)
	mux.HandleFunc("POST /tab", bridge.handleTab)

	srv := &http.Server{Addr: ":" + port, Handler: loggingMiddleware(corsMiddleware(authMiddleware(mux)))}

	// Graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		slog.Info("shutting down, saving state...")
		cleanupCancel()
		bridge.SaveState()
		markCleanExit()
		shutdownCtx, shutdownDone := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownDone()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown", "err", err)
		}
	}()

	slog.Info("🦀 PINCH! PINCH!", "port", port, "cdp", cdpURL)
	if token != "" {
		slog.Info("auth enabled")
	} else {
		slog.Info("auth disabled (set BRIDGE_TOKEN to enable)")
	}

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
