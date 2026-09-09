package main

import (
	"bufio"
	"os"
)

type lineScanner struct {
	scanner *bufio.Scanner
}

func newLineScanner() *lineScanner {
	return &lineScanner{scanner: bufio.NewScanner(os.Stdin)}
}

func (ls *lineScanner) ReadLine() (string, bool) {
	if ls.scanner.Scan() {
		return ls.scanner.Text(), true
	}
	return "", false
}
