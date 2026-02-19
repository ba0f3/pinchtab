package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

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
