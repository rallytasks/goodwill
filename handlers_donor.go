package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

func (s *server) handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	donor, err := s.authenticatedDonor(r)
	if err != nil || donor == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	adminPhone := os.Getenv("ADMIN_PHONE")
	// Normalize: compare just digits
	donorDigits := strings.TrimPrefix(donor.Phone, "+1")
	canFeedback := adminPhone != "" && donorDigits == adminPhone

	jsonOK(w, map[string]interface{}{
		"id":                  donor.ID,
		"phone":               donor.Phone,
		"name":                donor.Name,
		"email":               donor.Email,
		"created_at":          donor.CreatedAt,
		"can_submit_feedback": canFeedback,
	})
}

func (s *server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
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
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	_, err = s.db.Exec("UPDATE donors SET name = ?, email = ? WHERE id = ?", req.Name, req.Email, donor.ID)
	if err != nil {
		jsonError(w, "failed to update profile", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"status": "updated"})
}
