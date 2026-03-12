package alexa

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/rborale/mindvault/internal/config"
	"github.com/rborale/mindvault/internal/n8n"
)

type Handler struct {
	cfg       *config.Config
	n8nClient *n8n.Client
	logger    *slog.Logger
}

func NewHandler(cfg *config.Config, n8nClient *n8n.Client, logger *slog.Logger) *Handler {
	return &Handler{
		cfg:       cfg,
		n8nClient: n8nClient,
		logger:    logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read request body", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if h.cfg.AlexaVerifyRequests {
		if err := VerifyRequest(r, body); err != nil {
			h.logger.Warn("alexa request verification failed", "error", err)
			http.Error(w, "request verification failed", http.StatusForbidden)
			return
		}
	}

	var alexaReq Request
	if err := json.Unmarshal(body, &alexaReq); err != nil {
		h.logger.Error("failed to parse alexa request", "error", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if h.cfg.AlexaSkillID != "" && alexaReq.Session.Application.ApplicationID != h.cfg.AlexaSkillID {
		h.logger.Warn("skill ID mismatch",
			"expected", h.cfg.AlexaSkillID,
			"got", alexaReq.Session.Application.ApplicationID,
		)
		http.Error(w, "skill ID mismatch", http.StatusForbidden)
		return
	}

	if err := h.validateTimestamp(alexaReq.Request.Timestamp); err != nil {
		h.logger.Warn("request timestamp too old", "error", err)
		http.Error(w, "request too old", http.StatusBadRequest)
		return
	}

	var resp Response
	switch alexaReq.Request.Type {
	case "LaunchRequest":
		resp = h.handleLaunch()
	case "IntentRequest":
		resp = h.handleIntent(r, alexaReq)
	case "SessionEndedRequest":
		resp = NewTextResponse("Goodbye!", true)
	default:
		resp = NewTextResponse("I'm not sure how to handle that.", true)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleLaunch() Response {
	return NewRepromptResponse(
		"Welcome! I'm your personal assistant. How can I help you today?",
		"You can ask me anything. What would you like to know?",
	)
}

func (h *Handler) handleIntent(r *http.Request, alexaReq Request) Response {
	intent := alexaReq.Request.Intent

	if intent.Name == "AMAZON.HelpIntent" {
		return NewRepromptResponse(
			"You can ask me any question and I'll do my best to help. What would you like to know?",
			"What can I help you with?",
		)
	}
	if intent.Name == "AMAZON.StopIntent" || intent.Name == "AMAZON.CancelIntent" {
		return NewTextResponse("Goodbye!", true)
	}

	agentPath := h.cfg.AgentForIntent(intent.Name)

	query := h.extractQuery(intent)
	if query == "" {
		return NewRepromptResponse(
			"I didn't catch that. Could you please repeat your question?",
			"What would you like to ask?",
		)
	}

	h.logger.Info("calling n8n agent",
		"intent", intent.Name,
		"agent", agentPath,
		"query", query,
	)

	agentReq := n8n.AgentRequest{
		Query:     query,
		SessionID: alexaReq.Session.SessionID,
		UserID:    alexaReq.Session.User.UserID,
		Metadata: map[string]string{
			"source": "alexa",
			"intent": intent.Name,
			"locale": alexaReq.Request.Locale,
		},
	}

	agentResp, err := h.n8nClient.CallAgent(r.Context(), agentPath, agentReq)
	if err != nil {
		h.logger.Error("n8n agent call failed", "error", err, "agent", agentPath)
		return NewTextResponse(
			"I'm sorry, I had trouble processing your request. Please try again later.",
			true,
		)
	}

	responseText := strings.TrimSpace(agentResp.Response)
	if responseText == "" {
		responseText = "I processed your request but didn't get a response. Please try again."
	}

	return NewTextResponseWithCard(responseText, intent.Name, responseText, true)
}

// extractQuery pulls the user's query from known slot names.
// Alexa skills typically use a catch-all slot to capture free-form speech.
func (h *Handler) extractQuery(intent Intent) string {
	slotPriority := []string{"query", "question", "message", "text", "input"}

	for _, name := range slotPriority {
		if slot, ok := intent.Slots[name]; ok && slot.Value != "" {
			return slot.Value
		}
	}

	// Fall back to the first non-empty slot value
	for _, slot := range intent.Slots {
		if slot.Value != "" {
			return slot.Value
		}
	}

	return ""
}

func (h *Handler) validateTimestamp(ts string) error {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return err
	}
	if time.Since(t) > 150*time.Second {
		return http.ErrAbortHandler
	}
	return nil
}
