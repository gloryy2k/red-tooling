package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/user/rt/internal/audit"
	"github.com/user/rt/internal/checklist"
	"github.com/user/rt/internal/credentials"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/findings"
	"github.com/user/rt/internal/operator"
	"github.com/user/rt/internal/report"
	"github.com/user/rt/internal/scope"
	"github.com/user/rt/internal/session"
)

func jsonResp(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	opts := report.Options{}
	data, err := report.Gather(s.DB, s.EngID, opts)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	total, tested := scope.Stats(s.DB, s.EngID)
	_, checkDone := checklist.Stats(s.DB, s.EngID)

	overview := map[string]interface{}{
		"engagement":     data.Engagement,
		"stats":          data.Stats,
		"chain_intact":   data.ChainIntact,
		"cred_count":     data.CredCount,
		"session_count":  len(data.Sessions),
		"generated_at":   data.GeneratedAt,
		"scope_total":    total,
		"scope_tested":   tested,
		"checklist_done": checkDone,
	}
	jsonResp(w, overview)
}

func (s *Server) handleFindings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		s.handleCreateFinding(w, r)
		return
	}
	list, err := findings.List(s.DB, s.EngID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, list)
}

func (s *Server) handleCreateFinding(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Priority    string   `json:"priority"`
		Mitre       []string `json:"mitre"`
		EvidenceIDs []int64  `json:"evidence_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid JSON", 400)
		return
	}
	if req.Title == "" {
		jsonErr(w, "title required", 400)
		return
	}
	op := operatorFromCtx(r)
	f, err := findings.Create(s.DB, s.EngID, req.Title, req.Description, req.Priority, op, req.EvidenceIDs, req.Mitre)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, f)
}

func (s *Server) handleFindingAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		jsonErr(w, "invalid path", 400)
		return
	}
	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		jsonErr(w, "invalid finding ID", 400)
		return
	}
	action := parts[4]
	op := operatorFromCtx(r)

	switch action {
	case "verify":
		var req struct {
			Status string `json:"status"`
			Note   string `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "invalid JSON", 400)
			return
		}
		if err := findings.Verify(s.DB, id, req.Status, op, req.Note); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})

	case "recommend":
		var req struct {
			Recommendation string `json:"recommendation"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "invalid JSON", 400)
			return
		}
		if err := findings.SetRecommendation(s.DB, id, req.Recommendation, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})

	default:
		jsonErr(w, "unknown action", 400)
	}
}

func (s *Server) handleEvidence(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		s.handleSubmitEvidence(w, r)
		return
	}
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}
	ev, err := evidence.Timeline(s.DB, s.EngID, limit)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, ev)
}

func (s *Server) handleSubmitEvidence(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID  string   `json:"session_id"`
		Action     string   `json:"action"`
		Input      string   `json:"input"`
		Output     string   `json:"output"`
		ExitCode   int      `json:"exit_code"`
		DurationMs int      `json:"duration_ms"`
		CWD        string   `json:"cwd"`
		Tags       []string `json:"tags"`
		Priority   string   `json:"priority"`
		Operator   string   `json:"operator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid JSON", 400)
		return
	}
	if req.SessionID == "" || req.Action == "" {
		jsonErr(w, "session_id and action required", 400)
		return
	}

	op := req.Operator
	if op == "" {
		op = operatorFromCtx(r)
	}

	ev, err := evidence.Insert(s.DB, req.SessionID, req.Action, req.Input, req.Output,
		req.ExitCode, req.DurationMs, req.CWD, req.Tags, req.Priority, op)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	flagResult, _ := evidence.AutoFlag(s.DB, ev, s.EngID, op)

	type EvidenceResponse struct {
		ID        int64                    `json:"id"`
		Hash      string                   `json:"hash"`
		PrevHash  string                   `json:"prev_hash"`
		Timestamp string                   `json:"timestamp"`
		AutoFlag  *evidence.AutoFlagResult `json:"auto_flag,omitempty"`
	}

	resp := EvidenceResponse{
		ID:        ev.ID,
		Hash:      ev.Hash,
		PrevHash:  ev.PrevHash,
		Timestamp: ev.Timestamp,
		AutoFlag:  flagResult,
	}

	s.Hub.Broadcast(map[string]interface{}{
		"type":      "evidence",
		"id":        ev.ID,
		"action":    ev.Action,
		"input":     ev.Input,
		"exit_code": ev.ExitCode,
		"timestamp": ev.Timestamp,
		"operator":  op,
	})

	w.WriteHeader(201)
	jsonResp(w, resp)
}

