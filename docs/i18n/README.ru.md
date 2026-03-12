<p align="center">
  <img src="../../assets/logo.png" width="180" alt="Hintly logo">
</p>

<h1 align="center">Hintly</h1>

<p align="center">AI-помощник для терминала, который превращает естественный язык в исполняемые команды</p>

<p align="center">
  <a href="../../README.md">简体中文</a> | <a href="README.en.md">English</a> | <a href="README.ja.md">日本語</a> | <a href="README.ko.md">한국어</a> | <a href="README.es.md">Español</a> | <a href="README.fr.md">Français</a> | <a href="README.de.md">Deutsch</a> | <strong>Русский</strong> | <a href="README.pt-br.md">Português (BR)</a>
</p>

## Основные возможности

- Забыли команду? Просто спросите `hint`.
- Преобразует запрос на естественном языке в готовую команду.
- Автоматически добавляет контекст (`GOOS`, дистрибутив, shell, текущий каталог).
- Для опасных команд требуется ручное подтверждение.

## Быстрый старт

```bash
go mod tidy
go build ./cmd/hint
./hint -init
./hint "проверить статус блокировки sshd в fail2ban"
```

## Скриншот

![Hintly screenshot](../../assets/image.png)
