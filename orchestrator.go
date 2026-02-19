package // Launch starts a new Pinchtab instance on a named profile.
// Stop kills a running instance.
// List returns all instances with live status.
// Logs returns the log buffer for an instance.
// AllTabs aggregates tabs from all running instances.
// ProxyScreencastURL returns the WebSocket URL for a child instance's screencast.
main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
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

func (o *Orchestrator) Launch(name, port string, headless bool) (*Instance, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, inst := range o.instances {
		if inst.Port == port && inst.Status == "running" {
			return nil, fmt.Errorf("port %s already in use by instance %q", port, inst.Name)
		}
	}

	id := fmt.Sprintf("%s-%s", name, port)
	if inst, ok := o.instances[id]; ok && inst.Status == "running" {
		return nil, fmt.Errorf("instance %q already running", id)
	}

	profilePath := filepath.Join(o.baseDir, name)
	os.MkdirAll(filepath.Join(profilePath, "Default"), 0755)

	ctx, cancel := context.WithCancel(context.Background())

	headlessStr := "true"
	if !headless {
		headlessStr = "false"
	}

	cmd := exec.CommandContext(ctx, o.binary)
	cmd.Env = append(os.Environ(),
		"BRIDGE_PORT="+port,
		"BRIDGE_PROFILE="+profilePath,
		"BRIDGE_HEADLESS="+headlessStr,
		"BRIDGE_NO_RESTORE=true",

		"BRIDGE_NO_DASHBOARD=true",
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	logBuf := newRingBuffer(64 * 1024)
	cmd.Stdout = logBuf
	cmd.Stderr = logBuf

	inst := &Instance{
		ID:        id,
		Name:      name,
		Profile:   profilePath,
		Port:      port,
		Headless:  headless,
		Status:    "starting",
		StartedAt: time.Now(),
		URL:       fmt.Sprintf("http://localhost:%s", port),
		cmd:       cmd,
		cancel:    cancel,
		logBuf:    logBuf,
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start: %w", err)
	}

	inst.PID = cmd.Process.Pid
	o.instances[id] = inst

	go o.monitor(inst)

	return inst, nil
}

func (o *Orchestrator) monitor(inst *Instance) {

	healthy := false
	for i := 0; i < 30; i++ {
		time.Sleep(500 * time.Millisecond)
		resp, err := o.client.Get(inst.URL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				healthy = true
				break
			}
		}
	}

	o.mu.Lock()
	if healthy {
		inst.Status = "running"
		slog.Info("instance ready", "id", inst.ID, "port", inst.Port)
	} else {
		inst.Status = "error"
		inst.Error = "health check timeout after 15s"
		slog.Error("instance failed to start", "id", inst.ID)
	}
	o.mu.Unlock()

	err := inst.cmd.Wait()
	o.mu.Lock()
	if inst.Status != "stopped" {
		inst.Status = "stopped"
		if err != nil {
			inst.Error = err.Error()
		}
	}
	o.mu.Unlock()
	slog.Info("instance exited", "id", inst.ID)
}

func (o *Orchestrator) Stop(id string) error {
	o.mu.Lock()
	inst, ok := o.instances[id]
	if !ok {
		o.mu.Unlock()
		return fmt.Errorf("instance %q not found", id)
	}
	inst.Status = "stopped"
	o.mu.Unlock()

	req, _ := http.NewRequest("POST", inst.URL+"/shutdown", nil)
	resp, err := o.client.Do(req)
	if err == nil {
		resp.Body.Close()

		time.Sleep(2 * time.Second)
	}

	if inst.cmd.ProcessState == nil || !inst.cmd.ProcessState.Exited() {
		_ = syscall.Kill(-inst.cmd.Process.Pid, syscall.SIGKILL)
		inst.cancel()
	}

	return nil
}

func (o *Orchestrator) List() []Instance {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make([]Instance, 0, len(o.instances))
	for _, inst := range o.instances {
		copy := *inst
		copy.cmd = nil
		copy.cancel = nil
		copy.logBuf = nil

		if inst.Status == "running" {
			if tabs, err := o.fetchTabs(inst.URL); err == nil {
				copy.TabCount = len(tabs)
			}
		}

		result = append(result, copy)
	}
	return result
}

