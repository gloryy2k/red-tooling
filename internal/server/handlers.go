package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/user/rt/internal/attachments"
	"github.com/user/rt/internal/comments"
	"github.com/user/rt/internal/audit"
	"github.com/user/rt/internal/checklist"
	"github.com/user/rt/internal/credentials"
	"github.com/user/rt/internal/engagement"
	"github.com/user/rt/internal/evidence"
	"github.com/user/rt/internal/findings"
	"github.com/user/rt/internal/operator"
	"github.com/user/rt/internal/report"
	"github.com/user/rt/internal/scope"
	"github.com/user/rt/internal/session"
	"github.com/user/rt/internal/templates"
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

func (s *Server) handleDailyActivity(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.Query(`
		SELECT date(timestamp) AS day, COUNT(*) AS cnt
		FROM evidence e
		JOIN sessions ss ON e.session_id = ss.id
		WHERE ss.engagement_id = ? AND e.is_deleted = 0
		GROUP BY day ORDER BY day`, s.EngID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	type dayCount struct {
		Day   string `json:"day"`
		Count int    `json:"count"`
	}
	var data []dayCount
	for rows.Next() {
		var d dayCount
		rows.Scan(&d.Day, &d.Count)
		data = append(data, d)
	}
	jsonResp(w, data)
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
	s.Hub.Broadcast(map[string]interface{}{
		"type":     "finding.create",
		"id":       f.ID,
		"title":    f.Title,
		"priority": f.Priority,
		"operator": op,
	})
	jsonResp(w, f)
}

func (s *Server) handleFindingAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		jsonErr(w, "invalid path", 400)
		return
	}
	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		jsonErr(w, "invalid finding ID", 400)
		return
	}
	op := operatorFromCtx(r)

	// GET /api/findings/:id — single finding detail (no sub-action)
	if r.Method == "GET" && len(parts) < 5 {
		f, err := findings.Get(s.DB, id)
		if err != nil {
			jsonErr(w, err.Error(), 404)
			return
		}
		jsonResp(w, f)
		return
	}

	// DELETE /api/findings/:id (only when no sub-action)
	if r.Method == "DELETE" && len(parts) < 5 {
		if err := findings.Delete(s.DB, id, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		s.Hub.Broadcast(map[string]interface{}{"type": "finding.delete", "id": id, "operator": op})
		jsonResp(w, map[string]string{"status": "ok"})
		return
	}

	// PUT /api/findings/:id — update finding
	if r.Method == "PUT" {
		var req struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Priority    string   `json:"priority"`
			Mitre       []string `json:"mitre"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "invalid JSON", 400)
			return
		}
		if err := findings.Update(s.DB, id, req.Title, req.Description, req.Priority, req.Mitre, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		s.Hub.Broadcast(map[string]interface{}{"type": "finding.update", "id": id, "operator": op})
		jsonResp(w, map[string]string{"status": "ok"})
		return
	}

	// POST /api/findings/:id/:action
	if len(parts) < 5 {
		jsonErr(w, "action required", 400)
		return
	}
	action := parts[4]

	if action == "comments" {
		s.handleComments(w, r)
		return
	}

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
		s.Hub.Broadcast(map[string]interface{}{"type": "finding.verify", "id": id, "status": req.Status, "operator": op})
		jsonResp(w, map[string]string{"status": "ok"})

	case "link":
		var req struct {
			EvidenceIDs []int64 `json:"evidence_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "invalid JSON", 400)
			return
		}
		if err := findings.LinkEvidence(s.DB, id, req.EvidenceIDs, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		s.Hub.Broadcast(map[string]interface{}{"type": "finding.link", "id": id, "evidence_ids": req.EvidenceIDs, "operator": op})
		jsonResp(w, map[string]string{"status": "ok"})

	case "unlink":
		var req struct {
			EvidenceIDs []int64 `json:"evidence_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "invalid JSON", 400)
			return
		}
		if err := findings.UnlinkEvidence(s.DB, id, req.EvidenceIDs, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		s.Hub.Broadcast(map[string]interface{}{"type": "finding.unlink", "id": id, "evidence_ids": req.EvidenceIDs, "operator": op})
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
		s.Hub.Broadcast(map[string]interface{}{"type": "finding.recommend", "id": id, "operator": op})
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

	evBroadcast := map[string]interface{}{
		"type":      "evidence",
		"id":        ev.ID,
		"action":    ev.Action,
		"input":     ev.Input,
		"exit_code": ev.ExitCode,
		"timestamp": ev.Timestamp,
		"operator":  op,
	}
	if ev.Priority != "" {
		evBroadcast["priority"] = ev.Priority
	}
	if len(ev.Tags) > 0 {
		evBroadcast["tags"] = ev.Tags
	}
	s.Hub.Broadcast(evBroadcast)

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

func (s *Server) handleEvidenceDetail(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		jsonErr(w, "invalid path", 400)
		return
	}
	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		jsonErr(w, "invalid evidence ID", 400)
		return
	}
	ev, err := evidence.Get(s.DB, id)
	if err != nil {
		jsonErr(w, err.Error(), 404)
		return
	}
	jsonResp(w, ev)
}

func (s *Server) handleCredAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		jsonErr(w, "invalid path", 400)
		return
	}
	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		jsonErr(w, "invalid credential ID", 400)
		return
	}
	op := operatorFromCtx(r)

	if r.Method == "DELETE" {
		if err := credentials.Delete(s.DB, s.EngID, id, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		s.Hub.Broadcast(map[string]interface{}{"type": "cred.delete", "id": id, "operator": op})
		jsonResp(w, map[string]string{"status": "ok"})
		return
	}

	// GET /api/creds/:id/reveal
	if r.Method == "GET" && len(parts) >= 5 && parts[4] == "reveal" {
		secret, err := credentials.Reveal(s.DB, s.EngID, id, op)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"secret": secret})
		return
	}

	jsonErr(w, "method not allowed", 405)
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
	if r.Method == "POST" {
		var req struct {
			Name string `json:"name"`
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Role == "" {
			jsonErr(w, "name and role required", 400)
			return
		}
		op, err := operator.Add(s.DB, s.EngID, req.Name, req.Role, operatorFromCtx(r))
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"id": op.ID, "role": op.Role, "api_key": op.APIKey})
		return
	}

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

func (s *Server) handleOperatorAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/operators/"), "/")
	opID := parts[0]
	if opID == "" {
		jsonErr(w, "operator ID required", 400)
		return
	}
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	caller := operatorFromCtx(r)

	if action == "rotate" && r.Method == "POST" {
		newKey, err := operator.RotateKey(s.DB, s.EngID, opID, caller)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		s.Sessions.DeleteByOperator(opID)
		jsonResp(w, map[string]string{"api_key": newKey})
		return
	}

	if r.Method == "PUT" {
		var req struct {
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Role == "" {
			jsonErr(w, "role required", 400)
			return
		}
		if opID == caller {
			jsonErr(w, "cannot change your own role", 403)
			return
		}
		if err := operator.UpdateRole(s.DB, s.EngID, opID, req.Role, caller); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		s.Sessions.DeleteByOperator(opID)
		jsonResp(w, map[string]string{"ok": "true"})
		return
	}

	if r.Method == "DELETE" {
		if opID == caller {
			jsonErr(w, "cannot remove yourself", 403)
			return
		}
		if err := operator.Remove(s.DB, s.EngID, opID, caller); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		s.Sessions.DeleteByOperator(opID)
		jsonResp(w, map[string]string{"ok": "true"})
		return
	}

	jsonErr(w, "method not allowed", 405)
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

func (s *Server) handleAttachments(w http.ResponseWriter, r *http.Request) {
	evIDStr := r.URL.Query().Get("evidence_id")
	findingIDStr := r.URL.Query().Get("finding_id")

	if findingIDStr != "" {
		fid, err := strconv.ParseInt(findingIDStr, 10, 64)
		if err != nil {
			jsonErr(w, "invalid finding_id", 400)
			return
		}
		list, err := attachments.ListByFinding(s.DB, fid)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, list)
		return
	}

	if evIDStr != "" {
		eid, err := strconv.ParseInt(evIDStr, 10, 64)
		if err != nil {
			jsonErr(w, "invalid evidence_id", 400)
			return
		}
		list, err := attachments.ListByEvidence(s.DB, eid)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, list)
		return
	}

	list, err := attachments.ListByEngagement(s.DB, s.EngID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, list)
}

func (s *Server) handleAttachmentContent(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		jsonErr(w, "invalid path", 400)
		return
	}
	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		jsonErr(w, "invalid attachment ID", 400)
		return
	}

	content, filename, filetype, err := attachments.GetContent(s.DB, id)
	if err != nil {
		jsonErr(w, err.Error(), 404)
		return
	}

	contentType := "application/octet-stream"
	switch filetype {
	case "screenshot":
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".png":
			contentType = "image/png"
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".gif":
			contentType = "image/gif"
		}
	case "pdf":
		contentType = "application/pdf"
	case "txt", "log", "config":
		contentType = "text/plain; charset=utf-8"
	case "json":
		contentType = "application/json"
	case "csv":
		contentType = "text/csv"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.Write(content)
}

func (s *Server) handleComments(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		jsonErr(w, "invalid path — expected /api/findings/:id/comments", 400)
		return
	}
	findingID, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		jsonErr(w, "invalid finding ID", 400)
		return
	}

	comments.EnsureTable(s.DB)

	// DELETE /api/findings/:id/comments/:commentID
	if r.Method == "DELETE" && len(parts) >= 6 {
		commentID, err := strconv.ParseInt(parts[5], 10, 64)
		if err != nil {
			jsonErr(w, "invalid comment ID", 400)
			return
		}
		op := operatorFromCtx(r)
		if err := comments.Delete(s.DB, commentID, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})
		return
	}

	if r.Method == "POST" {
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "invalid JSON", 400)
			return
		}
		if req.Content == "" {
			jsonErr(w, "content required", 400)
			return
		}
		op := operatorFromCtx(r)
		c, err := comments.Add(s.DB, findingID, op, req.Content)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		s.Hub.Broadcast(map[string]interface{}{"type": "comment.new", "finding_id": findingID, "comment": c})
		w.WriteHeader(201)
		jsonResp(w, c)
		return
	}

	list, err := comments.ListByFinding(s.DB, findingID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	if list == nil {
		list = []comments.Comment{}
	}
	jsonResp(w, list)
}

func (s *Server) handlePresence(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]interface{}{"online": s.Hub.OnlineCount()})
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var req struct {
			Name    string `json:"name"`
			Format  string `json:"format"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "invalid JSON", 400)
			return
		}
		if req.Name == "" {
			jsonErr(w, "name required", 400)
			return
		}
		if req.Format == "" {
			req.Format = "markdown"
		}
		op := operatorFromCtx(r)
		t, err := templates.Create(s.DB, s.EngID, req.Name, req.Format, req.Content, false, op)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		w.WriteHeader(201)
		jsonResp(w, t)
		return
	}

	templates.EnsureTable(s.DB)
	templates.SeedDefaults(s.DB, s.EngID, "system")
	list, err := templates.List(s.DB, s.EngID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	jsonResp(w, list)
}