func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	ev, err := evidence.Timeline(s.DB, s.EngID, 500)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	type TimelineEntry struct {
		ID        int64    `json:"id"`
		Timestamp string   `json:"timestamp"`
		Action    string   `json:"action"`
		Input     string   `json:"input"`
		Output    string   `json:"output"`
		ExitCode  int      `json:"exit_code"`
		Tags      []string `json:"tags"`
		Priority  string   `json:"priority"`
		Mitre     []string `json:"mitre"`
	}

	var entries []TimelineEntry
	for _, e := range ev {
		output := e.Output
		if len(output) > 500 {
			output = output[:497] + "..."
		}
		entries = append(entries, TimelineEntry{
			ID:        e.ID,
			Timestamp: e.Timestamp,
			Action:    e.Action,
			Input:     e.Input,
			Output:    output,
			ExitCode:  e.ExitCode,
			Tags:      e.Tags,
			Priority:  e.Priority,
			Mitre:     e.Mitre,
		})
	}
	jsonResp(w, entries)
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		s.handleSessionAction(w, r)
		return
	}
	list, err := session.List(s.DB, s.EngID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, list)
}

func (s *Server) handleSessionAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action   string `json:"action"`
		ID       string `json:"id"`
		Name     string `json:"name"`
		Source   string `json:"source"`
		Operator string `json:"operator"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid JSON", 400)
		return
	}

	op := req.Operator
	if op == "" {
		op = operatorFromCtx(r)
	}

	switch req.Action {
	case "start":
		if req.ID == "" || req.Name == "" {
			jsonErr(w, "id and name required", 400)
			return
		}
		if req.Source == "" {
			req.Source = "remote"
		}
		if err := session.Create(s.DB, req.ID, s.EngID, req.Name, req.Source, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		s.Hub.Broadcast(map[string]interface{}{
			"type":     "session",
			"action":   "start",
			"id":       req.ID,
			"name":     req.Name,
			"operator": op,
		})
		w.WriteHeader(201)
		jsonResp(w, map[string]string{"status": "started", "session_id": req.ID})

	case "stop":
		if req.ID == "" {
			jsonErr(w, "id required", 400)
			return
		}
		if err := session.Stop(s.DB, req.ID, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		s.Hub.Broadcast(map[string]interface{}{
			"type":   "session",
			"action": "stop",
			"id":     req.ID,
		})
		jsonResp(w, map[string]string{"status": "stopped"})

	default:
		jsonErr(w, "action must be 'start' or 'stop'", 400)
	}
}

func (s *Server) handleCreds(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		s.handleSubmitCred(w, r)
		return
	}
	creds, err := credentials.List(s.DB, s.EngID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, creds)
}

func (s *Server) handleSubmitCred(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username   string `json:"username"`
		Secret     string `json:"secret"`
		SecretType string `json:"secret_type"`
		Host       string `json:"host"`
		Operator   string `json:"operator"`
		EvidenceID int64  `json:"evidence_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid JSON", 400)
		return
	}
	if req.Username == "" || req.Secret == "" {
		jsonErr(w, "username and secret required", 400)
		return
	}
	if req.SecretType == "" {
		req.SecretType = "password"
	}

	op := req.Operator
	if op == "" {
		op = operatorFromCtx(r)
	}

	if err := credentials.Store(s.DB, s.EngID, req.Username, req.Secret, req.SecretType, req.Host, op, req.EvidenceID); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	s.Hub.Broadcast(map[string]interface{}{
		"type":        "credential",
		"username":    req.Username,
		"secret_type": req.SecretType,
		"host":        req.Host,
		"operator":    op,
	})

	w.WriteHeader(201)
	jsonResp(w, map[string]string{"status": "stored"})
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}
	entries, err := audit.List(s.DB, limit)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, entries)
}

