package orchestrator

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pinchtab/pinchtab/internal/web"
)

func (o *Orchestrator) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /instances", o.handleList)

	// Agent-friendly routes: start/stop by profile ID
	mux.HandleFunc("POST /start/{id}", o.handleStartByID)
	mux.HandleFunc("POST /stop/{id}", o.handleStopByID)

	// Dashboard-compatible aliases
	mux.HandleFunc("POST /instances/launch", o.handleLaunchCompat)
	mux.HandleFunc("POST /instances/{id}/stop", o.handleStopByInstanceID)
	mux.HandleFunc("GET /instances/{id}/logs", o.handleLogsByID)
	mux.HandleFunc("GET /instances/tabs", o.handleAllTabs)
	mux.HandleFunc("POST /profiles/{name}/stop", o.handleStopProfileByPath)
	mux.HandleFunc("GET /profiles/{name}/instance", o.handleProfileInstance)
	mux.HandleFunc("GET /instances/{id}/proxy/screencast", o.handleProxyScreencast)
}

func (o *Orchestrator) handleList(w http.ResponseWriter, r *http.Request) {
	web.JSON(w, 200, o.List())
}

func (o *Orchestrator) handleStartByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if o.profiles == nil {
		web.Error(w, 500, fmt.Errorf("profile manager not configured"))
		return
	}
	name, err := o.profiles.FindByID(id)
	if err != nil {
		web.Error(w, 404, err)
		return
	}

	var req struct {
		Port     string `json:"port"`
		Headless bool   `json:"headless"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Port == "" {
		req.Port = "0"
	}

	inst, err := o.Launch(name, req.Port, req.Headless)
	if err != nil {
		web.Error(w, 409, err)
		return
	}
	web.JSON(w, 201, inst)
}

func (o *Orchestrator) handleStopByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if o.profiles == nil {
		web.Error(w, 500, fmt.Errorf("profile manager not configured"))
		return
	}
	name, err := o.profiles.FindByID(id)
	if err != nil {
		web.Error(w, 404, err)
		return
	}
	if err := o.StopProfile(name); err != nil {
		web.Error(w, 404, err)
		return
	}
	web.JSON(w, 200, map[string]string{"status": "stopped", "id": id, "name": name})
}

func (o *Orchestrator) handleLaunchCompat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Port     string `json:"port"`
		Headless bool   `json:"headless"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.Error(w, 400, fmt.Errorf("invalid JSON"))
		return
	}
	if req.Name == "" || req.Port == "" {
		web.Error(w, 400, fmt.Errorf("name and port required"))
		return
	}

	inst, err := o.Launch(req.Name, req.Port, req.Headless)
	if err != nil {
		web.Error(w, 409, err)
		return
	}
	web.JSON(w, 201, inst)
}

func (o *Orchestrator) handleStopByInstanceID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := o.Stop(id); err != nil {
		web.Error(w, 404, err)
		return
	}
	web.JSON(w, 200, map[string]string{"status": "stopped", "id": id})
}

func (o *Orchestrator) handleLogsByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	logs, err := o.Logs(id)
	if err != nil {
		web.Error(w, 404, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(logs))
}

func (o *Orchestrator) handleAllTabs(w http.ResponseWriter, r *http.Request) {
	web.JSON(w, 200, o.AllTabs())
}

func (o *Orchestrator) handleStopProfileByPath(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := o.StopProfile(name); err != nil {
		web.Error(w, 404, err)
		return
	}
	web.JSON(w, 200, map[string]string{"status": "stopped", "name": name})
}

func (o *Orchestrator) handleProfileInstance(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	instances := o.List()
	for _, inst := range instances {
		if inst.Name == name && (inst.Status == "running" || inst.Status == "starting") {
			web.JSON(w, 200, map[string]any{
				"name":    name,
				"running": inst.Status == "running",
				"status":  inst.Status,
				"port":    inst.Port,
				"id":      inst.ID,
			})
			return
		}
	}
	web.JSON(w, 200, map[string]any{
		"name":    name,
		"running": false,
		"status":  "stopped",
		"port":    "",
	})
}

func (o *Orchestrator) handleProxyScreencast(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tabID := r.URL.Query().Get("tabId")

	o.mu.RLock()
	inst, ok := o.instances[id]
	o.mu.RUnlock()
	if !ok || inst.Status != "running" {
		web.Error(w, 404, fmt.Errorf("instance not found or not running"))
		return
	}

	targetURL := fmt.Sprintf("ws://localhost:%s/screencast?tabId=%s", inst.Port, tabID)
	web.JSON(w, 200, map[string]string{"wsUrl": targetURL})
}

// handleLogs removed — use handleLogsByID (path-param) instead
