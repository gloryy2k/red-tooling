package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/user/rt/internal/config"
	"github.com/user/rt/internal/operator"
)

type Server struct {
	DB          *sql.DB
	EngID       string
	EngName     string
	Listen      string
	TLSCert     string
	TLSKey      string
	Hub         *WSHub
	Sessions    *SessionStore
	LockTimeout time.Duration
	lastActivity time.Time
	activityMu   sync.Mutex
}

type contextKey string

const operatorKey contextKey = "operator"
const operatorRoleKey contextKey = "operator_role"

func New(db *sql.DB, engID, engName, listen, tlsCert, tlsKey string) *Server {
	return &Server{
		DB:           db,
		EngID:        engID,
		EngName:      engName,
		Listen:       listen,
		TLSCert:      tlsCert,
		TLSKey:       tlsKey,
		Hub:          NewWSHub(),
		Sessions:     NewSessionStore(8 * time.Hour),
		LockTimeout:  30 * time.Minute,
		lastActivity: time.Now(),
	}
}

func (s *Server) touchActivity() {
	s.activityMu.Lock()
	s.lastActivity = time.Now()
	s.activityMu.Unlock()
}

func (s *Server) idleDuration() time.Duration {
	s.activityMu.Lock()
	d := time.Since(s.lastActivity)
	s.activityMu.Unlock()
	return d
}

func (s *Server) autoLockLoop() {
	if s.LockTimeout <= 0 {
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if s.idleDuration() >= s.LockTimeout {
			s.Sessions.ExpireAll()
			s.Hub.Broadcast([]byte(`{"type":"lock","reason":"inactivity"}`))
			fmt.Printf("  [auto-lock] Sessions expired after %s of inactivity\n", s.LockTimeout)
			s.touchActivity()
		}
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API routes — read
	mux.HandleFunc("/api/overview", s.withAuth("overview", s.handleOverview))
	mux.HandleFunc("/api/findings", s.withAuth("findings", s.handleFindings))
	mux.HandleFunc("/api/evidence", s.withAuth("evidence", s.handleEvidence))
	mux.HandleFunc("/api/timeline", s.withAuth("timeline", s.handleTimeline))
	mux.HandleFunc("/api/sessions", s.withAuth("sessions", s.handleSessions))
	mux.HandleFunc("/api/creds", s.withAuth("creds", s.handleCreds))
	mux.HandleFunc("/api/audit", s.withAuth("audit", s.handleAudit))
	mux.HandleFunc("/api/operators", s.withAuth("operators", s.handleOperators))
	mux.HandleFunc("/api/operators/", s.withAuth("operators", s.handleOperatorAction))
	mux.HandleFunc("/api/report", s.withAuth("report", s.handleReport))
	// API routes — write / detail
	mux.HandleFunc("/api/findings/", s.withAuth("findings", s.handleFindingAction))
	mux.HandleFunc("/api/presence", s.withAuth("overview", s.handlePresence))
	mux.HandleFunc("/api/evidence/", s.withAuth("evidence", s.handleEvidenceDetail))
	mux.HandleFunc("/api/creds/", s.withAuth("creds", s.handleCredAction))
	mux.HandleFunc("/api/attachments", s.withAuth("evidence", s.handleAttachments))
	mux.HandleFunc("/api/attachments/", s.withAuth("evidence", s.handleAttachmentContent))
	mux.HandleFunc("/api/templates", s.withAuth("report", s.handleTemplates))
	mux.HandleFunc("/api/templates/", s.withAuth("report", s.handleTemplateAction))
	mux.HandleFunc("/api/scope", s.withAuth("scope", s.handleScope))
	mux.HandleFunc("/api/scope/tested", s.withAuth("scope", s.handleScopeTested))
	mux.HandleFunc("/api/checklist", s.withAuth("checklist", s.handleChecklist))
	mux.HandleFunc("/api/checklist/toggle", s.withAuth("checklist", s.handleChecklistToggle))
	mux.HandleFunc("/api/context", s.withAuth("overview", s.handleContext))
	mux.HandleFunc("/api/overview/activity", s.withAuth("overview", s.handleDailyActivity))
	mux.HandleFunc("/api/settings", s.withAuth("operators", s.handleSettings))
	// Auth routes (no withAuth — these handle their own auth)
	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.HandleFunc("/api/auth/logout", s.handleLogout)
	mux.HandleFunc("/api/auth/me", s.handleMe)
	mux.HandleFunc("/api/auth/setup", s.handleSetup)
	// WebSocket
	mux.HandleFunc("/ws/live", s.handleWS)

	// Frontend (static)
	mux.HandleFunc("/", s.handleFrontend)

	handler := s.securityHeaders(s.rateLimit(mux))

	useTLS := s.shouldUseTLS()

	if useTLS {
		certFile, keyFile, err := s.ensureTLS()
		if err != nil {
			return fmt.Errorf("TLS setup: %w", err)
		}

		tlsCfg := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		srv := &http.Server{
			Addr:      s.Listen,
			Handler:   handler,
			TLSConfig: tlsCfg,
		}

		fmt.Printf("\n  RT Dashboard — https://%s\n", s.Listen)
		fmt.Printf("  Engagement: %s\n", s.EngName)
		fmt.Printf("  TLS: %s\n\n", certFile)
		fmt.Printf("  Use 'rt operator list' to see API keys.\n")
		fmt.Printf("  Press Ctrl+C to stop.\n\n")

		go s.Hub.Run()
		go s.autoLockLoop()
		return srv.ListenAndServeTLS(certFile, keyFile)
	}

	srv := &http.Server{
		Addr:    s.Listen,
		Handler: handler,
	}

	fmt.Printf("\n  RT Dashboard — http://%s\n", s.Listen)
	fmt.Printf("  Engagement: %s\n", s.EngName)
	fmt.Printf("  WARNING: No TLS — localhost only!\n\n")
	if s.LockTimeout > 0 {
		fmt.Printf("  Auto-lock: %s\n", s.LockTimeout)
	}
	fmt.Printf("  Press Ctrl+C to stop.\n\n")

	go s.Hub.Run()
	go s.autoLockLoop()
	return srv.ListenAndServe()
}

func (s *Server) shouldUseTLS() bool {
	if s.TLSCert != "" && s.TLSKey != "" {
		return true
	}
	host, _, _ := net.SplitHostPort(s.Listen)
	return host != "" && host != "localhost" && host != "127.0.0.1" && host != "::1"
}

func (s *Server) ensureTLS() (string, string, error) {
	if s.TLSCert != "" && s.TLSKey != "" {
		return s.TLSCert, s.TLSKey, nil
	}

	certPath := filepath.Join(config.TLSDir(), "server.crt")
	keyPath := filepath.Join(config.TLSDir(), "server.key")

	if _, err := os.Stat(certPath); err == nil {
		return certPath, keyPath, nil
	}

	return generateSelfSigned(certPath, keyPath, s.Listen)
}

func generateSelfSigned(certPath, keyPath, listen string) (string, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	host, _, _ := net.SplitHostPort(listen)

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{Organization: []string{"RT Red Team Logger"}, CommonName: host},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	if ip := net.ParseIP(host); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{host}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}

	certFile, err := os.Create(certPath)
	if err != nil {
		return "", "", err
	}
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certFile.Close()

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", err
	}
	keyFile, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", err
	}
	pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	keyFile.Close()

	fmt.Printf("  [tls] Auto-generated self-signed certificate\n")
	return certPath, keyPath, nil
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws: wss:")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		if s.shouldUseTLS() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// Rate limiter: per-IP, token bucket
type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*bucket
}