func (o *Orchestrator) Logs(id string) (string, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	inst, ok := o.instances[id]
	if !ok {
		return "", fmt.Errorf("instance %q not found", id)
	}
	return inst.logBuf.String(), nil
}

func (o *Orchestrator) AllTabs() []instanceTab {
	o.mu.RLock()
	instances := make([]*Instance, 0)
	for _, inst := range o.instances {
		if inst.Status == "running" {
			instances = append(instances, inst)
		}
	}
	o.mu.RUnlock()

	var all []instanceTab
	for _, inst := range instances {
		tabs, err := o.fetchTabs(inst.URL)
		if err != nil {
			continue
		}
		for _, t := range tabs {
			all = append(all, instanceTab{
				InstanceID:   inst.ID,
				InstanceName: inst.Name,
				InstancePort: inst.Port,
				TabID:        t.ID,
				URL:          t.URL,
			})
		}
	}
	return all
}

type instanceTab struct {
	InstanceID   string `json:"instanceId"`
	InstanceName string `json:"instanceName"`
	InstancePort string `json:"instancePort"`
	TabID        string `json:"tabId"`
	URL          string `json:"url"`
}

type remoteTab struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func (o *Orchestrator) fetchTabs(baseURL string) ([]remoteTab, error) {
	resp, err := o.client.Get(baseURL + "/screencast/tabs")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tabs []remoteTab
	json.NewDecoder(resp.Body).Decode(&tabs)
	return tabs, nil
}

func (o *Orchestrator) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /instances", o.handleList)
	mux.HandleFunc("POST /instances/launch", o.handleLaunch)
	mux.HandleFunc("POST /instances/{id}/stop", o.handleStop)
	mux.HandleFunc("GET /instances/{id}/logs", o.handleLogs)
	mux.HandleFunc("GET /instances/tabs", o.handleAllTabs)
	mux.HandleFunc("GET /instances/{id}/proxy/screencast", o.handleProxyScreencast)
}

func (o *Orchestrator) handleList(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, 200, o.List())
}

func (o *Orchestrator) handleLaunch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Port     string `json:"port"`
		Headless *bool  `json:"headless"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, 400, fmt.Errorf("invalid JSON"))
		return
	}
	if req.Name == "" || req.Port == "" {
		jsonErr(w, 400, fmt.Errorf("name and port required"))
		return
	}
	headless := true
	if req.Headless != nil {
		headless = *req.Headless
	}

	inst, err := o.Launch(req.Name, req.Port, headless)
	if err != nil {
		jsonErr(w, 409, err)
		return
	}
	jsonResp(w, 201, inst)
}

func (o *Orchestrator) handleStop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := o.Stop(id); err != nil {
		jsonErr(w, 404, err)
		return
	}
	jsonResp(w, 200, map[string]string{"status": "stopped", "id": id})
}

func (o *Orchestrator) handleLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	logs, err := o.Logs(id)
	if err != nil {
		jsonErr(w, 404, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(logs))
}

func (o *Orchestrator) handleAllTabs(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, 200, o.AllTabs())
}

func (o *Orchestrator) handleProxyScreencast(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tabID := r.URL.Query().Get("tabId")

	o.mu.RLock()
	inst, ok := o.instances[id]
	o.mu.RUnlock()
	if !ok || inst.Status != "running" {
		http.Error(w, "instance not found or not running", 404)
		return
	}

	targetURL := fmt.Sprintf("ws://localhost:%s/screencast?tabId=%s", inst.Port, tabID)
	jsonResp(w, 200, map[string]string{"wsUrl": targetURL})
}

func (o *Orchestrator) ScreencastURL(instanceID, tabID string) string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	inst, ok := o.instances[instanceID]
	if !ok {
		return ""
	}
	return fmt.Sprintf("ws://localhost:%s/screencast?tabId=%s", inst.Port, tabID)
}

func (o *Orchestrator) Shutdown() {
	o.mu.RLock()
	ids := make([]string, 0, len(o.instances))
	for id, inst := range o.instances {
		if inst.Status == "running" {
			ids = append(ids, id)
		}
	}
	o.mu.RUnlock()

	for _, id := range ids {
		slog.Info("stopping instance", "id", id)
		o.Stop(id)
	}
}

func readAll(r io.Reader, limit int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, limit))
}
