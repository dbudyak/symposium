package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Handlers struct {
	db *DB
}

func NewHandlers(db *DB) *Handlers {
	return &Handlers{db: db}
}

func (h *Handlers) GetMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	before := r.URL.Query().Get("before")

	msgs, err := h.db.GetMessages(ctx, limit, before)
	if err != nil {
		log.Printf("Error getting messages: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to fetch messages"})
		return
	}

	if msgs == nil {
		msgs = []Message{}
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (h *Handlers) GetMessagesSince(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	afterID := r.URL.Query().Get("after")

	if afterID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "after parameter is required"})
		return
	}

	msgs, err := h.db.GetMessagesSince(ctx, afterID)
	if err != nil {
		log.Printf("Error getting messages since: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to fetch messages"})
		return
	}

	if msgs == nil {
		msgs = []Message{}
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (h *Handlers) PostMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check cooldown
	remaining, err := h.db.GetCooldownRemaining(ctx)
	if err != nil {
		log.Printf("Error checking cooldown: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Internal error"})
		return
	}
	if remaining > 0 {
		writeJSON(w, http.StatusTooManyRequests, ErrorResponse{
			Error:      "Please wait before sending another message",
			RetryAfter: remaining,
		})
		return
	}

	var req PostMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Message content is required"})
		return
	}
	if len(content) > 500 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Message must be under 500 characters"})
		return
	}

	msg, err := h.db.InsertHumanMessage(ctx, content)
	if err != nil {
		log.Printf("Error inserting message: %v", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "Failed to send message"})
		return
	}

	writeJSON(w, http.StatusCreated, msg)
}

func (h *Handlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	running, err := h.db.GetOrchestratorRunning(ctx)
	if err != nil {
		log.Printf("Error getting orchestrator status: %v", err)
	}

	count, err := h.db.GetMessageCount(ctx)
	if err != nil {
		log.Printf("Error getting message count: %v", err)
	}

	cooldown, err := h.db.GetCooldownRemaining(ctx)
	if err != nil {
		log.Printf("Error getting cooldown: %v", err)
	}

	writeJSON(w, http.StatusOK, StatusResponse{
		IsRunning:       running,
		MessageCount:    count,
		Agents:          agentList,
		CooldownSeconds: cooldown,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
