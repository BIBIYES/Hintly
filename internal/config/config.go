package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	appName    = "hint"
	configFile = "config.yaml"
)

// Config stores runtime settings for AI requests.
type Config struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return errors.New("base_url is required")
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return errors.New("api_key is required")
	}
	if strings.TrimSpace(c.Model) == "" {
		return errors.New("model is required")
	}
	return nil
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}
	return filepath.Join(home, ".config", appName, configFile), nil
}

func Load() (Config, error) {
	var cfg Config

	path, err := ConfigPath()
	if err != nil {
		return cfg, err
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, fmt.Errorf("config not found at %s", path)
		}
		return cfg, fmt.Errorf("stat config: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetDefault("model", "gpt-4o")
	if err := v.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Save(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	v := viper.New()
	v.Set("base_url", strings.TrimSpace(cfg.BaseURL))
	v.Set("api_key", strings.TrimSpace(cfg.APIKey))
	v.Set("model", strings.TrimSpace(cfg.Model))
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return os.Chmod(path, 0o600)
}

func InitInteractive(in io.Reader, out io.Writer) error {
	s := bufio.NewScanner(in)
	fmt.Fprintln(out, "初始化 hint 配置")

	baseURL := prompt(s, out, "BaseURL", "https://api.openai.com/v1")
	apiKey := prompt(s, out, "API Key", "")
	model := prompt(s, out, "Model Name", "gpt-4o")

	cfg := Config{BaseURL: baseURL, APIKey: apiKey, Model: model}
	if err := Save(cfg); err != nil {
		return err
	}

	path, _ := ConfigPath()
	fmt.Fprintf(out, "配置已保存: %s\n", path)
	return nil
}

func prompt(scanner *bufio.Scanner, out io.Writer, key, fallback string) string {
	if fallback == "" {
		fmt.Fprintf(out, "%s: ", key)
	} else {
		fmt.Fprintf(out, "%s (默认 %s): ", key, fallback)
	}
	if !scanner.Scan() {
		return fallback
	}
	v := strings.TrimSpace(scanner.Text())
	if v == "" {
		return fallback
	}
	return v
}
