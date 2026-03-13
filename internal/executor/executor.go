package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
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

	c := commandForOS(command)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}

// RunCapture executes command and returns captured output and exit code.
func RunCapture(command string, timeout time.Duration) (string, int, error) {
	if strings.TrimSpace(command) == "" {
		return "", -1, fmt.Errorf("empty command")
	}
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	c := commandForOS(command)
	c = exec.CommandContext(ctx, c.Path, c.Args[1:]...)
	c.Env = os.Environ()
	out, err := c.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return strings.TrimSpace(string(out)), -1, fmt.Errorf("command timed out after %s", timeout)
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return strings.TrimSpace(string(out)), exitErr.ExitCode(), nil
		}
		return strings.TrimSpace(string(out)), -1, err
	}
	return strings.TrimSpace(string(out)), 0, nil
}

func commandForOS(command string) *exec.Cmd {
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
	return c
}
