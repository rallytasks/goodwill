package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

// handleNPSCheck returns whether the NPS survey should be shown.
// Shows every 5 logins, only if no response in last 90 days.
func (s *server) handleNPSCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	donor, err := s.authenticatedDonor(r)
	if err != nil || donor == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Show NPS every 5 logins
	if donor.LoginCount == 0 || donor.LoginCount%5 != 0 {
		jsonOK(w, map[string]bool{"show_nps": false})
		return
	}

	// Check if already responded recently (last 90 days)
	var recentCount int
	s.db.QueryRow(
		"SELECT COUNT(*) FROM nps_responses WHERE donor_id = ? AND created_at > datetime('now', '-90 days')",
		donor.ID,
	).Scan(&recentCount)

	jsonOK(w, map[string]bool{"show_nps": recentCount == 0})
}

// handleNPSSubmit records an NPS score and optional comment.
func (s *server) handleNPSSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	donor, err := s.authenticatedDonor(r)
	if err != nil || donor == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Score   int    `json:"score"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Score < 0 || req.Score > 10 {
		jsonError(w, "score must be 0-10", http.StatusBadRequest)
		return
	}

	// Idempotency: don't allow duplicate submissions within 1 minute
	var recent int
	s.db.QueryRow(
		"SELECT COUNT(*) FROM nps_responses WHERE donor_id = ? AND created_at > datetime('now', '-1 minute')",
		donor.ID,
	).Scan(&recent)
	if recent > 0 {
		jsonOK(w, map[string]string{"status": "already_recorded"})
		return
	}

	comment := strings.TrimSpace(req.Comment)
	if len(comment) > 2000 {
		comment = comment[:2000]
	}

	_, err = s.db.Exec(
		"INSERT INTO nps_responses (donor_id, score, comment) VALUES (?, ?, ?)",
		donor.ID, req.Score, comment,
	)
	if err != nil {
		jsonError(w, "failed to save response", http.StatusInternalServerError)
		return
	}

	log.Printf("[nps] donor %d scored %d", donor.ID, req.Score)
	jsonOK(w, map[string]string{"status": "recorded"})
}

// handleNPSReport returns NPS data for the reporting dashboard. Admin only.
func (s *server) handleNPSReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	donor, err := s.authenticatedDonor(r)
	if err != nil || donor == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Admin check
	adminPhone := os.Getenv("ADMIN_PHONE")
	donorDigits := strings.TrimPrefix(donor.Phone, "+1")
	if adminPhone == "" || donorDigits != adminPhone {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	// Calculate NPS
	var totalResponses, promoters, passives, detractors int
	rows, err := s.db.Query("SELECT score FROM nps_responses ORDER BY created_at DESC")
	if err != nil {
		jsonError(w, "failed to fetch NPS data", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var score int
		rows.Scan(&score)
		totalResponses++
		if score >= 9 {
			promoters++
		} else if score >= 7 {
			passives++
		} else {
			detractors++
		}
	}

	var npsScore float64
	if totalResponses > 0 {
		npsScore = float64(promoters-detractors) / float64(totalResponses) * 100
	}

	// Get recent comments
	type npsComment struct {
		Score     int    `json:"score"`
		Comment   string `json:"comment"`
		CreatedAt string `json:"created_at"`
	}

	commentRows, err := s.db.Query(`
		SELECT score, comment, created_at
		FROM nps_responses
		WHERE comment != ''
		ORDER BY created_at DESC
		LIMIT 50
	`)
	if err != nil {
		jsonError(w, "failed to fetch comments", http.StatusInternalServerError)
		return
	}
	defer commentRows.Close()

	comments := make([]npsComment, 0)
	for commentRows.Next() {
		var c npsComment
		commentRows.Scan(&c.Score, &c.Comment, &c.CreatedAt)
		comments = append(comments, c)
	}

	jsonOK(w, map[string]interface{}{
		"nps_score":       npsScore,
		"total_responses": totalResponses,
		"promoters":       promoters,
		"passives":        passives,
		"detractors":      detractors,
		"comments":        comments,
	})
}
