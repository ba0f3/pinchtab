package main

import (
	_ "embed"
	"os"
	"path/filepath"
	"time"
)

// Body size limit for POST handlers.
const maxBodySize = 1 << 20 // 1MB

// Target type for Chrome DevTools Protocol.
const targetTypePage = "page"

// Snapshot filter values.
const filterInteractive = "interactive"

// Action kinds for /action endpoint.
const (
	actionClick  = "click"
	actionType   = "type"
	actionFill   = "fill"
	actionPress  = "press"
	actionFocus  = "focus"
	actionHover  = "hover"
	actionSelect = "select"
	actionScroll = "scroll"
)

// Tab actions for /tab endpoint.
const (
	tabActionNew   = "new"
	tabActionClose = "close"
)

//go:embed stealth.js
var stealthScript string

var (
	port            = envOr("BRIDGE_PORT", "9867")
	cdpURL          = os.Getenv("CDP_URL") // empty = launch Chrome ourselves
	token           = os.Getenv("BRIDGE_TOKEN")
	stateDir        = envOr("BRIDGE_STATE_DIR", filepath.Join(homeDir(), ".pinchtab"))
	headless        = os.Getenv("BRIDGE_HEADLESS") == "true"
	noRestore       = os.Getenv("BRIDGE_NO_RESTORE") == "true"
	profileDir      = envOr("BRIDGE_PROFILE", filepath.Join(homeDir(), ".pinchtab", "chrome-profile"))
	actionTimeout   = 15 * time.Second
	navigateTimeout = 30 * time.Second
	shutdownTimeout = 10 * time.Second
	waitNavDelay    = 1 * time.Second
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}
