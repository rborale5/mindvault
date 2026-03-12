package db

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Handler struct {
	DB     *sql.DB
	Logger *slog.Logger
}

// POST /internal/user-facts
// Body: {"user_id":"...","fact":"...","category":"..."}
func (h *Handler) SaveUserFact(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		Fact     string `json:"fact"`
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.UserID == "" || req.Fact == "" {
		http.Error(w, `{"error":"user_id and fact are required"}`, http.StatusBadRequest)
		return
	}
	if req.Category == "" {
		req.Category = "general"
	}

	var id int
	err := h.DB.QueryRowContext(r.Context(),
		`INSERT INTO user_facts (user_id, fact, category) VALUES ($1, $2, $3) RETURNING id`,
		req.UserID, req.Fact, req.Category,
	).Scan(&id)
	if err != nil {
		h.Logger.Error("save_user_fact failed", "error", err)
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": id, "fact": req.Fact, "category": req.Category, "saved": true})
}

// GET /internal/user-facts?user_id=xxx
func (h *Handler) GetUserFacts(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, `{"error":"user_id is required"}`, http.StatusBadRequest)
		return
	}

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT fact, category, created_at FROM user_facts WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`,
		userID,
	)
	if err != nil {
		h.Logger.Error("get_user_facts failed", "error", err)
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Fact struct {
		Fact      string    `json:"fact"`
		Category  string    `json:"category"`
		CreatedAt time.Time `json:"created_at"`
	}
	facts := []Fact{}
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.Fact, &f.Category, &f.CreatedAt); err != nil {
			continue
		}
		facts = append(facts, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(facts)
}

// POST /internal/reminders
// Body: {"user_id":"...","reminder":"...","due_at":"2024-01-01T10:00:00Z"} (due_at optional)
func (h *Handler) SaveReminder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string  `json:"user_id"`
		Reminder string  `json:"reminder"`
		DueAt    *string `json:"due_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if req.UserID == "" || req.Reminder == "" {
		http.Error(w, `{"error":"user_id and reminder are required"}`, http.StatusBadRequest)
		return
	}

	var id int
	var dueAt interface{}
	if req.DueAt != nil && *req.DueAt != "" {
		dueAt = *req.DueAt
	}

	err := h.DB.QueryRowContext(r.Context(),
		`INSERT INTO reminders (user_id, reminder, due_at) VALUES ($1, $2, $3) RETURNING id`,
		req.UserID, req.Reminder, dueAt,
	).Scan(&id)
	if err != nil {
		h.Logger.Error("save_reminder failed", "error", err)
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": id, "reminder": req.Reminder, "saved": true})
}

// GET /internal/reminders?user_id=xxx
func (h *Handler) GetReminders(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, `{"error":"user_id is required"}`, http.StatusBadRequest)
		return
	}

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT id, reminder, due_at, created_at FROM reminders WHERE user_id = $1 AND completed = false ORDER BY due_at ASC NULLS LAST LIMIT 20`,
		userID,
	)
	if err != nil {
		h.Logger.Error("get_reminders failed", "error", err)
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Reminder struct {
		ID        int        `json:"id"`
		Reminder  string     `json:"reminder"`
		DueAt     *time.Time `json:"due_at"`
		CreatedAt time.Time  `json:"created_at"`
	}
	reminders := []Reminder{}
	for rows.Next() {
		var rem Reminder
		if err := rows.Scan(&rem.ID, &rem.Reminder, &rem.DueAt, &rem.CreatedAt); err != nil {
			continue
		}
		reminders = append(reminders, rem)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reminders)
}
