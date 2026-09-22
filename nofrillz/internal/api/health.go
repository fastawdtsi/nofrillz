package api

import (
	"net/http"

	"nofrillz/internal/app"
)

type HealthHandler struct {
	app *app.App
}

func NewHealthHandler(app *app.App) *HealthHandler {
	return &HealthHandler{
		app: app,
	}
}

func (h *HealthHandler) AddRoutes(sm *http.ServeMux, middleware func(http.HandlerFunc) http.HandlerFunc) {
	sm.HandleFunc("GET /health", middleware(h.Health))
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r)
	if !AllowReadByIP(w, h.app, ip) {
		return
	}

	w.WriteHeader(http.StatusOK)
}
