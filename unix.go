// +build darwin freebsd linux netbsd openbsd

package main

import (
	"os"
	"os/exec"
	"strings"
)

// IsTerminal returns true if f is a terminal.
func IsTerminal(f *os.File) bool {
	cmd := exec.Command("test", "-t", "0")
	cmd.Stdin = f
	return cmd.Run() == nil
}

func MakeRaw(f *os.File) error {
	return stty(f, "-icanon", "-echo").Run()
}

func RestoreTerm(f *os.File) error {
	return stty(f, "icanon", "echo").Run()
}

// helpers

func stty(f *os.File, args ...string) *exec.Cmd {
	c := exec.Command("stty", args...)
	c.Stdin = f
	return c
}

func tput(what string) (string, error) {
	c := exec.Command("tput", what)
	c.Stderr = os.Stderr
	out, err := c.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
