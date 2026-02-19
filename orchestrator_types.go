package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Orchestrator struct {
	instances map[string]*Instance
	baseDir   string
	binary    string
	mu        sync.RWMutex
	client    *http.Client
}

type Instance struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Profile   string    `json:"profile"`
	Port      string    `json:"port"`
	Headless  bool      `json:"headless"`
	Status    string    `json:"status"`
	PID       int       `json:"pid,omitempty"`
	StartedAt time.Time `json:"startedAt"`
	Error     string    `json:"error,omitempty"`
	TabCount  int       `json:"tabCount"`
	URL       string    `json:"url"`

	cmd    *exec.Cmd
	cancel context.CancelFunc
	logBuf *ringBuffer
}

type ringBuffer struct {
	mu   sync.Mutex
	data []byte
	max  int
}

func newRingBuffer(max int) *ringBuffer {
	return &ringBuffer{max: max, data: make([]byte, 0, max)}
}

func (rb *ringBuffer) Write(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.data = append(rb.data, p...)
	if len(rb.data) > rb.max {
		rb.data = rb.data[len(rb.data)-rb.max:]
	}
	return len(p), nil
}

func (rb *ringBuffer) String() string {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return string(rb.data)
}

func NewOrchestrator(baseDir string) *Orchestrator {
	binDir := filepath.Join(filepath.Dir(baseDir), "bin")
	stableBin := filepath.Join(binDir, "pinchtab")

	needsBuild := true
	if fi, err := os.Stat(stableBin); err == nil {
		if time.Since(fi.ModTime()) < time.Hour {
			needsBuild = false
		}
	}

	if needsBuild {
		os.MkdirAll(binDir, 0755)

		exe, _ := os.Executable()
		if exe != "" {
			if data, err := os.ReadFile(exe); err == nil {
				if err := os.WriteFile(stableBin, data, 0755); err == nil {
					slog.Info("installed pinchtab binary", "path", stableBin)
				}
			}
		}
	}

	binary := stableBin
	if _, err := os.Stat(binary); err != nil {
		binary, _ = os.Executable()
		if binary == "" {
			binary = os.Args[0]
		}
	}

	return &Orchestrator{
		instances: make(map[string]*Instance),
		baseDir:   baseDir,
		binary:    binary,
		client:    &http.Client{Timeout: 3 * time.Second},
	}
}
