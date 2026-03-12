package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/rborale/mindvault/internal/n8n"
)

type Handler struct {
	n8nClient *n8n.Client
	logger    *slog.Logger
}

func NewHandler(n8nClient *n8n.Client, logger *slog.Logger) *Handler {
	return &Handler{
		n8nClient: n8nClient,
		logger:    logger,
	}
}

// ServeHTTP handles POST /api/agents/{name}
// Accepts a JSON body with "query" and optional "metadata", forwards it to
// the corresponding n8n webhook, and returns the agent's response.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("name")
	if agentName == "" {
		http.Error(w, `{"error":"agent name is required"}`, http.StatusBadRequest)
		return
	}

	var req n8n.AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Query == "" {
		http.Error(w, `{"error":"query field is required"}`, http.StatusBadRequest)
		return
	}

	h.logger.Info("api agent call", "agent", agentName, "query", req.Query)

	resp, err := h.n8nClient.CallAgent(r.Context(), agentName, req)
	if err != nil {
		h.logger.Error("n8n agent call failed", "error", err, "agent", agentName)
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "failed to get response from agent",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
