package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/user/rt/internal/audit"
	"github.com/user/rt/internal/operator"
)

// POST /api/auth/login — authenticate with API key, create session
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.APIKey == "" {
		http.Error(w, `{"error":"api_key required"}`, http.StatusBadRequest)
		return
	}

	op, err := operator.Authenticate(s.DB, s.EngID, req.APIKey)
	if err != nil {
		http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
		return
	}

	sess, err := s.Sessions.Create(op.ID, op.Role)
	if err != nil {
		http.Error(w, `{"error":"session creation failed"}`, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "rt_session",
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.shouldUseTLS(),
		MaxAge:   int(s.Sessions.ttl.Seconds()),
	})

	audit.Log(s.DB, op.ID, "auth.login", "session", sess.Token[:16]+"...", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"operator": op.ID,
		"role":     op.Role,
	})
}

// POST /api/auth/logout — destroy session
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if cookie, err := r.Cookie("rt_session"); err == nil {
		s.Sessions.Delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "rt_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// GET /api/auth/me — return current user info
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var opID, role string

	// Check session cookie
	if cookie, err := r.Cookie("rt_session"); err == nil {
		if sess := s.Sessions.Get(cookie.Value); sess != nil {
			opID = sess.OperatorID
			role = sess.Role
		}
	}

	// Check API key
	if opID == "" {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			op, err := operator.Authenticate(s.DB, s.EngID, apiKey)
			if err == nil {
				opID = op.ID
				role = op.Role
			}
		}
	}

	// Localhost solo mode
	if opID == "" && s.isLocalhost() && !s.hasOperators() {
		opID = "local"
		role = "lead"
	}

	if opID == "" {
		http.Error(w, `{"error":"not authenticated"}`, http.StatusUnauthorized)
		return
	}

	// Build permissions map for frontend RBAC
	perms := operator.RolePermissions[role]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"operator":    opID,
		"role":        role,
		"permissions": perms,
	})
}

// POST /api/auth/setup — first-run: create lead operator (only when no operators exist)
func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if s.hasOperators() {
		http.Error(w, `{"error":"setup already completed"}`, http.StatusForbidden)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}

	op, err := operator.Add(s.DB, s.EngID, req.Name, "lead", "system")
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	// Auto-login the new lead
	sess, err := s.Sessions.Create(op.ID, op.Role)
	if err != nil {
		http.Error(w, `{"error":"session creation failed"}`, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "rt_session",
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   s.shouldUseTLS(),
		MaxAge:   int(s.Sessions.ttl.Seconds()),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"operator": op.ID,
		"role":     op.Role,
		"api_key":  op.APIKey,
	})
}

// sessionCookieSecure returns the Secure flag for cookies.
func (s *Server) sessionCookieExpiry() time.Time {
	return time.Now().Add(s.Sessions.ttl)
}
