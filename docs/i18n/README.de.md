<p align="center">
  <img src="../../assets/logo.png" width="180" alt="Hintly logo">
</p>

<h1 align="center">Hintly</h1>

<p align="center">Ein KI-Terminalassistent, der natürliche Sprache in ausführbare Befehle umwandelt</p>

<p align="center">
  <a href="../../README.md">简体中文</a> | <a href="README.en.md">English</a> | <a href="README.ja.md">日本語</a> | <a href="README.ko.md">한국어</a> | <a href="README.es.md">Español</a> | <a href="README.fr.md">Français</a> | <strong>Deutsch</strong> | <a href="README.ru.md">Русский</a> | <a href="README.pt-br.md">Português (BR)</a>
</p>

## Highlights

- Befehl vergessen? Frage direkt `hint`.
- Wandelt natürliche Sprache in sofort ausführbare Shell-Befehle um.
- Fügt automatisch Kontext hinzu (`GOOS`, Distribution, Shell, aktuelles Verzeichnis).
- Gefährliche Befehle erfordern eine manuelle Bestätigung.

## Schnellstart

```bash
go mod tidy
go build ./cmd/hint
./hint -init
./hint "fail2ban sshd sperrstatus prüfen"
```

## Screenshot

![Hintly screenshot](../../assets/image.png)