func (s *Server) handleOperators(w http.ResponseWriter, r *http.Request) {
	ops, err := operator.List(s.DB, s.EngID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	type SafeOp struct {
		ID         string `json:"id"`
		Role       string `json:"role"`
		CreatedAt  string `json:"created_at"`
		LastSeenAt string `json:"last_seen_at"`
	}
	var safe []SafeOp
	for _, o := range ops {
		safe = append(safe, SafeOp{
			ID:         o.ID,
			Role:       o.Role,
			CreatedAt:  o.CreatedAt,
			LastSeenAt: o.LastSeenAt,
		})
	}
	jsonResp(w, safe)
}

func (s *Server) handleScope(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var req struct {
			Hosts string `json:"hosts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "invalid JSON", 400)
			return
		}
		op := operatorFromCtx(r)
		count, err := scope.Add(s.DB, s.EngID, req.Hosts, op)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]int{"added": count})
		return
	}

	hosts, err := scope.List(s.DB, s.EngID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	total, tested := scope.Stats(s.DB, s.EngID)
	jsonResp(w, map[string]interface{}{
		"hosts":  hosts,
		"total":  total,
		"tested": tested,
	})
}

func (s *Server) handleScopeTested(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Host      string `json:"host"`
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid JSON", 400)
		return
	}
	op := operatorFromCtx(r)
	if err := scope.MarkTested(s.DB, s.EngID, req.Host, req.SessionID, op); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "ok"})
}

func (s *Server) handleChecklist(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var req struct {
			Category string `json:"category"`
			Item     string `json:"item"`
			Preset   string `json:"preset"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "invalid JSON", 400)
			return
		}
		if req.Preset != "" {
			if err := checklist.LoadPreset(s.DB, s.EngID, req.Preset); err != nil {
				jsonErr(w, err.Error(), 500)
				return
			}
			jsonResp(w, map[string]string{"status": "ok"})
			return
		}
		if err := checklist.Add(s.DB, s.EngID, req.Category, req.Item); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})
		return
	}

	items, err := checklist.List(s.DB, s.EngID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	total, done := checklist.Stats(s.DB, s.EngID)
	jsonResp(w, map[string]interface{}{
		"items": items,
		"total": total,
		"done":  done,
	})
}

func (s *Server) handleChecklistToggle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID         int64  `json:"id"`
		Checked    bool   `json:"checked"`
		EvidenceID int64  `json:"evidence_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid JSON", 400)
		return
	}
	op := operatorFromCtx(r)
	if req.Checked {
		if err := checklist.Check(s.DB, req.ID, op, req.EvidenceID); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
	} else {
		if err := checklist.Uncheck(s.DB, req.ID); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
	}
	jsonResp(w, map[string]string{"status": "ok"})
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "html"
	}
	opts := report.Options{
		CriticalOnly: r.URL.Query().Get("critical") == "true",
		VerifiedOnly: r.URL.Query().Get("verified") == "true",
		ExecOnly:     r.URL.Query().Get("exec") == "true",
		TechOnly:     r.URL.Query().Get("tech") == "true",
	}

	data, err := report.Gather(s.DB, s.EngID, opts)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	switch strings.ToLower(format) {
	case "html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(report.RenderHTML(data, opts)))
	case "markdown", "md":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte(report.RenderMarkdown(data, opts)))
	default:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(report.RenderHTML(data, opts)))
	}
}

func operatorFromCtx(r *http.Request) string {
	if op, ok := r.Context().Value(operatorKey).(string); ok && op != "" {
		return op
	}
	return "dashboard"
}
