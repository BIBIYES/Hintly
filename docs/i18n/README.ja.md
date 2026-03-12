<p align="center">
  <img src="../../assets/logo.png" width="180" alt="Hintly logo">
</p>

<h1 align="center">Hintly</h1>

<p align="center">自然言語を実行可能なコマンドに変換する AI ターミナルアシスタント</p>

<p align="center">
  <a href="../../README.md">简体中文</a> | <a href="README.en.md">English</a> | <strong>日本語</strong> | <a href="README.ko.md">한국어</a> | <a href="README.es.md">Español</a> | <a href="README.fr.md">Français</a> | <a href="README.de.md">Deutsch</a> | <a href="README.ru.md">Русский</a> | <a href="README.pt-br.md">Português (BR)</a>
</p>

## 特長

- コマンドを忘れても `hint` に直接質問できます。
- 自然言語の要望をそのまま実行可能なコマンドへ変換します。
- `GOOS`、ディストリビューション、Shell、作業ディレクトリを自動注入します。
- 危険コマンドは実行前に手動確認が必要です。

## クイックスタート

```bash
go mod tidy
go build ./cmd/hint
./hint -init
./hint "fail2ban の sshd BAN 状態を確認"
```

## スクリーンショット

![Hintly screenshot](../../assets/image.png)
