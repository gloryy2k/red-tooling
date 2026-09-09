package scope

import (
	"testing"
)

func TestParseHosts(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"10.0.0.1", 1},
		{"10.0.0.1,10.0.0.2,10.0.0.3", 3},
		{"10.0.0.1, 10.0.0.2 , 10.0.0.3", 3},
		{" , , ", 0},
		{"192.168.1.0/24,10.10.10.0/24", 2},
		{"", 0},
	}

	for _, tt := range tests {
		hosts := parseHosts(tt.input)
		if len(hosts) != tt.want {
			t.Errorf("parseHosts(%q) = %d hosts, want %d", tt.input, len(hosts), tt.want)
		}
	}
}
