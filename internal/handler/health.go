package handler

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// HealthHandler manages liveness, readiness, and info endpoints.
type HealthHandler struct {
	appName   string
	version   string
	appEnv    string
	startTime time.Time
	ready     atomic.Bool
}

type infoResponse struct {
	App     string `json:"app"`
	Version string `json:"version"`
	Env     string `json:"env"`
	Uptime  string `json:"uptime"`
	Status  string `json:"status"`
}

type healthResponse struct {
	Status string `json:"status"`
}

// NewHealthHandler creates a new HealthHandler. Call SetReady(true) once the app is fully initialised.
func NewHealthHandler(appName, version, appEnv string) *HealthHandler {
	return &HealthHandler{
		appName:   appName,
		version:   version,
		appEnv:    appEnv,
		startTime: time.Now(),
	}
}

// SetReady marks the app as ready (or not ready) to receive traffic.
func (h *HealthHandler) SetReady(ready bool) {
	h.ready.Store(ready)
}

// Liveness handles GET /healthz — returns 200 as long as the process is alive.
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// Readiness handles GET /readyz — returns 503 until SetReady(true) is called.
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	if !h.ready.Load() {
		writeJSON(w, http.StatusServiceUnavailable, healthResponse{Status: "not ready"})
		return
	}
	writeJSON(w, http.StatusOK, healthResponse{Status: "ready"})
}

// Info handles GET / — returns app metadata and uptime.
func (h *HealthHandler) Info(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, infoResponse{
		App:     h.appName,
		Version: h.version,
		Env:     h.appEnv,
		Uptime:  time.Since(h.startTime).Round(time.Second).String(),
		Status:  "running",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
