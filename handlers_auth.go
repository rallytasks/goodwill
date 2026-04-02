package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	phoneRegex    = regexp.MustCompile(`^\+1\d{10}$`)
	rateLimitMap  sync.Map
	rateLimitMax  = 5
	rateLimitWindow = 10 * time.Minute
)

type rateLimitEntry struct {
	count     int
	resetAt   time.Time
}

func checkRateLimit(key string) bool {
	now := time.Now()
	val, _ := rateLimitMap.LoadOrStore(key, &rateLimitEntry{count: 0, resetAt: now.Add(rateLimitWindow)})
	entry := val.(*rateLimitEntry)

	if now.After(entry.resetAt) {
		entry.count = 0
		entry.resetAt = now.Add(rateLimitWindow)
	}

	entry.count++
	return entry.count <= rateLimitMax
}

func normalizePhone(phone string) string {
	digits := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")
	if len(digits) == 10 {
		digits = "1" + digits
	}
	if len(digits) == 11 && digits[0] == '1' {
		return "+" + digits
	}
	return ""
}

func (s *server) handleSendCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	phone := normalizePhone(req.Phone)
	if phone == "" {
		jsonError(w, "invalid phone number", http.StatusBadRequest)
		return
	}

	if !checkRateLimit("send:" + phone) {
		jsonError(w, "too many requests, try again later", http.StatusTooManyRequests)
		return
	}

	// Send verification code via Twilio Verify
	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	serviceSID := os.Getenv("TWILIO_VERIFY_SERVICE_SID")

	twilioURL := fmt.Sprintf("https://verify.twilio.com/v2/Services/%s/Verifications", serviceSID)

	data := url.Values{}
	data.Set("To", phone)
	data.Set("Channel", "sms")

	client := &http.Client{Timeout: 10 * time.Second}
	twilioReq, _ := http.NewRequest("POST", twilioURL, strings.NewReader(data.Encode()))
	twilioReq.SetBasicAuth(accountSID, authToken)
	twilioReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(twilioReq)
	if err != nil {
		jsonError(w, "failed to send code", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Twilio error: %s\n", body)
		jsonError(w, "failed to send verification code", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]string{"status": "code_sent"})
}

func (s *server) handleVerifyCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	phone := normalizePhone(req.Phone)
	if phone == "" {
		jsonError(w, "invalid phone number", http.StatusBadRequest)
		return
	}

	if !checkRateLimit("verify:" + phone) {
		jsonError(w, "too many attempts, try again later", http.StatusTooManyRequests)
		return
	}

	// Verify code via Twilio Verify
	accountSID := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_AUTH_TOKEN")
	serviceSID := os.Getenv("TWILIO_VERIFY_SERVICE_SID")

	twilioURL := fmt.Sprintf("https://verify.twilio.com/v2/Services/%s/VerificationCheck", serviceSID)

	data := url.Values{}
	data.Set("To", phone)
	data.Set("Code", req.Code)

	client := &http.Client{Timeout: 10 * time.Second}
	twilioReq, _ := http.NewRequest("POST", twilioURL, strings.NewReader(data.Encode()))
	twilioReq.SetBasicAuth(accountSID, authToken)
	twilioReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(twilioReq)
	if err != nil {
		jsonError(w, "verification failed", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var twilioResp struct {
		Status string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&twilioResp)

	if twilioResp.Status != "approved" {
		jsonError(w, "invalid code", http.StatusUnauthorized)
		return
	}

	// Find or create donor
	var donorID int64
	err = s.db.QueryRow("SELECT id FROM donors WHERE phone = ?", phone).Scan(&donorID)
	if err != nil {
		result, err := s.db.Exec("INSERT INTO donors (phone) VALUES (?)", phone)
		if err != nil {
			jsonError(w, "failed to create account", http.StatusInternalServerError)
			return
		}
		donorID, _ = result.LastInsertId()
	}

	// Increment login count
	s.db.Exec("UPDATE donors SET login_count = COALESCE(login_count, 0) + 1 WHERE id = ?", donorID)

	// Create session
	token, err := s.createSession(donorID)
	if err != nil {
		jsonError(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	s.setSessionCookie(w, token)
	jsonOK(w, map[string]string{"status": "authenticated"})
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		s.db.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
	}

	s.clearSessionCookie(w)
	jsonOK(w, map[string]string{"status": "logged_out"})
}