type bucket struct {
	tokens    int
	lastReset time.Time
}

var limiter = &rateLimiter{clients: make(map[string]*bucket)}

func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		limiter.mu.Lock()
		b, ok := limiter.clients[ip]
		if !ok {
			b = &bucket{tokens: 60, lastReset: time.Now()}
			limiter.clients[ip] = b
		}
		if time.Since(b.lastReset) > time.Minute {
			b.tokens = 60
			b.lastReset = time.Now()
		}
		if b.tokens <= 0 {
			limiter.mu.Unlock()
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		b.tokens--
		limiter.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func (s *Server) isLocalhost() bool {
	host, _, _ := net.SplitHostPort(s.Listen)
	return host == "" || host == "localhost" || host == "127.0.0.1"
}

func (s *Server) hasOperators() bool {
	ops, err := operator.List(s.DB, s.EngID)
	return err == nil && len(ops) > 0
}

func (s *Server) withAuth(resource string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var opID, role string

		// 1. Check session cookie (dashboard browser sessions)
		if cookie, err := r.Cookie("rt_session"); err == nil {
			if sess := s.Sessions.Get(cookie.Value); sess != nil {
				opID = sess.OperatorID
				role = sess.Role
			}
		}

		// 2. Check API key header (remote agents + backward compat)
		if opID == "" {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}
			if apiKey != "" {
				op, err := operator.Authenticate(s.DB, s.EngID, apiKey)
				if err == nil {
					opID = op.ID
					role = op.Role
				}
			}
		}

		// 3. Localhost with no operators = solo mode (lead access, first-run)
		if opID == "" && s.isLocalhost() && !s.hasOperators() {
			opID = "local"
			role = "lead"
		}

		if opID == "" {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}

		s.touchActivity()

		if !operator.HasPermission(role, resource) {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, fmt.Sprintf(`{"error":"role '%s' cannot access '%s'"}`, role, resource), http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), operatorKey, opID)
		ctx = context.WithValue(ctx, operatorRoleKey, role)
		handler(w, r.WithContext(ctx))
	}
}

func clientIP2(r *http.Request) string {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}
