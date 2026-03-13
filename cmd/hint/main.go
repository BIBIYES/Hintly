package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"hint/internal/agent"
	"hint/internal/ai"
	"hint/internal/config"
	"hint/internal/executor"
	"hint/internal/ui"
	"hint/pkg/sysinfo"
)

func main() {
	initConfig := flag.Bool("init", false, "初始化配置")
	flag.Parse()

	if *initConfig {
		if err := config.InitInteractive(os.Stdin, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		path, _ := config.ConfigPath()
		fmt.Fprintf(os.Stderr, "读取配置失败: %v\n请先执行: hint -init\n配置路径: %s\n", err, path)
		os.Exit(1)
	}

	env := sysinfo.Detect()
	client := ai.NewClient(cfg)

	query := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if query == "" {
		if err := agent.Run(os.Stdin, os.Stdout, client, env); err != nil {
			fmt.Fprintf(os.Stderr, "Agent 模式运行失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	result, err := ui.Run(query, client, env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "UI 运行失败: %v\n", err)
		os.Exit(1)
	}

	if result.Cancelled {
		// Clear screen for a clean cancel exit.
		fmt.Print("\033[2J\033[H")
		return
	}

	if result.Execute {
		fmt.Printf("\n$ %s\n", result.Command)
		if err := executor.Run(result.Command); err != nil {
			fmt.Fprintf(os.Stderr, "\n命令执行失败: %v\n", err)
			os.Exit(1)
		}
	}
}
