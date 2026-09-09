package importer

import (
	"testing"
)

func TestMapNucleiSeverity(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"critical", "critical"},
		{"CRITICAL", "critical"},
		{"high", "high"},
		{"High", "high"},
		{"medium", "medium"},
		{"low", "low"},
		{"info", "info"},
		{"unknown", "info"},
		{"", "info"},
	}

	for _, tt := range tests {
		got := mapNucleiSeverity(tt.input)
		if got != tt.want {
			t.Errorf("mapNucleiSeverity(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
