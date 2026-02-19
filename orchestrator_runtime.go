package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

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
		copyInst := *inst
		copyInst.cmd = nil
		copyInst.cancel = nil
		copyInst.logBuf = nil

		if inst.Status == "running" {
			if tabs, err := o.fetchTabs(inst.URL); err == nil {
				copyInst.TabCount = len(tabs)
			}
		}

		result = append(result, copyInst)
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
		for _, tab := range tabs {
			all = append(all, instanceTab{
				InstanceID:   inst.ID,
				InstanceName: inst.Name,
				InstancePort: inst.Port,
				TabID:        tab.ID,
				URL:          tab.URL,
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
	if err := json.NewDecoder(resp.Body).Decode(&tabs); err != nil {
		return nil, err
	}
	return tabs, nil
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
