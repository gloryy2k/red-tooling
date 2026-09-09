package rules

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/user/rt/internal/config"
	"gopkg.in/yaml.v3"
)

type Rule struct {
	Name          string   `yaml:"name"`
	Patterns      []string `yaml:"patterns"`
	NotPatterns   []string `yaml:"not_patterns,omitempty"`
	MatchMode     string   `yaml:"match_mode,omitempty"` // "any" (default), "all"
	Tag           string   `yaml:"tag"`
	Priority      string   `yaml:"priority"`
	Mitre         string   `yaml:"mitre"`
	Confidence    int      `yaml:"confidence,omitempty"` // 0-100, default 80
	AutoCred      bool     `yaml:"auto_cred"`
	AutoMilestone bool     `yaml:"auto_milestone"`
	Regex         bool     `yaml:"regex,omitempty"` // true = patterns are real regex

	compiled    []*regexp.Regexp
	compiledNot []*regexp.Regexp
}

type RulesConfig struct {
	Rules []Rule `yaml:"rules"`
}

type Match struct {
	RuleName      string
	Tag           string
	Priority      string
	Mitre         string
	Confidence    int
	AutoCred      bool
	AutoMilestone bool
	Matched       string
}

var loaded *RulesConfig

func Load() (*RulesConfig, error) {
	if loaded != nil {
		return loaded, nil
	}

	path := filepath.Join(config.Home(), "rules.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		loaded = defaultRules()
		compile(loaded)
		return loaded, nil
	}

	var rc RulesConfig
	if err := yaml.Unmarshal(data, &rc); err != nil {
		return nil, err
	}

	compile(&rc)
	loaded = &rc
	return loaded, nil
}

func Reload() (*RulesConfig, error) {
	loaded = nil
	return Load()
}

func MatchOutput(output string) []Match {
	rc, err := Load()
	if err != nil || rc == nil {
		return nil
	}

	var matches []Match
	seen := make(map[string]bool)

	for _, rule := range rc.Rules {
		if seen[rule.Name] {
			continue
		}

		matched, matchedPat := evalRule(&rule, output)
		if !matched {
			continue
		}

		conf := rule.Confidence
		if conf == 0 {
			conf = 80
		}

		matches = append(matches, Match{
			RuleName:      rule.Name,
			Tag:           rule.Tag,
			Priority:      rule.Priority,
			Mitre:         rule.Mitre,
			Confidence:    conf,
			AutoCred:      rule.AutoCred,
			AutoMilestone: rule.AutoMilestone,
			Matched:       matchedPat,
		})
		seen[rule.Name] = true
	}

	return matches
}

func evalRule(rule *Rule, output string) (bool, string) {
	// Check not_patterns first — if any match, rule is excluded
	for _, re := range rule.compiledNot {
		if re.MatchString(output) {
			return false, ""
		}
	}

	if rule.MatchMode == "all" {
		// All patterns must match
		var lastMatch string
		for i, re := range rule.compiled {
			if !re.MatchString(output) {
				return false, ""
			}
			lastMatch = rule.Patterns[i]
		}
		return len(rule.compiled) > 0, lastMatch
	}

	// Default: any pattern matches
	for i, re := range rule.compiled {
		if re.MatchString(output) {
			return true, rule.Patterns[i]
		}
	}
	return false, ""
}

func HighestPriority(matches []Match, current string) string {
	order := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1, "info": 0, "": -1}
	best := current
	for _, m := range matches {
		if order[m.Priority] > order[best] {
			best = m.Priority
		}
	}
	return best
}

func CollectTags(matches []Match, existing []string) []string {
	set := make(map[string]bool)
	for _, t := range existing {
		set[t] = true
	}
	for _, m := range matches {
		if m.Tag != "" {
			set[m.Tag] = true
		}
		if m.Mitre != "" {
			set[m.Mitre] = true
		}
	}
	var out []string
	for t := range set {
		out = append(out, t)
	}
	return out
}

func compile(rc *RulesConfig) {
	for i := range rc.Rules {
		rc.Rules[i].compiled = compilePatterns(rc.Rules[i].Patterns, rc.Rules[i].Regex)
		rc.Rules[i].compiledNot = compilePatterns(rc.Rules[i].NotPatterns, rc.Rules[i].Regex)
	}
}

func compilePatterns(patterns []string, isRegex bool) []*regexp.Regexp {
	var out []*regexp.Regexp
	for _, pat := range patterns {
		var expr string
		if isRegex {
			expr = "(?i)" + pat
		} else {
			expr = "(?i)" + regexp.QuoteMeta(pat)
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			continue
		}
		out = append(out, re)
	}
	return out
}

