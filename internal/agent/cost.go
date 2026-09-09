package agent

import (
	"database/sql"
	"fmt"
	"time"
)

type CostEntry struct {
	ID           int64
	EngagementID string
	SessionID    string
	Model        string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	CreatedAt    string
}

type CostSummary struct {
	TotalCostUSD  float64
	TotalInput    int
	TotalOutput   int
	ByModel       map[string]ModelCost
	EntryCount    int
}

type ModelCost struct {
	Model        string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	Calls        int
}

// Known model pricing (USD per 1K tokens) as of 2026
var modelPricing = map[string][2]float64{
	"claude-sonnet-5":   {0.003, 0.015},
	"claude-opus-5":     {0.015, 0.075},
	"claude-haiku-4-5":  {0.0008, 0.004},
	"gpt-4o":            {0.005, 0.015},
	"gpt-4o-mini":       {0.00015, 0.0006},
}

// EnsureCostTable creates the cost tracking table if it doesn't exist.
func EnsureCostTable(db *sql.DB) {
	db.Exec(`CREATE TABLE IF NOT EXISTS agent_cost (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		engagement_id TEXT NOT NULL,
		session_id TEXT,
		model TEXT NOT NULL,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		cost_usd REAL DEFAULT 0,
		created_at TEXT DEFAULT (datetime('now'))
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_cost_eng ON agent_cost(engagement_id)`)
}

// TrackCost records a token usage entry.
func TrackCost(db *sql.DB, engID, sessionID, model string, inputTokens, outputTokens int) error {
	EnsureCostTable(db)

	costUSD := calculateCost(model, inputTokens, outputTokens)

	_, err := db.Exec(
		`INSERT INTO agent_cost (engagement_id, session_id, model, input_tokens, output_tokens, cost_usd, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		engID, sessionID, model, inputTokens, outputTokens, costUSD,
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// GetSummary returns cost summary for an engagement.
func GetSummary(db *sql.DB, engID string) (*CostSummary, error) {
	EnsureCostTable(db)

	rows, err := db.Query(
		`SELECT model, SUM(input_tokens), SUM(output_tokens), SUM(cost_usd), COUNT(*)
		 FROM agent_cost WHERE engagement_id = ? GROUP BY model`, engID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := &CostSummary{
		ByModel: make(map[string]ModelCost),
	}

	for rows.Next() {
		var mc ModelCost
		if err := rows.Scan(&mc.Model, &mc.InputTokens, &mc.OutputTokens, &mc.CostUSD, &mc.Calls); err != nil {
			return nil, err
		}
		summary.ByModel[mc.Model] = mc
		summary.TotalInput += mc.InputTokens
		summary.TotalOutput += mc.OutputTokens
		summary.TotalCostUSD += mc.CostUSD
		summary.EntryCount += mc.Calls
	}

	return summary, rows.Err()
}

// GetEntries returns recent cost entries.
func GetEntries(db *sql.DB, engID string, limit int) ([]CostEntry, error) {
	EnsureCostTable(db)

	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Query(
		`SELECT id, engagement_id, COALESCE(session_id,''), model, input_tokens, output_tokens, cost_usd, created_at
		 FROM agent_cost WHERE engagement_id = ? ORDER BY created_at DESC LIMIT ?`, engID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []CostEntry
	for rows.Next() {
		var e CostEntry
		if err := rows.Scan(&e.ID, &e.EngagementID, &e.SessionID, &e.Model,
			&e.InputTokens, &e.OutputTokens, &e.CostUSD, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func calculateCost(model string, inputTokens, outputTokens int) float64 {
	pricing, ok := modelPricing[model]
	if !ok {
		return 0
	}
	inputCost := float64(inputTokens) / 1000.0 * pricing[0]
	outputCost := float64(outputTokens) / 1000.0 * pricing[1]
	return inputCost + outputCost
}

// FormatCost formats USD amount.
func FormatCost(usd float64) string {
	if usd < 0.01 {
		return fmt.Sprintf("$%.4f", usd)
	}
	return fmt.Sprintf("$%.2f", usd)
}

// FormatTokens formats token count with K/M suffix.
func FormatTokens(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
