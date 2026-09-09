package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/rt/internal/credentials"
)

var credCmd = &cobra.Command{
	Use:   "cred <username> <secret> [--host HOST] [--type TYPE]",
	Short: "Store a credential",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, engID, err := getActiveSessionDB()
		if err != nil {
			return err
		}

		username := args[0]
		secret := args[1]
		host, _ := cmd.Flags().GetString("host")
		secretType, _ := cmd.Flags().GetString("type")
		if secretType == "" {
			secretType = detectSecretType(secret)
		}

		operator := getOperator()
		if err := credentials.Store(database, engID, username, secret, secretType, host, operator, 0); err != nil {
			return err
		}

		masked := maskCred(secret)
		fmt.Printf("  Credential stored: %s : %s (%s)", username, masked, secretType)
		if host != "" {
			fmt.Printf(" @ %s", host)
		}
		fmt.Println()
		return nil
	},
}

var credsCmd = &cobra.Command{
	Use:   "creds",
	Short: "List stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, engName, err := requireDB()
		if err != nil {
			return err
		}

		engID := strings.ToLower(strings.ReplaceAll(engName, " ", "-"))
		show, _ := cmd.Flags().GetBool("show")
		csv, _ := cmd.Flags().GetBool("csv")

		if csv {
			operator := getOperator()
			csvData, err := credentials.ExportCSV(database, engID, operator)
			if err != nil {
				return err
			}
			fmt.Print(csvData)
			return nil
		}

		if show {
			operator := getOperator()
			creds, err := credentials.RevealAll(database, engID, operator)
			if err != nil {
				return err
			}
			if len(creds) == 0 {
				fmt.Println("  No credentials stored.")
				return nil
			}

			fmt.Println()
			fmt.Printf("  %-4s %-30s %-40s %-12s %-20s\n", "ID", "USERNAME", "SECRET", "TYPE", "HOST")
			fmt.Printf("  %-4s %-30s %-40s %-12s %-20s\n",
				strings.Repeat("-", 4), strings.Repeat("-", 30), strings.Repeat("-", 40),
				strings.Repeat("-", 12), strings.Repeat("-", 20))
			for _, c := range creds {
				secret := c.Secret
				if len(secret) > 38 {
					secret = secret[:35] + "..."
				}
				fmt.Printf("  %-4d %-30s %-40s %-12s %-20s\n",
					c.ID, truncate(c.Username, 30), secret, c.SecretType, truncate(c.Host, 20))
			}
			fmt.Println()
			fmt.Printf("  \033[33m[!] Credential view audit-logged\033[0m\n\n")
			return nil
		}

		// Masked view (default)
		creds, err := credentials.List(database, engID)
		if err != nil {
			return err
		}
		if len(creds) == 0 {
			fmt.Println("  No credentials stored.")
			return nil
		}

		fmt.Println()
		fmt.Printf("  %-4s %-30s %-20s %-12s %-20s\n", "ID", "USERNAME", "SECRET", "TYPE", "HOST")
		fmt.Printf("  %-4s %-30s %-20s %-12s %-20s\n",
			strings.Repeat("-", 4), strings.Repeat("-", 30), strings.Repeat("-", 20),
			strings.Repeat("-", 12), strings.Repeat("-", 20))
		for _, c := range creds {
			fmt.Printf("  %-4d %-30s %-20s %-12s %-20s\n",
				c.ID, truncate(c.Username, 30), "********", c.CredType, truncate(c.Host, 20))
		}
		fmt.Println()
		fmt.Printf("  %d credential(s). Use 'rt creds --show' to reveal.\n\n", len(creds))
		return nil
	},
}

func init() {
	credCmd.Flags().String("host", "", "Target host/IP")
	credCmd.Flags().String("type", "", "Secret type (password, ntlm, hash, kerberos_tgs)")
	credCmd.Flags().String("ntlm", "", "NTLM hash (alternative to positional secret)")

	credsCmd.Flags().Bool("show", false, "Show plaintext secrets (audit-logged)")
	credsCmd.Flags().Bool("csv", false, "Export as CSV (audit-logged)")
}

func detectSecretType(secret string) string {
	if len(secret) == 65 && secret[32] == ':' {
		return "ntlm"
	}
	if strings.HasPrefix(secret, "$krb5tgs$") {
		return "kerberos_tgs"
	}
	if strings.HasPrefix(secret, "$krb5asrep$") {
		return "kerberos_asrep"
	}
	if len(secret) == 32 && isHex(secret) {
		return "hash"
	}
	return "password"
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func maskCred(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