func defaultRules() *RulesConfig {
	return &RulesConfig{
		Rules: []Rule{
			{
				Name:       "credential_found",
				Regex:      true,
				Patterns:   []string{`\bpassword\s*[:=]\s*\S+`, `NTLM\s*:\s*[a-fA-F0-9]{32}`, `\bhash\s*[:=]`, `\bTGT\b`, `\bkrbtgt\b`, `\[\+\]\s*Valid credentials`, `Successfully authenticated`},
				Tag:        "credential",
				Priority:   "high",
				Confidence: 90,
				Mitre:      "T1003",
				AutoCred:   true,
			},
			{
				Name:          "admin_access",
				Regex:         true,
				Patterns:      []string{`Domain\s+Admin`, `Enterprise\s+Admin`, `NT AUTHORITY\\SYSTEM`, `root@`, `\buid=0\b`, `\bAdministrator\b`},
				NotPatterns:   []string{`failed`, `denied`, `error.*admin`},
				Tag:           "privesc",
				Priority:      "critical",
				Confidence:    85,
				Mitre:         "T1078",
				AutoMilestone: true,
			},
			{
				Name:       "adcs_vuln",
				Regex:      true,
				Patterns:   []string{`ESC[1-9]\d*\b`, `Vulnerable.*Certificate`, `Certificate Template.*Enroll`},
				Tag:        "adcs",
				Priority:   "critical",
				Confidence: 90,
				Mitre:      "T1649",
			},
			{
				Name:       "sensitive_file",
				Regex:      true,
				Patterns:   []string{`\.kdbx\b`, `\bid_rsa\b`, `\bweb\.config\b`, `\.env\b`, `\b/etc/shadow\b`, `\bSAM\b`, `\bNTDS\.DIT\b`, `\bsecrets\.yml\b`, `\.pfx\b`, `\.p12\b`},
				Tag:        "loot",
				Priority:   "high",
				Confidence: 75,
				Mitre:      "T1005",
			},
			{
				Name:       "sqli_found",
				Regex:      true,
				Patterns:   []string{`SQL injection`, `sqlmap identified`, `\binjectable\b`, `UNION\s+SELECT`},
				Tag:        "sqli",
				Priority:   "high",
				Confidence: 90,
				Mitre:      "T1190",
			},
			{
				Name:       "rce_found",
				Regex:      true,
				Patterns:   []string{`remote code execution`, `\bRCE\b`, `command injection`, `\bOS command\b`},
				Tag:        "rce",
				Priority:   "critical",
				Confidence: 90,
				Mitre:      "T1190",
			},
			{
				Name:       "network_pivot",
				Regex:      true,
				Patterns:   []string{`\bpivot\b`, `tunnel\s+established`, `\bsocks[45]?\b`, `proxychains`, `port\s+forward`},
				Tag:        "lateral",
				Priority:   "medium",
				Confidence: 70,
				Mitre:      "T1090",
			},
			{
				Name:       "kerberoast",
				Regex:      true,
				Patterns:   []string{`Kerberoast`, `\$krb5tgs\$`, `GetUserSPNs`, `ServicePrincipalName`},
				Tag:        "kerberoast",
				Priority:   "high",
				Confidence: 95,
				Mitre:      "T1558.003",
			},
			{
				Name:       "asreproast",
				Regex:      true,
				Patterns:   []string{`AS-REP`, `\$krb5asrep\$`, `GetNPUsers`, `DONT_REQ_PREAUTH`},
				Tag:        "asreproast",
				Priority:   "high",
				Confidence: 95,
				Mitre:      "T1558.004",
			},
			{
				Name:       "bloodhound",
				Regex:      true,
				Patterns:   []string{`\bbloodhound\b`, `\bSharpHound\b`, `BloodHound\.py`},
				Tag:        "recon",
				Priority:   "medium",
				Confidence: 85,
				Mitre:      "T1087",
			},
			{
				Name:       "secretsdump",
				Regex:      true,
				Patterns:   []string{`secretsdump`, `DumpSecrets`, `SAM hashes`, `NTDS\.DIT`, `Dumping local SAM`, `Dumping LSA Secrets`},
				Tag:        "credential",
				Priority:   "critical",
				Confidence: 95,
				Mitre:      "T1003.003",
				AutoCred:   true,
			},
			{
				Name:       "mimikatz",
				Regex:      true,
				Patterns:   []string{`mimikatz`, `sekurlsa`, `logonpasswords`, `\bwdigest\b`, `kerberos\s+tickets`},
				Tag:        "credential",
				Priority:   "critical",
				Confidence: 95,
				Mitre:      "T1003.001",
				AutoCred:   true,
			},
			{
				Name:       "dcsync",
				Regex:      true,
				Patterns:   []string{`DCSync`, `DRS(GetNC|Replication)`, `lsadump::dcsync`},
				Tag:        "credential",
				Priority:   "critical",
				Confidence: 95,
				Mitre:      "T1003.006",
				AutoCred:   true,
			},
			{
				Name:       "golden_ticket",
				Regex:      true,
				Patterns:   []string{`golden\s*ticket`, `kerberos::golden`, `ticketer\.py.*-nthash`},
				Tag:        "persistence",
				Priority:   "critical",
				Confidence: 95,
				Mitre:      "T1558.001",
				AutoMilestone: true,
			},
			{
				Name:       "silver_ticket",
				Regex:      true,
				Patterns:   []string{`silver\s*ticket`, `kerberos::silver`},
				Tag:        "persistence",
				Priority:   "high",
				Confidence: 90,
				Mitre:      "T1558.002",
			},
			{
				Name:       "zerologon",
				Regex:      true,
				Patterns:   []string{`CVE-2020-1472`, `zerologon`, `Zerolog`},
				Tag:        "exploit",
				Priority:   "critical",
				Confidence: 95,
				Mitre:      "T1210",
			},
			{
				Name:       "printnightmare",
				Regex:      true,
				Patterns:   []string{`CVE-2021-3452[78]`, `PrintNightmare`, `nightmare\.py`},
				Tag:        "exploit",
				Priority:   "critical",
				Confidence: 95,
				Mitre:      "T1210",
			},
			{
				Name:       "coerce_attack",
				Regex:      true,
				Patterns:   []string{`PetitPotam`, `Coercer`, `DFSCoerce`, `PrinterBug`, `ShadowCoerce`},
				Tag:        "coerce",
				Priority:   "high",
				Confidence: 85,
				Mitre:      "T1187",
			},
			{
				Name:       "ntlm_relay",
				Regex:      true,
				Patterns:   []string{`ntlmrelayx`, `Relay.*succeeded`, `smbrelayx`},
				Tag:        "relay",
				Priority:   "high",
				Confidence: 90,
				Mitre:      "T1557.001",
			},
			{
				Name:       "lsass_dump",
				Regex:      true,
				Patterns:   []string{`lsass.*dump`, `procdump.*lsass`, `nanodump`, `MiniDump.*lsass`},
				Tag:        "credential",
				Priority:   "critical",
				Confidence: 90,
				Mitre:      "T1003.001",
				AutoCred:   true,
			},
			{
				Name:       "lateral_movement",
				Regex:      true,
				Patterns:   []string{`psexec`, `wmiexec`, `smbexec`, `atexec`, `dcomexec`, `evil-winrm`, `\bwinrs\b`},
				NotPatterns: []string{`failed`, `error`, `denied`},
				Tag:        "lateral",
				Priority:   "high",
				Confidence: 80,
				Mitre:      "T1021",
			},
			{
				Name:       "persistence_installed",
				Regex:      true,
				Patterns:   []string{`scheduled task.*created`, `service.*installed`, `registry.*Run\b.*added`, `\bweb\s*shell\b.*uploaded`},
				Tag:        "persistence",
				Priority:   "critical",
				Confidence: 85,
				Mitre:      "T1053",
				AutoMilestone: true,
			},
			{
				Name:       "data_exfil",
				Regex:      true,
				Patterns:   []string{`exfiltrat`, `data.*transfer.*complete`, `upload.*success`},
				NotPatterns: []string{`test`, `example`},
				Tag:        "exfil",
				Priority:   "high",
				Confidence: 70,
				Mitre:      "T1041",
			},
			{
				Name:       "password_spray_success",
				Regex:      true,
				MatchMode:  "all",
				Patterns:   []string{`spray`, `valid.*password|password.*valid|\[\+\]`},
				Tag:        "credential",
				Priority:   "high",
				Confidence: 85,
				Mitre:      "T1110.003",
				AutoCred:   true,
			},
		},
	}
}

func WriteDefault() error {
	path := filepath.Join(config.Home(), "rules.yml")
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	rc := defaultRules()
	data, err := yaml.Marshal(rc)
	if err != nil {
		return err
	}

	header := strings.Join([]string{
		"# RT Auto-Flag Rules",
		"# Each rule matches evidence output and applies tags/priority/MITRE.",
		"#",
		"# Fields:",
		"#   regex: true       — patterns are real regex (default: false = literal substring)",
		"#   match_mode: all   — ALL patterns must match (default: any = at least one)",
		"#   not_patterns: []  — if any match, rule is excluded (reduces false positives)",
		"#   confidence: 0-100 — how sure this detection is (default: 80)",
		"#   auto_cred: true   — auto-extract and store credentials",
		"#   auto_milestone: true — auto-create milestone on match",
		"",
		"",
	}, "\n")

	return os.WriteFile(path, []byte(header+string(data)), 0600)
}
