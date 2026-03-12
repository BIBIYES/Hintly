package executor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

var dangerousPatterns = []string{
	"rm -rf /",
	"mkfs",
	"> /dev/sda",
	">/dev/sda",
	"dd if=",
	"shutdown -h now",
	"reboot",
}

// IsDangerous performs a conservative blacklist check.
func IsDangerous(command string) bool {
	cmd := strings.ToLower(strings.TrimSpace(command))
	for _, p := range dangerousPatterns {
		if strings.Contains(cmd, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// Run executes command in current shell and streams stdout/stderr in real-time.
func Run(command string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("empty command")
	}

	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.Command("cmd", "/C", command)
	} else {
		shell := os.Getenv("SHELL")
		if strings.TrimSpace(shell) == "" {
			shell = "/bin/sh"
		}
		c = exec.Command(shell, "-lc", command)
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}
