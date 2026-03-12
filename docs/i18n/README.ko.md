<p align="center">
  <img src="../../assets/logo.png" width="180" alt="Hintly logo">
</p>

<h1 align="center">Hintly</h1>

<p align="center">자연어를 실행 가능한 명령어로 바꿔주는 AI 터미널 도우미</p>

<p align="center">
  <a href="../../README.md">简体中文</a> | <a href="README.en.md">English</a> | <a href="README.ja.md">日本語</a> | <strong>한국어</strong> | <a href="README.es.md">Español</a> | <a href="README.fr.md">Français</a> | <a href="README.de.md">Deutsch</a> | <a href="README.ru.md">Русский</a> | <a href="README.pt-br.md">Português (BR)</a>
</p>

## 주요 기능

- 명령어가 기억나지 않으면 `hint` 에 바로 질문하세요.
- 자연어 요구사항을 실행 가능한 명령어로 변환합니다.
- `GOOS`, 배포판, Shell, 현재 디렉터리를 자동으로 반영합니다.
- 위험 명령어는 실행 전에 수동 확인이 필요합니다.

## 빠른 시작

```bash
go mod tidy
go build ./cmd/hint
./hint -init
./hint "fail2ban sshd 차단 상태 확인"
```

## 스크린샷

![Hintly screenshot](../../assets/image.png)
