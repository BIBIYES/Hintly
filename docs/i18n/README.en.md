<p align="center">
  <img src="../../assets/logo.png" width="180" alt="Hintly logo">
</p>

<h1 align="center">Hintly</h1>

<p align="center">An AI terminal assistant that turns natural language into executable commands</p>

<p align="center">
  <a href="../../README.md">简体中文</a> | <strong>English</strong>
</p>

## Docs

- Chinese (main): `README.md`
- English: `docs/i18n/README.en.md`

## Highlights

- One-shot command mode: `hint "your goal"`.
- Agent conversation mode: run `hint` without prompt to enter iterative execution mode.
- Includes environment context (`GOOS`, distro, shell, current directory).
- Built-in safety check for dangerous commands.

## Quick Start

```bash
go mod tidy
go build ./cmd/hint
./hint -init
```

One-shot mode:

```bash
./hint "check fail2ban sshd ban status"
```

Agent mode:

```bash
./hint
```

## Linux Install/Update Script

Install or update to latest:

```bash
curl -fsSL https://raw.githubusercontent.com/BIBIYES/Hintly/main/scripts/install-linux.sh | bash
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/BIBIYES/Hintly/main/scripts/install-linux.sh | VERSION=v1.0.3 bash
```

Install to user directory (no sudo):

```bash
curl -fsSL https://raw.githubusercontent.com/BIBIYES/Hintly/main/scripts/install-linux.sh | INSTALL_DIR="$HOME/.local/bin" bash
```

## One-shot Mode Keys

- `Enter`: execute current command.
- `r`: retry with previous unsatisfied command context.
- `e`: edit command and execute with `Enter`.
- `Esc` / `Ctrl+C`: cancel and exit.

## Agent Mode Notes

- The agent iterates through think -> execute -> observe until done or max steps.
- UI output shows thought, command, and command output on each step.
- Type `exit` or `quit` to leave agent mode.

## Screenshot

![Hintly screenshot](../../assets/image.png)
