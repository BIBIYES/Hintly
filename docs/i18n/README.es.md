<p align="center">
  <img src="../../assets/logo.png" width="180" alt="Hintly logo">
</p>

<h1 align="center">Hintly</h1>

<p align="center">Asistente de terminal con IA que convierte lenguaje natural en comandos ejecutables</p>

<p align="center">
  <a href="../../README.md">简体中文</a> | <a href="README.en.md">English</a> | <a href="README.ja.md">日本語</a> | <a href="README.ko.md">한국어</a> | <strong>Español</strong> | <a href="README.fr.md">Français</a> | <a href="README.de.md">Deutsch</a> | <a href="README.ru.md">Русский</a> | <a href="README.pt-br.md">Português (BR)</a>
</p>

## Puntos clave

- ¿Olvidaste un comando? Pregunta directamente a `hint`.
- Convierte tu intención en lenguaje natural en comandos listos para ejecutar.
- Inyecta contexto del entorno (`GOOS`, distro, shell, directorio actual).
- Incluye confirmación manual para comandos peligrosos.

## Inicio rápido

```bash
go mod tidy
go build ./cmd/hint
./hint -init
./hint "ver estado de baneo sshd en fail2ban"
```

## Captura

![Hintly screenshot](../../assets/image.png)
