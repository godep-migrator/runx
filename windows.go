// +build windows

package main

import (
	"os"
)

// IsTerminal returns false on Windows.
func IsTerminal(f *os.File) bool {
	return false
}

// MakeRaw is a no-op on windows. It returns nil.
func MakeRaw(f *os.File) error {
	return nil
}

// RestoreTerm is a no-op on windows. It returns nil.
func RestoreTerm(f *os.File) error {
	return nil
}
