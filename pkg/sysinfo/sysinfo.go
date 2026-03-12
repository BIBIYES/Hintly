package sysinfo

import (
	"bufio"
	"os"
	"runtime"
	"strings"
)

// Env is injected into AI prompts so commands match runtime context.
type Env struct {
	GOOS   string
	Distro string
	Shell  string
	PWD    string
}

func Detect() Env {
	cwd, _ := os.Getwd()
	return Env{
		GOOS:   runtime.GOOS,
		Distro: distro(),
		Shell:  shell(),
		PWD:    cwd,
	}
}

func shell() string {
	if s := strings.TrimSpace(os.Getenv("SHELL")); s != "" {
		return s
	}
	if c := strings.TrimSpace(os.Getenv("ComSpec")); c != "" {
		return c
	}
	return "unknown"
}

func distro() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		f, err := os.Open("/etc/os-release")
		if err != nil {
			return "Linux"
		}
		defer f.Close()

		s := bufio.NewScanner(f)
		for s.Scan() {
			line := s.Text()
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
			}
		}
		return "Linux"
	default:
		return runtime.GOOS
	}
}
