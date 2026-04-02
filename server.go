package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type server struct {
	db  *sql.DB
	mux *http.ServeMux
}

func newServer(db *sql.DB) *server {
	s := &server{db: db, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *server) routes() {
	// Static files
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Pages
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/dashboard", s.handleDashboard)

	// Auth API
	s.mux.HandleFunc("/api/auth/send-code", s.handleSendCode)
	s.mux.HandleFunc("/api/auth/verify-code", s.handleVerifyCode)
	s.mux.HandleFunc("/api/auth/logout", s.handleLogout)

	// Donor API
	s.mux.HandleFunc("/api/donor/profile", s.handleProfile)
	s.mux.HandleFunc("/api/donor/update-profile", s.handleUpdateProfile)

	// Donation API
	s.mux.HandleFunc("/api/donations", s.handleDonations)
	s.mux.HandleFunc("/api/donations/create", s.handleCreateDonation)
	s.mux.HandleFunc("/api/receipt/", s.handleReceipt)

	// Feedback API
	s.mux.HandleFunc("/api/feedback", s.handleFeedback)

	// Health check
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func (s *server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
