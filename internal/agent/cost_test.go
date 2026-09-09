package agent

import (
	"math"
	"testing"
)

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		model  string
		input  int
		output int
		want   float64
	}{
		{"claude-sonnet-5", 1000, 1000, 0.003 + 0.015},
		{"claude-opus-5", 1000, 1000, 0.015 + 0.075},
		{"gpt-4o-mini", 10000, 5000, 0.0015 + 0.003},
		{"unknown-model", 1000, 1000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := calculateCost(tt.model, tt.input, tt.output)
			if math.Abs(got-tt.want) > 0.0001 {
				t.Errorf("calculateCost(%q, %d, %d) = %f, want %f", tt.model, tt.input, tt.output, got, tt.want)
			}
		})
	}
}

func TestFormatCost(t *testing.T) {
	if got := FormatCost(1.50); got != "$1.50" {
		t.Errorf("FormatCost(1.50) = %q", got)
	}
	if got := FormatCost(0.001); got != "$0.0010" {
		t.Errorf("FormatCost(0.001) = %q", got)
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{500, "500"},
		{1500, "1.5K"},
		{1500000, "1.5M"},
	}
	for _, tt := range tests {
		got := FormatTokens(tt.n)
		if got != tt.want {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
