package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const maxBodySize = 1 << 20

const targetTypePage = "page"

const filterInteractive = "interactive"

const (
	actionClick      = "click"
	actionType       = "type"
	actionFill       = "fill"
	actionPress      = "press"
	actionFocus      = "focus"
	actionHover      = "hover"
	actionSelect     = "select"
	actionScroll     = "scroll"
	actionHumanClick = "humanClick"
	actionHumanType  = "humanType"
)

const (
	tabActionNew   = "new"
	tabActionClose = "close"
)

//go:embed stealth.js
var stealthScript string

//go:embed readability.js
var readabilityJS string

//go:embed welcome.html
var welcomeHTML string

var (
	port             = envOr("BRIDGE_PORT", "9867")
	cdpURL           = os.Getenv("CDP_URL")
	token            = os.Getenv("BRIDGE_TOKEN")
	stateDir         = envOr("BRIDGE_STATE_DIR", filepath.Join(homeDir(), ".pinchtab"))
	headless         = os.Getenv("BRIDGE_HEADLESS") == "true"
	noRestore        = os.Getenv("BRIDGE_NO_RESTORE") == "true"
	profileDir       = envOr("BRIDGE_PROFILE", filepath.Join(homeDir(), ".pinchtab", "chrome-profile"))
	chromeVersion    = envOr("BRIDGE_CHROME_VERSION", "144.0.7559.133")
	timezone         = os.Getenv("BRIDGE_TIMEZONE")
	blockImages      = os.Getenv("BRIDGE_BLOCK_IMAGES") == "true"
	blockMedia       = os.Getenv("BRIDGE_BLOCK_MEDIA") == "true"
	chromeBinary     = os.Getenv("CHROME_BINARY")
	chromeExtraFlags = os.Getenv("CHROME_FLAGS")
	noAnimations     = os.Getenv("BRIDGE_NO_ANIMATIONS") == "true"
	stealthLevel     = envOr("BRIDGE_STEALTH", "light")
	actionTimeout    = 15 * time.Second
	navigateTimeout  = 30 * time.Second
	shutdownTimeout  = 10 * time.Second
	waitNavDelay     = 1 * time.Second
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

type Config struct {
	Port        string `json:"port"`
	CdpURL      string `json:"cdpUrl,omitempty"`
	Token       string `json:"token,omitempty"`
	StateDir    string `json:"stateDir"`
	ProfileDir  string `json:"profileDir"`
	Headless    bool   `json:"headless"`
	NoRestore   bool   `json:"noRestore"`
	TimeoutSec  int    `json:"timeoutSec,omitempty"`
	NavigateSec int    `json:"navigateSec,omitempty"`
}

func loadConfig() {

	configPath := envOr("BRIDGE_CONFIG", filepath.Join(homeDir(), ".pinchtab", "config.json"))

	if data, err := os.ReadFile(configPath); err == nil {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			slog.Warn("invalid JSON in config file, ignoring", "path", configPath, "err", err)
		} else {

			if cfg.Port != "" && os.Getenv("BRIDGE_PORT") == "" {
				port = cfg.Port
			}
			if cfg.CdpURL != "" && os.Getenv("CDP_URL") == "" {
				cdpURL = cfg.CdpURL
			}
			if cfg.Token != "" && os.Getenv("BRIDGE_TOKEN") == "" {
				token = cfg.Token
			}
			if cfg.StateDir != "" && os.Getenv("BRIDGE_STATE_DIR") == "" {
				stateDir = cfg.StateDir
			}
			if cfg.ProfileDir != "" && os.Getenv("BRIDGE_PROFILE") == "" {
				profileDir = cfg.ProfileDir
			}
			if cfg.Headless && os.Getenv("BRIDGE_HEADLESS") == "" {
				headless = true
			}
			if cfg.NoRestore && os.Getenv("BRIDGE_NO_RESTORE") == "" {
				noRestore = true
			}
			if cfg.TimeoutSec > 0 && os.Getenv("BRIDGE_TIMEOUT") == "" {
				actionTimeout = time.Duration(cfg.TimeoutSec) * time.Second
			}
			if cfg.NavigateSec > 0 && os.Getenv("BRIDGE_NAV_TIMEOUT") == "" {
				navigateTimeout = time.Duration(cfg.NavigateSec) * time.Second
			}
		}
	}
}

func defaultConfig() Config {
	return Config{
		Port:        "9867",
		StateDir:    filepath.Join(homeDir(), ".pinchtab"),
		ProfileDir:  filepath.Join(homeDir(), ".pinchtab", "chrome-profile"),
		Headless:    true,
		NoRestore:   false,
		TimeoutSec:  15,
		NavigateSec: 30,
	}
}

func handleConfigCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: pinchtab config <command>")
		fmt.Println("Commands:")
		fmt.Println("  init    - Create default config file")
		fmt.Println("  show    - Show current configuration")
		return
	}

	switch os.Args[2] {
	case "init":
		configPath := filepath.Join(homeDir(), ".pinchtab", "config.json")

		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config file already exists at %s\n", configPath)
			fmt.Print("Overwrite? (y/N): ")
			var response string
			_, _ = fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				return
			}
		}

		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			fmt.Printf("Error creating directory: %v\n", err)
			os.Exit(1)
		}

		cfg := defaultConfig()
		data, _ := json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			fmt.Printf("Error writing config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Config file created at %s\n", configPath)
		fmt.Println("\nExample with auth token:")
		fmt.Println(`{
  "port": "9867",
  "token": "your-secret-token",
  "headless": true,
  "stateDir": "` + cfg.StateDir + `",
  "profileDir": "` + cfg.ProfileDir + `"
}`)

	case "show":
		fmt.Println("Current configuration:")
		fmt.Printf("  Port:       %s\n", port)
		fmt.Printf("  CDP URL:    %s\n", cdpURL)
		fmt.Printf("  Token:      %s\n", maskToken(token))
		fmt.Printf("  State Dir:  %s\n", stateDir)
		fmt.Printf("  Profile:    %s\n", profileDir)
		fmt.Printf("  Headless:   %v\n", headless)
		fmt.Printf("  No Restore: %v\n", noRestore)
		fmt.Printf("  Timeouts:   action=%v navigate=%v\n", actionTimeout, navigateTimeout)

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[2])
		os.Exit(1)
	}
}

func maskToken(t string) string {
	if t == "" {
		return "(none)"
	}
	if len(t) <= 8 {
		return "***"
	}
	return t[:4] + "..." + t[len(t)-4:]
}
