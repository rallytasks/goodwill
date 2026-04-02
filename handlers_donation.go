package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func generateReceiptNumber() string {
	now := time.Now()
	token := generateToken()[:6]
	return fmt.Sprintf("GW-%s-%s", now.Format("20060102"), strings.ToUpper(token))
}

func (s *server) handleCreateDonation(w http.ResponseWriter, r *http.Request) {
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
		DonationDate     string `json:"donation_date"`
		Location         string `json:"location"`
		ItemsDescription string `json:"items_description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.ItemsDescription == "" {
		jsonError(w, "items description is required", http.StatusBadRequest)
		return
	}

	if req.DonationDate == "" {
		req.DonationDate = time.Now().Format("2006-01-02")
	}

	receiptNumber := generateReceiptNumber()

	result, err := s.db.Exec(`
		INSERT INTO donations (donor_id, receipt_number, donation_date, location, items_description)
		VALUES (?, ?, ?, ?, ?)
	`, donor.ID, receiptNumber, req.DonationDate, req.Location, req.ItemsDescription)
	if err != nil {
		jsonError(w, "failed to create donation", http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	jsonOK(w, map[string]interface{}{
		"id":             id,
		"receipt_number": receiptNumber,
		"status":         "created",
	})
}

func (s *server) handleDonations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	donor, err := s.authenticatedDonor(r)
	if err != nil || donor == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := s.db.Query(`
		SELECT id, receipt_number, donation_date, location, items_description, created_at
		FROM donations
		WHERE donor_id = ?
		ORDER BY created_at DESC
	`, donor.ID)
	if err != nil {
		jsonError(w, "failed to fetch donations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var donations []Donation
	for rows.Next() {
		var d Donation
		rows.Scan(&d.ID, &d.ReceiptNumber, &d.DonationDate, &d.Location, &d.ItemsDescription, &d.CreatedAt)
		d.DonorID = donor.ID
		donations = append(donations, d)
	}

	if donations == nil {
		donations = []Donation{}
	}

	jsonOK(w, donations)
}

func (s *server) handleReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	donor, err := s.authenticatedDonor(r)
	if err != nil || donor == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract receipt number from URL path: /api/receipt/GW-XXXXXXXX-YYYYYY
	receiptNumber := strings.TrimPrefix(r.URL.Path, "/api/receipt/")
	if receiptNumber == "" {
		jsonError(w, "receipt number required", http.StatusBadRequest)
		return
	}

	var donation Donation
	err = s.db.QueryRow(`
		SELECT id, donor_id, receipt_number, donation_date, location, items_description, created_at
		FROM donations
		WHERE receipt_number = ? AND donor_id = ?
	`, receiptNumber, donor.ID).Scan(
		&donation.ID, &donation.DonorID, &donation.ReceiptNumber,
		&donation.DonationDate, &donation.Location, &donation.ItemsDescription, &donation.CreatedAt,
	)
	if err != nil {
		jsonError(w, "receipt not found", http.StatusNotFound)
		return
	}

	jsonOK(w, map[string]interface{}{
		"receipt":    donation,
		"donor_name": donor.Name,
		"charity":   "Goodwill Industries of Denver",
		"ein":       "84-0404583",
		"statement": "No goods or services were provided in exchange for this donation.",
	})
}
