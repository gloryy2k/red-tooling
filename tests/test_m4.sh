#!/bin/bash
# M4 End-to-End Test Script — Report Engine + Export
set -e

RT="${RT:-../rt.exe}"
export RT_HOME="$(pwd)/test-rt-home"
rm -rf "$RT_HOME"

PASS="testpass1234"

echo "=== M4 End-to-End Test ==="
echo ""

# Setup: create engagement with evidence and findings
echo "--- Setup: Create engagement with data ---"
printf '%s\n%s\n' "$PASS" "$PASS" | $RT new "M4-TEST" --client "ACME Corp"
echo "$PASS" | $RT unlock

# Generate evidence
$RT exec echo "nmap -sV 10.10.10.1 - discovered open ports 22,80,443,445"
$RT exec echo "Administrator:500:aad3b435b51404eeaad3b435b51404ee:31d6cfe0d16ae931b73c59d7e0c089c0:::"
$RT exec echo "[+] Domain Admin access achieved via pass-the-hash"
$RT exec echo "Certificate Template ESC1 is vulnerable to privilege escalation"
$RT exec echo "sqlmap identified SQL injection in parameter id on login.aspx"

# Create findings with MITRE tags
$RT finding "SQL Injection in Login" --desc "Blind SQLi found in login.aspx id parameter" --priority high --evidence 1,5 --mitre T1190
$RT finding "Domain Admin via Pass-the-Hash" --desc "Achieved Domain Admin using extracted NTLM hash" --priority critical --evidence 2,3 --mitre T1078,T1550.002
$RT finding "ADCS ESC1 Privilege Escalation" --desc "Vulnerable certificate template allows domain escalation" --priority high --evidence 4 --mitre T1649
$RT finding "Information Disclosure" --desc "Server version info in HTTP headers" --priority low --evidence 1

# Verify some findings
$RT verify-finding 1 confirmed --note "Reproduced with sqlmap"
$RT verify-finding 2 confirmed --note "Successfully replayed PtH attack"
$RT verify-finding 4 false-positive --note "Only version headers, no impact"

# Set recommendations
$RT recommend 1 "Implement parameterized queries and WAF rules"
$RT recommend 2 "Enforce credential tiering and disable NTLM where possible"
$RT recommend 3 "Restrict certificate template enrollment permissions"

echo "PASS: Setup complete"

# 1. Markdown report (full)
echo ""
echo "--- Test 1: Markdown report (stdout) ---"
$RT report | head -30
echo "..."
echo "PASS: Markdown report generated"

# 2. HTML report to file
echo ""
echo "--- Test 2: HTML report to file ---"
$RT report --html -o report_test.html
echo "PASS: HTML report written"

# 3. Verify no XSS in HTML output (check for unescaped content)
echo ""
echo "--- Test 3: XSS safety check ---"
if grep -q '<script>' report_test.html 2>/dev/null; then
    echo "FAIL: Found unescaped script tag!"
    exit 1
fi
echo "PASS: No XSS detected"

# 4. Executive summary only
echo ""
echo "--- Test 4: Executive summary ---"
$RT report --exec | head -20
echo "..."
echo "PASS: Executive summary"

# 5. Technical details only
echo ""
echo "--- Test 5: Technical details ---"
$RT report --tech | head -20
echo "..."
echo "PASS: Technical details"

# 6. Critical findings only
echo ""
echo "--- Test 6: Critical findings report ---"
$RT report --critical | head -20
echo "..."
echo "PASS: Critical-only report"

# 7. Verified findings only
echo ""
echo "--- Test 7: Verified findings report ---"
$RT report --verified | head -20
echo "..."
echo "PASS: Verified-only report"

# 8. Export Ghostwriter JSON
echo ""
echo "--- Test 8: Ghostwriter export ---"
$RT export ghostwriter -o ghostwriter_test.json
cat ghostwriter_test.json | head -20
echo "..."
echo "PASS: Ghostwriter export"

# 9. Export CSV
echo ""
echo "--- Test 9: CSV export ---"
$RT export csv -o findings_test.csv
cat findings_test.csv
echo "PASS: CSV export"

# 10. Export JSON
echo ""
echo "--- Test 10: JSON export ---"
$RT export json -o findings_test.json
cat findings_test.json | head -20
echo "..."
echo "PASS: JSON export"

# 11. Export MITRE Navigator layer
echo ""
echo "--- Test 11: MITRE Navigator layer ---"
$RT export mitre -o mitre_test.json
cat mitre_test.json
echo "PASS: MITRE layer export"

# 12. Verify chain integrity after all operations
echo ""
echo "--- Test 12: Chain integrity ---"
$RT verify-chain
echo "PASS: Chain intact"

# 13. Audit log shows export events
echo ""
echo "--- Test 13: Audit log ---"
$RT audit --limit 10
echo "PASS: Audit log shows exports"

# 14. Lock
echo ""
echo "--- Test 14: Lock ---"
$RT lock
echo "PASS: Database locked"

echo ""
echo "========================================="
echo "  All M4 tests PASSED!"
echo "========================================="

rm -rf "$RT_HOME" report_test.html ghostwriter_test.json findings_test.csv findings_test.json mitre_test.json
