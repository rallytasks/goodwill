package main

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func (s *server) isAdmin(donor *Donor) bool {
	adminPhone := os.Getenv("ADMIN_PHONE")
	donorDigits := strings.TrimPrefix(donor.Phone, "+1")
	return adminPhone != "" && donorDigits == adminPhone
}

// handleDonorReport returns donor stats by zip code. Admin only.
func (s *server) handleDonorReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	donor, err := s.authenticatedDonor(r)
	if err != nil || donor == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !s.isAdmin(donor) {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	type zipStats struct {
		ZipCode    string `json:"zip_code"`
		DonorCount int    `json:"donor_count"`
		Donations  int    `json:"donations"`
	}

	rows, err := s.db.Query(`
		SELECT COALESCE(d.zip_code, 'Unknown') as zip,
			COUNT(DISTINCT d.id) as donor_count,
			COUNT(dn.id) as donation_count
		FROM donors d
		LEFT JOIN donations dn ON dn.donor_id = d.id
		WHERE d.zip_code != ''
		GROUP BY zip
		ORDER BY donor_count DESC
	`)
	if err != nil {
		jsonError(w, "failed to fetch report", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	stats := make([]zipStats, 0)
	for rows.Next() {
		var s zipStats
		rows.Scan(&s.ZipCode, &s.DonorCount, &s.Donations)
		stats = append(stats, s)
	}

	// Summary stats
	var totalDonors, totalDonations int
	s.db.QueryRow("SELECT COUNT(*) FROM donors").Scan(&totalDonors)
	s.db.QueryRow("SELECT COUNT(*) FROM donations").Scan(&totalDonations)

	jsonOK(w, map[string]interface{}{
		"by_zip":          stats,
		"total_donors":    totalDonors,
		"total_donations": totalDonations,
	})
}

// handleReportCSV exports full donor + donation data as CSV. Admin only.
func (s *server) handleReportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	donor, err := s.authenticatedDonor(r)
	if err != nil || donor == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if !s.isAdmin(donor) {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	rows, err := s.db.Query(`
		SELECT d.name, d.email, d.phone, COALESCE(d.zip_code, ''),
			COALESCE(dn.receipt_number, ''), COALESCE(dn.donation_date, ''),
			COALESCE(dn.location, ''), COALESCE(dn.items_description, ''),
			COALESCE(dn.created_at, '')
		FROM donors d
		LEFT JOIN donations dn ON dn.donor_id = d.id
		ORDER BY d.id, dn.created_at DESC
	`)
	if err != nil {
		jsonError(w, "failed to export data", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=goodwill-donations-%s.csv", strings.ReplaceAll(r.URL.Query().Get("_t"), " ", "")))

	writer := csv.NewWriter(w)
	writer.Write([]string{"Donor Name", "Email", "Phone", "Zip Code", "Receipt Number", "Donation Date", "Location", "Items", "Created At"})

	for rows.Next() {
		var name, email, phone, zip, receipt, date, location, items, created string
		rows.Scan(&name, &email, &phone, &zip, &receipt, &date, &location, &items, &created)
		writer.Write([]string{name, email, phone, zip, receipt, date, location, items, created})
	}

	writer.Flush()
}
