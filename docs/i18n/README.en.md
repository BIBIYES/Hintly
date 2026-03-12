<p align="center">
  <img src="../../assets/logo.png" width="180" alt="Hintly logo">
</p>

<h1 align="center">Hintly</h1>

<p align="center">An AI terminal assistant that turns natural language into executable commands</p>

<p align="center">
  <a href="../../README.md">简体中文</a> | <strong>English</strong> | <a href="README.ja.md">日本語</a> | <a href="README.ko.md">한국어</a> | <a href="README.es.md">Español</a> | <a href="README.fr.md">Français</a> | <a href="README.de.md">Deutsch</a> | <a href="README.ru.md">Русский</a> | <a href="README.pt-br.md">Português (BR)</a>
</p>

## Highlights

- Forgot a command? Ask `hint` directly.
- Write your intent in plain language and get runnable commands.
- Includes environment context (`GOOS`, distro, shell, current directory).
- Built-in safety check for dangerous commands.

## Quick Start

```bash
go mod tidy
go build ./cmd/hint
./hint -init
./hint "check fail2ban sshd ban status"
```

## Screenshot

![Hintly screenshot](../../assets/image.png)
