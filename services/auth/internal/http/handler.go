package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"pz1.2/services/auth/internal/service"
	"pz1.2/shared/logger"
)

type Handler struct {
	authService *service.AuthService
	log         *zap.Logger
}

func NewHandler(authService *service.AuthService, log *zap.Logger) *Handler {
	return &Handler{authService: authService, log: log}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/login", h.handleLogin)
	mux.HandleFunc("GET /v1/auth/verify", h.handleVerify)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context()).With(zap.String("component", "handler"))

	var req service.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.Warn("invalid request body", zap.Error(err))
		h.respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	resp, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		l.Warn("login failed", zap.String("username", req.Username))
		h.respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	l.Info("login successful", zap.String("username", req.Username))
	h.respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleVerify(w http.ResponseWriter, r *http.Request) {
	l := logger.FromContext(r.Context()).With(zap.String("component", "handler"))

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		l.Warn("missing authorization header")
		h.respondJSON(w, http.StatusUnauthorized, service.VerifyResponse{
			Valid: false,
			Error: "missing authorization header",
		})
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		l.Warn("invalid authorization format")
		h.respondJSON(w, http.StatusUnauthorized, service.VerifyResponse{
			Valid: false,
			Error: "invalid authorization format",
		})
		return
	}

	token := parts[1]
	resp, err := h.authService.Verify(token)
	if err != nil {
		l.Warn("token verification failed", zap.Bool("has_auth", true))
		h.respondJSON(w, http.StatusUnauthorized, resp)
		return
	}

	l.Info("token verified", zap.String("subject", resp.Subject))
	h.respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