func (s *Server) handleTemplateAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		jsonErr(w, "invalid path", 400)
		return
	}
	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		jsonErr(w, "invalid template ID", 400)
		return
	}
	op := operatorFromCtx(r)

	if r.Method == "GET" {
		t, err := templates.Get(s.DB, id)
		if err != nil {
			jsonErr(w, err.Error(), 404)
			return
		}
		jsonResp(w, t)
		return
	}

	if r.Method == "PUT" {
		var req struct {
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "invalid JSON", 400)
			return
		}
		if err := templates.Update(s.DB, id, req.Name, req.Content, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})
		return
	}

	if r.Method == "DELETE" {
		if err := templates.Delete(s.DB, id, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})
		return
	}

	jsonErr(w, "method not allowed", 405)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		eng, err := engagement.Get(s.DB, s.EngID)
		if err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		roe := engagement.GetROE(s.DB, s.EngID)
		jsonResp(w, map[string]interface{}{
			"id":         eng.ID,
			"name":       eng.Name,
			"client":     eng.Client,
			"status":     eng.Status,
			"start_date": eng.StartDate,
			"end_date":   eng.EndDate,
			"roe":        roe,
		})
		return
	}

	if r.Method == "PUT" {
		var req struct {
			Name      string `json:"name"`
			Client    string `json:"client"`
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
			Status    string `json:"status"`
			ROE       string `json:"roe"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonErr(w, "invalid JSON", 400)
			return
		}
		op := operatorFromCtx(r)
		if req.Name != "" {
			if err := engagement.Update(s.DB, s.EngID, req.Name, req.Client, req.StartDate, req.EndDate, req.Status, op); err != nil {
				jsonErr(w, err.Error(), 500)
				return
			}
			s.EngName = req.Name
		}
		if err := engagement.SetROE(s.DB, s.EngID, req.ROE, op); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "ok"})
		return
	}

	jsonErr(w, "method not allowed", 405)
}

func (s *Server) handleContext(w http.ResponseWriter, r *http.Request) {
	eng, err := engagement.Get(s.DB, s.EngID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	hosts, _ := scope.List(s.DB, s.EngID)
	scopeTotal, scopeTested := scope.Stats(s.DB, s.EngID)

	findingsList, _ := findings.List(s.DB, s.EngID)
	findingSummary := make([]map[string]interface{}, 0, len(findingsList))
	for _, f := range findingsList {
		findingSummary = append(findingSummary, map[string]interface{}{
			"id":       f.ID,
			"title":    f.Title,
			"priority": f.Priority,
			"status":   f.Verified,
			"mitre":    f.Mitre,
		})
	}

	checkTotal, checkDone := checklist.Stats(s.DB, s.EngID)
	items, _ := checklist.List(s.DB, s.EngID)
	checkItems := make([]map[string]interface{}, 0, len(items))
	for _, it := range items {
		checkItems = append(checkItems, map[string]interface{}{
			"id":       it.ID,
			"category": it.Category,
			"item":     it.Item,
			"checked":  it.Done,
		})
	}

	ctx := map[string]interface{}{
		"engagement": map[string]interface{}{
			"id":         eng.ID,
			"name":       eng.Name,
			"client":     eng.Client,
			"status":     eng.Status,
			"start_date": eng.StartDate,
			"end_date":   eng.EndDate,
		},
		"scope": map[string]interface{}{
			"hosts":  hosts,
			"total":  scopeTotal,
			"tested": scopeTested,
		},
		"findings": findingSummary,
		"checklist": map[string]interface{}{
			"items": checkItems,
			"total": checkTotal,
			"done":  checkDone,
		},
	}
	jsonResp(w, ctx)
}

func operatorFromCtx(r *http.Request) string {
	if op, ok := r.Context().Value(operatorKey).(string); ok && op != "" {
		return op
	}
	return "dashboard"
}
