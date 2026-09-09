package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/rt/internal/config"
	"gopkg.in/yaml.v3"
)

type Playbook struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Timeout     int      `yaml:"timeout"`
	Inputs      []Input  `yaml:"inputs"`
	Steps       []Step   `yaml:"steps"`
}

type Input struct {
	Name     string `yaml:"name"`
	Required bool   `yaml:"required"`
	Default  string `yaml:"default"`
}

type Step struct {
	Name      string   `yaml:"name"`
	Cmd       string   `yaml:"cmd"`
	Tags      []string `yaml:"tags"`
	Timeout   int      `yaml:"timeout"`
	DependsOn string   `yaml:"depends_on"`
	Condition string   `yaml:"condition"`
	Parser    string   `yaml:"parser"`
	OnFail    string   `yaml:"on_fail"` // continue, stop (default: stop)
}

// LoadPlaybook reads a playbook from the playbooks directory or an absolute path.
func LoadPlaybook(nameOrPath string) (*Playbook, error) {
	var path string

	if filepath.IsAbs(nameOrPath) || strings.Contains(nameOrPath, string(os.PathSeparator)) || strings.Contains(nameOrPath, "/") {
		path = nameOrPath
	} else {
		path = filepath.Join(config.Home(), "playbooks", nameOrPath+".yml")
		if _, err := os.Stat(path); err != nil {
			path = filepath.Join(config.Home(), "playbooks", nameOrPath+".yaml")
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load playbook %q: %w", nameOrPath, err)
	}

	var pb Playbook
	if err := yaml.Unmarshal(data, &pb); err != nil {
		return nil, fmt.Errorf("parse playbook: %w", err)
	}

	if pb.Name == "" {
		pb.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	return &pb, nil
}

// ResolveInputs substitutes {input_name} placeholders in step commands.
func ResolveInputs(pb *Playbook, vars map[string]string) error {
	for _, inp := range pb.Inputs {
		if _, ok := vars[inp.Name]; !ok {
			if inp.Required {
				return fmt.Errorf("required input %q not provided", inp.Name)
			}
			if inp.Default != "" {
				vars[inp.Name] = inp.Default
			}
		}
	}

	for i := range pb.Steps {
		pb.Steps[i].Cmd = substituteVars(pb.Steps[i].Cmd, vars)
		pb.Steps[i].Condition = substituteVars(pb.Steps[i].Condition, vars)
	}

	return nil
}

func substituteVars(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// EvalCondition evaluates simple conditions like "domain != ''" or "target != ''".
func EvalCondition(cond string) bool {
	if cond == "" {
		return true
	}

	cond = strings.TrimSpace(cond)

	if strings.Contains(cond, "!=") {
		parts := strings.SplitN(cond, "!=", 2)
		left := strings.TrimSpace(strings.Trim(parts[0], "'\""))
		right := strings.TrimSpace(strings.Trim(parts[1], "'\""))
		return left != right
	}

	if strings.Contains(cond, "==") {
		parts := strings.SplitN(cond, "==", 2)
		left := strings.TrimSpace(strings.Trim(parts[0], "'\""))
		right := strings.TrimSpace(strings.Trim(parts[1], "'\""))
		return left == right
	}

	return cond != "" && cond != "false" && cond != "0"
}

// ListPlaybooks returns available playbooks from the playbooks directory.
func ListPlaybooks() ([]string, error) {
	dir := filepath.Join(config.Home(), "playbooks")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext == ".yml" || ext == ".yaml" {
			names = append(names, strings.TrimSuffix(e.Name(), ext))
		}
	}
	return names, nil
}
