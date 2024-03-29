package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"shidai/internal/config"
	"shidai/internal/registry"
)

type CommandRequest struct {
	Command string                 `json:"command"`
	Args    map[string]interface{} `json:"args"`
	Config  config.Config          `json:"config"`
}

func ExecuteCommandHandler(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	slog.Info("Handle", "command", req.Command, "arguments", req.Args)

	executor, exists := registry.GetCommandExecutor(req.Command)
	if !exists {
		slog.Error("Not supported", "command", req.Command)

		http.Error(w, "Command not supported", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ctx = context.WithValue(ctx, config.ConfigContextKey, req.Config)
	slog.Debug("Context", "ctx", ctx, "config", req.Config)

	if err := executor.Execute(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("SUCCESS!", "command", req.Command)

	json.NewEncoder(w).Encode(map[string]string{"status": "success"}) //nolint:errcheck
}
