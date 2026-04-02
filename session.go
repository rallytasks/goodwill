package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"time"
)

const sessionCookieName = "goodwill_session"
const sessionDuration = 30 * 24 * time.Hour

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *server) createSession(donorID int64) (string, error) {
	token := generateToken()
	expires := time.Now().Add(sessionDuration)
	_, err := s.db.Exec(
		"INSERT INTO sessions (token, donor_id, expires_at) VALUES (?, ?, ?)",
		token, donorID, expires,
	)
	return token, err
}

func (s *server) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionDuration.Seconds()),
	})
}

func (s *server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func (s *server) authenticatedDonor(r *http.Request) (*Donor, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}

	var donor Donor
	err = s.db.QueryRow(`
		SELECT d.id, d.phone, d.name, d.email, COALESCE(d.zip_code, ''), d.created_at
		FROM donors d
		JOIN sessions s ON s.donor_id = d.id
		WHERE s.token = ? AND s.expires_at > ?
	`, cookie.Value, time.Now()).Scan(&donor.ID, &donor.Phone, &donor.Name, &donor.Email, &donor.ZipCode, &donor.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &donor, err
}
