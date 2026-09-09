package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

var (
	stdinReader     *bufio.Reader
	stdinReaderOnce sync.Once
)

func getStdinReader() *bufio.Reader {
	stdinReaderOnce.Do(func() {
		stdinReader = bufio.NewReader(os.Stdin)
	})
	return stdinReader
}

// readPassphrase reads a passphrase from terminal (hidden) or stdin (piped).
func readPassphrase(prompt string) ([]byte, error) {
	fmt.Print(prompt)

	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		pass, err := term.ReadPassword(fd)
		fmt.Println()
		return pass, err
	}

	reader := getStdinReader()
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return nil, err
	}
	return []byte(strings.TrimRight(line, "\r\n")), nil
}
