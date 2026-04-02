package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	validFeedbackTypes   = map[string]bool{"bug": true, "feature": true, "other": true}
	validFeedbackUrgency = map[string]bool{"normal": true, "critical": true}
)

var githubHTTPClient = &http.Client{Timeout: 10 * time.Second}

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

	// Create GitHub issue asynchronously
	go s.createGitHubIssue(id, req.Type, req.Urgency, req.Body)

	// Write to repo markdown files so Claude Code sees it on next session
	if req.Urgency == "critical" {
		go s.appendToRepoFile("MUST_HAVES.md", req.Body, req.Urgency, "Must-Have Requirements",
			"Items below were submitted via the feedback widget with high urgency.\nClaude Code MUST implement these.")
	} else {
		go s.appendToRepoFile("CONSIDER_FOR_ROADMAP.md", req.Body, req.Urgency, "Consider for Roadmap",
			"Items below were submitted via the feedback widget for consideration.\nClaude Code should evaluate these when planning future work.")
	}

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
		SELECT id, body, type, urgency, status, admin_notes, github_issue_url, created_at, COALESCE(updated_at, '')
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
		ID             int64  `json:"id"`
		Body           string `json:"body"`
		Type           string `json:"type"`
		Urgency        string `json:"urgency"`
		Status         string `json:"status"`
		AdminNotes     string `json:"admin_notes,omitempty"`
		GitHubIssueURL string `json:"github_issue_url,omitempty"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at,omitempty"`
	}

	items := make([]feedbackItem, 0)
	for rows.Next() {
		var f feedbackItem
		rows.Scan(&f.ID, &f.Body, &f.Type, &f.Urgency, &f.Status, &f.AdminNotes, &f.GitHubIssueURL, &f.CreatedAt, &f.UpdatedAt)
		items = append(items, f)
	}

	jsonOK(w, items)
}

// --- GitHub integration ---

func truncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}

func (s *server) createGitHubIssue(feedbackID int64, feedbackType, urgency, body string) {
	ghPAT := os.Getenv("GH_PAT")
	ghRepo := os.Getenv("GH_REPO")
	if ghPAT == "" || ghRepo == "" {
		return
	}

	title := fmt.Sprintf("[%s] %s", feedbackType, truncateRunes(body, 80))

	labels := []string{"feedback", feedbackType}
	if urgency == "critical" {
		labels = append(labels, "priority:critical")
	}

	issueBody := fmt.Sprintf("**Type:** %s\n**Urgency:** %s\n**Feedback ID:** %d\n\n---\n\n%s",
		feedbackType, urgency, feedbackID, body)

	ghResult, ok := s.postGitHubIssue(ghPAT, ghRepo, title, issueBody, labels)
	if !ok {
		log.Printf("[feedback] retrying issue creation without labels")
		ghResult, ok = s.postGitHubIssue(ghPAT, ghRepo, title, issueBody, nil)
	}
	if !ok {
		return
	}

	log.Printf("[feedback] created GitHub issue #%d: %s", ghResult.Number, ghResult.HTMLURL)
	if ghResult.HTMLURL != "" {
		s.db.Exec("UPDATE feedback_requests SET github_issue_url = ? WHERE id = ?", ghResult.HTMLURL, feedbackID)
	}
}

type ghIssueResult struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
}

func (s *server) postGitHubIssue(pat, repo, title, body string, labels []string) (ghIssueResult, bool) {
	issuePayload := map[string]interface{}{
		"title": title,
		"body":  body,
	}
	if len(labels) > 0 {
		issuePayload["labels"] = labels
	}

	payload, _ := json.Marshal(issuePayload)

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues", repo)
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(payload))
	if err != nil {
		log.Printf("[feedback] failed to create GitHub issue request: %v", err)
		return ghIssueResult{}, false
	}
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := githubHTTPClient.Do(req)
	if err != nil {
		log.Printf("[feedback] failed to create GitHub issue: %v", err)
		return ghIssueResult{}, false
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		log.Printf("[feedback] GitHub API returned %d: %s", resp.StatusCode, string(respBody))
		return ghIssueResult{}, false
	}

	var result ghIssueResult
	json.Unmarshal(respBody, &result)
	return result, true
}

func (s *server) appendToRepoFile(filePath, body, urgency, title, description string) {
	ghPAT := os.Getenv("GH_PAT")
	ghRepo := os.Getenv("GH_REPO")
	if ghPAT == "" || ghRepo == "" {
		log.Printf("[feedback] skipping %s update: no GitHub config", filePath)
		return
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", ghRepo, filePath)

	getReq, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("[feedback] failed to create GET request for %s: %v", filePath, err)
		return
	}
	getReq.Header.Set("Authorization", "Bearer "+ghPAT)
	getReq.Header.Set("Accept", "application/vnd.github+json")

	resp, err := githubHTTPClient.Do(getReq)
	if err != nil {
		log.Printf("[feedback] failed to fetch %s: %v", filePath, err)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var existingContent string
	var sha string

	if resp.StatusCode == 200 {
		var fileResp struct {
			Content string `json:"content"`
			SHA     string `json:"sha"`
		}
		if err := json.Unmarshal(respBody, &fileResp); err == nil {
			sha = fileResp.SHA
			s2 := strings.ReplaceAll(fileResp.Content, "\n", "")
			decoded, err := base64.StdEncoding.DecodeString(s2)
			if err == nil {
				existingContent = string(decoded)
			}
		}
	}

	timestamp := time.Now().UTC().Format("2006-01-02 15:04 UTC")
	priorityLabel := strings.ToUpper(urgency)
	newEntry := fmt.Sprintf("\n## [%s] %s\n%s\n_Submitted: %s_\n", priorityLabel, timestamp, body, timestamp)

	var newContent string
	if existingContent == "" {
		newContent = fmt.Sprintf("# %s\n\n%s\n%s", title, description, newEntry)
	} else {
		newContent = existingContent + newEntry
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(newContent))
	commitMsg := fmt.Sprintf("feat: add %s item (%s)", filePath, urgency)

	updatePayload := map[string]interface{}{
		"message": commitMsg,
		"content": encoded,
	}
	if sha != "" {
		updatePayload["sha"] = sha
	}

	payloadBytes, _ := json.Marshal(updatePayload)
	putReq, err := http.NewRequest("PUT", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		log.Printf("[feedback] failed to create PUT request for %s: %v", filePath, err)
		return
	}
	putReq.Header.Set("Authorization", "Bearer "+ghPAT)
	putReq.Header.Set("Accept", "application/vnd.github+json")
	putReq.Header.Set("Content-Type", "application/json")

	putResp, err := githubHTTPClient.Do(putReq)
	if err != nil {
		log.Printf("[feedback] failed to update %s: %v", filePath, err)
		return
	}
	defer putResp.Body.Close()
	putBody, _ := io.ReadAll(putResp.Body)

	if putResp.StatusCode >= 300 {
		log.Printf("[feedback] GitHub API returned %d updating %s: %s", putResp.StatusCode, filePath, string(putBody))
		return
	}

	log.Printf("[feedback] updated %s with %s requirement", filePath, urgency)
}
