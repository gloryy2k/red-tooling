package credentials

import (
	"regexp"
	"strings"
)

type ParsedCred struct {
	Username   string
	Secret     string
	SecretType string
	Host       string
}

var credPatterns = []struct {
	name       string
	re         *regexp.Regexp
	secretType string
	extract    func([]string) *ParsedCred
}{
	// secretsdump NTLM: Administrator:500:aad3b435b51404eeaad3b435b51404ee:31d6cfe0d16ae931b73c59d7e0c089c0:::
	{
		name:       "ntlm_hash",
		re:         regexp.MustCompile(`(?m)^([^:\s]+):(\d+):([a-fA-F0-9]{32}):([a-fA-F0-9]{32}):::`),
		secretType: "ntlm",
		extract: func(m []string) *ParsedCred {
			return &ParsedCred{
				Username:   m[1],
				Secret:     m[3] + ":" + m[4],
				SecretType: "ntlm",
			}
		},
	},
	// secretsdump with domain: DOMAIN\user:RID:LM:NTLM:::
	{
		name:       "ntlm_domain",
		re:         regexp.MustCompile(`(?m)^([A-Za-z0-9._-]+\\[A-Za-z0-9._-]+):(\d+):([a-fA-F0-9]{32}):([a-fA-F0-9]{32}):::`),
		secretType: "ntlm",
		extract: func(m []string) *ParsedCred {
			return &ParsedCred{
				Username:   m[1],
				Secret:     m[3] + ":" + m[4],
				SecretType: "ntlm",
			}
		},
	},
	// NetNTLMv2: user::DOMAIN:challenge:response
	{
		name:       "netntlmv2",
		re:         regexp.MustCompile(`(?m)([A-Za-z0-9._-]+)::([A-Za-z0-9._-]+):([a-fA-F0-9]+):([a-fA-F0-9]+):`),
		secretType: "netntlmv2",
		extract: func(m []string) *ParsedCred {
			return &ParsedCred{
				Username:   m[2] + "\\" + m[1],
				Secret:     m[0],
				SecretType: "netntlmv2",
			}
		},
	},
	// Kerberoast: $krb5tgs$23$*user$DOMAIN$...
	{
		name:       "kerberoast",
		re:         regexp.MustCompile(`(\$krb5tgs\$\d+\$\*([^$*]+)\$([^$*]+)\$.+)`),
		secretType: "kerberos_tgs",
		extract: func(m []string) *ParsedCred {
			return &ParsedCred{
				Username:   m[3] + "\\" + m[2],
				Secret:     m[1],
				SecretType: "kerberos_tgs",
			}
		},
	},
	// AS-REP: $krb5asrep$23$user@DOMAIN:...
	{
		name:       "asreproast",
		re:         regexp.MustCompile(`(\$krb5asrep\$\d+\$([^@:]+)@([^:]+):.+)`),
		secretType: "kerberos_asrep",
		extract: func(m []string) *ParsedCred {
			return &ParsedCred{
				Username:   m[3] + "\\" + m[2],
				Secret:     m[1],
				SecretType: "kerberos_asrep",
			}
		},
	},
	// Cleartext: [+] user:password  or  user : password
	{
		name:       "cleartext_cred",
		re:         regexp.MustCompile(`(?m)\[?\+\]?\s+(?:Valid credentials?:?\s+)?([A-Za-z0-9._\\@-]+)\s*:\s*([^\s]+)`),
		secretType: "password",
		extract: func(m []string) *ParsedCred {
			return &ParsedCred{
				Username:   m[1],
				Secret:     m[2],
				SecretType: "password",
			}
		},
	},
	// mimikatz wdigest: Username : admin  Password : P@ssw0rd
	{
		name:       "mimikatz_wdigest",
		re:         regexp.MustCompile(`(?m)Username\s*:\s*([^\s]+)\s*\n.*?Password\s*:\s*([^\s]+)`),
		secretType: "password",
		extract: func(m []string) *ParsedCred {
			if m[2] == "(null)" {
				return nil
			}
			return &ParsedCred{
				Username:   m[1],
				Secret:     m[2],
				SecretType: "password",
			}
		},
	},
}

var falsePositiveUsernames = map[string]bool{
	"categoryinfo":          true,
	"fullyqualifiederrorid": true,
	"targetobject":          true,
	"errordetails":          true,
	"pscomputername":        true,
	"erroraction":           true,
	"warningaction":         true,
	"informationaction":     true,
	"errorvariable":         true,
	"warningvariable":       true,
	"informationvariable":   true,
	"outbuffer":             true,
	"outvariable":           true,
	"pipelinevariable":      true,
}

// ParseCredentials extracts credentials from command output.
func ParseCredentials(output, host string) []ParsedCred {
	var results []ParsedCred
	seen := make(map[string]bool)

	for _, pat := range credPatterns {
		matches := pat.re.FindAllStringSubmatch(output, -1)
		for _, m := range matches {
			cred := pat.extract(m)
			if cred == nil {
				continue
			}
			if cred.Host == "" {
				cred.Host = host
			}

			key := strings.ToLower(cred.Username + "|" + cred.SecretType + "|" + cred.Host)
			if seen[key] {
				continue
			}
			seen[key] = true

			// Skip machine accounts for NTLM (ending in $)
			if cred.SecretType == "ntlm" && strings.HasSuffix(cred.Username, "$") {
				continue
			}
			// Skip empty/null secrets
			if cred.Secret == "" || cred.Secret == "(null)" {
				continue
			}
			// Skip PowerShell error property names parsed as usernames
			if falsePositiveUsernames[strings.ToLower(cred.Username)] {
				continue
			}

			results = append(results, *cred)
		}
	}

	return results
}
