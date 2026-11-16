---
name: telegram
description: Send messages or files to a user via the Telegram Bot API. Use this skill when the user asks to send a Telegram notification, alert, status update, summary, or file/report to their Telegram chat.
---

# Telegram Sender

Send messages or files via [send.py](./scripts/send.py) (`python-telegram-bot` via `uv`).

## Configuration

Checked in order:

1. Env vars: `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`
2. Git-ignored `<skill-dir>/settings.json` (`{"bot_token": "...", "chat_id": "..."}`, see [settings.example.json](./settings.example.json))

**If unconfigured**: `send.py` raises an error and exits `1` explaining what is missing. Ask the user for `bot_token` (`@BotFather`) and `chat_id` (from `https://api.telegram.org/bot<TOKEN>/getUpdates`), save to `<skill-dir>/settings.json` with `chmod 600`, and verify `git check-ignore -q <skill-dir>/settings.json`. Never echo `bot_token`.

## Usage & Options

```bash
# Plain text or stdin
mise x uv -- uv run --script <skill-dir>/scripts/send.py -m "Build finished: all tests passed."
cat <<'EOF' | mise x uv -- uv run --script <skill-dir>/scripts/send.py -p HTML
<b>Deploy complete</b>: <code>v1.2.3</code>
EOF

# File attachment (>4096 chars, logs, or reports)
mise x uv -- uv run --script <skill-dir>/scripts/send.py -f ./report.md -c "Verification report"
```

- `-m, --message <text>` (max 4096 chars) | `-f, --file <path>` | `-c, --caption <text>` (max 1024 chars)
- `-p, --parse-mode <HTML|Markdown|MarkdownV2>` (prefer `HTML`) | `-s, --silent` | `-r, --reply-to <id>` | `-t, --chat-id <id>`

## Session / Express Notification Format

When a main agent sends a task or `/express` completion summary, pipe HTML via `stdin` (`cat <<'EOF'`, never URL-encode `%0A` in `-m`):

```bash
cat <<'EOF' | mise x uv -- uv run --script ~/.gemini/config/plugins/opus/skills/telegram/scripts/send.py -p HTML
<b>✅ Issue #<ID> Merged — <Title></b>
• <b>PR:</b> <a href="<PR_URL>">#<PR_NUMBER></a> (<code><short-sha></code> on <code>main</code>)
• <b>Changes:</b> <1-3 concise points of what changed>
• <b>Verified:</b> <AC1..ACn + test/lint checks passed>
EOF
```

Escape `<`, `>`, and `&` as `&lt;`, `&gt;`, `&amp;` inside dynamic text when using `-p HTML`.

## Rules

1. **No secret leaks**: Never send or print tokens, keys, or `.env` contents.
2. **Verify delivery**: Confirm exit code `0` and `"ok":true` in JSON output. Surface the error to the user if delivery fails.
