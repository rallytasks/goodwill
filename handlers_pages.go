package main

import (
	"net/http"
)

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// If already authenticated, redirect to dashboard
	donor, _ := s.authenticatedDonor(r)
	if donor != nil {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}

	http.ServeFile(w, r, "static/index.html")
}

func (s *server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	donor, _ := s.authenticatedDonor(r)
	if donor == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.ServeFile(w, r, "static/dashboard.html")
}

func (s *server) handleReporting(w http.ResponseWriter, r *http.Request) {
	donor, _ := s.authenticatedDonor(r)
	if donor == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	http.ServeFile(w, r, "static/reporting.html")
}
