package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var (
	validFeedbackTypes   = map[string]bool{"bug": true, "feature": true, "other": true}
	validFeedbackUrgency = map[string]bool{"normal": true, "critical": true}
)

func (s *server) handleFeedback(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleFeedbackList(w, r)
	case http.MethodPost:
		s.handleFeedbackSubmit(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *server) handleFeedbackSubmit(w http.ResponseWriter, r *http.Request) {
	donor, err := s.authenticatedDonor(r)
	if err != nil || donor == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Rate limit: max 10 per hour per donor
	var recentCount int
	s.db.QueryRow(
		"SELECT COUNT(*) FROM feedback_requests WHERE donor_id = ? AND created_at > datetime('now', '-1 hour')",
		donor.ID,
	).Scan(&recentCount)
	if recentCount >= 10 {
		jsonError(w, "too many submissions, try again later", http.StatusTooManyRequests)
		return
	}

	var req struct {
		Body    string `json:"body"`
		Type    string `json:"type"`
		Urgency string `json:"urgency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		jsonError(w, "feedback body is required", http.StatusBadRequest)
		return
	}
	if len(req.Body) > 10000 {
		jsonError(w, "please keep feedback under 10,000 characters", http.StatusBadRequest)
		return
	}

	if !validFeedbackTypes[req.Type] {
		req.Type = "feature"
	}
	if !validFeedbackUrgency[req.Urgency] {
		req.Urgency = "normal"
	}

	result, err := s.db.Exec(
		"INSERT INTO feedback_requests (donor_id, body, type, urgency) VALUES (?, ?, ?, ?)",
		donor.ID, req.Body, req.Type, req.Urgency,
	)
	if err != nil {
		jsonError(w, "failed to save feedback", http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	log.Printf("[feedback] donor %d submitted %s (%s): %.60s", donor.ID, req.Type, req.Urgency, req.Body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      id,
		"success": true,
		"message": fmt.Sprintf("Feedback #%d submitted. Thank you!", id),
	})
}

func (s *server) handleFeedbackList(w http.ResponseWriter, r *http.Request) {
	donor, err := s.authenticatedDonor(r)
	if err != nil || donor == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := s.db.Query(`
		SELECT id, body, type, urgency, status, admin_notes, created_at, COALESCE(updated_at, '')
		FROM feedback_requests
		WHERE donor_id = ?
		ORDER BY id DESC
	`, donor.ID)
	if err != nil {
		jsonError(w, "failed to fetch feedback", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type feedbackItem struct {
		ID         int64  `json:"id"`
		Body       string `json:"body"`
		Type       string `json:"type"`
		Urgency    string `json:"urgency"`
		Status     string `json:"status"`
		AdminNotes string `json:"admin_notes,omitempty"`
		CreatedAt  string `json:"created_at"`
		UpdatedAt  string `json:"updated_at,omitempty"`
	}

	items := make([]feedbackItem, 0)
	for rows.Next() {
		var f feedbackItem
		rows.Scan(&f.ID, &f.Body, &f.Type, &f.Urgency, &f.Status, &f.AdminNotes, &f.CreatedAt, &f.UpdatedAt)
		items = append(items, f)
	}

	jsonOK(w, items)
}
