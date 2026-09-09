package capture

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/user/rt/internal/evidence"
)

// Session holds the state for an active capture session.
type Session struct {
	DB        *sql.DB
	SessionID string
	Operator  string

	mu       sync.Mutex
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	done     chan struct{}
	running  bool
}

// Start begins an interactive shell capture session.
func Start(db *sql.DB, sessionID, operator string) (*Session, error) {
	s := &Session{
		DB:        db,
		SessionID: sessionID,
		Operator:  operator,
		done:      make(chan struct{}),
	}

	shell := getShell()
	s.cmd = exec.Command(shell)
	s.cmd.Env = append(os.Environ(), "RT_SESSION="+sessionID)

	stdinPipe, err := s.cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	s.stdin = stdinPipe

	stdoutPipe, err := s.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	s.cmd.Stderr = s.cmd.Stdout

	if err := s.cmd.Start(); err != nil {
		return nil, fmt.Errorf("start shell: %w", err)
	}

	s.running = true
	go s.captureOutput(stdoutPipe)

	return s, nil
}

// RunCommand executes a single command and records evidence with auto-flag.
func RunCommand(db *sql.DB, sessionID, engID, operator, command, cwd string) (*evidence.Evidence, *evidence.AutoFlagResult, error) {
	shell := getShell()
	var args []string
	if runtime.GOOS == "windows" {
		args = []string{"/C", command}
	} else {
		args = []string{"-c", command}
	}

	cmd := exec.Command(shell, args...)
	cmd.Dir = cwd

	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start).Milliseconds()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	outputStr := string(output)
	if len(outputStr) > 1024*1024 {
		outputStr = outputStr[:1024*1024] + "\n[output truncated at 1MB]"
	}

	ev, insertErr := evidence.Insert(db, sessionID, "command", command, outputStr, exitCode, int(duration), cwd, nil, "", operator)
	if insertErr != nil {
		return nil, nil, fmt.Errorf("record evidence: %w", insertErr)
	}

	// Auto-flag
	flagResult, _ := evidence.AutoFlag(db, ev, engID, operator)

	return ev, flagResult, nil
}

// ExecInteractive runs a command interactively, capturing stdin/stdout while passing through to terminal.
func ExecInteractive(db *sql.DB, sessionID, engID, operator, command string) error {
	shell := getShell()
	var args []string
	if runtime.GOOS == "windows" {
		args = []string{"/C", command}
	} else {
		args = []string{"-c", command}
	}

	cwd, _ := os.Getwd()
	cmd := exec.Command(shell, args...)
	cmd.Dir = cwd

	var outputBuf strings.Builder
	cmd.Stdout = io.MultiWriter(os.Stdout, &outputBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &outputBuf)
	cmd.Stdin = os.Stdin

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start).Milliseconds()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	outputStr := outputBuf.String()
	if len(outputStr) > 1024*1024 {
		outputStr = outputStr[:1024*1024] + "\n[output truncated at 1MB]"
	}

	ev, _ := evidence.Insert(db, sessionID, "command", command, outputStr, exitCode, int(duration), cwd, nil, "", operator)
	if ev != nil {
		flagResult, _ := evidence.AutoFlag(db, ev, engID, operator)
		evidence.PrintAutoFlagResult(flagResult)
	}

	return nil // don't propagate command exit code as error
}

func (s *Session) captureOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}
}

func (s *Session) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	s.running = false
	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	return nil
}

func getShell() string {
	if runtime.GOOS == "windows" {
		if ps, _ := exec.LookPath("powershell.exe"); ps != "" {
			return ps
		}
		return "cmd.exe"
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	return "/bin/sh"
}
