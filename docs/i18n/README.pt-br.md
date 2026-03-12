<p align="center">
  <img src="../../assets/logo.png" width="180" alt="Hintly logo">
</p>

<h1 align="center">Hintly</h1>

<p align="center">Assistente de terminal com IA que transforma linguagem natural em comandos executáveis</p>

<p align="center">
  <a href="../../README.md">简体中文</a> | <a href="README.en.md">English</a> | <a href="README.ja.md">日本語</a> | <a href="README.ko.md">한국어</a> | <a href="README.es.md">Español</a> | <a href="README.fr.md">Français</a> | <a href="README.de.md">Deutsch</a> | <a href="README.ru.md">Русский</a> | <strong>Português (BR)</strong>
</p>

## Destaques

- Esqueceu um comando? Pergunte direto ao `hint`.
- Converte intenção em linguagem natural para comandos prontos para executar.
- Injeta contexto do ambiente (`GOOS`, distro, shell e diretório atual).
- Comandos perigosos exigem confirmação manual.

## Início rápido

```bash
go mod tidy
go build ./cmd/hint
./hint -init
./hint "verificar status de banimento sshd no fail2ban"
```

## Captura de tela

![Hintly screenshot](../../assets/image.png)
